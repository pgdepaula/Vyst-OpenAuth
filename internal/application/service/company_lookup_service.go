package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

var (
	// ErrNoProvidersAvailable is returned when all configured external providers fail.
	ErrNoProvidersAvailable = errors.New("no company data providers available")

	// ErrQueryTooShort is returned when a text search query is too short.
	ErrQueryTooShort = errors.New("search query must be at least 3 characters")
)

// CompanyLookupService orchestrates company data retrieval.
// It implements provider fallback, caching, and rate-limiting patterns.
type CompanyLookupService struct {
	repo        company.CompanyInfoRepository
	providers   []ports.CompanyDataPort
	logger      ports.Logger
	publisher   event.Publisher
	metrics     ports.MetricsService
	rateLimiter ports.RateLimiter
}

// NewCompanyLookupService creates a new instance of CompanyLookupService.
func NewCompanyLookupService(
	repo company.CompanyInfoRepository,
	providers []ports.CompanyDataPort,
	logger ports.Logger,
	publisher event.Publisher,
	metrics ports.MetricsService,
	rateLimiter ports.RateLimiter,
) *CompanyLookupService {
	return &CompanyLookupService{
		repo:        repo,
		providers:   providers,
		logger:      logger,
		publisher:   publisher,
		metrics:     metrics,
		rateLimiter: rateLimiter,
	}
}

// GetByCNPJ retrieves company information by CNPJ.
// It tries to find the data in the cache first. If not found or expired, it falls back to external providers.
func (s *CompanyLookupService) GetByCNPJ(ctx context.Context, tenantID, cnpj string) (*company.CompanyInfo, error) {
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordDuration("company_lookup_getByCNPJ", time.Since(startTime).Seconds(), nil)
		}
	}()

	s.logger.WithContext(ctx).Debug("looking up company by CNPJ", "cnpj", cnpj)

	normalizedCNPJ := company.NormalizeCNPJ(cnpj)
	if !company.ValidateCNPJ(normalizedCNPJ) {
		return nil, company.ErrCNPJInvalid
	}

	if err := s.enforceLookupRateLimit(ctx, tenantID); err != nil {
		return nil, err
	}

	cachedInfo := s.getFreshCachedCompanyInfo(ctx, normalizedCNPJ)
	if cachedInfo != nil {
		return cachedInfo, nil
	}

	info, successfulProvider, err := s.fetchCompanyInfoFromProviders(ctx, normalizedCNPJ)
	if err != nil {
		return nil, err
	}

	s.saveCompanyInfoCache(ctx, normalizedCNPJ, info)

	// 4. Publish Domain Event
	s.publishCompanyInfoFetched(ctx, normalizedCNPJ, successfulProvider, true)

	return info, nil
}

func (s *CompanyLookupService) enforceLookupRateLimit(ctx context.Context, tenantID string) error {
	if s.rateLimiter == nil {
		return nil
	}
	key := "company_lookup_api:" + tenantID
	if tenantID == "" {
		key = "company_lookup_api:anonymous"
	}
	allowed, err := s.rateLimiter.Allow(ctx, key)
	if err != nil || !allowed {
		s.logger.WithContext(ctx).Warn("rate limit exceeded for company lookup", "tenant_id", tenantID)
		return errors.New("rate limit exceeded")
	}
	return nil
}

func (s *CompanyLookupService) getFreshCachedCompanyInfo(ctx context.Context, normalizedCNPJ string) *company.CompanyInfo {
	if s.repo == nil {
		return nil
	}

	cachedInfo, err := s.repo.GetByCNPJ(ctx, normalizedCNPJ)
	if err == nil && cachedInfo != nil {
		if time.Since(cachedInfo.LastFetchedAt) <= 7*24*time.Hour {
			s.logger.WithContext(ctx).Debug("company info found in cache", "cnpj", normalizedCNPJ)
			return cachedInfo
		}
		s.logger.WithContext(ctx).Debug("company info found in cache but expired, fetching new", "cnpj", normalizedCNPJ)
		return nil
	}
	if err != nil && !errors.Is(err, company.ErrCompanyInfoNotFound) {
		s.logger.WithContext(ctx).Warn("failed to fetch company info from cache", "error", err, "cnpj", normalizedCNPJ)
	}
	return nil
}

func (s *CompanyLookupService) fetchCompanyInfoFromProviders(ctx context.Context, normalizedCNPJ string) (*company.CompanyInfo, string, error) {
	var lastErr error
	for _, provider := range s.providers {
		s.logger.WithContext(ctx).Debug("trying provider", "provider", provider.Name(), "cnpj", normalizedCNPJ)

		info, err := provider.GetByCNPJ(ctx, normalizedCNPJ)
		if err == nil && info != nil {
			return info, provider.Name(), nil
		}

		s.logger.WithContext(ctx).Warn("provider failed", "provider", provider.Name(), "cnpj", normalizedCNPJ, "error", err)
		lastErr = err
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", ErrNoProvidersAvailable
}

func (s *CompanyLookupService) saveCompanyInfoCache(ctx context.Context, normalizedCNPJ string, info *company.CompanyInfo) {
	if s.repo == nil {
		return
	}
	if err := s.repo.Save(ctx, info); err != nil {
		s.logger.WithContext(ctx).Error("failed to save company info to cache", "error", err, "cnpj", normalizedCNPJ)
	}
}

// SearchByName performs a text search for companies.
// It implements debounce/caching concepts by returning directly from the external providers
// (it's hard to reliably cache free-text searches locally unless doing a local fuzzy search over all cached).
// We might just use DB search for already known companies and maybe one provider for external.
func (s *CompanyLookupService) SearchByName(ctx context.Context, tenantID, query string, limit int) ([]*company.CompanyInfo, error) {
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordDuration("company_lookup_searchByName", time.Since(startTime).Seconds(), nil)
		}
	}()

	if len(query) < 3 {
		return nil, ErrQueryTooShort
	}

	if s.rateLimiter != nil {
		key := "company_search_api:" + tenantID
		if tenantID == "" {
			key = "company_search_api:anonymous"
		}
		allowed, err := s.rateLimiter.Allow(ctx, key)
		if err != nil || !allowed {
			s.logger.WithContext(ctx).Warn("rate limit exceeded for company search", "tenant_id", tenantID)
			return nil, errors.New("rate limit exceeded")
		}
	}

	// First try searching from our local cached repository for speed & reducing provider limits
	if s.repo != nil {
		localResults, err := s.repo.SearchByName(ctx, query, limit)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to search company info locally", "error", err)
		}

		// If we have local results and they are sufficient, return them
		if len(localResults) > 0 {
			return localResults, nil
		}
	}

	// Fallback to external API for search
	var results []*company.CompanyInfo
	var lastErr error

	for _, provider := range s.providers {
		pResults, pErr := provider.SearchByName(ctx, query, limit)
		if pErr == nil {
			results = append(results, pResults...)
			break // if one provider succeeds, we stop (or we could aggregate). Let's stop.
		} else if errors.Is(pErr, company.ErrSearchNotSupported) {
			s.logger.WithContext(ctx).Debug("provider does not support search", "provider", provider.Name())
			continue // try next provider
		}
		s.logger.WithContext(ctx).Warn("provider search failed", "provider", provider.Name(), "error", pErr)
		lastErr = pErr
	}

	if results == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, ErrNoProvidersAvailable
	}

	return results, nil
}

// publishCompanyInfoFetched publishes the event.
func (s *CompanyLookupService) publishCompanyInfoFetched(ctx context.Context, cnpj, provider string, found bool) {
	// Attempt to extract UserID from context if available in the real implementation
	// For now, leave it empty or fetch from context

	payload := company.CompanyInfoFetchedPayload{
		CNPJ:     cnpj,
		Provider: provider,
		Found:    found,
	}

	evt := event.Event{
		ID:        uuid.New().String(),
		Type:      event.CompanyInfoFetched,
		Source:    "identity-api",
		Timestamp: time.Now(),
		Payload:   payload,
	}

	if err := s.publisher.Publish(ctx, evt); err != nil {
		s.logger.WithContext(ctx).Error("failed to publish company info fetched event", "error", err)
	}
}

// Lookup determines whether the input is a CNPJ or a name and calls the appropriate method.
func (s *CompanyLookupService) Lookup(ctx context.Context, tenantID, query string, limit int) ([]*company.CompanyInfo, error) {
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordDuration("company_lookup_lookup", time.Since(startTime).Seconds(), nil)
		}
	}()

	s.logger.WithContext(ctx).Debug("looking up company", "query", query)

	if len(query) < 3 {
		return nil, ErrQueryTooShort
	}

	// Clean input to check if it's potentially a CNPJ
	normalizedQuery := company.NormalizeCNPJ(query)
	if len(normalizedQuery) == 14 && company.ValidateCNPJ(normalizedQuery) {
		// Valid CNPJ, do an exact lookup
		info, err := s.GetByCNPJ(ctx, tenantID, normalizedQuery)
		if err != nil {
			if errors.Is(err, company.ErrCompanyInfoNotFound) || errors.Is(err, ErrNoProvidersAvailable) {
				return []*company.CompanyInfo{}, nil
			}
			return nil, err
		}
		if info != nil {
			return []*company.CompanyInfo{info}, nil
		}
		return []*company.CompanyInfo{}, nil
	}

	// Execute search by name
	return s.SearchByName(ctx, tenantID, query, limit)
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
)

type CompanySyncService struct {
	companyRepo      company.Repository
	companyLookupSvc *CompanyLookupService
	eventBus         event.Bus
	logger           ports.Logger
}

func NewCompanySyncService(
	companyRepo company.Repository,
	companyLookupSvc *CompanyLookupService,
	eventBus event.Bus,
	logger ports.Logger,
) *CompanySyncService {
	return &CompanySyncService{
		companyRepo:      companyRepo,
		companyLookupSvc: companyLookupSvc,
		eventBus:         eventBus,
		logger:           logger,
	}
}

// SyncAll iterates over all active companies and updates their status based on external data.
// It processes in batches to avoid high memory usage.
func (s *CompanySyncService) SyncAll(ctx context.Context) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Starting company data synchronization job")

	limit := 50
	offset := 0
	totalProcessed := 0
	totalUpdated := 0
	totalErrors := 0

	for {
		companies, err := s.companyRepo.ListAllActive(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to list companies: %w", err)
		}

		if len(companies) == 0 {
			break
		}

		for _, comp := range companies {
			updated, err := s.syncCompany(ctx, comp)
			if err != nil {
				logger.Error("Failed to sync company", "company_id", comp.ID, "cnpj", comp.CNPJ, "error", err)
				totalErrors++
				continue
			}
			if updated {
				totalUpdated++
			}
			totalProcessed++
		}

		offset += limit

		// Avoid indefinite loops if pagination is broken, though offsets should work
		if len(companies) < limit {
			break
		}
	}

	logger.Info("Company synchronization finished",
		"processed", totalProcessed,
		"updated", totalUpdated,
		"errors", totalErrors,
	)
	return nil
}

func (s *CompanySyncService) syncCompany(ctx context.Context, comp *company.Company) (bool, error) {
	if s.companyLookupSvc == nil {
		return false, nil
	}

	details, err := s.companyLookupSvc.GetByCNPJ(ctx, comp.TenantID, comp.CNPJ)
	if err != nil {
		return false, err // Transitional error, retry next time
	}
	if details == nil {
		// Not found in external source? Maybe log warning but don't suspend immediately?
		// Or maybe it means it's really invalid?
		// Let's assume strict: if not found, we don't change status automatically yet, too risky.
		return false, nil
	}

	// Map external status to domain status
	// BrasilAPI: ATIVA, BAIXADA, INAPTA, SUSPENSA, NULA
	var newStatus company.CompanyStatus
	switch details.Situacao {
	case "ATIVA":
		newStatus = company.StatusActive
	case "BAIXADA", "INAPTA", "NULA":
		newStatus = company.StatusSuspended // We treat invalid as suspended
	case "SUSPENSA":
		newStatus = company.StatusSuspended
	default:
		// Unknown status, assume active or keep current?
		// Keep current is safer
		return false, nil
	}

	if comp.Status != newStatus {
		oldStatus := comp.Status
		comp.Status = newStatus
		comp.UpdatedAt = time.Now()

		if err := s.companyRepo.Update(ctx, comp); err != nil {
			return false, fmt.Errorf("failed to update company status: %w", err)
		}

		// Emit event
		eventType := event.CompanyUpdated
		if newStatus == company.StatusSuspended {
			eventType = event.CompanySuspended
		} else if newStatus == company.StatusActive {
			eventType = event.CompanyActivated
		}

		go func() {
			if err := s.eventBus.Publish(ctx, event.Event{
				ID:            uuid.New().String(),
				Type:          eventType,
				AggregateType: "company",
				AggregateID:   comp.ID,
				Source:        "CompanySyncService",
				Timestamp:     time.Now(),
				TenantID:      comp.TenantID,
				Payload: map[string]interface{}{
					"company_id": comp.ID,
					"old_status": oldStatus,
					"new_status": newStatus,
					"reason":     "External data sync",
				},
			}); err != nil {
				s.logger.WithContext(ctx).Warn("Failed to publish company sync event", "company_id", comp.ID, "error", err)
			}
		}()

		s.logger.Info("Company status updated via sync",
			"company_id", comp.ID,
			"old_status", oldStatus,
			"new_status", newStatus,
		)
		return true, nil
	}

	return false, nil
}

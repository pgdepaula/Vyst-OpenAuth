package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/redis/go-redis/v9"
)

// CachedCompanyUserRepository is a decorator that adds caching to a CompanyUserRepository.
type CachedCompanyUserRepository struct {
	delegate company.CompanyUserRepository
	client   *redis.Client
	ttl      time.Duration
	logger   *slog.Logger
}

// NewCachedCompanyUserRepository creates a new CachedCompanyUserRepository.
// ttl defaults to 5 minutes if not provided (or 0).
func NewCachedCompanyUserRepository(delegate company.CompanyUserRepository, client *redis.Client, ttl time.Duration, logger *slog.Logger) *CachedCompanyUserRepository {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &CachedCompanyUserRepository{
		delegate: delegate,
		client:   client,
		ttl:      ttl,
		logger:   logger,
	}
}

func (r *CachedCompanyUserRepository) AddUser(ctx context.Context, cu *company.CompanyUser) error {
	if err := r.delegate.AddUser(ctx, cu); err != nil {
		return err
	}
	r.invalidateUserCache(ctx, cu.UserID)
	return nil
}

func (r *CachedCompanyUserRepository) RemoveUser(ctx context.Context, companyID, userID string) error {
	if err := r.delegate.RemoveUser(ctx, companyID, userID); err != nil {
		return err
	}
	r.invalidateUserCache(ctx, userID)
	r.invalidateRoleCache(ctx, companyID, userID)
	return nil
}

func (r *CachedCompanyUserRepository) GetUserRole(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
	key := fmt.Sprintf("company:user_role:%s:%s", companyID, userID)

	// 1. Try Cache
	val, err := r.client.Get(ctx, key).Result()
	if err == nil {
		if val == "NOT_FOUND" {
			return "", company.ErrUserNotMember
		}
		return company.CompanyRole(val), nil
	} else if err != redis.Nil {
		r.logger.Error("Redis cache error", "key", key, "error", err)
	}

	// 2. Delegate
	role, err := r.delegate.GetUserRole(ctx, companyID, userID)
	if err != nil {
		if err == company.ErrUserNotMember {
			// Cache negative result for shorter duration (one minute)
			_ = r.client.Set(ctx, key, "NOT_FOUND", time.Minute).Err()
			return "", err
		}
		return "", err
	}

	// 3. Update Cache
	if err := r.client.Set(ctx, key, string(role), r.ttl).Err(); err != nil {
		r.logger.Error("Redis set error", "key", key, "error", err)
	}

	return role, nil
}

func (r *CachedCompanyUserRepository) GetCompaniesForUser(ctx context.Context, userID string) ([]*company.CompanyUser, error) {
	key := fmt.Sprintf("company:user_companies:%s", userID)

	// 1. Try Cache
	val, err := r.client.Get(ctx, key).Result()
	if err == nil {
		var cached []*company.CompanyUser
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			return cached, nil
		}
	} else if err != redis.Nil {
		r.logger.Error("Redis cache error", "key", key, "error", err)
	}

	// 2. Delegate
	companies, err := r.delegate.GetCompaniesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 3. Update Cache
	encoded, err := json.Marshal(companies)
	if err != nil {
		r.logger.Error("Failed to marshal companies for cache", "error", err)
		return companies, nil
	}

	if err := r.client.Set(ctx, key, encoded, r.ttl).Err(); err != nil {
		r.logger.Error("Redis set error", "key", key, "error", err)
	}

	return companies, nil
}

func (r *CachedCompanyUserRepository) GetUsersForCompany(ctx context.Context, companyID string) ([]*company.CompanyUser, error) {
	// Not caching this list for now as it can be large and change frequently
	return r.delegate.GetUsersForCompany(ctx, companyID)
}

func (r *CachedCompanyUserRepository) UpdateUserRole(ctx context.Context, companyID, userID string, role company.CompanyRole) error {
	if err := r.delegate.UpdateUserRole(ctx, companyID, userID, role); err != nil {
		return err
	}
	r.invalidateUserCache(ctx, userID)
	r.invalidateRoleCache(ctx, companyID, userID)
	return nil
}

func (r *CachedCompanyUserRepository) UpdateUserStatus(ctx context.Context, companyID, userID string, status company.MembershipStatus) error {
	if err := r.delegate.UpdateUserStatus(ctx, companyID, userID, status); err != nil {
		return err
	}
	r.invalidateUserCache(ctx, userID)
	// Role cache might not change, but status affects effective access so maybe clear it just in case
	r.invalidateRoleCache(ctx, companyID, userID)
	return nil
}

func (r *CachedCompanyUserRepository) invalidateUserCache(ctx context.Context, userID string) {
	key := fmt.Sprintf("company:user_companies:%s", userID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Error("Failed to invalidate user cache", "key", key, "error", err)
	}
}

func (r *CachedCompanyUserRepository) invalidateRoleCache(ctx context.Context, companyID, userID string) {
	key := fmt.Sprintf("company:user_role:%s:%s", companyID, userID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Error("Failed to invalidate role cache", "key", key, "error", err)
	}
}

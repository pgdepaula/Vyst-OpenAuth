// Package service contains application services that orchestrate domain logic.
// CompanyService handles all company-related operations.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// CreateCompanyRequest contains the data to create a new company.
type CreateCompanyRequest struct {
	CNPJ               string          `json:"cnpj"`
	RazaoSocial        string          `json:"razao_social"`
	NomeFantasia       string          `json:"nome_fantasia"`
	Endereco           company.Address `json:"endereco"`
	RepresentanteLegal string          `json:"representante_legal"`
}

// AddUserToCompanyRequest contains the data to add a user to a company.
type AddUserToCompanyRequest struct {
	CompanyID string              `json:"company_id"`
	UserID    string              `json:"user_id"`
	Role      company.CompanyRole `json:"role"`
	InvitedBy string              `json:"invited_by"`
}

// CompanyWithRole contains a company and the user's role in it.
type CompanyWithRole struct {
	Company *company.Company    `json:"company"`
	Role    company.CompanyRole `json:"role"`
	Status  string              `json:"status"`
}

// CompanyService handles company-related operations.
// It orchestrates company creation, user management, and company context switching.
type CompanyService struct {
	tm               ports.TransactionManager
	companyRepo      company.Repository
	companyUserRepo  company.CompanyUserRepository
	userRepo         user.Repository
	eventBus         event.Bus
	outboxPub        ports.OutboxPublisher
	companyLookupSvc *CompanyLookupService
	logger           ports.Logger
}

// NewCompanyService creates a new CompanyService.
func NewCompanyService(
	tm ports.TransactionManager,
	companyRepo company.Repository,
	companyUserRepo company.CompanyUserRepository,
	userRepo user.Repository,
	eventBus event.Bus,
	outboxPub ports.OutboxPublisher,
	companyLookupSvc *CompanyLookupService,
	logger ports.Logger,
) *CompanyService {
	return &CompanyService{
		tm:               tm,
		companyRepo:      companyRepo,
		companyUserRepo:  companyUserRepo,
		userRepo:         userRepo,
		eventBus:         eventBus,
		outboxPub:        outboxPub,
		companyLookupSvc: companyLookupSvc,
		logger:           logger,
	}
}

// CreateCompany creates a new company and adds the creator as admin.
// Returns the created company or an error if validation fails.
func (s *CompanyService) CreateCompany(
	ctx context.Context,
	tenantID string,
	creatorUserID string,
	req CreateCompanyRequest,
) (*company.Company, error) {
	logger := s.logger.WithContext(ctx)
	logger.Info("Creating company",
		"tenant_id", tenantID,
		"creator", creatorUserID,
		"cnpj", company.MaskCNPJ(req.CNPJ),
	)

	normalizedCNPJ, err := s.validateCreateCompanyCNPJ(ctx, logger, tenantID, req.CNPJ)
	if err != nil {
		return nil, err
	}

	// 3. Check if CNPJ already exists
	existing, err := s.companyRepo.GetByCNPJ(ctx, normalizedCNPJ)
	if err == nil && existing != nil {
		logger.Warn("CNPJ already registered", "cnpj", company.MaskCNPJ(req.CNPJ))
		return nil, company.ErrCNPJTaken
	}
	// If error is not ErrNotFound, it's a real error
	if err != nil && !errors.Is(err, company.ErrNotFound) {
		logger.Error("Failed to check CNPJ existence", "error", err)
		return nil, fmt.Errorf("checking CNPJ: %w", err)
	}

	// 4. Validate that creator user exists
	creatorUser, err := s.userRepo.GetByID(ctx, creatorUserID)
	if err != nil {
		logger.Error("Creator user not found", "user_id", creatorUserID, "error", err)
		return nil, fmt.Errorf("creator user not found: %w", err)
	}

	// 5. Create company entity
	companyID := uuid.New().String()
	now := time.Now()

	newCompany := &company.Company{
		ID:                 companyID,
		TenantID:           tenantID,
		CNPJ:               normalizedCNPJ,
		RazaoSocial:        req.RazaoSocial,
		NomeFantasia:       req.NomeFantasia,
		Endereco:           req.Endereco,
		RepresentanteLegal: req.RepresentanteLegal,
		Status:             company.StatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// 6. Execute everything in a single transaction
	err = s.tm.RunInTransaction(ctx, func(ctx context.Context) error {
		// 6a. Create company
		if err := s.companyRepo.Create(ctx, newCompany); err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}

		// 6b. Add creator as admin
		cu, err := company.NewCompanyUser(companyID, creatorUserID, company.RoleAdmin, creatorUserID)
		if err != nil {
			return fmt.Errorf("failed to create company user: %w", err)
		}
		if err := s.companyUserRepo.AddUser(ctx, cu); err != nil {
			return fmt.Errorf("failed to add creator as admin: %w", err)
		}

		// 6c. Publish event to outbox
		if err := s.outboxPub.Publish(ctx, event.Event{
			ID:            uuid.New().String(),
			Type:          event.CompanyCreated,
			AggregateType: "company",
			AggregateID:   companyID,
			Source:        EventSource,
			Timestamp:     now,
			TenantID:      tenantID,
			Payload: company.CompanyCreatedPayload{
				CompanyID:   companyID,
				CNPJ:        normalizedCNPJ,
				RazaoSocial: req.RazaoSocial,
				TenantID:    tenantID,
				CreatedBy:   creatorUserID,
			},
		}); err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to create company", "error", err)
		return nil, err
	}

	// 7. Publish to in-memory bus for real-time listeners (fire-and-forget)
	s.publishAsync(ctx, event.Event{
		ID:            uuid.New().String(),
		Type:          event.CompanyCreated,
		AggregateType: "company",
		AggregateID:   companyID,
		Source:        EventSource,
		Timestamp:     now,
		TenantID:      tenantID,
		Payload: company.CompanyCreatedPayload{
			CompanyID:   companyID,
			CNPJ:        normalizedCNPJ,
			RazaoSocial: req.RazaoSocial,
			TenantID:    tenantID,
			CreatedBy:   creatorUserID,
		},
	})

	logger.Info("Company created successfully",
		"company_id", companyID,
		"cnpj", company.MaskCNPJ(normalizedCNPJ),
		"razao_social", req.RazaoSocial,
		"creator", creatorUser.Email,
	)

	return newCompany, nil
}

// GetCompanyByID retrieves a company by its ID.
func (s *CompanyService) GetCompanyByID(ctx context.Context, companyID string) (*company.Company, error) {
	return s.companyRepo.GetByID(ctx, companyID)
}

func (s *CompanyService) validateCreateCompanyCNPJ(ctx context.Context, logger ports.Logger, tenantID, cnpj string) (string, error) {
	normalizedCNPJ := company.NormalizeCNPJ(cnpj)
	if !company.ValidateCNPJ(normalizedCNPJ) {
		logger.Warn("Invalid CNPJ provided", "cnpj", company.MaskCNPJ(cnpj))
		return "", company.ErrCNPJInvalid
	}
	if company.IsBlacklistedCNPJ(normalizedCNPJ) {
		logger.Warn("Blacklisted CNPJ provided", "cnpj", company.MaskCNPJ(cnpj))
		return "", company.ErrCNPJInvalid
	}
	return normalizedCNPJ, s.validateExternalCNPJStatus(ctx, logger, tenantID, normalizedCNPJ, cnpj)
}

func (s *CompanyService) validateExternalCNPJStatus(ctx context.Context, logger ports.Logger, tenantID, normalizedCNPJ, rawCNPJ string) error {
	if s.companyLookupSvc == nil {
		return nil
	}
	details, err := s.companyLookupSvc.GetByCNPJ(ctx, tenantID, normalizedCNPJ)
	if err != nil {
		logger.Warn("Failed to validate CNPJ externally", "error", err)
		return nil
	}
	if details == nil {
		return nil
	}
	switch details.Situacao {
	case "BAIXADA", "NULA", "INAPTA":
		msg := fmt.Sprintf("CNPJ status is %s", details.Situacao)
		logger.Warn(msg, "cnpj", company.MaskCNPJ(rawCNPJ), "status", details.Situacao)
		return fmt.Errorf("%w: %s", company.ErrCNPJInvalid, msg)
	default:
		return nil
	}
}

// GetCompanyByCNPJ retrieves a company by its CNPJ.
func (s *CompanyService) GetCompanyByCNPJ(ctx context.Context, cnpj string) (*company.Company, error) {
	normalizedCNPJ := company.NormalizeCNPJ(cnpj)
	return s.companyRepo.GetByCNPJ(ctx, normalizedCNPJ)
}

// GetCompaniesForUser returns all companies a user belongs to with their roles.
func (s *CompanyService) GetCompaniesForUser(ctx context.Context, userID string) ([]*CompanyWithRole, error) {
	logger := s.logger.WithContext(ctx)
	logger.Debug("Getting companies for user", "user_id", userID)

	memberships, err := s.companyUserRepo.GetCompaniesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting user memberships: %w", err)
	}

	result := make([]*CompanyWithRole, 0, len(memberships))
	for _, m := range memberships {
		c, err := s.companyRepo.GetByID(ctx, m.CompanyID)
		if err != nil {
			logger.Warn("Company not found for membership", "company_id", m.CompanyID)
			continue
		}
		result = append(result, &CompanyWithRole{
			Company: c,
			Role:    m.Role,
			Status:  string(m.Status),
		})
	}

	logger.Debug("Found companies for user", "user_id", userID, "count", len(result))
	return result, nil
}

// AddUserToCompany adds a user to a company with the specified role.
func (s *CompanyService) AddUserToCompany(ctx context.Context, req AddUserToCompanyRequest) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Adding user to company",
		"company_id", req.CompanyID,
		"user_id", req.UserID,
		"role", req.Role,
		"invited_by", req.InvitedBy,
	)

	// Validate role
	if !req.Role.IsValid() {
		return company.ErrInvalidRole
	}

	// Verify company exists
	comp, err := s.companyRepo.GetByID(ctx, req.CompanyID)
	if err != nil {
		return fmt.Errorf("company not found: %w", err)
	}

	// Verify user exists
	_, err = s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user is already a member
	existingRole, err := s.companyUserRepo.GetUserRole(ctx, req.CompanyID, req.UserID)
	if err == nil && existingRole != "" {
		return company.ErrAlreadyMember
	}

	// Create and add company user
	cu, err := company.NewCompanyUser(req.CompanyID, req.UserID, req.Role, req.InvitedBy)
	if err != nil {
		return err
	}

	if err := s.companyUserRepo.AddUser(ctx, cu); err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	// Publish event
	s.publishAsync(ctx, event.Event{
		ID:            uuid.New().String(),
		Type:          event.CompanyUserAdded,
		AggregateType: "company",
		AggregateID:   req.CompanyID,
		Source:        EventSource,
		Timestamp:     time.Now(),
		TenantID:      comp.TenantID,
		Payload: company.CompanyUserAddedPayload{
			CompanyID: req.CompanyID,
			UserID:    req.UserID,
			Role:      req.Role.String(),
			InvitedBy: req.InvitedBy,
		},
	})

	logger.Info("User added to company successfully",
		"company_id", req.CompanyID,
		"user_id", req.UserID,
		"role", req.Role,
	)

	return nil
}

// RemoveUserFromCompany removes a user from a company.
func (s *CompanyService) RemoveUserFromCompany(ctx context.Context, companyID, userID, removedBy, reason string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Removing user from company",
		"company_id", companyID,
		"user_id", userID,
		"removed_by", removedBy,
	)

	// Verify company exists
	comp, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("company not found: %w", err)
	}

	// Verify user is a member
	_, err = s.companyUserRepo.GetUserRole(ctx, companyID, userID)
	if err != nil {
		return company.ErrUserNotMember
	}

	// Remove user
	if err := s.companyUserRepo.RemoveUser(ctx, companyID, userID); err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	// Publish event
	s.publishAsync(ctx, event.Event{
		ID:            uuid.New().String(),
		Type:          event.CompanyUserRemoved,
		AggregateType: "company",
		AggregateID:   companyID,
		Source:        EventSource,
		Timestamp:     time.Now(),
		TenantID:      comp.TenantID,
		Payload: company.CompanyUserRemovedPayload{
			CompanyID: companyID,
			UserID:    userID,
			RemovedBy: removedBy,
			Reason:    reason,
		},
	})

	logger.Info("User removed from company successfully",
		"company_id", companyID,
		"user_id", userID,
	)

	return nil
}

// SwitchCompany switches the user's active company context.
// This updates the user's ActiveCompanyID which will be included in subsequent JWTs.
func (s *CompanyService) SwitchCompany(ctx context.Context, userID, companyID string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Switching company context", "user_id", userID, "company_id", companyID)

	// Verify user has access to this company
	role, err := s.companyUserRepo.GetUserRole(ctx, companyID, userID)
	if err != nil {
		logger.Warn("User not a member of company", "user_id", userID, "company_id", companyID)
		return company.ErrUserNotMember
	}

	// Verify membership is active
	memberships, err := s.companyUserRepo.GetCompaniesForUser(ctx, userID)
	if err != nil {
		return err
	}

	var isActive bool
	for _, m := range memberships {
		if m.CompanyID == companyID && m.Status == company.MembershipActive {
			isActive = true
			break
		}
	}
	if !isActive {
		return company.ErrUserNotMember
	}

	// Update user's active company
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	u.ActiveCompanyID = companyID
	u.IdentityType = user.IdentityTypeCompany
	u.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, u); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("Company context switched successfully",
		"user_id", userID,
		"company_id", companyID,
		"role", role,
	)

	return nil
}

// ClearCompanyContext clears the user's active company context.
// This switches the user back to individual (pessoa física) mode.
func (s *CompanyService) ClearCompanyContext(ctx context.Context, userID string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Clearing company context", "user_id", userID)

	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	u.ActiveCompanyID = ""
	u.IdentityType = user.IdentityTypeIndividual
	u.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, u); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("Company context cleared", "user_id", userID)
	return nil
}

// GetUserRoleInCompany returns the user's role in a specific company.
func (s *CompanyService) GetUserRoleInCompany(ctx context.Context, companyID, userID string) (company.CompanyRole, error) {
	return s.companyUserRepo.GetUserRole(ctx, companyID, userID)
}

// UpdateUserRole updates a user's role in a company.
func (s *CompanyService) UpdateUserRole(ctx context.Context, companyID, userID string, newRole company.CompanyRole, updatedBy string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Updating user role",
		"company_id", companyID,
		"user_id", userID,
		"new_role", newRole,
		"updated_by", updatedBy,
	)

	if !newRole.IsValid() {
		return company.ErrInvalidRole
	}

	// Verify user is a member
	currentRole, err := s.companyUserRepo.GetUserRole(ctx, companyID, userID)
	if err != nil {
		return company.ErrUserNotMember
	}

	if currentRole == newRole {
		// No change needed
		return nil
	}

	if err := s.companyUserRepo.UpdateUserRole(ctx, companyID, userID, newRole); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	logger.Info("User role updated",
		"company_id", companyID,
		"user_id", userID,
		"old_role", currentRole,
		"new_role", newRole,
	)

	return nil
}

// SuspendCompany suspends a company, preventing its use for login.
func (s *CompanyService) SuspendCompany(ctx context.Context, companyID, suspendedBy, reason string) error {
	logger := s.logger.WithContext(ctx)
	logger.Warn("Suspending company",
		"company_id", companyID,
		"suspended_by", suspendedBy,
		"reason", reason,
	)

	comp, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return err
	}

	comp.Suspend()

	if err := s.companyRepo.Update(ctx, comp); err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	// Publish event
	s.publishAsync(ctx, event.Event{
		ID:            uuid.New().String(),
		Type:          event.CompanySuspended,
		AggregateType: "company",
		AggregateID:   companyID,
		Source:        EventSource,
		Timestamp:     time.Now(),
		TenantID:      comp.TenantID,
		Payload: company.CompanySuspendedPayload{
			CompanyID:   companyID,
			SuspendedBy: suspendedBy,
			Reason:      reason,
		},
	})

	return nil
}

// ActivateCompany activates a suspended company.
func (s *CompanyService) ActivateCompany(ctx context.Context, companyID, activatedBy string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Activating company",
		"company_id", companyID,
		"activated_by", activatedBy,
	)

	comp, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return err
	}

	comp.Activate()

	if err := s.companyRepo.Update(ctx, comp); err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	// Publish event
	s.publishAsync(ctx, event.Event{
		ID:            uuid.New().String(),
		Type:          event.CompanyActivated,
		AggregateType: "company",
		AggregateID:   companyID,
		Source:        EventSource,
		Timestamp:     time.Now(),
		TenantID:      comp.TenantID,
		Payload: company.CompanyActivatedPayload{
			CompanyID:   companyID,
			ActivatedBy: activatedBy,
		},
	})

	return nil
}

// RequestJoin allows a user to request to join a company.
// The user is added with Pending status.
func (s *CompanyService) RequestJoin(ctx context.Context, companyID, userID string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("User requesting to join company", "company_id", companyID, "user_id", userID)

	// Verify company exists
	_, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("company not found: %w", err)
	}

	// Verify user exists
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if user is already a member (active or pending)
	existingRole, err := s.companyUserRepo.GetUserRole(ctx, companyID, userID)
	if err == nil && existingRole != "" {
		return company.ErrAlreadyMember
	}

	// Create request as Pending Member
	cu, err := company.NewCompanyUser(companyID, userID, company.RoleMember, userID) // Self-invited
	if err != nil {
		return err
	}
	cu.Status = company.MembershipPending

	if err := s.companyUserRepo.AddUser(ctx, cu); err != nil {
		return fmt.Errorf("failed to create join request: %w", err)
	}

	logger.Info("Join request created", "company_id", companyID, "user_id", userID)
	return nil
}

// ApproveMember approves a pending member.
func (s *CompanyService) ApproveMember(ctx context.Context, companyID, adminID, targetUserID string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Approving member", "company_id", companyID, "target_user_id", targetUserID, "admin_id", adminID)

	// 1. Verify Admin permissions
	role, err := s.companyUserRepo.GetUserRole(ctx, companyID, adminID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if role != company.RoleAdmin {
		return errors.New("only admins can approve members")
	}

	// 2. Approve by updating status to Active
	if err := s.companyUserRepo.UpdateUserStatus(ctx, companyID, targetUserID, company.MembershipActive); err != nil {
		return fmt.Errorf("failed to approve member: %w", err)
	}

	logger.Info("Member approved", "company_id", companyID, "target_user_id", targetUserID)
	return nil
}

// RejectMember rejects a pending member (removes them).
func (s *CompanyService) RejectMember(ctx context.Context, companyID, adminID, targetUserID string) error {
	logger := s.logger.WithContext(ctx)
	logger.Info("Rejecting member", "company_id", companyID, "target_user_id", targetUserID, "admin_id", adminID)

	// 1. Verify Admin permissions
	role, err := s.companyUserRepo.GetUserRole(ctx, companyID, adminID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if role != company.RoleAdmin {
		return errors.New("only admins can reject members")
	}

	// 2. Remove user (Rejecting basically deletes the request)
	if err := s.companyUserRepo.RemoveUser(ctx, companyID, targetUserID); err != nil {
		return fmt.Errorf("failed to reject member: %w", err)
	}

	logger.Info("Member rejected", "company_id", companyID, "target_user_id", targetUserID)
	return nil
}

func (s *CompanyService) publishAsync(ctx context.Context, evt event.Event) {
	logger := s.logger.WithContext(ctx)
	go func() {
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			logger.Warn("Failed to publish domain event", "event_type", evt.Type, "error", err)
		}
	}()
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/invitation"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

type InvitationService struct {
	invRepo            invitation.Repository
	userRepo           user.Repository
	companyRepo        company.Repository
	companyUserRepo    company.CompanyUserRepository
	notificationSvc    ports.NotificationService
	invitationDuration time.Duration
	logger             ports.Logger
}

func NewInvitationService(
	invRepo invitation.Repository,
	userRepo user.Repository,
	companyRepo company.Repository,
	companyUserRepo company.CompanyUserRepository,
	notificationSvc ports.NotificationService,
	logger ports.Logger,
) *InvitationService {
	return &InvitationService{
		invRepo:            invRepo,
		userRepo:           userRepo,
		companyRepo:        companyRepo,
		companyUserRepo:    companyUserRepo,
		notificationSvc:    notificationSvc,
		invitationDuration: 7 * 24 * time.Hour, // Default 7 days
		logger:             logger,
	}
}

// InviteUser creates an invitation for a user to join a company.
// invitorID is the ID of the user sending the invitation (must be admin).
func (s *InvitationService) InviteUser(ctx context.Context, invitorID, companyID, email string, role company.CompanyRole) error {
	// 1. Verify invitor has permission (Admin)
	invitorRole, err := s.companyUserRepo.GetUserRole(ctx, companyID, invitorID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if invitorRole != company.RoleAdmin {
		return fmt.Errorf("only admins can invite users")
	}

	// 2. Check if user is already a member
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		// User exists, check if they are already in the company
		_, err := s.companyUserRepo.GetUserRole(ctx, companyID, existingUser.ID)
		if err == nil {
			return fmt.Errorf("user is already a member of this company")
		}
	}

	// 3. Check for existing pending invitation
	existingInv, err := s.invRepo.GetByEmailAndCompany(ctx, email, companyID)
	if err == nil && existingInv.Status == invitation.StatusPending {
		if time.Now().Before(existingInv.ExpiresAt) {
			return fmt.Errorf("pending invitation already exists for this email")
		}
		// If expired, we can create a new one (or update existing, but creating new logic is simpler for now)
	}

	// 4. Create Invitation
	inv := invitation.NewInvitation(companyID, email, role, s.invitationDuration)
	if err := s.invRepo.Create(ctx, inv); err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	// 5. Send Email
	companyName := "Company" // Default
	comp, err := s.companyRepo.GetByID(ctx, companyID)
	if err == nil {
		companyName = comp.RazaoSocial
	}

	subject := fmt.Sprintf("Invitation to join %s", companyName)
	body := fmt.Sprintf("You have been invited to join %s as a %s. Click here to accept: %s/invitations/%s/accept", companyName, role, "http://localhost:8080", inv.Token) // URL should be config

	if err := s.notificationSvc.SendEmail(email, subject, body); err != nil {
		s.logger.Error("Failed to send invitation email", "email", email, "error", err)
		// We don't rollback the invitation, but maybe we should? For now, just log.
	}

	return nil
}

// AcceptInvitation processes the acceptance of an invitation.
func (s *InvitationService) AcceptInvitation(ctx context.Context, token, userID string) error {
	// 1. Get Invitation
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil {
		return invitation.ErrNotFound
	}

	// 2. Validate Invitation
	if err := inv.Accept(); err != nil {
		return err
	}

	// 3. Verify user matches invitation email?
	// Note: Usually we require the user to be logged in. If the email doesn't match the logged-in user,
	// strict security would block it. But often we allow accepting with *any* account or force creation.
	// For Vyst, let's enforce email match if user exists?
	// Or simply add the userID provided to the company.

	// Let's verify the user e-mail matches the invitation e-mail for security.
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if u.Email != inv.Email {
		return fmt.Errorf("invitation email %s does not match user email %s", inv.Email, u.Email)
	}

	// 4. Add User to Company
	// Create CompanyUser domain entity
	// We don't track who invited in the invitation acceptance flow clearly from stored invitation yet?
	// Ah, Invitation entity doesn't have InvitorID.
	// For now, let's put "system" or empty string if allowed.
	// We can update Invitation entity later to store InvitorID if needed.

	cu, err := company.NewCompanyUser(inv.CompanyID, userID, inv.Role, "")
	if err != nil {
		return fmt.Errorf("failed to create company user entity: %w", err)
	}

	err = s.companyUserRepo.AddUser(ctx, cu)
	if err != nil {
		return fmt.Errorf("failed to add user to company: %w", err)
	}

	// 5. Update Invitation Status
	if err := s.invRepo.Update(ctx, inv); err != nil {
		// This is bad if step 4 succeeded. Should be in transaction.
		s.logger.Error("Failed to update invitation status", "invitation_id", inv.ID, "error", err)
		return fmt.Errorf("failed to update invitation status: %w", err)
	}

	return nil
}

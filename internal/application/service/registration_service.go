package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/document"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// RegisterCommand contains the data needed to register a new user with tenant.
type RegisterCommand struct {
	Email      string
	Password   string
	TenantName string
	CPF        string
}

// RegisterResult contains the result of a successful registration.
type RegisterResult struct {
	User   *user.User
	Tenant *tenant.Tenant
}

// RegistrationService handles the complete user registration flow.
// It orchestrates: tenant creation, user creation, password hashing,
// ReBAC tuple assignment, and event publishing - all in a single transaction.
type RegistrationService struct {
	tm          ports.TransactionManager
	userRepo    user.Repository
	tenantRepo  tenant.Repository
	policyRepo  policy.Repository
	hasher      ports.PasswordHasher
	outboxPub   ports.OutboxPublisher
	eventBus    event.Bus
	notifier    ports.NotificationService
	documentSvc *DocumentService
}

const EventSource = "vyst-identity"

// NewRegistrationService creates a new registration service.
func NewRegistrationService(
	tm ports.TransactionManager,
	userRepo user.Repository,
	tenantRepo tenant.Repository,
	policyRepo policy.Repository,
	hasher ports.PasswordHasher,
	outboxPub ports.OutboxPublisher,
	eventBus event.Bus,
	notifier ports.NotificationService,
	documentSvc *DocumentService,
) *RegistrationService {
	return &RegistrationService{
		tm:          tm,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		policyRepo:  policyRepo,
		hasher:      hasher,
		outboxPub:   outboxPub,
		eventBus:    eventBus,
		notifier:    notifier,
		documentSvc: documentSvc,
	}
}

// RegisterWithTenant creates a new tenant and user in a single transaction.
// This ensures atomicity: either everything succeeds or nothing is created.
func (s *RegistrationService) RegisterWithTenant(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
	// 1. Hash password outside transaction (CPU-intensive)
	hashedPassword, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 2. Generate IDs
	tenantID := uuid.New().String()
	userID := uuid.New().String()
	now := time.Now()

	// 3. Create entities
	newTenant := &tenant.Tenant{
		ID:        tenantID,
		Name:      cmd.TenantName,
		Status:    tenant.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Validate CPF if provided
	var cpfVO document.CPF
	if cmd.CPF != "" {
		if s.documentSvc == nil {
			// Should not happen if initialized correctly, but defensive coding
			return nil, fmt.Errorf("document service not initialized")
		}
		var err error
		cpfVO, err = s.documentSvc.ValidateAndNormalizeCPF(ctx, cmd.CPF)
		if err != nil {
			return nil, fmt.Errorf("invalid CPF: %w", err)
		}
	}

	newUser, err := user.NewUser(userID, cmd.Email, hashedPassword, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user entity: %w", err)
	}
	// Set validated CPF
	newUser.CPF = cpfVO

	// Set verification token
	verificationToken := uuid.New().String()
	newUser.VerificationToken = verificationToken
	newUser.VerificationTokenExpiresAt = now.Add(24 * time.Hour)
	newUser.Status = user.StatusPending

	// 4. Execute everything in a single transaction
	err = s.tm.RunInTransaction(ctx, func(ctx context.Context) error {
		// 4a. Create Tenant
		if err := s.tenantRepo.Create(ctx, newTenant); err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
		}

		// Set RLS context for the transaction so we can insert the user
		if err := s.tenantRepo.SetCurrentTenant(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to set tenant context: %w", err)
		}

		// 4b. Create User
		if err := s.userRepo.Create(ctx, newUser); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// 4c. Create ReBAC tuple (user is owner of tenant)
		if err := s.policyRepo.WriteTuple(ctx, policy.Tuple{
			TenantID: tenantID,
			Subject:  "user:" + userID,
			Relation: "owner",
			Object:   "tenant:" + tenantID,
		}); err != nil {
			return fmt.Errorf("failed to assign permissions: %w", err)
		}

		// 4d. Write to outbox (TenantProvisioned event)
		if err := s.outboxPub.Publish(ctx, event.Event{
			ID:        uuid.New().String(),
			Type:      event.TenantProvisioned,
			Source:    EventSource,
			Timestamp: now,
			TenantID:  tenantID,
			Payload: event.TenantProvisionedPayload{
				TenantID: tenantID,
				Name:     newTenant.Name,
			},
		}); err != nil {
			return fmt.Errorf("failed to publish tenant event: %w", err)
		}

		// 4e. Write to outbox (UserCreated event)
		if err := s.outboxPub.Publish(ctx, event.Event{
			ID:        uuid.New().String(),
			Type:      event.UserCreated,
			Source:    EventSource,
			Timestamp: now,
			TenantID:  tenantID,
			Payload: event.UserCreatedPayload{
				UserID:   userID,
				Email:    newUser.Email,
				TenantID: tenantID,
				Role:     "owner",
			},
		}); err != nil {
			return fmt.Errorf("failed to publish user event: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 6. Publish to in-memory bus (for real-time listeners like Kill Switch)
	// This is fire-and-forget, after commit
	go func() {
		if err := s.eventBus.Publish(ctx, event.Event{
			ID:        uuid.New().String(),
			Type:      event.UserCreated,
			Source:    EventSource,
			Timestamp: now,
			TenantID:  tenantID,
			Payload: event.UserCreatedPayload{
				UserID:   userID,
				Email:    newUser.Email,
				TenantID: tenantID,
				Role:     "owner",
			},
		}); err != nil {
			log.Printf("Failed to publish user-created event: %v", err)
		}
	}()

	// 7. Send Verification Email
	// Using the notification service which handles async execution internally
	verificationLink := fmt.Sprintf("http://localhost:4200/auth/verify-email?token=%s", verificationToken)
	subject := "Verify your email address"
	body := fmt.Sprintf(`
		<h1>Welcome to Vyst Identity!</h1>
		<p>Please click the link below to verify your email address and activate your account:</p>
		<p><a href="%s">Verify Email</a></p>
		<p>This link will expire in 24 hours.</p>
	`, verificationLink)
	if err := s.notifier.SendEmail(cmd.Email, subject, body); err != nil {
		log.Printf("Failed to send verification email: %v", err)
	}

	return &RegisterResult{
		User:   newUser,
		Tenant: newTenant,
	}, nil
}

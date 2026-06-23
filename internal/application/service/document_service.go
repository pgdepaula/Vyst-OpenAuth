package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/document"
)

// DocumentService handles document validation interactions.
type DocumentService struct {
	logger           ports.Logger
	verificationPort ports.DocumentVerificationPort
	repo             document.VerificationRepository
}

// NewDocumentService creates a new instance of DocumentService.
func NewDocumentService(logger ports.Logger, verificationPort ports.DocumentVerificationPort, repo document.VerificationRepository) *DocumentService {
	return &DocumentService{
		logger:           logger,
		verificationPort: verificationPort,
		repo:             repo,
	}
}

// ValidateCPF validates a CPF string.
func (s *DocumentService) ValidateCPF(cpf string) error {
	_, err := document.NewCPF(cpf)
	return err
}

// ValidateAndNormalizeCPF validates a CPF and returns its Value Object.
// Returns error if validation fails.
func (s *DocumentService) ValidateAndNormalizeCPF(ctx context.Context, cpf string) (document.CPF, error) {
	// 1. Static Validation & Creation attempt
	// We use NewCPF to validate and normalize at once if possible,
	// but the original logic had distinct steps (validate, then external check, then repo).
	// Let's keep the logic similar but wrap result.

	// First, let's just use the domain validation.
	// Note: NewCPF normalizes internally.
	cpfVO, err := document.NewCPF(cpf)
	if err != nil {
		return document.CPF{}, fmt.Errorf("invalid CPF: %w", err)
	}

	// Get normalized string for external checks (logic remains similar)
	// Accessing internal number is not possible outside package, but we have Value() or we can re-normalize input string.
	// Actually NewCPF guarantees it is valid, but we might want to check against external sources.

	// We need the string for External API.
	// Since 'number' field is private, we can use Value() (it returns string) or String() (formatted).
	// The previous code normalized it manually.

	// To access the raw number for external validation:
	// We can rely on value string being essentially what we want?
	// Let's use internal helpers if needed or just use what we have.
	// Wait, `cpfVO.Value()` returns `driver.Value` which is `interface{}`.

	// Let's assume we can get the string via String() and remove formatting,
	// OR we can just re-normalize the input 'cpf' string since we know it's valid.
	value, err := cpfVO.Value()
	if err != nil {
		return document.CPF{}, fmt.Errorf("invalid CPF value: %w", err)
	}
	normalized, ok := value.(string)
	if !ok {
		return document.CPF{}, fmt.Errorf("invalid CPF value type")
	}

	verificationSource := "OFFLINE_ALGO"
	isValid := true // Algorithmic validation passed (NewCPF succeeded)

	// 2. External Validation (if configured)
	if s.verificationPort != nil {
		res, err := s.verificationPort.VerifyCPF(ctx, normalized)
		if err != nil {
			s.logger.Warn("External CPF verification failed (fallback to offline)", "error", err)
			// Fallback: stay with OFFLINE_ALGO result
		} else {
			verificationSource = "EXTERNAL_API"
			isValid = res.Valid
			if !res.Valid {
				s.logger.Warn("External CPF verification returned invalid", "cpf", cpfVO.Mask(), "reason", res.Situation)
				return document.CPF{}, fmt.Errorf("CPF invalid according to external source: %s", res.Situation)
			}
		}
	}

	// 3. Log Verification (Async)
	if s.repo != nil {
		go func() {
			record := &document.VerificationRecord{
				ID:        uuid.New().String(),
				Document:  cpfVO.Mask(), // Store masked for privacy/audit
				Type:      "CPF",
				Source:    verificationSource,
				Valid:     isValid,
				CreatedAt: time.Now(),
			}
			// Creates a detached context for async logging
			if err := s.repo.Save(context.Background(), record); err != nil {
				s.logger.Error("Failed to save verification record", "error", err)
			}
		}()
	}

	return cpfVO, nil
}

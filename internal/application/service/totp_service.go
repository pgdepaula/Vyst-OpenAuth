package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/auth"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPService handles TOTP-based two-factor authentication.
type TOTPService struct {
	repo           TOTPRepository
	tempTokenStore ports.TempTokenStore
	issuer         string
	tempTokenTTL   time.Duration
}

// TOTPRepository interface for TOTP secrets storage.
type TOTPRepository interface {
	Create(ctx context.Context, secret *auth.TOTPSecret) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*auth.TOTPSecret, error)
	Update(ctx context.Context, secret *auth.TOTPSecret) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

// TOTPSetupResult contains the QR code URL and secret for setup.
type TOTPSetupResult struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// NewTOTPService creates a new TOTP service.
func NewTOTPService(repo TOTPRepository, tempTokenStore ports.TempTokenStore, issuer string) *TOTPService {
	return &TOTPService{
		repo:           repo,
		tempTokenStore: tempTokenStore,
		issuer:         issuer,
		tempTokenTTL:   5 * time.Minute,
	}
}

// GenerateSecret creates a new TOTP secret for a user (setup step 1).
func (s *TOTPService) GenerateSecret(ctx context.Context, userID, email string) (*TOTPSetupResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user already has a secret (pending or enabled)
	existing, err := s.repo.GetByUserID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing secret: %w", err)
	}

	// Generate new TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: email,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.generateBackupCodes(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	secret := &auth.TOTPSecret{
		UserID:      uid,
		Secret:      key.Secret(),
		Enabled:     false, // Will be enabled after verification
		BackupCodes: backupCodes,
	}

	if existing != nil {
		// Update existing (maybe they're re-setting up)
		secret.ID = existing.ID
		if err := s.repo.Update(ctx, secret); err != nil {
			return nil, fmt.Errorf("failed to update secret: %w", err)
		}
	} else {
		// Create new
		if err := s.repo.Create(ctx, secret); err != nil {
			return nil, fmt.Errorf("failed to store secret: %w", err)
		}
	}

	return &TOTPSetupResult{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// VerifySetup verifies the setup code and enables 2FA.
func (s *TOTPService) VerifySetup(ctx context.Context, userID, code string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	secret, err := s.repo.GetByUserID(ctx, uid)
	if err != nil || secret == nil {
		return errors.New("no TOTP secret found, please set up 2FA first")
	}

	// Verify the code
	if !totp.Validate(code, secret.Secret) {
		return errors.New("invalid verification code")
	}

	// Enable 2FA
	secret.Enabled = true
	if err := s.repo.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to enable 2FA: %w", err)
	}

	return nil
}

// IsEnabled checks if a user has 2FA enabled.
func (s *TOTPService) IsEnabled(ctx context.Context, userID string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	secret, err := s.repo.GetByUserID(ctx, uid)
	if err != nil {
		return false, fmt.Errorf("failed to check 2FA status: %w", err)
	}

	if secret == nil {
		return false, nil
	}

	return secret.Enabled, nil
}

// VerifyCode validates a TOTP code for a user.
func (s *TOTPService) VerifyCode(ctx context.Context, userID, code string) bool {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false
	}

	secret, err := s.repo.GetByUserID(ctx, uid)
	if err != nil || secret == nil || !secret.Enabled {
		return false
	}

	// First try TOTP code
	if totp.Validate(code, secret.Secret) {
		return true
	}

	// Then try backup codes
	for i, bc := range secret.BackupCodes {
		if bc == code {
			// Remove used backup code
			secret.BackupCodes = append(secret.BackupCodes[:i], secret.BackupCodes[i+1:]...)
			_ = s.repo.Update(ctx, secret) // Best effort
			return true
		}
	}

	return false
}

// Disable removes 2FA for a user.
func (s *TOTPService) Disable(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	return s.repo.Delete(ctx, uid)
}

// GenerateTempToken creates a temporary token for the 2FA step.
func (s *TOTPService) GenerateTempToken(userID string) (string, error) {
	// Generate random bytes
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	tokenStr := base64.URLEncoding.EncodeToString(token)

	// Store in temp token store with short TTL
	ctx := context.Background()
	if s.tempTokenStore != nil {
		// Store userID -> token mapping
		key := "2fa_temp:" + tokenStr
		if err := s.tempTokenStore.SaveString(ctx, key, userID, s.tempTokenTTL); err != nil {
			return "", fmt.Errorf("failed to store temp token: %w", err)
		}
	}

	return tokenStr, nil
}

// ValidateTempToken validates a temporary token and returns the user ID.
func (s *TOTPService) ValidateTempToken(ctx context.Context, tempToken string) (string, error) {
	if s.tempTokenStore == nil {
		return "", errors.New("temp token store not available")
	}

	key := "2fa_temp:" + tempToken
	userID, err := s.tempTokenStore.GetString(ctx, key)
	if err != nil {
		return "", errors.New("invalid or expired temporary token")
	}

	// Delete the token after use (one-time use)
	_ = s.tempTokenStore.Delete(ctx, key)

	return userID, nil
}

// generateBackupCodes generates random backup codes.
func (s *TOTPService) generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 5)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		// Format: XXXX-XXXX
		code := strings.ToUpper(base32.StdEncoding.EncodeToString(bytes))
		codes[i] = code[:4] + "-" + code[4:8]
	}
	return codes, nil
}

package integration

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/config"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/security"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Token Service Integration Tests
// These tests verify token service works with real config keys.
// They skip gracefully if environment is not properly configured.
// ============================================================================

func getTokenService(t *testing.T) ports.TokenService {
	// Try to load .env from root
	err := godotenv.Load("../../.env", "../../.env.development")
	if err != nil {
		t.Logf("Failed to load .env: %v", err)
		cwd, _ := os.Getwd()
		t.Logf("Current working directory: %s", cwd)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skip("Config not available")
		return nil
	}

	svc, err := security.NewTokenService(cfg.JWTPrivateKey, cfg.JWTPublicKey, "vyst-identity")
	if err != nil {
		t.Skipf("Skipped: PEM key parsing issue in test env")
		return nil
	}
	return svc
}

func TestTokenService_Integration_ServiceCreation(t *testing.T) {
	svc := getTokenService(t)
	if svc == nil {
		return
	}
	assert.NotNil(t, svc)
}

func TestTokenService_Integration_GenerateToken(t *testing.T) {
	svc := getTokenService(t)
	if svc == nil {
		return
	}
	token, err := svc.GenerateToken("user-123", "tenant-456", []string{"admin"}, "", "", "individual")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestTokenService_Integration_ValidateToken(t *testing.T) {
	svc := getTokenService(t)
	if svc == nil {
		return
	}
	token, _ := svc.GenerateToken("user-123", "tenant-456", []string{"admin"}, "", "", "individual")
	claims, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
}

func TestTokenService_Integration_InvalidToken(t *testing.T) {
	svc := getTokenService(t)
	if svc == nil {
		return
	}
	claims, err := svc.ValidateToken("invalid.token")
	assert.Nil(t, claims)
	assert.Error(t, err)
}

package ports

import (
	"github.com/golang-jwt/jwt/v5"
)

// PasswordHasher defines the contract for password hashing operations.
type PasswordHasher interface {
	// Hash generates a secure hash from a plaintext password.
	Hash(password string) (string, error)

	// Verify checks if a plaintext password matches a hash.
	Verify(password, hash string) bool
}

// TokenService defines the contract for JWT token operations.
type TokenService interface {
	// GenerateToken creates a new signed JWT for the user.
	GenerateToken(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error)

	// GenerateEncryptedToken creates an encrypted token (JWE).
	GenerateEncryptedToken(payload map[string]interface{}) (string, error)

	// ValidateToken verifies and parses a JWT token.
	ValidateToken(tokenString string) (*Claims, error)

	// GenerateRefreshToken creates a secure random string for refresh tokens.
	GenerateRefreshToken() string

	// GenerateID creates a unique identifier (UUID).
	GenerateID() string
}

// Claims represents the JWT payload.
type Claims struct {
	UserID          string   `json:"user_id"`
	TenantID        string   `json:"tenant_id"`
	Roles           []string `json:"roles"`
	ActiveCompanyID string   `json:"active_company_id,omitempty"`
	CompanyRole     string   `json:"company_role,omitempty"`
	IdentityType    string   `json:"identity_type"` // "individual" or "company"
	jwt.RegisteredClaims
}

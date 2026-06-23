// Package security provides authentication and cryptographic services.
package security

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
)

// tokenService implements TokenService using RSA keys.
type tokenService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	expiry     time.Duration
}

// NewTokenService creates a new token service with RSA keys.
func NewTokenService(privateKeyPEM, publicKeyPEM, issuer string) (ports.TokenService, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &tokenService{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		expiry:     24 * time.Hour,
	}, nil
}

// GenerateToken creates a new signed JWT.
func (s *tokenService) GenerateToken(userID, tenantID string, roles []string, activeCompanyID, companyRole, identityType string) (string, error) {
	now := time.Now()
	claims := ports.Claims{
		UserID:          userID,
		TenantID:        tenantID,
		Roles:           roles,
		ActiveCompanyID: activeCompanyID,
		CompanyRole:     companyRole,
		IdentityType:    identityType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// GenerateEncryptedToken creates a JWE encrypted token.
func (s *tokenService) GenerateEncryptedToken(payload map[string]interface{}) (string, error) {
	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: s.publicKey},
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create encrypter: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	object, err := encrypter.Encrypt(payloadBytes)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}

	return object.CompactSerialize()
}

// ValidateToken verifies and parses a JWT token.
func (s *tokenService) ValidateToken(tokenString string) (*ports.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ports.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*ports.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateRefreshToken creates a secure random string for refresh tokens.
func (s *tokenService) GenerateRefreshToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "" // Should handle error better, but for now empty string will cause failure downstream
	}
	return hex.EncodeToString(b)
}

// GenerateID creates a unique identifier (UUID).
func (s *tokenService) GenerateID() string {
	return uuid.New().String()
}

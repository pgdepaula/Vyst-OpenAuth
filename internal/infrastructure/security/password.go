// Package security provides authentication and cryptographic services.
package security

import (
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"golang.org/x/crypto/bcrypt"
)

// bcryptHasher implements PasswordHasher using bcrypt.
type bcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new bcrypt password hasher.
// Default cost is 14 (secure but not too slow).
func NewBcryptHasher() ports.PasswordHasher {
	return &bcryptHasher{cost: 14}
}

// NewBcryptHasherWithCost creates a hasher with custom cost.
func NewBcryptHasherWithCost(cost int) ports.PasswordHasher {
	return &bcryptHasher{cost: cost}
}

// Hash generates a bcrypt hash from the password.
func (h *bcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(bytes), err
}

// Verify checks if the password matches the hash.
func (h *bcryptHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

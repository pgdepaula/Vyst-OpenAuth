// Package session contains the Session domain entity and repository interface.
package session

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("session not found")

// Session represents an active user session.
type Session struct {
	ID           string    // Unique session identifier
	UserID       string    // ID of the user who owns this session
	RefreshToken string    // Refresh token associated with this session
	UserAgent    string    // User agent string of the client
	IPAddress    string    // IP address of the client
	ExpiresAt    time.Time // When the session/refresh token expires
	CreatedAt    time.Time // When the session was created
	Revoked      bool      // Whether the session has been manually revoked
}

// NewSession creates a new session instance.
func NewSession(id, userID, refreshToken, userAgent, ipAddress string, expiresAt time.Time) *Session {
	return &Session{
		ID:           id,
		UserID:       userID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		Revoked:      false,
	}
}

// IsValid checks if the session is valid (not expired and not revoked).
func (s *Session) IsValid() bool {
	return !s.Revoked && time.Now().Before(s.ExpiresAt)
}

// Repository defines the contract for session persistence.
type Repository interface {
	// Create persists a new session.
	Create(ctx context.Context, session *Session) error

	// GetByID retrieves a session by its ID.
	GetByID(ctx context.Context, id string) (*Session, error)

	// GetByRefreshToken retrieves a session by its refresh token.
	GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)

	// Update updates an existing session.
	Update(ctx context.Context, session *Session) error

	// Delete removes a session.
	Delete(ctx context.Context, id string) error

	// DeleteByUserID removes all sessions for a user (e.g. logout all devices).
	DeleteByUserID(ctx context.Context, userID string) error
}

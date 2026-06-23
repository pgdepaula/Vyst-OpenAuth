package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/pgdepaula/vyst-openauth/internal/domain/session"
	"github.com/redis/go-redis/v9"
)

type SessionStore struct {
	client *redis.Client
}

func NewSessionStore(client *redis.Client) *SessionStore {
	return &SessionStore{client: client}
}

func (s *SessionStore) SaveSession(ctx context.Context, key string, session *webauthn.SessionData, ttl time.Duration) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.client.Set(ctx, "webauthn:"+key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save session to redis: %w", err)
	}
	return nil
}

func (s *SessionStore) GetSession(ctx context.Context, key string) (*webauthn.SessionData, error) {
	data, err := s.client.Get(ctx, "webauthn:"+key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session from redis: %w", err)
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Delete after retrieval (one-time use)
	s.client.Del(ctx, "webauthn:"+key)

	return &session, nil
}

// SaveString stores a string value with TTL.
func (s *SessionStore) SaveString(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := s.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save string to redis: %w", err)
	}
	return nil
}

// GetString retrieves a string value.
func (s *SessionStore) GetString(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("key not found")
		}
		return "", fmt.Errorf("failed to get string from redis: %w", err)
	}
	return val, nil
}

// Delete removes a key.
// Delete removes a key.
func (s *SessionStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// --- session.Repository Implementation ---

// Create persists a new session.
func (s *SessionStore) Create(ctx context.Context, sess *session.Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	pipe := s.client.Pipeline()

	// 1. Store session data
	pipe.Set(ctx, "session:"+sess.ID, data, time.Until(sess.ExpiresAt))

	// 2. Store refresh token mapping
	pipe.Set(ctx, "refresh:"+sess.RefreshToken, sess.ID, time.Until(sess.ExpiresAt))

	// 3. Add to user sessions set
	pipe.SAdd(ctx, "user:"+sess.UserID+":sessions", sess.ID)
	pipe.Expire(ctx, "user:"+sess.UserID+":sessions", 30*24*time.Hour) // Keep user mapping for 30 days

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session in redis: %w", err)
	}
	return nil
}

// GetByID retrieves a session by its ID.
func (s *SessionStore) GetByID(ctx context.Context, id string) (*session.Session, error) {
	data, err := s.client.Get(ctx, "session:"+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, session.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	return &sess, nil
}

// GetByRefreshToken retrieves a session by its refresh token.
func (s *SessionStore) GetByRefreshToken(ctx context.Context, refreshToken string) (*session.Session, error) {
	id, err := s.client.Get(ctx, "refresh:"+refreshToken).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, session.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Update updates an existing session.
func (s *SessionStore) Update(ctx context.Context, sess *session.Session) error {
	// For now, just re-save the session data.
	// If refresh token changed, we'd need to handle that, but typically it doesn't change on simple updates.
	// If it does, we should delete old mapping.

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := s.client.Set(ctx, "session:"+sess.ID, data, time.Until(sess.ExpiresAt)).Err(); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteByUserID removes all sessions for a user.
func (s *SessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	// Get all session IDs
	key := "user:" + userID + ":sessions"
	sessionIDs, err := s.client.SMembers(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, id := range sessionIDs {
		// Get session to find refresh token (to delete mapping)
		// This is best-effort.
		sess, _ := s.GetByID(ctx, id)

		pipe.Del(ctx, "session:"+id)
		if sess != nil {
			pipe.Del(ctx, "refresh:"+sess.RefreshToken)
		}
	}
	pipe.Del(ctx, key)

	_, err = pipe.Exec(ctx)
	return err
}

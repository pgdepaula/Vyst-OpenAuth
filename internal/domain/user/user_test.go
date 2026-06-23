package user_test

import (
	"testing"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// NewUser Validation Tests
// ============================================================================

func TestNewUser_ValidInput_ReturnsUser(t *testing.T) {
	u, err := user.NewUser("user-123", "test@example.com", "hashed_password", "tenant-456")

	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "user-123", u.ID)
	assert.Equal(t, "test@example.com", u.Email)
	assert.Equal(t, "hashed_password", u.PasswordHash)
	assert.Equal(t, "tenant-456", u.TenantID)
}

func TestNewUser_GeneratesTimestamps(t *testing.T) {
	before := time.Now().Add(-1 * time.Second)

	u, err := user.NewUser("user-123", "test@example.com", "hashed_password", "tenant-456")

	after := time.Now().Add(1 * time.Second)

	require.NoError(t, err)
	assert.True(t, u.CreatedAt.After(before), "CreatedAt should be after test start")
	assert.True(t, u.CreatedAt.Before(after), "CreatedAt should be before test end")
	assert.True(t, u.UpdatedAt.After(before), "UpdatedAt should be after test start")
	assert.True(t, u.UpdatedAt.Before(after), "UpdatedAt should be before test end")
	assert.Equal(t, u.CreatedAt, u.UpdatedAt, "CreatedAt and UpdatedAt should be equal on creation")
}

// ============================================================================
// NewUser Required Field Validation Tests
// ============================================================================

func TestNewUser_EmptyID_ReturnsError(t *testing.T) {
	u, err := user.NewUser("", "test@example.com", "hashed_password", "tenant-456")

	assert.Nil(t, u)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")
}

func TestNewUser_EmptyEmail_ReturnsError(t *testing.T) {
	u, err := user.NewUser("user-123", "", "hashed_password", "tenant-456")

	assert.Nil(t, u)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestNewUser_EmptyPasswordHash_ReturnsError(t *testing.T) {
	u, err := user.NewUser("user-123", "test@example.com", "", "tenant-456")

	assert.Nil(t, u)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestNewUser_EmptyTenantID_ReturnsError(t *testing.T) {
	u, err := user.NewUser("user-123", "test@example.com", "hashed_password", "")

	assert.Nil(t, u)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant")
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestNewUser_AllFieldsEmpty_ReturnsFirstError(t *testing.T) {
	u, err := user.NewUser("", "", "", "")

	assert.Nil(t, u)
	assert.Error(t, err)
	// Should return the first validation error (ID)
	assert.Contains(t, err.Error(), "id")
}

func TestNewUser_WhitespaceID_ReturnsUser(t *testing.T) {
	// Note: Current implementation doesn't trim whitespace
	// This test documents current behavior
	u, err := user.NewUser("  ", "test@example.com", "hash", "tenant")

	require.NoError(t, err) // Current behavior allows whitespace
	assert.NotNil(t, u)
	assert.Equal(t, "  ", u.ID)
}

func TestNewUser_VeryLongEmail_ReturnsUser(t *testing.T) {
	longEmail := "verylongemail" + "@" + "verylongdomain.com"

	u, err := user.NewUser("user-123", longEmail, "hash", "tenant")

	require.NoError(t, err)
	assert.Equal(t, longEmail, u.Email)
}

func TestNewUser_SpecialCharactersInEmail_ReturnsUser(t *testing.T) {
	specialEmail := "user+tag@example.com"

	u, err := user.NewUser("user-123", specialEmail, "hash", "tenant")

	require.NoError(t, err)
	assert.Equal(t, specialEmail, u.Email)
}

// ============================================================================
// User Struct Direct Tests
// ============================================================================

func TestUser_FieldsAreAccessible(t *testing.T) {
	now := time.Now()
	u := &user.User{
		ID:           "id",
		Email:        "email@test.com",
		PasswordHash: "hash",
		TenantID:     "tenant",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, "id", u.ID)
	assert.Equal(t, "email@test.com", u.Email)
	assert.Equal(t, "hash", u.PasswordHash)
	assert.Equal(t, "tenant", u.TenantID)
	assert.Equal(t, now, u.CreatedAt)
	assert.Equal(t, now, u.UpdatedAt)
}

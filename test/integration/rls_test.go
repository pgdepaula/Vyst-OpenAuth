package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRLS_Isolation(t *testing.T) {
	env := SetupTestEnv(t)
	ctx := context.Background()

	// 1. Create Tenants
	tenantA := uuid.New().String()
	tenantB := uuid.New().String()

	_, err := env.DB.ExecContext(ctx, "INSERT INTO tenants (id, name) VALUES ($1, 'Tenant A')", tenantA)
	require.NoError(t, err)
	_, err = env.DB.ExecContext(ctx, "INSERT INTO tenants (id, name) VALUES ($1, 'Tenant B')", tenantB)
	require.NoError(t, err)

	// 2. Insert Users (Must set RLS context)
	// Insert User A for Tenant A
	txA, err := env.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = txA.ExecContext(ctx, "SET LOCAL app.current_tenant = '"+tenantA+"'")
	require.NoError(t, err)
	_, err = txA.ExecContext(ctx, "INSERT INTO users (email, password_hash, tenant_id) VALUES ('user@a.com', 'hash', $1)", tenantA)
	require.NoError(t, err)
	err = txA.Commit()
	require.NoError(t, err)

	// Insert User B for Tenant B
	txB, err := env.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = txB.ExecContext(ctx, "SET LOCAL app.current_tenant = '"+tenantB+"'")
	require.NoError(t, err)
	_, err = txB.ExecContext(ctx, "INSERT INTO users (email, password_hash, tenant_id) VALUES ('user@b.com', 'hash', $1)", tenantB)
	require.NoError(t, err)
	err = txB.Commit()
	require.NoError(t, err)

	// 3. Verify Isolation
	// Query as Tenant A
	txQueryA, err := env.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, txQueryA.Rollback())
	}()
	_, err = txQueryA.ExecContext(ctx, "SET LOCAL app.current_tenant = '"+tenantA+"'")
	require.NoError(t, err)

	rowsA, err := txQueryA.QueryContext(ctx, "SELECT email FROM users")
	require.NoError(t, err)
	var emailsA []string
	for rowsA.Next() {
		var email string
		err := rowsA.Scan(&email)
		require.NoError(t, err)
		emailsA = append(emailsA, email)
	}
	assert.Len(t, emailsA, 1)
	assert.Contains(t, emailsA, "user@a.com")
	assert.NotContains(t, emailsA, "user@b.com")

	// Query as Tenant B
	txQueryB, err := env.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, txQueryB.Rollback())
	}()
	_, err = txQueryB.ExecContext(ctx, "SET LOCAL app.current_tenant = '"+tenantB+"'")
	require.NoError(t, err)

	rowsB, err := txQueryB.QueryContext(ctx, "SELECT email FROM users")
	require.NoError(t, err)
	var emailsB []string
	for rowsB.Next() {
		var email string
		err := rowsB.Scan(&email)
		require.NoError(t, err)
		emailsB = append(emailsB, email)
	}
	assert.Len(t, emailsB, 1)
	assert.Contains(t, emailsB, "user@b.com")
	assert.NotContains(t, emailsB, "user@a.com")

	// Query without Tenant Context (Should fail or return nothing depending on policy implementation details,
	// but since we use COALESCE(..., '0000...'), it should return nothing matching the real tenants)
	txQueryNone, err := env.DB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, txQueryNone.Rollback())
	}()
	// No SET LOCAL

	rowsNone, err := txQueryNone.QueryContext(ctx, "SELECT email FROM users")
	require.NoError(t, err)
	var emailsNone []string
	for rowsNone.Next() {
		var email string
		err := rowsNone.Scan(&email)
		require.NoError(t, err)
		emailsNone = append(emailsNone, email)
	}
	assert.Empty(t, emailsNone, "Should not see any users when no tenant context is set")
}

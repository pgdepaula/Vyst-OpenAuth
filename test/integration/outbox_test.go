package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/messaging/outbox"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutbox_Transactional(t *testing.T) {
	env := SetupTestEnv(t)
	ctx := context.Background()

	tm := postgres.NewTransactionManager(env.Pool)
	publisher := outbox.NewPublisher(env.Pool)

	// Setup Tenant
	tenantID := uuid.New().String()
	_, err := env.DB.ExecContext(ctx, "INSERT INTO tenants (id, name) VALUES ($1, 'Tenant Outbox')", tenantID)
	require.NoError(t, err)

	// Sanity Check: Can vyst_app write/read outbox_events?
	_, err = env.Pool.Exec(ctx, "INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload) VALUES ('Sanity', '123', 'Test', '{}')")
	require.NoError(t, err, "Sanity insert failed")
	var sanityCount int
	err = env.Pool.QueryRow(ctx, "SELECT count(*) FROM outbox_events WHERE aggregate_id = '123'").Scan(&sanityCount)
	require.NoError(t, err, "Sanity select failed")
	require.Equal(t, 1, sanityCount, "Sanity check failed: expected 1 row")

	t.Run("Commit_PersistsEvent", func(t *testing.T) {
		userID := uuid.New().String()
		t.Logf("Test UserID: %s", userID)
		evt := event.Event{
			ID:   uuid.New().String(),
			Type: event.UserCreated,
			Payload: event.UserCreatedPayload{
				UserID: userID,
				Email:  "outbox@test.com",
			},
		}

		err := tm.RunInTransaction(ctx, func(ctxTx context.Context) error {
			// 1. Insert User (Simulate Repo)
			executor := postgres.GetExecutor(ctxTx, env.Pool)

			// Set Tenant Context
			_, err := executor.Exec(ctxTx, "SET LOCAL app.current_tenant = '"+tenantID+"'")
			if err != nil {
				return err
			}

			_, err = executor.Exec(ctxTx, "INSERT INTO users (id, email, password_hash, tenant_id) VALUES ($1, 'outbox@test.com', 'hash', $2)", userID, tenantID)
			if err != nil {
				return err
			}

			// 2. Publish Event
			return publisher.Publish(ctxTx, evt)
		})
		require.NoError(t, err)

		// Verify User Exists (Use AdminDB to bypass RLS)
		var count int
		err = env.AdminDB.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id = $1", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Verify Event Exists using AdminDB
		err = env.AdminDB.QueryRowContext(ctx, "SELECT count(*) FROM outbox_events WHERE aggregate_id = $1", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Rollback_DiscardsEvent", func(t *testing.T) {
		userID := uuid.New().String()
		evt := event.Event{
			ID:   uuid.New().String(),
			Type: event.UserCreated,
			Payload: event.UserCreatedPayload{
				UserID: userID,
				Email:  "rollback@test.com",
			},
		}

		err := tm.RunInTransaction(ctx, func(ctxTx context.Context) error {
			executor := postgres.GetExecutor(ctxTx, env.Pool)

			// Set Tenant Context
			_, err := executor.Exec(ctxTx, "SET LOCAL app.current_tenant = '"+tenantID+"'")
			if err != nil {
				return err
			}

			_, err = executor.Exec(ctxTx, "INSERT INTO users (id, email, password_hash, tenant_id) VALUES ($1, 'rollback@test.com', 'hash', $2)", userID, tenantID)
			if err != nil {
				return err
			}

			err = publisher.Publish(ctxTx, evt)
			if err != nil {
				return err
			}

			// Force Rollback
			return assert.AnError
		})
		require.Error(t, err)

		// Verify User Does NOT Exist
		var count int
		err = env.AdminDB.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE id = $1", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Verify Event Does NOT Exist
		err = env.AdminDB.QueryRowContext(ctx, "SELECT count(*) FROM outbox_events WHERE aggregate_id = $1", userID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

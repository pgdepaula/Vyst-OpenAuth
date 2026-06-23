package integration

import (
	"context"
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReBAC_ComplexScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	env := SetupTestEnv(t)
	repo := postgres.NewPolicyRepository(env.Pool)
	ctx := context.Background()

	// Tenant for isolation
	tenantID := "11111111-1111-1111-1111-111111111111"

	// Scenario: "Approval > 5k"
	// We model this using two permissions: "approve_standard" and "approve_high_value"
	// Roles:
	// - Team Lead: Can approve standard
	// - Manager: Can approve high_value AND standard (via inheritance)

	// Entities
	managerRole := "role:manager"
	teamLeadRole := "role:team_lead"

	aliceManager := "user:alice_manager"
	bobTeamLead := "user:bob_team_lead"

	permStandard := "permission:approve_standard"
	permHighValue := "permission:approve_high_value"

	// 0. Create Tenant
	_, err := env.Pool.Exec(ctx, "INSERT INTO tenants (id, name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())", tenantID, "Test Tenant")
	require.NoError(t, err)

	// Setup Relationships
	tuples := []policy.Tuple{
		// 1. Assign Users to Roles
		{TenantID: tenantID, Subject: aliceManager, Relation: "member", Object: managerRole},
		{TenantID: tenantID, Subject: bobTeamLead, Relation: "member", Object: teamLeadRole},

		// 2. Role Inheritance: Manager "is a" Team Lead (inherits permissions)
		// NOTE: The engine recurses on 'member'. So we say Manager is a member of Team Lead group?
		// Let's trace: Check(Alice, granted, permStandard)
		// Base: permStandard -> granted -> TeamLead
		// Recurse: TeamLead -> member -> Manager
		// Recurse: Manager -> member -> Alice
		// So yes, Manager must be a 'member' of TeamLead for Alice to inherit TeamLead's perms.
		{TenantID: tenantID, Subject: managerRole, Relation: "member", Object: teamLeadRole},

		// 3. Assign Permissions to Roles
		{TenantID: tenantID, Subject: teamLeadRole, Relation: "granted", Object: permStandard},
		{TenantID: tenantID, Subject: managerRole, Relation: "granted", Object: permHighValue},
	}

	for _, tuple := range tuples {
		err := repo.WriteTuple(ctx, tuple)
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		subject  string
		relation string
		object   string
		want     bool
	}{
		// Standard Approval
		{"Team Lead can approve standard", bobTeamLead, "granted", permStandard, true},
		{"Manager can approve standard (inherited)", aliceManager, "granted", permStandard, true},

		// High Value Approval
		{"Team Lead CANNOT approve high value", bobTeamLead, "granted", permHighValue, false},
		{"Manager can approve high value", aliceManager, "granted", permHighValue, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Check(ctx, tt.subject, tt.relation, tt.object)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "Check(%s, %s, %s)", tt.subject, tt.relation, tt.object)
		})
	}
}

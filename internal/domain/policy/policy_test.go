package policy_test

import (
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Tuple Struct Tests
// ============================================================================

func TestTuple_FieldsAreAccessible(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "user:alice",
		Relation: "owner",
		Object:   "document:doc-456",
	}

	assert.Equal(t, "tenant-123", tuple.TenantID)
	assert.Equal(t, "user:alice", tuple.Subject)
	assert.Equal(t, "owner", tuple.Relation)
	assert.Equal(t, "document:doc-456", tuple.Object)
}

func TestTuple_CommonSubjectPatterns(t *testing.T) {
	tests := []struct {
		name    string
		subject string
	}{
		{"user subject", "user:user-123"},
		{"group subject", "group:admins"},
		{"role subject", "role:manager"},
		{"service subject", "service:auth-api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := policy.Tuple{Subject: tt.subject}
			assert.Equal(t, tt.subject, tuple.Subject)
		})
	}
}

func TestTuple_CommonRelationPatterns(t *testing.T) {
	tests := []struct {
		name     string
		relation string
	}{
		{"owner relation", "owner"},
		{"viewer relation", "viewer"},
		{"editor relation", "editor"},
		{"member relation", "member"},
		{"admin relation", "admin"},
		{"granted relation", "granted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := policy.Tuple{Relation: tt.relation}
			assert.Equal(t, tt.relation, tuple.Relation)
		})
	}
}

func TestTuple_CommonObjectPatterns(t *testing.T) {
	tests := []struct {
		name   string
		object string
	}{
		{"tenant object", "tenant:tenant-123"},
		{"document object", "document:doc-456"},
		{"permission object", "permission:approve_high_value"},
		{"role object", "role:manager"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tuple := policy.Tuple{Object: tt.object}
			assert.Equal(t, tt.object, tuple.Object)
		})
	}
}

func TestTuple_EmptyFieldsAreValid(t *testing.T) {
	tuple := policy.Tuple{}

	assert.Empty(t, tuple.TenantID)
	assert.Empty(t, tuple.Subject)
	assert.Empty(t, tuple.Relation)
	assert.Empty(t, tuple.Object)
}

// ============================================================================
// CheckRequest Struct Tests
// ============================================================================

func TestCheckRequest_FieldsAreAccessible(t *testing.T) {
	req := policy.CheckRequest{
		Subject:  "user:alice",
		Relation: "owner",
		Object:   "document:doc-456",
	}

	assert.Equal(t, "user:alice", req.Subject)
	assert.Equal(t, "owner", req.Relation)
	assert.Equal(t, "document:doc-456", req.Object)
}

func TestCheckRequest_CanBeCreatedFromTuple(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "user:alice",
		Relation: "owner",
		Object:   "document:doc-456",
	}

	req := policy.CheckRequest{
		Subject:  tuple.Subject,
		Relation: tuple.Relation,
		Object:   tuple.Object,
	}

	assert.Equal(t, tuple.Subject, req.Subject)
	assert.Equal(t, tuple.Relation, req.Relation)
	assert.Equal(t, tuple.Object, req.Object)
}

// ============================================================================
// CheckResult Struct Tests
// ============================================================================

func TestCheckResult_AllowedTrue(t *testing.T) {
	result := policy.CheckResult{
		Allowed: true,
		Reason:  "Direct relationship found",
	}

	assert.True(t, result.Allowed)
	assert.Equal(t, "Direct relationship found", result.Reason)
}

func TestCheckResult_AllowedFalse(t *testing.T) {
	result := policy.CheckResult{
		Allowed: false,
		Reason:  "No matching relationship",
	}

	assert.False(t, result.Allowed)
	assert.Equal(t, "No matching relationship", result.Reason)
}

func TestCheckResult_EmptyReason(t *testing.T) {
	result := policy.CheckResult{
		Allowed: true,
		Reason:  "",
	}

	assert.True(t, result.Allowed)
	assert.Empty(t, result.Reason)
}

// ============================================================================
// ReBAC Pattern Tests (Documentation through tests)
// ============================================================================

func TestReBAC_UserToRolePattern(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "user:alice",
		Relation: "member",
		Object:   "role:manager",
	}

	assert.Equal(t, "user:alice", tuple.Subject)
	assert.Equal(t, "member", tuple.Relation)
	assert.Equal(t, "role:manager", tuple.Object)
}

func TestReBAC_RoleInheritancePattern(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "role:manager",
		Relation: "member",
		Object:   "role:team_lead",
	}

	assert.Equal(t, "role:manager", tuple.Subject)
	assert.Equal(t, "member", tuple.Relation)
	assert.Equal(t, "role:team_lead", tuple.Object)
}

func TestReBAC_PermissionGrantPattern(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "role:manager",
		Relation: "granted",
		Object:   "permission:approve_high_value",
	}

	assert.Equal(t, "role:manager", tuple.Subject)
	assert.Equal(t, "granted", tuple.Relation)
	assert.Equal(t, "permission:approve_high_value", tuple.Object)
}

func TestReBAC_TenantOwnershipPattern(t *testing.T) {
	tuple := policy.Tuple{
		TenantID: "tenant-123",
		Subject:  "user:alice",
		Relation: "owner",
		Object:   "tenant:tenant-123",
	}

	assert.Equal(t, "user:alice", tuple.Subject)
	assert.Equal(t, "owner", tuple.Relation)
	assert.Equal(t, "tenant:tenant-123", tuple.Object)
}

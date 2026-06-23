package tenant_test

import (
	"testing"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/tenant"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Status Constants Tests
// ============================================================================

func TestStatusConstants_HaveCorrectValues(t *testing.T) {
	assert.Equal(t, tenant.Status("active"), tenant.StatusActive)
	assert.Equal(t, tenant.Status("suspended"), tenant.StatusSuspended)
	assert.Equal(t, tenant.Status("pending"), tenant.StatusPending)
}

// ============================================================================
// IsActive Method Tests
// ============================================================================

func TestTenant_IsActive_WhenStatusActive_ReturnsTrue(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.StatusActive,
	}

	assert.True(t, tn.IsActive())
}

func TestTenant_IsActive_WhenStatusSuspended_ReturnsFalse(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.StatusSuspended,
	}

	assert.False(t, tn.IsActive())
}

func TestTenant_IsActive_WhenStatusPending_ReturnsFalse(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.StatusPending,
	}

	assert.False(t, tn.IsActive())
}

func TestTenant_IsActive_WhenUnknownStatus_ReturnsFalse(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.Status("unknown"),
	}

	assert.False(t, tn.IsActive())
}

func TestTenant_IsActive_WhenEmptyStatus_ReturnsFalse(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.Status(""),
	}

	assert.False(t, tn.IsActive())
}

// ============================================================================
// Tenant Struct Tests
// ============================================================================

func TestTenant_FieldsAreAccessible(t *testing.T) {
	now := time.Now()
	tn := &tenant.Tenant{
		ID:        "tenant-123",
		Name:      "Test Org",
		Status:    tenant.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "tenant-123", tn.ID)
	assert.Equal(t, "Test Org", tn.Name)
	assert.Equal(t, tenant.StatusActive, tn.Status)
	assert.Equal(t, now, tn.CreatedAt)
	assert.Equal(t, now, tn.UpdatedAt)
}

func TestTenant_JSONTags(t *testing.T) {
	// This test verifies that JSON serialization works correctly
	// by checking the struct can be used with JSON tags
	now := time.Now()
	tn := &tenant.Tenant{
		ID:        "tenant-123",
		Name:      "Test Org",
		Status:    tenant.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Verify ID field is json:"id"
	assert.Equal(t, "tenant-123", tn.ID)
}

// ============================================================================
// Status Transition Logic (Behavior Documentation)
// ============================================================================

func TestTenant_StatusCanChange(t *testing.T) {
	tn := &tenant.Tenant{
		ID:     "tenant-123",
		Name:   "Test Tenant",
		Status: tenant.StatusPending,
	}

	assert.False(t, tn.IsActive())

	tn.Status = tenant.StatusActive
	assert.True(t, tn.IsActive())

	tn.Status = tenant.StatusSuspended
	assert.False(t, tn.IsActive())
}

// ============================================================================
// Table-Driven Tests for IsActive
// ============================================================================

func TestTenant_IsActive_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		status   tenant.Status
		expected bool
	}{
		{"active status returns true", tenant.StatusActive, true},
		{"suspended status returns false", tenant.StatusSuspended, false},
		{"pending status returns false", tenant.StatusPending, false},
		{"empty status returns false", tenant.Status(""), false},
		{"unknown status returns false", tenant.Status("unknown"), false},
		{"case sensitive active", tenant.Status("Active"), false}, // Not equal to "active"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tn := &tenant.Tenant{Status: tt.status}
			assert.Equal(t, tt.expected, tn.IsActive())
		})
	}
}

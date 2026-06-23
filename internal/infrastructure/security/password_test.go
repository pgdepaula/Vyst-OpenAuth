package security_test

import (
	"testing"

	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// NewBcryptHasher Tests
// ============================================================================

func TestNewBcryptHasher_ReturnsValidHasher(t *testing.T) {
	hasher := security.NewBcryptHasher()

	assert.NotNil(t, hasher)
}

func TestNewBcryptHasherWithCost_ReturnsValidHasher(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(10)

	assert.NotNil(t, hasher)
}

// ============================================================================
// Hash Tests
// ============================================================================

func TestBcryptHasher_Hash_ReturnsValidHash(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	hash, err := hasher.Hash("password123")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "password123", hash)
}

func TestBcryptHasher_Hash_DifferentHashesForSamePassword(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	hash1, err1 := hasher.Hash("password")
	hash2, err2 := hasher.Hash("password")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "Bcrypt should produce different hashes due to salt")
}

func TestBcryptHasher_Hash_EmptyPassword_ReturnsHash(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	hash, err := hasher.Hash("")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestBcryptHasher_Hash_LongPassword_DocumentsBcryptLimit(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)
	longPassword := string(make([]byte, 100))

	hash, err := hasher.Hash(longPassword)

	if err != nil {
		assert.Contains(t, err.Error(), "72")
	} else {
		assert.NotEmpty(t, hash)
	}
}

func TestBcryptHasher_Hash_SpecialCharacters_ReturnsHash(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	hash, err := hasher.Hash("p@$$w0rd!#%&*(){}[]")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestBcryptHasher_Hash_UnicodePassword_ReturnsHash(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	hash, err := hasher.Hash("senhaçøm¥€₹характери")

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

// ============================================================================
// Verify Tests
// ============================================================================

func TestBcryptHasher_Verify_CorrectPassword_ReturnsTrue(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)
	password := "correct_password"
	hash, _ := hasher.Hash(password)

	result := hasher.Verify(password, hash)

	assert.True(t, result)
}

func TestBcryptHasher_Verify_WrongPassword_ReturnsFalse(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)
	hash, _ := hasher.Hash("correct_password")

	result := hasher.Verify("wrong_password", hash)

	assert.False(t, result)
}

func TestBcryptHasher_Verify_InvalidHash_ReturnsFalse(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	result := hasher.Verify("password", "not_a_valid_bcrypt_hash")

	assert.False(t, result)
}

func TestBcryptHasher_Verify_EmptyHash_ReturnsFalse(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	result := hasher.Verify("password", "")

	assert.False(t, result)
}

func TestBcryptHasher_Verify_EmptyPassword_WithValidHash_ReturnsFalse(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)
	hash, _ := hasher.Hash("password")

	result := hasher.Verify("", hash)

	assert.False(t, result)
}

func TestBcryptHasher_Verify_EmptyPasswordHash_Matches(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)
	hash, _ := hasher.Hash("")

	result := hasher.Verify("", hash)

	assert.True(t, result, "Empty password should verify against its own hash")
}

// ============================================================================
// Cost Tests
// ============================================================================

func TestBcryptHasher_DifferentCosts_AllValidHashes(t *testing.T) {
	costs := []int{4, 10}
	password := "test_password"

	for _, cost := range costs {
		t.Run("cost_"+string(rune('0'+cost)), func(t *testing.T) {
			hasher := security.NewBcryptHasherWithCost(cost)

			hash, err := hasher.Hash(password)

			require.NoError(t, err)
			assert.True(t, hasher.Verify(password, hash))
		})
	}
}

// ============================================================================
// Cross-Hasher Verification Tests
// ============================================================================

func TestBcryptHasher_HashFromOneCost_VerifiesWithAnother(t *testing.T) {
	hasher1 := security.NewBcryptHasherWithCost(4)
	hasher2 := security.NewBcryptHasherWithCost(10)
	password := "cross_verify_password"

	hash, _ := hasher1.Hash(password)

	result := hasher2.Verify(password, hash)

	assert.True(t, result, "Hash from one hasher should verify with another")
}

// ============================================================================
// Table-Driven Tests
// ============================================================================

func TestBcryptHasher_Verify_TableDriven(t *testing.T) {
	hasher := security.NewBcryptHasherWithCost(4)

	tests := []struct {
		name           string
		password       string
		inputPassword  string
		expectedResult bool
	}{
		{"matching password", "password123", "password123", true},
		{"wrong password", "password123", "password456", false},
		{"case sensitive", "Password", "password", false},
		{"with spaces", "pass word", "pass word", true},
		{"spaces matter", "pass word", "password", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, _ := hasher.Hash(tt.password)
			result := hasher.Verify(tt.inputPassword, hash)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

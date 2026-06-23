// Package identity contains identity-related domain types.
// This is the core domain layer - no external dependencies allowed.
package identity

// IdentityType represents the type of identity (individual or company).
type IdentityType string

const (
	// IdentityTypeIndividual represents a natural person (pessoa física).
	IdentityTypeIndividual IdentityType = "individual"

	// IdentityTypeCompany represents a legal entity (pessoa jurídica).
	IdentityTypeCompany IdentityType = "company"
)

// IsValid checks if the identity type is valid.
func (t IdentityType) IsValid() bool {
	return t == IdentityTypeIndividual || t == IdentityTypeCompany
}

// String returns the string representation of the identity type.
func (t IdentityType) String() string {
	return string(t)
}

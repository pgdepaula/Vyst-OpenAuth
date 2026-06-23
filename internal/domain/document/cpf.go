// Package document contains validation logic for personal documents.
// This package is part of the core domain layer - no external dependencies allowed.
package document

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
)

// Domain errors for CPF operations.
var (
	// ErrCPFInvalid is returned when a CPF fails validation (length, check digits, or format).
	ErrCPFInvalid = errors.New("invalid CPF")

	// ErrCPFBlacklisted is returned when a CPF is in the blacklist or has all digits equal.
	ErrCPFBlacklisted = errors.New("CPF is blacklisted or invalid")
)

// CPF represents a valid Brazilian Individual Taxpayer Registry number.
// It is a value object that guarantees validity upon creation.
type CPF struct {
	number string // stored as 11 digits string
}

// NewCPF creates a new CPF Value Object.
// It validates the input string and returns an error if invalid.
func NewCPF(value string) (CPF, error) {
	if err := validateCPF(value); err != nil {
		return CPF{}, err
	}
	return CPF{number: normalizeCPF(value)}, nil
}

// String returns the formatted CPF (XXX.XXX.XXX-XX).
func (c CPF) String() string {
	return formatCPF(c.number)
}

// Value implements the driver.Valuer interface for database persistence.
// It returns the normalized string (digits only).
func (c CPF) Value() (driver.Value, error) {
	return c.number, nil
}

// Scan implements the sql.Scanner interface for database persistence.
func (c *CPF) Scan(value interface{}) error {
	if value == nil {
		*c = CPF{}
		return nil
	}
	s, ok := value.(string)
	if !ok {
		b, okB := value.([]byte)
		if !okB {
			return errors.New("invalid type for CPF")
		}
		s = string(b)
	}

	if s == "" {
		*c = CPF{}
		return nil
	}

	// We might store it normalized, but let's re-validate to be safe if DB was tempered
	newC, err := NewCPF(s)
	if err != nil {
		return err
	}
	*c = newC
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the CPF as a formatted string.
func (c CPF) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *CPF) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	newC, err := NewCPF(s)
	if err != nil {
		return err
	}
	*c = newC
	return nil
}

// Equals checks if two CPFs are equal.
func (c CPF) Equals(other CPF) bool {
	return c.number == other.number
}

// IsEmpty checks if the CPF is empty (zero value).
func (c CPF) IsEmpty() bool {
	return c.number == ""
}

// Mask returns a masked version of the CPF for display/logging.
// Format: XXX.XXX.***-**
func (c CPF) Mask() string {
	return MaskCPF(c.number)
}

// validateCPF validates a CPF using the official Brazilian algorithm (Mod 11).
// This is now an internal helper.
func validateCPF(cpf string) error {
	// Remove non-digit characters
	normalized := normalizeCPF(cpf)

	// Must have exactly 11 digits
	if len(normalized) != 11 {
		return ErrCPFInvalid
	}

	// Check for known invalid patterns (all same digit, e.g., 111.111.111-11)
	if isAllSameDigit(normalized) {
		return ErrCPFBlacklisted
	}

	// Validate first check digit (position 9)
	if !validateCheckDigit(normalized, 9) {
		return ErrCPFInvalid
	}

	// Validate second check digit (position 10)
	if !validateCheckDigit(normalized, 10) {
		return ErrCPFInvalid
	}

	return nil
}

// normalizeCPF removes all non-digit characters from a CPF string.
func normalizeCPF(cpf string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(cpf, "")
}

// formatCPF formats a CPF string to the standard format: XXX.XXX.XXX-XX
func formatCPF(normalized string) string {
	if len(normalized) != 11 {
		return normalized
	}
	return normalized[:3] + "." + normalized[3:6] + "." + normalized[6:9] + "-" + normalized[9:]
}

// Deprecated: ValidateCPF is exposed for backward compatibility during refactor,
// but should be avoided in favor of NewCPF.
func ValidateCPF(cpf string) error {
	return validateCPF(cpf)
}

// Deprecated: NormalizeCPF is exposed for backward compatibility.
func NormalizeCPF(cpf string) string {
	return normalizeCPF(cpf)
}

// Deprecated: FormatCPF is exposed for backward compatibility.
func FormatCPF(cpf string) string {
	return formatCPF(normalizeCPF(cpf))
}

// MaskCPF returns a masked version of the CPF for display/logging.
// Exposed as helper for cases where we only have a string.
func MaskCPF(cpf string) string {
	normalized := normalizeCPF(cpf)
	if len(normalized) != 11 {
		return "***.***.***-**"
	}
	return normalized[:3] + "." + normalized[3:6] + ".***-**"
}

// isAllSameDigit checks if all characters in the string are the same.
func isAllSameDigit(s string) bool {
	if len(s) == 0 {
		return true
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

// validateCheckDigit validates a specific check digit of the CPF.
func validateCheckDigit(cpf string, position int) bool {
	weightStart := position + 1
	sum := 0

	for i := 0; i < position; i++ {
		digit, err := strconv.Atoi(string(cpf[i]))
		if err != nil {
			return false
		}
		sum += digit * (weightStart - i)
	}

	remainder := sum % 11

	expectedDigit := 0
	if remainder >= 2 {
		expectedDigit = 11 - remainder
	}

	actualDigit, err := strconv.Atoi(string(cpf[position]))
	if err != nil {
		return false
	}

	return actualDigit == expectedDigit
}

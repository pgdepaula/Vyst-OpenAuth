// Package company contains the Company domain entity and repository interface.
// This file contains CNPJ validation logic - a Brazilian company registration number.
package company

import (
	"regexp"
	"strconv"
)

// ValidateCNPJ validates a CNPJ using the official Brazilian algorithm.
// A CNPJ consists of 14 digits: XX.XXX.XXX/XXXX-XX
// The last two digits are check digits calculated using a weighted sum algorithm.
//
// Returns true if the CNPJ is valid, false otherwise.
func ValidateCNPJ(cnpj string) bool {
	// Remove non-digit characters
	cnpj = NormalizeCNPJ(cnpj)

	// Must have exactly 14 digits
	if len(cnpj) != 14 {
		return false
	}

	// Check for known invalid patterns (all same digit)
	if isAllSameDigit(cnpj) {
		return false
	}

	// Validate first check digit (position 12)
	if !validateCheckDigit(cnpj, 12) {
		return false
	}

	// Validate second check digit (position 13)
	return validateCheckDigit(cnpj, 13)
}

// NormalizeCNPJ removes all non-digit characters from a CNPJ string.
// Returns only the 14 digits.
func NormalizeCNPJ(cnpj string) string {
	re := regexp.MustCompile(`\D`)
	return re.ReplaceAllString(cnpj, "")
}

// FormatCNPJ formats a CNPJ string to the standard format: XX.XXX.XXX/XXXX-XX
// If the input is not valid (not 14 digits after normalization), returns the input unchanged.
func FormatCNPJ(cnpj string) string {
	normalized := NormalizeCNPJ(cnpj)
	if len(normalized) != 14 {
		return cnpj
	}
	return normalized[:2] + "." + normalized[2:5] + "." + normalized[5:8] + "/" + normalized[8:12] + "-" + normalized[12:]
}

// MaskCNPJ returns a masked version of the CNPJ for display/logging.
// Format: XX.XXX.XXX/****-**
func MaskCNPJ(cnpj string) string {
	normalized := NormalizeCNPJ(cnpj)
	if len(normalized) != 14 {
		return "**.***.****/****-**"
	}
	return normalized[:2] + "." + normalized[2:5] + "." + normalized[5:8] + "/****-**"
}

// isAllSameDigit checks if all characters in the string are the same.
// Used to reject invalid CNPJs like 00000000000000, 11111111111111, etc.
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

// validateCheckDigit validates a specific check digit of the CNPJ.
// position is 12 for the first check digit, 13 for the second.
//
// The algorithm uses a weighted sum where:
// - For position 12: weights are [5,4,3,2,9,8,7,6,5,4,3,2]
// - For position 13: weights are [6,5,4,3,2,9,8,7,6,5,4,3,2]
//
// The sum is divided by 11, and the check digit is:
// - 0 if remainder < 2
// - 11 - remainder otherwise
func validateCheckDigit(cnpj string, position int) bool {
	var weights []int
	if position == 12 {
		weights = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	} else {
		weights = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	}

	sum := 0
	for i := 0; i < position; i++ {
		digit, err := strconv.Atoi(string(cnpj[i]))
		if err != nil {
			return false
		}
		sum += digit * weights[i]
	}

	remainder := sum % 11
	expectedDigit := 0
	if remainder >= 2 {
		expectedDigit = 11 - remainder
	}

	actualDigit, err := strconv.Atoi(string(cnpj[position]))
	if err != nil {
		return false
	}

	return actualDigit == expectedDigit
}

// KnownInvalidCNPJs is a list of CNPJs that are technically valid but should be rejected.
// These are test/placeholder CNPJs that should not be used in production.
var KnownInvalidCNPJs = []string{
	"00000000000000",
	"11111111111111",
	"22222222222222",
	"33333333333333",
	"44444444444444",
	"55555555555555",
	"66666666666666",
	"77777777777777",
	"88888888888888",
	"99999999999999",
}

// IsBlacklistedCNPJ checks if a CNPJ is in the known invalid list.
func IsBlacklistedCNPJ(cnpj string) bool {
	normalized := NormalizeCNPJ(cnpj)
	for _, invalid := range KnownInvalidCNPJs {
		if normalized == invalid {
			return true
		}
	}
	return false
}

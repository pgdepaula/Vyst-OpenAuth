package ports

import (
	"context"
	"time"
)

// DocumentVerificationResult contains the result of a document verification against an external API.
type DocumentVerificationResult struct {
	Valid     bool      `json:"valid"`
	Name      string    `json:"name,omitempty"`
	Situation string    `json:"situation,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// DocumentVerificationPort defines the interface for external document verification services (e.g. Serpro).
type DocumentVerificationPort interface {
	// VerifyCPF verifies a CPF against an official database.
	VerifyCPF(ctx context.Context, cpf string) (*DocumentVerificationResult, error)
}

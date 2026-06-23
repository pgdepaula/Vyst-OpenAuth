package document

import (
	"context"
	"time"
)

// VerificationRecord represents a log of a document verification attempt.
type VerificationRecord struct {
	ID        string    `json:"id"`
	Document  string    `json:"document"` // Masked or Hashed
	Type      string    `json:"type"`     // CPF, CNPJ
	Source    string    `json:"source"`   // API Check, Offline
	Valid     bool      `json:"valid"`
	Metadata  string    `json:"metadata"` // JSON with details
	CreatedAt time.Time `json:"created_at"`
}

// VerificationRepository defines the persistence layer for document verifications.
type VerificationRepository interface {
	Save(ctx context.Context, record *VerificationRecord) error
	GetByDocument(ctx context.Context, documentType, documentValue string) ([]*VerificationRecord, error)
}

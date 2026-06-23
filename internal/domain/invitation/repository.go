package invitation

import "context"

type Repository interface {
	Create(ctx context.Context, inv *Invitation) error
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	GetByEmailAndCompany(ctx context.Context, email, companyID string) (*Invitation, error) // To prevent duplicate invites
	Update(ctx context.Context, inv *Invitation) error
	ListByCompany(ctx context.Context, companyID string) ([]*Invitation, error)
}

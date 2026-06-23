package handlers

import (
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
)

// UserResponse is the DTO for sending user data over HTTP.
// It decouples the internal domain model from the external API contract.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	TenantID  string    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToUserResponse converts a domain User entity to a UserResponse DTO.
func ToUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		TenantID:  u.TenantID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// Package handlers contains HTTP request handlers.
// Contains Swagger documentation annotations for all API endpoints.
package handlers

// ErrorResponse represents an error response.
// @Description Error response returned by the API
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

// MessageResponse represents a success message response.
// @Description Simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

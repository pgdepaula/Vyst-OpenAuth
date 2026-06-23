package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/redis"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
)

type WebAuthnHandler struct {
	service      *service.WebAuthnService
	authService  *service.AuthService
	sessionStore *redis.SessionStore
}

func NewWebAuthnHandler(service *service.WebAuthnService, authService *service.AuthService, sessionStore *redis.SessionStore) *WebAuthnHandler {
	return &WebAuthnHandler{
		service:      service,
		authService:  authService,
		sessionStore: sessionStore,
	}
}

// BeginRegistration initiates the passkey registration ceremony.
func (h *WebAuthnHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	// Get UserID from context (must be authenticated)
	userIDStr := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	options, session, err := h.service.BeginRegistration(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save session to Redis with a short TTL (e.g., 5 minutes)
	// Key: userID (assuming one active ceremony per user, or use a random ID)
	// Using userID is simpler but prevents concurrent registrations. Let's use userID for now.
	if err := h.sessionStore.SaveSession(r.Context(), "reg:"+userIDStr, session, 5*time.Minute); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, options)
}

// FinishRegistration completes the passkey registration ceremony.
func (h *WebAuthnHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	// Retrieve session from Redis
	session, err := h.sessionStore.GetSession(r.Context(), "reg:"+userIDStr)
	if err != nil {
		http.Error(w, "Session expired or invalid", http.StatusBadRequest)
		return
	}

	if err := h.service.FinishRegistration(r.Context(), userID, *session, r); err != nil {
		http.Error(w, "Registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// BeginLogin initiates the passkey login ceremony.
func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	options, session, userID, err := h.service.BeginLoginByEmail(r.Context(), req.Email)
	if err != nil {
		// Don't reveal user existence? For WebAuthn, we kinda have to if we return options.
		// But we can return generic error.
		http.Error(w, "Login failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Save session to Redis
	sessionKey := "login:" + userID.String()
	if err := h.sessionStore.SaveSession(r.Context(), sessionKey, session, 5*time.Minute); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Set cookie so client can send it back in FinishLogin
	http.SetCookie(w, &http.Cookie{
		Name:     "login_user_id",
		Value:    userID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Dev
		MaxAge:   300,
	})

	writeJSON(w, http.StatusOK, options)
}

// FinishLogin completes the passkey login ceremony and issues a JWT.
func (h *WebAuthnHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	// Get UserID from cookie
	cookie, err := r.Cookie("login_user_id")
	if err != nil {
		http.Error(w, "Missing login session", http.StatusBadRequest)
		return
	}
	userIDStr := cookie.Value
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Retrieve session
	session, err := h.sessionStore.GetSession(r.Context(), "login:"+userIDStr)
	if err != nil {
		http.Error(w, "Session expired or invalid", http.StatusBadRequest)
		return
	}

	_, err = h.service.FinishLogin(r.Context(), userID, *session, r)
	if err != nil {
		http.Error(w, "Login failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Login successful - issue JWT token
	tokenPair, err := h.authService.GenerateTokenForUser(r.Context(), userIDStr)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Clear login cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "login_user_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, LoginResponse{
		Token:     tokenPair.AccessToken,
		ExpiresIn: tokenPair.ExpiresIn,
	})
}

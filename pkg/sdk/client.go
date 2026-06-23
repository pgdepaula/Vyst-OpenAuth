// Package sdk provides a Go client for the Vyst Identity API.
// It handles authentication, automatic token refresh, and permission checks.
//
// Usage:
//
//	client := sdk.NewClient("https://auth.vyst.com.br")
//	err := client.Login(ctx, "user@example.com", "password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check permissions
//	allowed, err := client.Can(ctx, userID, "edit", "invoice")
//	if allowed {
//	    // proceed
//	}
//
//	// Make authenticated requests
//	resp, err := client.Get(ctx, "/api/v1/users")
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is the Vyst Identity SDK client.
type Client struct {
	baseURL      string
	httpClient   *http.Client
	accessToken  string
	refreshToken string
	expiresAt    time.Time
	mu           sync.RWMutex

	// Options
	autoRefresh    bool
	refreshBuffer  time.Duration
	onTokenRefresh func(accessToken, refreshToken string)
}

// Option is a function that configures the Client.
type Option func(*Client)

// NewClient creates a new Vyst Identity client.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		autoRefresh:   true,
		refreshBuffer: 5 * time.Minute, // Refresh 5 min before expiry
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithAutoRefresh enables/disables automatic token refresh.
func WithAutoRefresh(enabled bool) Option {
	return func(c *Client) {
		c.autoRefresh = enabled
	}
}

// WithRefreshBuffer sets the time before expiry to refresh tokens.
func WithRefreshBuffer(d time.Duration) Option {
	return func(c *Client) {
		c.refreshBuffer = d
	}
}

// WithOnTokenRefresh sets a callback for when tokens are refreshed.
// Useful for persisting new tokens.
func WithOnTokenRefresh(fn func(accessToken, refreshToken string)) Option {
	return func(c *Client) {
		c.onTokenRefresh = fn
	}
}

// WithTokens sets initial tokens (for resuming sessions).
func WithTokens(accessToken, refreshToken string) Option {
	return func(c *Client) {
		c.accessToken = accessToken
		c.refreshToken = refreshToken
	}
}

// LoginResponse is the response from the login endpoint.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Login authenticates with email and password.
func (c *Client) Login(ctx context.Context, email, password string) error {
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/auth/login", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	c.mu.Lock()
	c.accessToken = loginResp.AccessToken
	c.refreshToken = loginResp.RefreshToken
	c.expiresAt = time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)
	c.mu.Unlock()

	return nil
}

// RefreshIfNeeded refreshes the access token if it's about to expire.
func (c *Client) RefreshIfNeeded(ctx context.Context) error {
	c.mu.RLock()
	needsRefresh := time.Now().Add(c.refreshBuffer).After(c.expiresAt)
	refreshToken := c.refreshToken
	c.mu.RUnlock()

	if !needsRefresh || refreshToken == "" {
		return nil
	}

	return c.doRefresh(ctx, refreshToken)
}

func (c *Client) doRefresh(ctx context.Context, refreshToken string) error {
	body := fmt.Sprintf(`{"refresh_token":"%s"}`, refreshToken)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/auth/refresh", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	c.mu.Lock()
	c.accessToken = loginResp.AccessToken
	c.refreshToken = loginResp.RefreshToken
	c.expiresAt = time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)
	c.mu.Unlock()

	// Notify callback
	if c.onTokenRefresh != nil {
		c.onTokenRefresh(loginResp.AccessToken, loginResp.RefreshToken)
	}

	return nil
}

// PermissionCheckResponse is the response from the permission check endpoint.
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// Can checks if a user has permission to perform an action on a resource.
// This uses the ReBAC policy engine.
//
// Example:
//
//	allowed, err := client.Can(ctx, userID, "edit", "invoice:123")
func (c *Client) Can(ctx context.Context, userID, action, resource string) (bool, error) {
	if c.autoRefresh {
		if err := c.RefreshIfNeeded(ctx); err != nil {
			return false, fmt.Errorf("refresh token: %w", err)
		}
	}

	body := fmt.Sprintf(`{"user_id":"%s","action":"%s","resource":"%s"}`, userID, action, resource)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/authz/check", strings.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.getAccessToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("permission check request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("permission check failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var checkResp PermissionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}

	return checkResp.Allowed, nil
}

// Claims represents JWT claims.
type Claims struct {
	UserID   string   `json:"sub"`
	TenantID string   `json:"tenant_id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Exp      int64    `json:"exp"`
}

// ValidateToken validates a token and returns its claims.
func (c *Client) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/auth/introspect", strings.NewReader(fmt.Sprintf(`{"token":"%s"}`, token)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspect request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("introspect failed: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var claims Claims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &claims, nil
}

// Get makes an authenticated GET request.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, "GET", path, nil)
}

// Post makes an authenticated POST request.
func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.Do(ctx, "POST", path, body)
}

// Do makes an authenticated request.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if c.autoRefresh {
		if err := c.RefreshIfNeeded(ctx); err != nil {
			return nil, fmt.Errorf("refresh token: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.getAccessToken())
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// getAccessToken returns the current access token (thread-safe).
func (c *Client) getAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// GetAccessToken returns the current access token.
func (c *Client) GetAccessToken() string {
	return c.getAccessToken()
}

// GetRefreshToken returns the current refresh token.
func (c *Client) GetRefreshToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken
}

// IsAuthenticated returns true if the client has a valid (non-expired) token.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken != "" && time.Now().Before(c.expiresAt)
}

// Logout clears the current session.
func (c *Client) Logout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.accessToken = ""
	c.refreshToken = ""
	c.expiresAt = time.Time{}
}

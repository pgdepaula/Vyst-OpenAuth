// Package middleware contains HTTP middleware functions.
package middleware

import (
	"net/http"
)

// SecurityHeaders adds security headers to all responses.
// This helps prevent common web vulnerabilities.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS - Enforce HTTPS
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS Protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		// Allow same-origin, allow SSE connections, allow inline scripts for Vue.js
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (previously Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

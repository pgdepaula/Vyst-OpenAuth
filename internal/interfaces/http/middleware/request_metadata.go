package middleware

import (
	"context"
	"net/http"
	"strings"
)

// RequestMetadata is a middleware that extracts request metadata (UserAgent, IP)
// and places it into the context for downstream use.
func RequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract User-Agent
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			userAgent = "unknown"
		}
		ctx = context.WithValue(ctx, UserAgentKey, userAgent)

		// Extract IP Address (handles X-Forwarded-For, X-Real-IP, and RemoteAddr)
		ipAddress := getClientIP(r)
		ctx = context.WithValue(ctx, IPAddressKey, ipAddress)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getClientIP extracts the real client IP from the request.
// It checks X-Forwarded-For, X-Real-IP headers, and falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (common for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP (used by some proxies like Nginx)
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (remove port if present)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		// Handle IPv6 addresses enclosed in brackets
		if strings.Contains(addr, "[") {
			if bracketEnd := strings.LastIndex(addr, "]"); bracketEnd != -1 {
				return addr[1:bracketEnd] // Remove brackets
			}
		}
		return addr[:idx]
	}
	return addr
}

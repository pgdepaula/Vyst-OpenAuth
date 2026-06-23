package middleware

import (
	"net/http"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
)

// RequiredCompanyRole middleware checks if the user has the required role in the active company context.
// It assumes Auth middleware has already run and populated context with claims.
func RequiredCompanyRole(requiredRole company.CompanyRole, tokenSvc ports.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ClaimsKey).(*ports.Claims)
			if !ok {
				http.Error(w, "Unauthorized: No claims found", http.StatusUnauthorized)
				return
			}

			if claims.ActiveCompanyID == "" {
				http.Error(w, "Forbidden: No active company context", http.StatusForbidden)
				return
			}

			if claims.CompanyRole == "" {
				http.Error(w, "Forbidden: No role in active company", http.StatusForbidden)
				return
			}

			// Validate role hierarchy (simplistic for now)
			// Admin > Member > Viewer
			// If required is admin, user must be admin
			// If required is member, user must be member or admin
			// If required is viewer, user must be viewer, member or admin

			userRole := company.CompanyRole(claims.CompanyRole)

			if !hasSufficientRole(userRole, requiredRole) {
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasSufficientRole(userRole, requiredRole company.CompanyRole) bool {
	if userRole == company.RoleAdmin {
		return true // Admin can do anything
	}
	if requiredRole == company.RoleAdmin {
		return false // Only Admin can be Admin
	}

	if userRole == company.RoleMember {
		// Member can satisfy Member and Viewer
		return requiredRole == company.RoleMember || requiredRole == company.RoleViewer
	}

	if userRole == company.RoleViewer {
		return requiredRole == company.RoleViewer
	}

	return false
}

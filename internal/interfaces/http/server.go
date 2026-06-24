// Package http provides the HTTP server setup using Chi router.
package http

import (
	gohttp "net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/company"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/policy"
	"github.com/pgdepaula/vyst-openauth/internal/domain/user"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/graphql"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/graphql/resolvers"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/redis/go-redis/v9"
)

// ServerConfig holds the configuration for the HTTP server.
type ServerConfig struct {
	TokenService ports.TokenService
	RedisClient  *redis.Client
	DBPool       *pgxpool.Pool
	RateLimit    int
	RateWindow   time.Duration
}

// NewRouter creates and configures the Chi router with all routes.
func NewRouter(
	registrationSvc *service.RegistrationService,
	authSvc *service.AuthService,
	totpSvc *service.TOTPService,
	captchaSvc ports.CaptchaService,
	apiKeySvc *service.APIKeyService,
	companySvc *service.CompanyService,
	companyLookupSvc *service.CompanyLookupService,
	policySvc *service.PolicyService,
	webAuthnHandler *handlers.WebAuthnHandler,
	passwordHandler *handlers.PasswordHandler,
	policyHandler *handlers.PolicyHandler,
	tenantHandler *handlers.TenantHandler,
	apiKeyHandler *handlers.APIKeyHandler,
	companyHandler *handlers.CompanyHandler,
	documentHandler *handlers.DocumentHandler,
	userRepo user.Repository,
	policyRepo policy.RoleRepository,
	companyRepo company.Repository,
	eventBus event.Bus,
	cfg ServerConfig,
) *chi.Mux {
	// ... existing setup ...

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)  // RequestID must come before Logger
	r.Use(middleware.RequestLogger) // Use our custom structured logger
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)

	// Security headers middleware
	r.Use(middleware.SecurityHeaders)

	// Rate limiting
	if cfg.RedisClient != nil {
		rateLimiter := middleware.NewRateLimiter(cfg.RedisClient, cfg.RateLimit, cfg.RateWindow)
		r.Use(rateLimiter.Middleware)
	}

	// Initialize handlers
	// Quota Enforcer (needed for AuthHandler)
	var quotaEnforcer *middleware.QuotaEnforcer
	if cfg.RedisClient != nil {
		quotaEnforcer = middleware.NewQuotaEnforcer(cfg.RedisClient)
	}

	authHandler := handlers.NewAuthHandler(registrationSvc, authSvc, totpSvc, captchaSvc, quotaEnforcer)
	totpHandler := handlers.NewTOTPHandler(totpSvc, authSvc)
	healthHandler := handlers.NewHealthHandler()

	// SSE and Stats handlers (for real-time dashboard)
	var sseHandler *handlers.SSEHandler
	var statsHandler *handlers.StatsHandler
	if cfg.RedisClient != nil {
		sseHandler = handlers.NewSSEHandler(cfg.RedisClient)
		if cfg.DBPool != nil {
			statsHandler = handlers.NewStatsHandler(cfg.DBPool, cfg.RedisClient)
		}
	}

	// Quota Handler
	var quotaHandler *middleware.QuotaHandler
	if quotaEnforcer != nil {
		quotaHandler = middleware.NewQuotaHandler(quotaEnforcer)
	}

	// Health routes (no auth)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Public auth routes
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/forgot-password", passwordHandler.ForgotPassword)
	r.Post("/auth/reset-password", passwordHandler.ResetPassword)
	r.Get("/auth/verify-email", authHandler.VerifyEmail)
	r.Post("/auth/refresh", authHandler.RefreshToken)
	r.Post("/auth/logout", authHandler.Logout)

	// WebAuthn Public Routes (Login)
	r.Post("/auth/passkeys/login/begin", webAuthnHandler.BeginLogin)
	r.Post("/auth/passkeys/login/finish", webAuthnHandler.FinishLogin)

	// API v1 Routes
	r.Route("/api/v1", func(r chi.Router) {
		// SSE Events Stream (real-time dashboard)
		if sseHandler != nil {
			r.Get("/events/stream", sseHandler.StreamEvents)
		}

		// Stats endpoint
		if statsHandler != nil {
			r.Get("/stats", statsHandler.GetStats)
		}

		// Quota/Billing endpoints
		if quotaHandler != nil {
			r.Get("/plans", quotaHandler.GetPlans)
			r.Get("/tenants/{id}/usage", quotaHandler.GetUsage)
		}

		// Public Document Routes
		if documentHandler != nil {
			r.Post("/documents/validate-cpf", documentHandler.ValidateCPF)
		}

		// Protected API routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.TokenService, apiKeySvc))

			// API Keys Management
			r.Post("/api-keys", apiKeyHandler.CreateAPIKey)
			r.Get("/api-keys", apiKeyHandler.ListAPIKeys)
			r.Delete("/api-keys/{id}", apiKeyHandler.RevokeAPIKey)

			// Policy/Role Management
			r.Get("/roles", policyHandler.ListRoles)
			r.Post("/roles", policyHandler.CreateRole)
			r.Get("/roles/{id}", policyHandler.GetRole)
			r.Put("/roles/{id}", policyHandler.UpdateRole)
			r.Delete("/roles/{id}", policyHandler.DeleteRole)

			// Tenant Management (Onboarding)
			r.Post("/tenants", tenantHandler.CreateTenant)

			// Company Management
			if companyHandler != nil {
				r.Get("/companies/lookup", companyHandler.LookupCompany)
				r.Post("/companies", companyHandler.CreateCompany)
				r.Get("/companies", companyHandler.ListCompanies)
				r.Get("/companies/{id}", companyHandler.GetCompany)
				r.Post("/companies/{id}/users", companyHandler.AddUserToCompany)
				r.Delete("/companies/{id}/users/{userId}", companyHandler.RemoveUserFromCompany)

				// Invitations
				r.Post("/companies/{id}/invitations", companyHandler.InviteUser)
				r.Post("/invitations/{token}/accept", companyHandler.AcceptInvitation)

				// Join Requests & Approval
				r.Post("/companies/{id}/join-requests", companyHandler.RequestJoin)
				r.Post("/companies/{id}/members/{userId}/approve", companyHandler.ApproveMember)
				r.Post("/companies/{id}/members/{userId}/reject", companyHandler.RejectMember)

				// Company Context Switching
				r.Post("/auth/switch-company", companyHandler.SwitchCompany)
				r.Delete("/auth/company-context", companyHandler.ClearCompanyContext)
			}

			// Super Admin Routes (should be protected by role check)
			r.Get("/admin/tenants", tenantHandler.ListTenants)
			r.Post("/admin/tenants/{id}/suspend", tenantHandler.SuspendTenant)
		})

		// Authz/Permission Check (Protected)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.TokenService, apiKeySvc))
			r.Post("/authz/check", policyHandler.CheckPermission)
		})

		// Token introspection — public, no bearer required.
		// The submitted token is the subject being validated, not the caller credential.
		r.Post("/auth/introspect", authHandler.IntrospectToken)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.TokenService, apiKeySvc))
		r.Get("/auth/me", authHandler.Me)

		// WebAuthn Protected Routes (Registration)
		r.Post("/auth/passkeys/register/begin", webAuthnHandler.BeginRegistration)
		r.Post("/auth/passkeys/register/finish", webAuthnHandler.FinishRegistration)

		// 2FA Routes (Protected)
		if totpHandler != nil {
			r.Post("/auth/2fa/setup", totpHandler.Setup)
			r.Post("/auth/2fa/verify", totpHandler.Verify)
			r.Get("/auth/2fa/status", totpHandler.Status)
			r.Delete("/auth/2fa", totpHandler.Disable)
		}
	})

	// Public CAPTCHA config endpoint
	r.Get("/auth/captcha-config", authHandler.GetCaptchaSiteKey)

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// GraphQL
	srv := handler.NewDefaultServer(graphql.NewExecutableSchema(graphql.Config{Resolvers: &resolvers.Resolver{
		AuthService:    authSvc,
		PolicyService:  policySvc,
		CompanyService: companySvc,
		UserRepo:       userRepo,
		PolicyRepo:     policyRepo,
		CompanyRepo:    companyRepo,
		EventBus:       eventBus,
	}}))

	r.Handle("/query", srv)
	r.Handle("/playground", playground.Handler("GraphQL playground", "/query"))

	// UI Routes (Angular SPA) - Must be last!
	// Serve static files and handle SPA routing
	spaPath := "./web/vyst-ui/dist/vyst-ui/browser"
	spaBaseAbs, err := filepath.Abs(spaPath)
	if err != nil {
		panic(err)
	}
	r.Get("/*", func(w gohttp.ResponseWriter, r *gohttp.Request) {
		// Check if file exists in static directory
		reqPath := filepath.Clean("/" + r.URL.Path)
		relPath := strings.TrimPrefix(reqPath, "/")
		candidatePath, err := filepath.Abs(filepath.Join(spaBaseAbs, relPath))
		if err == nil && (candidatePath == spaBaseAbs || strings.HasPrefix(candidatePath, spaBaseAbs+string(os.PathSeparator))) {
			if info, statErr := os.Stat(candidatePath); statErr == nil && !info.IsDir() {
				gohttp.ServeFile(w, r, candidatePath)
				return
			}
		}

		// Otherwise serve index.html for SPA routing
		gohttp.ServeFile(w, r, filepath.Join(spaBaseAbs, "index.html"))
	})

	return r
}

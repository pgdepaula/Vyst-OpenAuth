package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	gohttp "net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/joho/godotenv"
	pb "github.com/pgdepaula/vyst-openauth/api/proto"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/config"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/eventbus"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/external/serpro"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/logger"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/messaging/outbox"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/notification/plivo"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/notification/smtp"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
	redispersistence "github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/redis"
	providerbrasilapi "github.com/pgdepaula/vyst-openauth/internal/infrastructure/provider/brasilapi"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/resilience"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/security"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/telemetry"
	internalgrpc "github.com/pgdepaula/vyst-openauth/internal/interfaces/grpc"
	internalhttp "github.com/pgdepaula/vyst-openauth/internal/interfaces/http"
	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/handlers"
	"github.com/pgdepaula/vyst-openauth/migrations"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

//nolint:gocyclo // Startup wiring is intentionally explicit to keep boot order auditable.
func main() {

	// 0. Initialize Logger
	appLogger := logger.New(logger.Config{
		Level:  "debug",
		Format: "json",
	})
	logger.SetGlobal(appLogger)

	// 1. Load .env file (highest priority for local overrides)
	if err := godotenv.Load(); err != nil {
		appLogger.Info("No .env file found")
	} else {
		appLogger.Info("✓ Loaded .env file")
	}

	// 2. Load environment-specific defaults (does not overwrite existing vars)
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	envFile := ".env." + env
	if err := godotenv.Load(envFile); err != nil {
		appLogger.Info(fmt.Sprintf("No %s file found", envFile))
	} else {
		appLogger.Info(fmt.Sprintf("✓ Loaded %s file", envFile))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		appLogger.Error(fmt.Sprintf("Failed to load config: %v", err))
		os.Exit(1)
	}
	appLogger.Info(fmt.Sprintf("✓ Config loaded (Port: %s)", cfg.Port))

	// 2. Initialize Telemetry
	shutdown, err := telemetry.InitTracer(context.Background(), "identity-api", cfg.EnableTelemetry)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Failed to init tracer: %v", err))
	} else {
		defer func() {
			if err := shutdown(context.Background()); err != nil {
				appLogger.Error(fmt.Sprintf("Failed to shutdown tracer: %v", err))
			}
		}()
	}

	// 2.5 Run Migrations
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		appLogger.Error(fmt.Sprintf("Failed to run migrations: %v", err))
		os.Exit(1)
	}

	// 3. Infrastructure
	// Database
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Failed to connect to database: %v", err))
		os.Exit(1)
	}
	defer db.Close()

	// Redis
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Failed to parse Redis URL: %v", err))
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpt)

	// Enable tracing
	if cfg.EnableTelemetry {
		if err := redisotel.InstrumentTracing(redisClient); err != nil {
			appLogger.Error(fmt.Sprintf("Failed to instrument Redis: %v", err))
			os.Exit(1)
		}
	}

	// Security
	tokenSvc, err := security.NewTokenService(cfg.JWTPrivateKey, cfg.JWTPublicKey, "vyst-identity")
	if err != nil {
		appLogger.Error(fmt.Sprintf("Failed to initialize token service: %v", err))
		os.Exit(1)
	}
	hasher := security.NewBcryptHasher()

	// Event Bus (in-memory for real-time events)
	eventBus := eventbus.NewInMemoryBus()

	// Repositories
	userRepo := postgres.NewUserRepository(db.Pool)
	tenantRepo := postgres.NewTenantRepository(db.Pool)

	// Policy Repo with Caching (Decorator)
	pgPolicyRepo := postgres.NewPolicyRepository(db.Pool)
	policyRepo := redispersistence.NewCachedPolicyRepository(pgPolicyRepo, redisClient, 30*time.Second)

	webAuthnRepo := postgres.NewWebAuthnRepository(db.Pool)
	outboxPub := outbox.NewPublisher(db.Pool)

	// Transaction Manager
	tm := postgres.NewTransactionManager(db.Pool)

	// Notification Adapters
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := "no-reply@auth.vyst.com.br"

	plivoAuthID := os.Getenv("PLIVO_AUTH_ID")
	plivoAuthToken := os.Getenv("PLIVO_AUTH_TOKEN")
	plivoSource := "Vyst"

	smtpAdapter := smtp.NewSMTPAdapter(smtpHost, smtpPort, smtpUser, smtpPass, smtpFrom)
	plivoAdapter := plivo.NewPlivoAdapter(plivoAuthID, plivoAuthToken, plivoSource)

	notificationSvc := service.NewNotificationService(smtpAdapter, plivoAdapter)

	// Role Repository
	roleRepo := postgres.NewRoleRepository(db.Pool)

	companyRepo := postgres.NewCompanyRepository(db.Pool)
	pgCompanyUserRepo := postgres.NewCompanyUserRepository(db.Pool)
	companyUserRepo := redispersistence.NewCachedCompanyUserRepository(pgCompanyUserRepo, redisClient, 5*time.Minute, appLogger.Logger)
	invitationRepo := postgres.NewInvitationRepository(db.Pool)
	webhookRepo := postgres.NewWebhookRepository(db.Pool)
	auditRepo := postgres.NewAuditRepository(db.Pool)
	verificationRepo := postgres.NewVerificationRepository(db.Pool)

	// 3. Initialize Application Services
	// Note: We pass the cached policyRepo to services
	policySvc := service.NewPolicyService(roleRepo, policyRepo)
	tenantSvc := service.NewTenantService(tenantRepo, userRepo, policyRepo)
	// Document Verification
	serproConfig := serpro.SerproConfig{
		BaseURL: os.Getenv("SERPRO_API_URL"),
		APIKey:  os.Getenv("SERPRO_API_KEY"),
	}
	serproAdapter := serpro.NewSerproAdapter(serproConfig)

	// Decorator Chain: Service -> Cache -> CircuitBreaker -> Metrics -> Adapter
	// 1. Metrics (inner-most, measures adapter directly)
	metricAdapter, err := telemetry.NewMetricDocumentVerificationPort(serproAdapter)
	if err != nil {
		appLogger.Error("Failed to init metrics adapter", "error", err)
		// non-fatal, proceed with raw adapter if metric fails (unlikely)
	}
	var decoratedAdapter ports.DocumentVerificationPort = serproAdapter
	if metricAdapter != nil {
		decoratedAdapter = metricAdapter
	}

	// 2. Circuit Breaker
	cbAdapter := resilience.NewCircuitBreakerDocumentVerificationPort(decoratedAdapter, "serpro-cpf", appLogger)

	// 3. Cache (outer-most, avoids calls if cached)
	cachedDocumentAdapter := redispersistence.NewCachedDocumentVerificationPort(cbAdapter, redisClient, 24*time.Hour, appLogger)

	documentSvc := service.NewDocumentService(appLogger, cachedDocumentAdapter, verificationRepo)
	registrationSvc := service.NewRegistrationService(tm, userRepo, tenantRepo, policyRepo, hasher, outboxPub, eventBus, notificationSvc, documentSvc)
	// Session Store
	sessionStore := redispersistence.NewSessionStore(redisClient)

	// Auth Service
	authSvc := service.NewAuthService(userRepo, policyRepo, companyUserRepo, sessionStore, hasher, tokenSvc, notificationSvc, appLogger)
	passwordSvc := service.NewPasswordService(userRepo, notificationSvc, hasher, cfg.FrontendURL)
	_ = service.NewOAuthService(userRepo, tokenSvc) // Initialized but currently unused in handlers

	webAuthnSvc, err := service.NewWebAuthnService(
		userRepo,
		webAuthnRepo,
		cfg.WebAuthnRPID,
		cfg.WebAuthnOrigin,
		cfg.WebAuthnRPName,
	)
	if err != nil {
		log.Fatalf("Failed to initialize WebAuthn service: %v", err)
	}

	// TOTP Service (2FA)
	totpRepo := postgres.NewTOTPRepository(db.Pool)
	totpSvc := service.NewTOTPService(totpRepo, sessionStore, cfg.TOTPIssuer)

	// CAPTCHA Service (Cloudflare Turnstile)
	captchaSvc := security.NewTurnstileService(cfg.TurnstileSiteKey, cfg.TurnstileSecretKey, appLogger)

	// Handlers
	documentHandler := handlers.NewDocumentHandler(documentSvc)
	webAuthnHandler := handlers.NewWebAuthnHandler(webAuthnSvc, authSvc, sessionStore)
	passwordHandler := handlers.NewPasswordHandler(passwordSvc)
	policyHandler := handlers.NewPolicyHandler(policySvc)
	tenantHandler := handlers.NewTenantHandler(tenantSvc)

	// API Keys
	apiKeyRepo := postgres.NewPostgresAPIKeyRepository(db.Pool)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, hasher)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeySvc)

	// Company Service

	// External Adapters
	lookupAPIAdapter := providerbrasilapi.NewBrasilAPIAdapter(nil, cfg.BrasilAPIURL)
	companyInfoRepo := postgres.NewCompanyInfoRepository(db.Pool)
	companyLookupSvc := service.NewCompanyLookupService(companyInfoRepo, []ports.CompanyDataPort{lookupAPIAdapter}, appLogger, eventBus, nil, nil)

	companySvc := service.NewCompanyService(tm, companyRepo, companyUserRepo, userRepo, eventBus, outboxPub, companyLookupSvc, appLogger)
	invitationSvc := service.NewInvitationService(invitationRepo, userRepo, companyRepo, companyUserRepo, notificationSvc, appLogger)
	webhookSvc := service.NewWebhookService(webhookRepo, eventBus, appLogger)
	_ = webhookSvc // Keep alive via event subscription
	auditSvc := service.NewAuditService(auditRepo, eventBus, appLogger)
	_ = auditSvc

	companyHandler := handlers.NewCompanyHandler(companySvc, companyLookupSvc, authSvc, invitationSvc)

	// 4. Setup gRPC Server (S2S token validation and RBAC)
	grpcServer := grpc.NewServer()
	identityGrpcServer := internalgrpc.NewServer(tokenSvc, policyRepo, companyUserRepo, appLogger)
	pb.RegisterIdentityServiceServer(grpcServer, identityGrpcServer)

	// Enable gRPC Server Reflection for runtime introspection (grpcurl, etc.)
	reflection.Register(grpcServer)

	// Subscribe to UserSuspended for Kill Switch
	eventBus.Subscribe(event.UserSuspended, func(ctx context.Context, e event.Event) error {
		payload, ok := e.Payload.(event.UserSuspendedPayload)
		if !ok {
			return nil
		}
		log.Printf("Kill Switch triggered for User: %s", payload.UserID)
		identityGrpcServer.TriggerKillSwitch(payload.UserID)
		return nil
	})

	// Start gRPC Server
	grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}
	go func() {
		log.Printf("Starting gRPC server on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// 5. Setup HTTP Server
	router := internalhttp.NewRouter(
		registrationSvc,
		authSvc,
		totpSvc,
		captchaSvc,
		apiKeySvc,
		companySvc,
		companyLookupSvc,
		policySvc,
		webAuthnHandler,
		passwordHandler,
		policyHandler,
		tenantHandler,
		apiKeyHandler,
		companyHandler,
		documentHandler,
		userRepo,
		roleRepo,
		companyRepo,
		eventBus,
		internalhttp.ServerConfig{
			TokenService: tokenSvc,
			RedisClient:  redisClient,
			DBPool:       db.Pool,
			RateLimit:    100,
			RateWindow:   time.Minute,
		},
	)

	httpServer := &gohttp.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP Server
	go func() {
		log.Printf("Starting HTTP server on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != gohttp.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 6. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}

func runMigrations(databaseURL string) error {
	log.Println("Running database migrations...")

	logEmbeddedMigrations()

	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer closeMigrator(m)

	if rerun, err := repairMigrationDrift(m, databaseURL); rerun || err != nil {
		if err != nil {
			return err
		}
		return runMigrations(databaseURL)
	}

	if rerun, err := migrateUpWithRecovery(m); rerun || err != nil {
		if err != nil {
			return err
		}
		return runMigrations(databaseURL)
	}

	log.Println("✓ Migrations applied successfully")
	return nil
}

func logEmbeddedMigrations() {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Printf("Failed to read embedded migrations: %v", err)
		return
	}
	for _, e := range entries {
		log.Printf("Embedded file: %s", e.Name())
	}
}

func closeMigrator(m *migrate.Migrate) {
	sourceErr, databaseErr := m.Close()
	if sourceErr != nil {
		log.Printf("Failed to close migration source: %v", sourceErr)
	}
	if databaseErr != nil {
		log.Printf("Failed to close migration database: %v", databaseErr)
	}
}

func repairMigrationDrift(m *migrate.Migrate, databaseURL string) (bool, error) {
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		log.Printf("Failed to get migration version: %v", err)
		return false, nil
	}

	log.Printf("Current DB Version: %d, Dirty: %v", version, dirty)
	if version > 0 {
		rerun, err := repairMissingTable(m, databaseURL, version, "tenants", "tenants")
		if rerun || err != nil {
			return rerun, err
		}
	}
	if version >= 20 {
		return repairMissingTable(m, databaseURL, version, "companies", "companies")
	}
	return false, nil
}

func repairMissingTable(m *migrate.Migrate, databaseURL string, version uint, tableName, label string) (bool, error) {
	exists, err := tableExists(databaseURL, tableName)
	if err != nil {
		log.Printf("Warning: Failed to check %s table existence: %v", label, err)
		return false, nil
	}
	if exists {
		return false, nil
	}

	log.Printf("Drift detected: Migrations at version %d but %q table is missing.", version, tableName)
	log.Println("Auto-repair: Dropping database to recover...")
	if err := m.Drop(); err != nil {
		log.Printf("Failed to drop database: %v", err)
		return false, nil
	}
	log.Println("Database dropped. Migrations will re-run.")
	return true, nil
}

func tableExists(databaseURL, tableName string) (bool, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close drift-check database connection: %v", err)
		}
	}()

	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)", tableName).Scan(&exists)
	return exists, err
}

func migrateUpWithRecovery(m *migrate.Migrate) (bool, error) {
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return recoverMigrationError(m, err)
	}
	return false, nil
}

func recoverMigrationError(m *migrate.Migrate, err error) (bool, error) {
	if strings.Contains(err.Error(), "no migration found for version 0") {
		return dropCorruptedDatabase(m), nil
	}
	if strings.Contains(err.Error(), "Dirty database version") {
		return false, recoverDirtyMigration(m)
	}
	return false, fmt.Errorf("failed to run migrate up: %w", err)
}

func dropCorruptedDatabase(m *migrate.Migrate) bool {
	log.Println("Corrupted version 0 state detected. Dropping database...")
	if err := m.Drop(); err != nil {
		log.Printf("Failed to drop database: %v", err)
		return false
	}
	log.Println("Database dropped. Migrations will re-run.")
	return true
}

func recoverDirtyMigration(m *migrate.Migrate) error {
	log.Println("Dirty database detected. Attempting recovery...")
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	if !dirty {
		return nil
	}

	log.Printf("Current dirty version: %d", version)
	targetVersion, err := migrationForceVersion(version)
	if err != nil {
		return err
	}
	if err := m.Force(targetVersion); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}

	log.Println("Retrying migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to retry migrate up: %w", err)
	}
	return nil
}

func migrationForceVersion(version uint) (int, error) {
	if version <= 2 {
		log.Println("Low version detected. Forcing version 0 to restart migrations from scratch...")
		return 0, nil
	}
	targetVersion, err := strconv.Atoi(strconv.FormatUint(uint64(version), 10))
	if err != nil {
		return 0, fmt.Errorf("migration version is too large: %w", err)
	}
	log.Printf("Forcing version %d to retry...", targetVersion)
	return targetVersion, nil
}

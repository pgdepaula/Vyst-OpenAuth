package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/application/service"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/config"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/eventbus"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/logger"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/provider/brasilapi"
)

func main() {
	// Load env
	_ = godotenv.Load()

	// Logger
	appLogger := logger.New(logger.Config{Level: "info", Format: "json"})

	// Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Database
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Deps
	companyRepo := postgres.NewCompanyRepository(db.Pool)
	eventBus := eventbus.NewInMemoryBus() // In real job, this might publish to obscure Kafka/RabbitMQ
	lookupAPIAdapter := brasilapi.NewBrasilAPIAdapter(nil, cfg.BrasilAPIURL)
	companyInfoRepo := postgres.NewCompanyInfoRepository(db.Pool)
	companyLookupSvc := service.NewCompanyLookupService(companyInfoRepo, []ports.CompanyDataPort{lookupAPIAdapter}, appLogger, eventBus, nil, nil)

	syncSvc := service.NewCompanySyncService(companyRepo, companyLookupSvc, eventBus, appLogger)

	// Run Sync
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := syncSvc.SyncAll(ctx); err != nil {
		appLogger.Error("Sync job failed", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Sync job completed successfully")
}

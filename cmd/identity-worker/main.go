package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/messaging/outbox"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database connection
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://vyst_app:vyst_app_secure_password@localhost:5432/vyst_identity?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Vyst Identity Worker started")

	// Create outbox processor with log publisher (for development)
	// In production, use RabbitMQ, Kafka, or NATS publisher
	publisher := outbox.NewLogPublisher()
	processor := outbox.NewProcessor(pool, publisher)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down worker...")
		cancel()
	}()

	// Start processing (blocks until context is cancelled)
	processor.Start(ctx)

	log.Println("Worker stopped")
}

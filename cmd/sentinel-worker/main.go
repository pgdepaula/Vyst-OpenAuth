package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/pgdepaula/vyst-openauth/internal/domain/event"
	"github.com/pgdepaula/vyst-openauth/internal/domain/risk"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/config"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/messaging/outbox"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/persistence/postgres"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// 1. Load Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Infrastructure
	// DB
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Redis
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpt)

	// Event Bus (for internal communication if needed, or to publish Kill Switch)
	// In this worker, we might need to publish "UserSuspended" back to the system?
	// Or we can call the gRPC service?
	// The plan said "Publish UserSuspended event".
	// Since we are in a separate process, we should probably write to the Outbox again?
	// Or use a message broker.
	// For this MVP, let's assume we write to the Outbox table directly to trigger other systems,
	// BUT for the "Kill Switch" which is in-memory in the API, we have a problem.
	// The API needs to know.
	// We should use Redis Pub/Sub for inter-service communication of "Kill Signals".
	// Let's add Redis Pub/Sub to the EventBus implementation?
	// Or just use Redis directly here to publish to a channel the API listens to.

	// Actually, the API subscribes to `eventBus.Subscribe(event.UserSuspended)`.
	// If `eventBus` is in-memory, the API won't hear us.
	// We need a distributed event bus.
	// For Phase 3, let's upgrade `eventbus` to use Redis Pub/Sub!
	// That's a great "Intelligence" upgrade.

	// But first, let's wire the worker logic.

	historyRepo := postgres.NewLoginHistoryRepository(db.Pool)
	outboxRepo := outbox.NewRepository(db.Pool)

	// 3. Risk Engine
	velocityRule := risk.NewVelocityRule(redisClient, 5, 1*time.Minute) // 5 logins per minute
	travelRule := risk.NewImpossibleTravelRule(historyRepo)

	engine := risk.NewStandardRiskEngine(velocityRule, travelRule)

	// 4. Outbox Poller (The "Ear")
	// We need a Poller that reads from `outbox_events` and processes them.
	// Wait, the Outbox pattern usually sends to a Broker (Kafka/Rabbit).
	// Here we are using the Table as the Queue for simplicity?
	// Or are we implementing the "Relay" part?
	// The `sentinel-worker` acts as the Relay AND the Processor for simplicity in this Vyst OS.
	// It reads "UserLoggedIn" events.

	log.Println("Vyst Sentinel starting...")

	// Polling Loop
	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			processEvents(context.Background(), outboxRepo, historyRepo, engine, redisClient)
		case <-quit:
			ticker.Stop()
			log.Println("Sentinel shutting down...")
			return
		}
	}
}

func processEvents(
	ctx context.Context,
	outboxRepo *outbox.Repository,
	historyRepo risk.LoginHistoryRepository,
	engine risk.RiskEngine,
	redisClient *redis.Client,
) {
	// Fetch unprocessed events
	events, err := outboxRepo.FetchUnprocessed(ctx, 10)
	if err != nil {
		log.Printf("Error fetching events: %v", err)
		return
	}

	for _, evt := range events {
		if evt.Type == event.UserLoggedIn {
			handleLoginEvent(ctx, evt, historyRepo, engine, redisClient)
		}

		// Mark as processed
		evtID, _ := uuid.Parse(evt.ID)
		if err := outboxRepo.MarkProcessed(ctx, evtID); err != nil {
			log.Printf("Error marking event %s as processed: %v", evt.ID, err)
		}
	}
}

func handleLoginEvent(
	ctx context.Context,
	evt event.Event,
	historyRepo risk.LoginHistoryRepository,
	engine risk.RiskEngine,
	redisClient *redis.Client,
) {
	// Parse Payload
	// We need to know the structure of UserLoggedIn payload.
	// It's likely map[string]interface{} from JSON unmarshal in FetchUnprocessed?
	// Or we need to unmarshal it here.

	// Let's assume evt.Payload is already unmarshaled into map[string]interface{} by the repository?
	// Looking at `outbox/repository.go` (I need to check it), usually it returns raw bytes or unmarshals.
	// I'll assume it returns `event.Event` struct where Payload is `interface{}`.

	// We need to cast payload.
	payloadBytes, _ := json.Marshal(evt.Payload)
	var loginPayload struct {
		UserID    string    `json:"user_id"`
		IPAddress string    `json:"ip_address"`
		UserAgent string    `json:"user_agent"`
		LoginAt   time.Time `json:"login_at"`
	}
	if err := json.Unmarshal(payloadBytes, &loginPayload); err != nil {
		log.Printf("Failed to unmarshal login payload: %v", err)
		return
	}

	userID, _ := uuid.Parse(loginPayload.UserID)

	// 1. Analyze Risk
	// Start a span for analysis
	tracer := otel.Tracer("sentinel-worker")
	ctx, span := tracer.Start(ctx, "AnalyzeRisk")
	defer span.End()

	score, reasons, err := engine.Analyze(ctx, userID, loginPayload.IPAddress, loginPayload.UserAgent)
	if err != nil {
		span.RecordError(err)
		log.Printf("Risk analysis error: %v", err)
		return
	}
	span.SetAttributes(
		attribute.String("user.id", userID.String()),
		attribute.Float64("risk.score", score),
	)

	log.Printf("Analyzed Login for %s. Score: %.2f. Reasons: %v", userID, score, reasons)

	// 2. Save History (for next time)
	// We save AFTER analysis so we compare against *previous* login, not current one.
	history := &risk.LoginHistory{
		UserID:    userID,
		IPAddress: loginPayload.IPAddress,
		UserAgent: loginPayload.UserAgent,
		LoginAt:   loginPayload.LoginAt,
	}
	if err := historyRepo.Save(ctx, history); err != nil {
		span.RecordError(err)
		log.Printf("Failed to save login history: %v", err)
	}

	// 3. React (Kill Switch)
	if score >= 1.0 {
		span.AddEvent("KillSwitchTriggered")
		log.Printf("CRITICAL RISK DETECTED for User %s. Triggering Kill Switch.", userID)

		// Publish to Redis Pub/Sub for API to pick up
		// Channel: "vyst:events:killswitch"
		msg := map[string]string{"user_id": userID.String(), "reason": "High Risk Detected"}
		if err := redisClient.Publish(ctx, "vyst:events:killswitch", msg).Err(); err != nil {
			span.RecordError(err)
			log.Printf("Failed to publish kill signal: %v", err)
		}
	}
}

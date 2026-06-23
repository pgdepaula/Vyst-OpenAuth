package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pgdepaula/vyst-openauth/internal/interfaces/http/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Starting Billing Worker...")

	// Redis Config
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Check Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	enforcer := middleware.NewQuotaEnforcer(rdb)

	// Run loop
	ticker := time.NewTicker(1 * time.Minute) // Run every minute for demo
	defer ticker.Stop()

	for range ticker.C {
		processBilling(context.Background(), enforcer)
	}
}

func processBilling(ctx context.Context, enforcer *middleware.QuotaEnforcer) {
	log.Println("Processing billing cycle...")

	// Get active tenants
	tenants, err := enforcer.GetActiveTenants(ctx)
	if err != nil {
		log.Printf("Error getting active tenants: %v", err)
		return
	}

	if len(tenants) == 0 {
		log.Println("No active tenants found.")
		return
	}

	for _, tenantID := range tenants {
		// Get usage
		usage, err := enforcer.GetUsage(ctx, tenantID)
		if err != nil {
			log.Printf("Error getting usage for tenant %s: %v", tenantID, err)
			continue
		}

		// Calculate cost (Mock)
		// Free: $0
		// Pro: $10 + $0.001 per auth
		// Enterprise: $500 flat
		var cost float64
		switch usage.PlanName {
		case "Free":
			cost = 0
		case "Pro":
			cost = 10.0 + (float64(usage.AuthsThisMonth) * 0.001)
		case "Enterprise":
			cost = 500.0
		default:
			cost = 0
		}

		if cost > 0 {
			log.Printf("🧾 INVOICE GENERATED | Tenant: %s | Plan: %s | Usage: %d | Amount: $%.2f",
				tenantID, usage.PlanName, usage.AuthsThisMonth, cost)

			// In a real system, we would insert this into a 'invoices' table or call a payment gateway.
		} else {
			log.Printf("Skipping invoice for free/zero cost tenant: %s", tenantID)
		}
	}
}

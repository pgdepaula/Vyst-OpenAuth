package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port            string
	GRPCPort        string
	DatabaseURL     string
	RedisURL        string
	JWTPrivateKey   string
	JWTPublicKey    string
	EnableTelemetry bool
	WebAuthnRPID    string
	WebAuthnOrigin  string
	WebAuthnRPName  string
	// CAPTCHA (Cloudflare Turnstile)
	TurnstileSiteKey   string
	TurnstileSecretKey string
	// 2FA (TOTP)
	TOTPIssuer string
	// Frontend URL for email links (password reset, verification)
	FrontendURL string
	// External APIs
	SerproAPIURL string
	SerproAPIKey string
	BrasilAPIURL string
}

func Load() (*Config, error) {
	dbURL, err := getEnvOrError("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	redisURL, err := getEnvOrError("REDIS_URL")
	if err != nil {
		return nil, err
	}
	jwtPrivate, err := getEnvOrError("JWT_PRIVATE_KEY")
	if err != nil {
		return nil, err
	}
	jwtPublic, err := getEnvOrError("JWT_PUBLIC_KEY")
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:            getEnv("PORT", "8080"),
		GRPCPort:        getEnv("GRPC_PORT", "50051"),
		DatabaseURL:     dbURL,
		RedisURL:        redisURL,
		JWTPrivateKey:   jwtPrivate,
		JWTPublicKey:    jwtPublic,
		EnableTelemetry: getEnvAsBool("ENABLE_TELEMETRY", true),
		WebAuthnRPID:    getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnOrigin:  getEnv("WEBAUTHN_ORIGIN", "http://localhost:3000"),
		WebAuthnRPName:  getEnv("WEBAUTHN_RP_NAME", "Vyst Identity"),
		// CAPTCHA
		TurnstileSiteKey:   getEnv("TURNSTILE_SITE_KEY", ""),
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", ""),
		// 2FA
		TOTPIssuer: getEnv("TOTP_ISSUER", "Vyst Identity"),
		// Frontend
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:4200"),
		// External APIs
		SerproAPIURL: getEnv("SERPRO_API_URL", ""),
		SerproAPIKey: getEnv("SERPRO_API_KEY", ""),
		BrasilAPIURL: getEnv("BRASIL_API_URL", "https://brasilapi.com.br/api/cnpj/v1"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return value == "true" || value == "1"
	}
	return fallback
}

func getEnvOrError(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s is required but not set", key)
}

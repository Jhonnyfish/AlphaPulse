package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	JWTExpiry          time.Duration
	RefreshTokenExpiry time.Duration
	AdminUsername      string
	AdminPassword      string
	AppVersion         string
	HTTPTimeout        time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtExpiry, err := parseDuration("JWT_EXPIRY", "24h")
	if err != nil {
		return nil, err
	}

	refreshExpiry, err := parseDuration("REFRESH_TOKEN_EXPIRY", "168h")
	if err != nil {
		return nil, err
	}

	httpTimeout, err := parseDuration("HTTP_TIMEOUT", "15s")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:               envOrDefault("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTExpiry:          jwtExpiry,
		RefreshTokenExpiry: refreshExpiry,
		AdminUsername:      envOrDefault("ADMIN_USERNAME", "admin"),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		AppVersion:         envOrDefault("APP_VERSION", "dev"),
		HTTPTimeout:        httpTimeout,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD is required")
	}

	return cfg, nil
}

func parseDuration(key, fallback string) (time.Duration, error) {
	value := envOrDefault(key, fallback)
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return duration, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

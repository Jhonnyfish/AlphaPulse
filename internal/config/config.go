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

	// DeepSeek / LLM config (OpenAI-compatible)
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string

	// Alpha300 Database (read-only)
	Alpha300DBEnabled  bool
	Alpha300DBHost     string
	Alpha300DBPort     string
	Alpha300DBName     string
	Alpha300DBUser     string
	Alpha300DBPassword string
	Alpha300DBSSLMode  string
	Alpha300DBMaxConns int
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

		DeepSeekAPIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
		DeepSeekModel:   envOrDefault("DEEPSEEK_MODEL", "deepseek-chat"),

		// Alpha300 Database (read-only)
		Alpha300DBEnabled:  envOrDefault("ALPHA300_DB_ENABLED", "false") == "true",
		Alpha300DBHost:     envOrDefault("ALPHA300_DB_HOST", "198.23.251.110"),
		Alpha300DBPort:     envOrDefault("ALPHA300_DB_PORT", "25432"),
		Alpha300DBName:     envOrDefault("ALPHA300_DB_NAME", "alpha300"),
		Alpha300DBUser:     envOrDefault("ALPHA300_DB_USER", "alpha300_app"),
		Alpha300DBPassword: os.Getenv("ALPHA300_DB_PASSWORD"),
		Alpha300DBSSLMode:  envOrDefault("ALPHA300_DB_SSL_MODE", "disable"),
		Alpha300DBMaxConns: 5,
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

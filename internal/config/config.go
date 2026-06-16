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

	// Tushare Pro API
	TushareToken       string
	TushareBaseURL     string
	TushareToken2      string
	TushareBaseURL2    string
	TushareActive      string // "1" or "2"
	TushareEnabled     bool
}

// ActiveTushareToken returns the token for the currently active profile.
func (c *Config) ActiveTushareToken() string {
	if c.TushareActive == "2" && c.TushareToken2 != "" {
		return c.TushareToken2
	}
	return c.TushareToken
}

// ActiveTushareBaseURL returns the base URL for the currently active profile.
func (c *Config) ActiveTushareBaseURL() string {
	if c.TushareActive == "2" && c.TushareBaseURL2 != "" {
		return c.TushareBaseURL2
	}
	return c.TushareBaseURL
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

		TushareToken:       os.Getenv("TUSHARE_TOKEN"),
		TushareBaseURL:     envOrDefault("TUSHARE_BASE_URL", "https://api.tushare.pro"),
		TushareToken2:      os.Getenv("TUSHARE_TOKEN2"),
		TushareBaseURL2:    envOrDefault("TUSHARE_BASE_URL2", "http://47.122.118.90:8080"),
		TushareActive:      envOrDefault("TUSHARE_ACTIVE", "1"),
		TushareEnabled:     envOrDefault("TUSHARE_ENABLED", "true") == "true",
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

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"alphapulse/internal/config"
	"alphapulse/internal/database"
	"alphapulse/internal/logger"
	"alphapulse/internal/services"

	"go.uber.org/zap"
)

// Backfill tool: populate historical Tushare data into local PostgreSQL.
//
// Usage:
//   go run ./cmd/backfill -start 20260101 -end 20260504
//   go run ./cmd/backfill -start 20260401          (to today)

func main() {
	startDate := flag.String("start", "", "start date YYYYMMDD (required)")
	endDate := flag.String("end", time.Now().Format("20060102"), "end date YYYYMMDD (default: today)")
	token := flag.String("token", "", "Tushare token (or set TUSHARE_TOKEN/TUSHARE_TOKEN2 env)")
	baseURL := flag.String("base-url", "", "Tushare base URL (default: active profile)")
	flag.Parse()

	if *startDate == "" {
		log.Fatal("-start flag is required (YYYYMMDD format)")
	}

	// Load config for database connection
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Resolve token: flag first, then active profile, then env
	tushareToken := *token
	if tushareToken == "" {
		tushareToken = cfg.ActiveTushareToken()
	}
	if tushareToken == "" {
		tushareToken = os.Getenv("TUSHARE_TOKEN")
	}
	if tushareToken == "" {
		log.Fatal("Tushare token required: use -token flag or set TUSHARE_TOKEN env")
	}

	// Resolve base URL: flag first, then active profile, then default
	tushareBaseURL := *baseURL
	if tushareBaseURL == "" {
		tushareBaseURL = cfg.ActiveTushareBaseURL()
	}

	ctx := context.Background()

	// Connect to database
	migrationPath := "migrations/001_initial.sql"
	db, err := database.New(ctx, cfg, migrationPath)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	// Create Tushare service and sync
	ts := services.NewTushareService(tushareToken, tushareBaseURL, 30*time.Second)
	eastMoneyService := services.NewEastMoneyService(30 * time.Second)
	sync := services.NewTushareSync(ts, eastMoneyService, db, logger.L())

	log.Printf("Starting backfill from %s to %s", *startDate, *endDate)

	if err := sync.RunBackfill(ctx, *startDate, *endDate); err != nil {
		logger.L().Fatal("backfill failed", zap.Error(err))
	}

	log.Printf("Backfill completed successfully")
}

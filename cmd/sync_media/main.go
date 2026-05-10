package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"alphapulse/internal/config"
	"alphapulse/internal/database"
	"alphapulse/internal/logger"
	"alphapulse/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil { log.Fatal(err) }
	ctx := context.Background()
	db, err := database.New(ctx, cfg, "migrations/001_initial.sql")
	if err != nil { log.Fatal(err) }
	defer db.Close()
	ts := services.NewTushareService(cfg.TushareToken, 30*time.Second)
	eastMoney := services.NewEastMoneyService(30 * time.Second)
	sync := services.NewTushareSync(ts, eastMoney, db, logger.L())
	
	fmt.Println("Syncing announcements...")
	if err := sync.SyncAnnouncements(ctx); err != nil {
		fmt.Printf("Announcements FAILED: %v\n", err)
	}
	fmt.Println("Syncing news...")
	if err := sync.SyncNews(ctx); err != nil {
		fmt.Printf("News FAILED: %v\n", err)
	}
	fmt.Println("Done!")
}

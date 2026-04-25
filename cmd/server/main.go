package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"alphapulse/internal/config"
	"alphapulse/internal/database"
	"alphapulse/internal/handlers"
	"alphapulse/internal/middleware"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	migrationPath := filepath.Join("migrations", "001_initial.sql")
	db, err := database.New(ctx, cfg, migrationPath)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	created, err := database.EnsureAdminUser(ctx, db, cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("ensure admin user: %v", err)
	}
	if created {
		log.Printf("created admin user %q", cfg.AdminUsername)
	}

	if *migrateOnly {
		log.Println("database migrations completed")
		return
	}

	eastMoneyService := services.NewEastMoneyService(cfg.HTTPTimeout)
	tencentService := services.NewTencentService(cfg.HTTPTimeout)
	alpha300Service := services.NewAlpha300Service(cfg.HTTPTimeout)
	authHandler := handlers.NewAuthHandler(db, cfg)
	watchlistHandler := handlers.NewWatchlistHandler(db)
	marketHandler := handlers.NewMarketHandler(eastMoneyService, tencentService, db)
	candidatesHandler := handlers.NewCandidatesHandler(alpha300Service, db)
	screenerHandler := handlers.NewScreenerHandler(alpha300Service, db)
	scoreHistoryHandler := handlers.NewScoreHistoryHandler(db)
	patternScannerHandler := handlers.NewPatternScannerHandler(eastMoneyService, tencentService, db)
	analyzeHandler := handlers.NewAnalyzeHandler(eastMoneyService, tencentService)
	systemHandler := handlers.NewSystemHandler(db, cfg.AppVersion, time.Now(), marketHandler.CacheStats())

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.CORS())

	authMiddleware := middleware.AuthRequired(cfg.JWTSecret)

	router.GET("/health", systemHandler.Health)
	router.GET("/api/system/info", systemHandler.Info)
	router.GET("/api/system/datasources",
		systemHandler.DataSourceHealth(
			eastMoneyService.HealthCheck,
			tencentService.HealthCheck,
		),
	)

	api := router.Group("/api")
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.GET("/verify", authMiddleware, authHandler.Verify)

	adminGroup := api.Group("/admin")
	adminGroup.Use(authMiddleware, middleware.RequireAdmin())
	adminGroup.POST("/invite-codes", authHandler.CreateInviteCode)
	adminGroup.GET("/invite-codes", authHandler.ListInviteCodes)
	adminGroup.DELETE("/invite-codes/:id", authHandler.DeleteInviteCode)

	watchlistGroup := api.Group("/watchlist")
	watchlistGroup.Use(authMiddleware)
	watchlistGroup.GET("", watchlistHandler.List)
	watchlistGroup.POST("", watchlistHandler.Add)
	watchlistGroup.DELETE("/:code", watchlistHandler.Delete)
	watchlistGroup.POST("/batch", watchlistHandler.BatchAdd)

	marketGroup := api.Group("/market")
	marketGroup.Use(authMiddleware)
	marketGroup.GET("/quote", marketHandler.Quote)
	marketGroup.GET("/kline", marketHandler.Kline)
	marketGroup.GET("/sectors", marketHandler.Sectors)
	marketGroup.GET("/overview", marketHandler.Overview)
	marketGroup.GET("/news", marketHandler.News)
	marketGroup.GET("/search", marketHandler.Search)
	marketGroup.GET("/top-movers", marketHandler.TopMovers)
	marketGroup.GET("/session", marketHandler.Session)
	marketGroup.GET("/trends", marketHandler.Trends)
	marketGroup.GET("/market-overview", marketHandler.MarketOverview)
	marketGroup.GET("/hot-concepts", marketHandler.HotConcepts)
	marketGroup.GET("/breadth", marketHandler.MarketBreadth)
	marketGroup.GET("/sentiment", marketHandler.MarketSentiment)

	candidatesGroup := api.Group("/candidates")
	candidatesGroup.Use(authMiddleware)
	candidatesGroup.GET("", candidatesHandler.Candidates)

	screenerGroup := api.Group("/screener")
	screenerGroup.Use(authMiddleware)
	screenerGroup.GET("", screenerHandler.Screener)

	patternScannerGroup := api.Group("/pattern-scanner")
	patternScannerGroup.Use(authMiddleware)
	patternScannerGroup.GET("", patternScannerHandler.Scan)

	scoreHistoryGroup := api.Group("/score-history")
	scoreHistoryGroup.Use(authMiddleware)
	scoreHistoryGroup.GET("/:code", scoreHistoryHandler.GetHistory)

	analyzeGroup := api.Group("/analyze")
	analyzeGroup.Use(authMiddleware)
	analyzeGroup.GET("", analyzeHandler.Analyze)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("AlphaPulse server running on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}

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
	"alphapulse/internal/logger"
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
	dragonTigerHandler := handlers.NewDragonTigerHandler(eastMoneyService)
	candidatesHandler := handlers.NewCandidatesHandler(alpha300Service, db)
	screenerHandler := handlers.NewScreenerHandler(alpha300Service, db)
	scoreHistoryHandler := handlers.NewScoreHistoryHandler(db)
	patternScannerHandler := handlers.NewPatternScannerHandler(eastMoneyService, tencentService, db)
	analyzeHandler := handlers.NewAnalyzeHandler(eastMoneyService, tencentService)
	trendHandler := handlers.NewTrendHandler(eastMoneyService, tencentService, db)
	compareHandler := handlers.NewCompareHandler(eastMoneyService, tencentService)
	portfolioHandler := handlers.NewPortfolioHandler(tencentService, eastMoneyService, db)
	tradingJournalHandler := handlers.NewTradingJournalHandler(db)
	strategiesHandler := handlers.NewStrategiesHandler(db)
	customAlertsHandler := handlers.NewCustomAlertsHandler(db, tencentService)
	stockNotesHandler := handlers.NewStockNotesHandler(db)
	fundFlowHandler := handlers.NewFundFlowHandler(eastMoneyService, logger.L())
	sectorRotationHandler := handlers.NewSectorRotationHandler(eastMoneyService, db, logger.L())
	investmentPlansHandler := handlers.NewInvestmentPlansHandler(logger.L())
	watchlistAnalysisHandler := handlers.NewWatchlistAnalysisHandler(db, tencentService, eastMoneyService, analyzeHandler, logger.L())
	systemHandler := handlers.NewSystemHandler(db, cfg.AppVersion, time.Now(), marketHandler.CacheStats())
	signalHandler := handlers.NewSignalHandler(alpha300Service, tencentService, eastMoneyService, logger.L())
	reportsHandler := handlers.NewReportsHandler(db, tencentService, eastMoneyService, analyzeHandler, watchlistHandler, logger.L())
	alertsHandler := handlers.NewAlertsHandler(db, analyzeHandler, logger.L())
	docsHandler := handlers.NewDocsHandler()
	dashboardHandler := handlers.NewDashboardHandler(db, tencentService, watchlistHandler, logger.L())
	watchlistHandler.SetAlpha300(alpha300Service)

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

	api.GET("/announcements", marketHandler.Announcements)

	dragonTigerGroup := api.Group("/dragon-tiger")
	dragonTigerGroup.Use(authMiddleware)
	dragonTigerGroup.GET("", dragonTigerHandler.GetDragonTiger)
	api.GET("/dragon-tiger-history", authMiddleware, dragonTigerHandler.GetHistory)
	api.GET("/institution-tracker", authMiddleware, dragonTigerHandler.GetInstitutionTracker)

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

	trendGroup := api.Group("")
	trendGroup.Use(authMiddleware)
	trendGroup.GET("/multi-trend", trendHandler.MultiTrend)
	trendGroup.GET("/correlation", trendHandler.Correlation)

	compareGroup := api.Group("/compare")
	compareGroup.Use(authMiddleware)
	compareGroup.GET("/sector", compareHandler.SectorCompare)
	compareGroup.GET("/backtest", compareHandler.BacktestCompare)

	portfolioGroup := api.Group("/portfolio")
	portfolioGroup.Use(authMiddleware)
	portfolioGroup.GET("", portfolioHandler.List)
	portfolioGroup.POST("", portfolioHandler.Add)
	portfolioGroup.PUT("/:id", portfolioHandler.Update)
	portfolioGroup.DELETE("/:id", portfolioHandler.Delete)
	portfolioGroup.GET("/analytics", portfolioHandler.Analytics)
	portfolioGroup.GET("/risk", portfolioHandler.Risk)
	tradingJournalGroup := api.Group("/trading-journal")
	tradingJournalGroup.Use(authMiddleware)
	tradingJournalGroup.GET("", tradingJournalHandler.List)
	tradingJournalGroup.POST("", tradingJournalHandler.Create)
	tradingJournalGroup.DELETE("/:id", tradingJournalHandler.Delete)
	tradingJournalGroup.GET("/stats", tradingJournalHandler.Stats)
	tradingJournalGroup.GET("/calendar", tradingJournalHandler.Calendar)

	tradeStrategyEvalGroup := api.Group("/trade-strategy-eval")
	tradeStrategyEvalGroup.Use(authMiddleware)
	tradeStrategyEvalGroup.GET("", tradingJournalHandler.StrategyEval)

	strategiesGroup := api.Group("/strategies")
	strategiesGroup.Use(authMiddleware)
	strategiesGroup.GET("", strategiesHandler.List)
	strategiesGroup.POST("", strategiesHandler.Create)
	strategiesGroup.PUT("/:id", strategiesHandler.Update)
	strategiesGroup.DELETE("/:id", strategiesHandler.Delete)
	strategiesGroup.POST("/:id/activate", strategiesHandler.Activate)
	strategiesGroup.POST("/:id/deactivate", strategiesHandler.Deactivate)

	customAlertsGroup := api.Group("/custom-alerts")
	customAlertsGroup.Use(authMiddleware)
	customAlertsGroup.GET("", customAlertsHandler.List)
	customAlertsGroup.POST("", customAlertsHandler.Create)
	customAlertsGroup.DELETE("/:id", customAlertsHandler.Delete)
	customAlertsGroup.GET("/check", customAlertsHandler.Check)

	stockNotesGroup := api.Group("/stock-notes")
	stockNotesGroup.Use(authMiddleware)
	stockNotesGroup.GET("/tags/all", stockNotesHandler.AllTags)
	stockNotesGroup.GET("/:code", stockNotesHandler.GetNotes)
	stockNotesGroup.POST("", stockNotesHandler.CreateNote)
	stockNotesGroup.PUT("/:id", stockNotesHandler.UpdateNote)
	stockNotesGroup.DELETE("/:id", stockNotesHandler.DeleteNote)

	fundFlowGroup := api.Group("/fund-flow")
	fundFlowGroup.Use(authMiddleware)
	fundFlowGroup.GET("/flow", fundFlowHandler.Flow)

	// Compat route: Python /flow → Go fund flow
	router.GET("/flow", authMiddleware, fundFlowHandler.Flow)

	sectorRotationGroup := api.Group("/sector-rotation")
	sectorRotationGroup.Use(authMiddleware)
	sectorRotationGroup.GET("", sectorRotationHandler.Rotation)
	sectorRotationGroup.GET("/history", sectorRotationHandler.RotationHistory)

	// Compat route: Python /api/sector-rotation-history
	api.GET("/sector-rotation-history", authMiddleware, sectorRotationHandler.RotationHistory)

	// Investment plans (Module 17)
	investmentPlansGroup := api.Group("/investment-plans")
	investmentPlansGroup.Use(authMiddleware)
	investmentPlansGroup.GET("", investmentPlansHandler.List)
	investmentPlansGroup.POST("", investmentPlansHandler.Upsert)
	investmentPlansGroup.DELETE("/:code", investmentPlansHandler.Delete)

	// System management (Module 21)
	api.GET("/system-status", authMiddleware, systemHandler.SystemStatus)
	api.GET("/status", authMiddleware, systemHandler.Status)
	api.POST("/cache/clear", authMiddleware, systemHandler.CacheClear)
	api.GET("/activity-log", authMiddleware, systemHandler.ActivityLog)
	api.GET("/slow-queries", authMiddleware, systemHandler.SlowQueries)
	api.GET("/performance-stats", authMiddleware, systemHandler.PerformanceStats)
	api.GET("/docs", docsHandler.Docs)

	// Watchlist analysis (Module 19)
	wlAnalysisGroup := api.Group("/watchlist-analysis")
	wlAnalysisGroup.Use(authMiddleware)
	wlAnalysisGroup.GET("/heatmap", watchlistAnalysisHandler.Heatmap)
	wlAnalysisGroup.GET("/sectors", watchlistAnalysisHandler.Sectors)
	wlAnalysisGroup.GET("/ranking", watchlistAnalysisHandler.Ranking)

	// Compat routes matching Python paths
	api.GET("/watchlist-heatmap", authMiddleware, watchlistAnalysisHandler.Heatmap)
	api.GET("/watchlist-sectors", authMiddleware, watchlistAnalysisHandler.Sectors)
	api.GET("/watchlist-ranking", authMiddleware, watchlistAnalysisHandler.Ranking)

	// Watchlist groups CRUD
	wlGroupsGroup := api.Group("/watchlist-groups")
	wlGroupsGroup.Use(authMiddleware)
	wlGroupsGroup.GET("", watchlistAnalysisHandler.GetGroups)
	wlGroupsGroup.POST("", watchlistAnalysisHandler.CreateGroup)
	wlGroupsGroup.PUT("/:id", watchlistAnalysisHandler.UpdateGroup)
	wlGroupsGroup.DELETE("/:id", watchlistAnalysisHandler.DeleteGroup)
	wlGroupsGroup.POST("/assign", watchlistAnalysisHandler.AssignStock)

	// Signal system (Module 18)
	api.GET("/anomalies", authMiddleware, signalHandler.Anomalies)
	api.GET("/signal-history", authMiddleware, signalHandler.SignalHistory)
	api.GET("/signal-calendar", authMiddleware, signalHandler.SignalCalendar)

	// Hot concept stocks (Module 12 remaining)
	marketGroup.GET("/hot-concepts/:code/stocks", marketHandler.HotConceptStocks)
	api.GET("/watchlist-concept-overlap", authMiddleware, marketHandler.WatchlistConceptOverlap)

	// Reports system (Module 16)
	router.GET("/reports", reportsHandler.RedirectToAPI)
	api.GET("/reports", authMiddleware, reportsHandler.ListReports)
	api.GET("/reports/:filename", authMiddleware, reportsHandler.GetReport)
	api.GET("/daily-report/latest", authMiddleware, reportsHandler.DailyReportLatest)
	api.GET("/daily-report/list", authMiddleware, reportsHandler.DailyReportList)
	api.POST("/daily-report/generate", authMiddleware, reportsHandler.DailyReportGenerate)
	api.GET("/daily-brief", authMiddleware, reportsHandler.DailyBrief)

	// Smart alerts (Module 22)
	api.GET("/alerts", authMiddleware, alertsHandler.Alerts)

	// Dashboard summary (composite endpoint)
	api.GET("/dashboard-summary", authMiddleware, dashboardHandler.DashboardSummary)

	// Watchlist sync (Alpha300 pool)
	watchlistGroup.POST("/sync", watchlistHandler.Sync)

	// Stock info (comprehensive single stock data)
	api.GET("/stockinfo", authMiddleware, analyzeHandler.StockInfo)

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

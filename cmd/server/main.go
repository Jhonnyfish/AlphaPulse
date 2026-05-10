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
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"

	_ "alphapulse/docs" // swagger generated docs
)

// @title           AlphaPulse API
// @version         3.0
// @description     AlphaPulse 股票综合分析服务 — Go backend REST API.
// @termsOfService  http://swagger.io/terms/

// @contact.name  API Support
// @contact.email support@alphapulse.local

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8899
// @BasePath  /api
// @schemes   http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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
	tushareSectorService := services.NewTushareSectorService(db)
	newsService := services.NewNewsService(db, eastMoneyService, logger.L())
	alpha300Service := services.NewAlpha300Service(cfg.HTTPTimeout)
	alpha300Cache := services.NewAlpha300Cache(alpha300Service)
	deepseekClient := services.NewDeepSeekClient(cfg.DeepSeekAPIKey, cfg.DeepSeekBaseURL, cfg.DeepSeekModel, logger.L())
	authHandler := handlers.NewAuthHandler(db, cfg, logger.L())
	watchlistHandler := handlers.NewWatchlistHandler(db, logger.L())
	marketHandler := handlers.NewMarketHandler(eastMoneyService, tushareSectorService, tencentService, db)
	dragonTigerHandler := handlers.NewDragonTigerHandler(eastMoneyService)
	candidatesHandler := handlers.NewCandidatesHandler(alpha300Cache, db)
	screenerHandler := handlers.NewScreenerHandler(alpha300Cache, db)
	scoreHistoryHandler := handlers.NewScoreHistoryHandler(db)
	patternScannerHandler := handlers.NewPatternScannerHandler(eastMoneyService, tencentService, db)
	analyzeHandler := handlers.NewAnalyzeHandler(eastMoneyService, tencentService, newsService, logger.L())
	trendHandler := handlers.NewTrendHandler(eastMoneyService, tencentService, db, logger.L())
	compareHandler := handlers.NewCompareHandler(eastMoneyService, tencentService)
	portfolioHandler := handlers.NewPortfolioHandler(tencentService, eastMoneyService, db, logger.L())
	tradingJournalHandler := handlers.NewTradingJournalHandler(db)
	strategiesHandler := handlers.NewStrategiesHandler(db)
	customAlertsHandler := handlers.NewCustomAlertsHandler(db, tencentService)
	stockNotesHandler := handlers.NewStockNotesHandler(db)
	fundFlowHandler := handlers.NewFundFlowHandler(eastMoneyService, logger.L())
	sectorRotationHandler := handlers.NewSectorRotationHandler(eastMoneyService, tushareSectorService, db, logger.L())
	investmentPlansHandler := handlers.NewInvestmentPlansHandler(logger.L())
	watchlistAnalysisHandler := handlers.NewWatchlistAnalysisHandler(db, tencentService, eastMoneyService, analyzeHandler, logger.L())
	perfTracker := services.NewPerfTracker()
	systemHandler := handlers.NewSystemHandler(db, cfg.AppVersion, time.Now(), marketHandler.CacheStats(), perfTracker)
	signalHandler := handlers.NewSignalHandler(alpha300Cache, tencentService, eastMoneyService, logger.L())
	reportsHandler := handlers.NewReportsHandler(db, tencentService, eastMoneyService, tushareSectorService, analyzeHandler, watchlistHandler, logger.L(), deepseekClient)
	alertsHandler := handlers.NewAlertsHandler(db, analyzeHandler, logger.L())
	defer alertsHandler.Stop()
	docsHandler := handlers.NewDocsHandler()
	dashboardHandler := handlers.NewDashboardHandler(db, tencentService, eastMoneyService, tushareSectorService, watchlistHandler, logger.L())
	watchlistHandler.SetAlpha300(alpha300Cache)
		watchlistHandler.SetOnChange(watchlistAnalysisHandler.InvalidateRankingCache)

	// Initialize Tushare data source (primary) if enabled
	var tushareDB *services.TushareDB
	if cfg.TushareEnabled && cfg.TushareToken != "" {
		tushareSvc := services.NewTushareService(cfg.TushareToken, cfg.HTTPTimeout)
		tushareDB = services.NewTushareDB(db, logger.L())
		tushareSync := services.NewTushareSync(tushareSvc, eastMoneyService, db, logger.L())

		analyzeHandler.SetTushareDB(tushareDB)
		reportsHandler.SetTushareDB(tushareDB)
	marketHandler.SetTushareDB(tushareDB)

		// Initial sync if tables are empty
		if !tushareDB.HasData(context.Background()) {
			go func() {
				syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				log.Println("[tushare] initial stock_basic sync starting...")
				if err := tushareSync.SyncStockBasic(syncCtx); err != nil {
					log.Printf("[tushare] initial sync failed: %v", err)
				}
			}()
		}

		logger.L().Info("Tushare data source enabled")
	} else if cfg.TushareEnabled {
		logger.L().Warn("Tushare enabled but no token provided")
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.TimingMiddleware(perfTracker))
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
	api.POST("/system/vitals", authMiddleware, systemHandler.ReceiveVitals)
	api.GET("/system/vitals", authMiddleware, systemHandler.GetVitals)
	api.GET("/docs", docsHandler.Docs)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Watchlist analysis (Module 19)
	wlAnalysisGroup := api.Group("/watchlist-analysis")
	wlAnalysisGroup.Use(authMiddleware)
	wlAnalysisGroup.GET("/heatmap", watchlistAnalysisHandler.Heatmap)
	wlAnalysisGroup.GET("/sectors", watchlistAnalysisHandler.Sectors)
	wlAnalysisGroup.GET("/ranking", watchlistAnalysisHandler.Ranking)
	wlAnalysisGroup.GET("/ranking/stream", watchlistAnalysisHandler.RankingStream)

	// Compat routes matching Python paths
	api.GET("/watchlist-heatmap", authMiddleware, watchlistAnalysisHandler.Heatmap)
	api.GET("/watchlist-sectors", authMiddleware, watchlistAnalysisHandler.Sectors)
	api.GET("/watchlist-ranking", authMiddleware, watchlistAnalysisHandler.Ranking)
	api.GET("/watchlist-ranking/stream", authMiddleware, watchlistAnalysisHandler.RankingStream)

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

	// Built-in scheduler for daily tasks
	scheduler := services.NewScheduler()
	scheduler.AddDailyJob("alpha300-sync", 9, 0, func() {
		log.Println("[scheduler] syncing Alpha300 top 10 to watchlist...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := watchlistHandler.SyncAlpha300TopN(ctx, 10); err != nil {
			log.Printf("[scheduler] alpha300 sync failed: %v", err)
		}
	})
	scheduler.AddDailyJob("daily-report", 15, 30, func() {
		log.Println("[scheduler] generating daily report...")
		reportsHandler.GenerateDailyReportAuto()
	})
		// Tushare daily sync at 16:00 after market close
		if tushareDB != nil {
			scheduler.AddDailyJob("tushare-daily-sync", 16, 0, func() {
				log.Println("[scheduler] tushare daily sync...")
				syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				tushareSvc := services.NewTushareService(cfg.TushareToken, cfg.HTTPTimeout)
				ts := services.NewTushareSync(tushareSvc, eastMoneyService, db, logger.L())
				ts.RunDaily(syncCtx)
			})
		}
		// Pre-fetch watchlist news at 16:10 so ranking reads from DB
		scheduler.AddDailyJob("watchlist-news-sync", 16, 10, func() {
			log.Println("[scheduler] syncing watchlist news to DB...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			rows, err := db.Query(ctx, `SELECT code FROM watchlist`)
			if err != nil {
				log.Printf("[scheduler] watchlist query failed: %v", err)
				return
			}
			defer rows.Close()
			var codes []string
			for rows.Next() {
				var code string
				if rows.Scan(&code) == nil {
					codes = append(codes, code)
				}
			}
			for _, code := range codes {
				codeCtx, codeCancel := context.WithTimeout(ctx, 15*time.Second)
				if _, err := newsService.GetStockNews(codeCtx, code, 10); err != nil {
					log.Printf("[scheduler] news sync failed for %s: %v", code, err)
				}
				if _, err := newsService.GetStockAnnouncements(codeCtx, code, 10); err != nil {
					log.Printf("[scheduler] announcements sync failed for %s: %v", code, err)
				}
				codeCancel()
			}
			log.Printf("[scheduler] watchlist news sync done for %d stocks", len(codes))
		})
		// Pre-compute ranking at 16:15 so users see results instantly
		scheduler.AddDailyJob("ranking-precompute", 16, 15, func() {
			watchlistAnalysisHandler.PreComputeRanking()
		})
	scheduler.AddDailyJob("news-cleanup", 3, 0, func() {
		log.Println("[scheduler] cleaning up old news data...")
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := newsService.CleanupOldData(cleanupCtx); err != nil {
			log.Printf("[scheduler] news cleanup failed: %v", err)
		}
	})
	defer scheduler.StopAll()

	// Scheduler status API
	api.GET("/system/scheduler", authMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "jobs": scheduler.Status()})
	})

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

	// Pre-compute ranking on startup if Tushare data is available
	if tushareDB != nil {
		go func() {
			if tushareDB.HasData(context.Background()) {
				log.Println("[startup] pre-computing ranking from TushareDB...")
				watchlistAnalysisHandler.PreComputeRanking()
			}
		}()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
}

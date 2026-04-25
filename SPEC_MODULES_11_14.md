# AlphaPulse Migration - Modules 11 & 14

## Context
Migrating Python stock-quote-service APIs to Go backend. Project uses:
- `go.uber.org/zap` for logging (use `zap.L().Error(...)`)
- `github.com/stretchr/testify` for tests
- `github.com/gin-gonic/gin` for HTTP
- `internal/errors` package for unified AppError
- `internal/cache` package for caching
- `internal/services/eastmoney.go` for EastMoney API calls
- `internal/models/` for data models

## Module 14: Announcements (1 endpoint)

### Existing Code
- `internal/models/analysis.go` line 44: `Announcement` struct (Title, URL, PublishedAt)
- `internal/services/eastmoney.go` line 397: `FetchStockAnnouncements(ctx, code, limit)` returns `[]Announcement`

### Task: Add standalone `/announcements` endpoint

**Python API (to match):**
```python
@app.get("/announcements")
def announcements(code: str = Query(...), limit: int = Query(10, ge=1, le=50)):
    items, cached = get_announcements_cached(code, limit)
    return {"code": code6(normalize_code(code)), "items": items, "source": "eastmoney", "cached": cached}
```

**Python item format:**
```python
{"title": "...", "date": "...", "url": "...", "art_code": "...", "source": "eastmoney"}
```

**Steps:**
1. Update `internal/models/analysis.go` `Announcement` struct to add `Date string` and `Source string` fields (keep existing Title, URL, PublishedAt for backward compat with analyze handler)
2. Update `internal/services/eastmoney.go` `FetchStockAnnouncements` to populate the new Date and Source fields
3. Add handler method `Announcements` to `internal/handlers/market.go` (since market.go already handles similar data-fetching endpoints)
4. Register route in `cmd/server/main.go` - add `api.GET("/announcements", marketHandler.Announcements)` (no auth required, matching Python)
5. Add test in `internal/handlers/market_test.go`

**Response format:**
```json
{
  "code": "600176",
  "items": [{"title": "...", "date": "...", "url": "...", "art_code": "...", "source": "eastmoney"}],
  "source": "eastmoney",
  "cached": false
}
```

## Module 11: Dragon-Tiger Board (3 endpoints)

### New Code Needed

**Models (`internal/models/dragon_tiger.go`):**
```go
type DragonTigerItem struct {
    Code       string              `json:"code"`
    Name       string              `json:"name"`
    Close      float64             `json:"close"`
    ChangePct  float64             `json:"change_pct"`
    NetBuy     float64             `json:"net_buy"`
    BuyTotal   float64             `json:"buy_total"`
    SellTotal  float64             `json:"sell_total"`
    Reason     string              `json:"reason"`
    TradeDate  string              `json:"trade_date"`
    Departments []DepartmentDetail `json:"departments"`
}

type DepartmentDetail struct {
    Name string  `json:"name"`
    Buy  float64 `json:"buy"`
    Sell float64 `json:"sell"`
    Net  float64 `json:"net"`
    Side string  `json:"side"`
}

type DragonTigerResponse struct {
    OK          bool              `json:"ok"`
    Items       []DragonTigerItem `json:"items"`
    Count       int               `json:"count"`
    TotalNetBuy float64           `json:"total_net_buy"`
    TotalNetSell float64          `json:"total_net_sell"`
    Cached      bool              `json:"cached"`
}

type DragonTigerHistoryResponse struct {
    OK               bool                       `json:"ok"`
    Dates            []string                   `json:"dates"`
    DailySummary     []DailySummary             `json:"daily_summary"`
    InstitutionStats []InstitutionStat           `json:"institution_stats"`
    RecurringStocks  []RecurringStock            `json:"recurring_stocks"`
    Cached           bool                        `json:"cached"`
}

type DailySummary struct {
    Date         string        `json:"date"`
    Count        int           `json:"count"`
    TotalNetBuy  float64       `json:"total_net_buy"`
    TotalNetSell float64       `json:"total_net_sell"`
    TopBuyers    []StockBrief  `json:"top_buyers"`
    TopSellers   []StockBrief  `json:"top_sellers"`
}

type StockBrief struct {
    Code   string  `json:"code"`
    Name   string  `json:"name"`
    NetBuy float64 `json:"net_buy"`
}

type InstitutionStat struct {
    Name        string   `json:"name"`
    Appearances int      `json:"appearances"`
    TotalNet    float64  `json:"total_net"`
    Dates       []string `json:"dates"`
}

type RecurringStock struct {
    Code        string   `json:"code"`
    Name        string   `json:"name"`
    Appearances int      `json:"appearances"`
    TotalNet    float64  `json:"total_net"`
    Dates       []string `json:"dates"`
}

type InstitutionTrackerResponse struct {
    OK          bool              `json:"ok"`
    Institutions []InstitutionStat `json:"institutions"`
    Period      string            `json:"period"`
    Cached      bool              `json:"cached"`
}
```

**Service methods (`internal/services/eastmoney.go`):**
Add to `EastMoneyService`:
1. `FetchDragonTiger(ctx) ([]DragonTigerItem, error)` - fetch latest dragon-tiger data
   - URL: `https://datacenter-web.eastmoney.com/api/data/v1/get?sortColumns=SECURITY_CODE&sortTypes=1&pageSize=50&pageNumber=1&reportName=RPT_DAILYBILLBOARD_DETAILSNEW&columns=ALL&filter=(TRADE_DATE>='YYYY-MM-DD')`
   - Use date from 3 days ago
   - Also fetch buy/sell department details from RPT_BILLBOARD_DAILYDETAILSBUY and RPT_BILLBOARD_DAILYDETAILSSELL
   
2. `FetchDragonTigerHistory(ctx, days int) (*DragonTigerHistoryResponse, error)` - fetch multi-day history
   - Call FetchDragonTiger per day, aggregate stats
   
3. `FetchInstitutionTracker(ctx, days int) ([]InstitutionStat, error)` - institution tracking
   - Reuse dragon-tiger history data, focus on institution aggregation

**Handler (`internal/handlers/dragon_tiger.go`):**
```go
type DragonTigerHandler struct {
    eastMoney *services.EastMoneyService
    // caches
    dragonTigerCache *cache.Cache[models.DragonTigerResponse]
    historyCache     *cache.Cache[models.DragonTigerHistoryResponse]
    institutionCache *cache.Cache[models.InstitutionTrackerResponse]
}
```

Methods:
1. `GetDragonTiger(c *gin.Context)` - GET /api/dragon-tiger
2. `GetHistory(c *gin.Context)` - GET /api/dragon-tiger-history?days=5
3. `GetInstitutionTracker(c *gin.Context)` - GET /api/institution-tracker?days=5

**Routes in cmd/server/main.go:**
```go
dragonTigerGroup := api.Group("/dragon-tiger")
dragonTigerGroup.Use(authMiddleware)
dragonTigerGroup.GET("", dragonTigerHandler.GetDragonTiger)

api.GET("/dragon-tiger-history", authMiddleware, dragonTigerHandler.GetHistory)
api.GET("/institution-tracker", authMiddleware, dragonTigerHandler.GetInstitutionTracker)
```

## Important Patterns

### Cache usage (from existing code):
```go
cache := cache.New[SomeType]()
if val, ok := cache.Get(key); ok {
    return val, nil
}
// ... fetch ...
cache.Set(key, result, ttl)
```

### Error response:
```go
writeError(c, http.StatusInternalServerError, "CODE", "message")
```

### Logging:
```go
zap.L().Error("message", zap.Error(err), zap.String("key", "value"))
```

### EastMoney JSON fetch pattern:
```go
func (s *EastMoneyService) getJSON(ctx context.Context, url string, params url.Values, result interface{}) error {
    // ... existing implementation
}
```

For dragon-tiger, use the datacenter-web.eastmoney.com URL pattern (different from push2his):
```go
params := url.Values{}
params.Set("sortColumns", "SECURITY_CODE")
params.Set("sortTypes", "1")
params.Set("pageSize", "50")
params.Set("pageNumber", "1")
params.Set("reportName", "RPT_DAILYBILLBOARD_DETAILSNEW")
params.Set("columns", "ALL")
params.Set("filter", "(TRADE_DATE>='2024-01-01')")
```

## Files to create/modify:
1. `internal/models/analysis.go` - Update Announcement struct
2. `internal/models/dragon_tiger.go` - NEW: dragon-tiger models
3. `internal/services/eastmoney.go` - Add FetchDragonTiger, FetchDragonTigerHistory, update FetchStockAnnouncements
4. `internal/handlers/market.go` - Add Announcements method
5. `internal/handlers/dragon_tiger.go` - NEW: dragon-tiger handler
6. `internal/handlers/dragon_tiger_test.go` - NEW: tests
7. `cmd/server/main.go` - Register new routes

## Verification:
1. `go build ./...` must pass
2. `go test ./... -count=1 -short` must pass
3. `curl http://localhost:8899/health` must return OK

package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/cache"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DragonTigerHandler struct {
	eastMoney        *services.EastMoneyService
	dragonTigerCache *cache.Cache[models.DragonTigerResponse]
	historyCache     *cache.Cache[models.DragonTigerHistoryResponse]
	institutionCache *cache.Cache[models.InstitutionTrackerResponse]
}

func NewDragonTigerHandler(eastMoney *services.EastMoneyService) *DragonTigerHandler {
	return &DragonTigerHandler{
		eastMoney:        eastMoney,
		dragonTigerCache: cache.New[models.DragonTigerResponse](),
		historyCache:     cache.New[models.DragonTigerHistoryResponse](),
		institutionCache: cache.New[models.InstitutionTrackerResponse](),
	}
}

// @Summary      获取龙虎榜数据
// @Description  获取最新的龙虎榜交易数据，包含净买入金额和机构信息
// @Tags         dragon-tiger
// @Produce      json
// @Success      200  {object}  models.DragonTigerResponse
// @Router       /api/dragon-tiger [get]
func (h *DragonTigerHandler) GetDragonTiger(c *gin.Context) {
	if cached, ok := h.dragonTigerCache.Get("latest"); ok {
		cached.Cached = true
		c.JSON(http.StatusOK, cached)
		return
	}

	items, err := h.eastMoney.FetchDragonTiger(c.Request.Context())
	if err != nil {
		zap.L().Error("fetch dragon tiger failed", zap.Error(err))
		writeError(c, http.StatusInternalServerError, "DRAGON_TIGER_FETCH_FAILED", "failed to fetch dragon-tiger data")
		return
	}

	response := models.DragonTigerResponse{
		OK:     true,
		Items:  items,
		Count:  len(items),
		Cached: false,
	}
	for _, item := range items {
		if item.NetBuy >= 0 {
			response.TotalNetBuy += item.NetBuy
		} else {
			response.TotalNetSell += -item.NetBuy
		}
	}

	h.dragonTigerCache.Set("latest", response, 5*time.Minute)
	c.JSON(http.StatusOK, response)
}

// @Summary      获取龙虎榜历史数据
// @Description  获取指定天数范围内的龙虎榜历史交易记录
// @Tags         dragon-tiger
// @Produce      json
// @Param        days  query  int  false  "查询天数，默认5天"
// @Success      200  {object}  models.DragonTigerHistoryResponse
// @Router       /api/dragon-tiger-history [get]
func (h *DragonTigerHandler) GetHistory(c *gin.Context) {
	days, ok := parseDragonTigerDays(c)
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("history:%d", days)
	if cached, ok := h.historyCache.Get(cacheKey); ok {
		cached.Cached = true
		c.JSON(http.StatusOK, cached)
		return
	}

	response, err := h.eastMoney.FetchDragonTigerHistory(c.Request.Context(), days)
	if err != nil {
		zap.L().Error("fetch dragon tiger history failed", zap.Error(err), zap.Int("days", days))
		writeError(c, http.StatusInternalServerError, "DRAGON_TIGER_HISTORY_FETCH_FAILED", "failed to fetch dragon-tiger history")
		return
	}

	h.historyCache.Set(cacheKey, *response, 10*time.Minute)
	c.JSON(http.StatusOK, response)
}

// @Summary      机构追踪
// @Description  获取机构席位交易追踪数据
// @Tags         dragon-tiger
// @Produce      json
// @Param        days  query  int  false  "查询天数，默认5天"
// @Success      200  {object}  models.InstitutionTrackerResponse
// @Router       /api/institution-tracker [get]
func (h *DragonTigerHandler) GetInstitutionTracker(c *gin.Context) {
	days, ok := parseDragonTigerDays(c)
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("institution:%d", days)
	if cached, ok := h.institutionCache.Get(cacheKey); ok {
		cached.Cached = true
		c.JSON(http.StatusOK, cached)
		return
	}

	institutions, err := h.eastMoney.FetchInstitutionTracker(c.Request.Context(), days)
	if err != nil {
		zap.L().Error("fetch institution tracker failed", zap.Error(err), zap.Int("days", days))
		writeError(c, http.StatusInternalServerError, "INSTITUTION_TRACKER_FETCH_FAILED", "failed to fetch institution tracker")
		return
	}

	response := models.InstitutionTrackerResponse{
		OK:           true,
		Institutions: institutions,
		Period:       fmt.Sprintf("%d days", days),
		Cached:       false,
	}
	h.institutionCache.Set(cacheKey, response, 10*time.Minute)
	c.JSON(http.StatusOK, response)
}

func parseDragonTigerDays(c *gin.Context) (int, bool) {
	days := 5
	if rawDays := strings.TrimSpace(c.Query("days")); rawDays != "" {
		parsedDays, err := strconv.Atoi(rawDays)
		if err != nil || parsedDays < 1 || parsedDays > 30 {
			writeError(c, http.StatusBadRequest, "INVALID_DAYS", "days must be between 1 and 30")
			return 0, false
		}
		days = parsedDays
	}
	return days, true
}

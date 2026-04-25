package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	alphacache "alphapulse/internal/cache"
	apperrors "alphapulse/internal/errors"
	"alphapulse/internal/models"
	"alphapulse/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FundFlowHandler handles fund flow / money flow endpoints.
type FundFlowHandler struct {
	eastMoney *services.EastMoneyService
	logger    *zap.Logger
	flowCache *alphacache.Cache[[]models.MoneyFlowDay]
}

// NewFundFlowHandler creates a new FundFlowHandler.
func NewFundFlowHandler(eastMoney *services.EastMoneyService, logger *zap.Logger) *FundFlowHandler {
	return &FundFlowHandler{
		eastMoney: eastMoney,
		logger:    logger,
		flowCache: alphacache.New[[]models.MoneyFlowDay](),
	}
}

// Flow godoc
// @Summary      Get money flow data for a stock
// @Description  Returns daily money flow data (main/small/middle/big/huge net inflows) for the given stock code
// @Tags         fund-flow
// @Accept       json
// @Produce      json
// @Param        code  query    string  true  "Stock code (e.g. 600176, sh600176)"
// @Param        days  query    int     false "Number of days (1-120, default 5)"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/fund-flow/flow [get]
func (h *FundFlowHandler) Flow(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		writeAppError(c, apperrors.BadRequest("code is required"))
		return
	}
	if err := services.ValidateStockCode(code); err != nil {
		writeAppError(c, apperrors.BadRequest("invalid code format: "+err.Error()))
		return
	}

	normalized := services.NormalizeCode(code)
	code6 := services.StockCode6(normalized)

	days := 5
	if rawDays := strings.TrimSpace(c.Query("days")); rawDays != "" {
		parsedDays, err := strconv.Atoi(rawDays)
		if err != nil || parsedDays < 1 || parsedDays > 120 {
			writeAppError(c, apperrors.BadRequest("days must be between 1 and 120"))
			return
		}
		days = parsedDays
	}

	cacheKey := code6 + ":" + strconv.Itoa(days)
	if cached, ok := h.flowCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, gin.H{
			"code":   code6,
			"items":  cached,
			"source": "eastmoney",
			"cached": true,
		})
		return
	}

	flows, err := h.eastMoney.FetchMoneyFlow(c.Request.Context(), normalized, days)
	if err != nil {
		h.logger.Warn("failed to fetch money flow",
			zap.String("code", code6),
			zap.Error(err),
		)
		writeAppError(c, apperrors.Internal(err))
		return
	}

	h.flowCache.Set(cacheKey, flows, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"code":   code6,
		"items":  flows,
		"source": "eastmoney",
		"cached": false,
	})
}

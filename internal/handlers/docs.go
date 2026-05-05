package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DocsHandler serves API documentation.
type DocsHandler struct{}

// NewDocsHandler creates a new DocsHandler.
func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// Docs handles GET /api/docs — returns API endpoint documentation.
// @Summary API 接口文档
// @Description 返回所有可用 API 端点的文档
// @Tags docs
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/docs [get]
func (h *DocsHandler) Docs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   "AlphaPulse 股票综合分析服务 v3.0 (Go)",
		"base_url":  "http://localhost:8899",
		"endpoints": gin.H{
			// ---- 核心行情 (Module 1) ----
			"market": gin.H{
				"GET /api/market/quote":           "实时行情 (?code=600176)",
				"GET /api/market/kline":           "K线数据 (?code=600176&period=daily&count=60)",
				"GET /api/market/sectors":         "板块列表 (?code=600176)",
				"GET /api/market/overview":        "板块概览 (东财行业板块)",
				"GET /api/market/news":            "新闻资讯 (?code=600176&limit=10)",
				"GET /api/market/search":          "股票搜索 (?q=茅台)",
				"GET /api/market/top-movers":      "涨跌幅排行",
				"GET /api/market/session":         "市场状态 (开盘/收盘/午休)",
				"GET /api/market/trends":          "市场趋势",
				"GET /api/market/market-overview": "市场概览 (大盘指数)",
				"GET /api/market/breadth":         "市场宽度 (涨跌家数)",
				"GET /api/market/sentiment":       "市场情绪",
				"GET /api/market/hot-concepts":    "热门概念",
				"GET /api/market/hot-concepts/:code/stocks": "概念成分股",
			},
			// ---- 自选股 (Module 2) ----
			"watchlist": gin.H{
				"GET /api/watchlist":        "自选股列表",
				"POST /api/watchlist":       "添加自选股 (?code=600176)",
				"DELETE /api/watchlist/:code": "删除自选股",
				"POST /api/watchlist/batch":  "批量添加自选股",
			},
			// ---- 仪表盘 (Module 3) ----
			"dashboard": gin.H{
				"GET /api/market/overview":  "仪表盘摘要 (含大盘指数)",
				"GET /api/market/session":   "市场状态",
				"GET /api/market/trends":    "市场趋势",
				"GET /api/market/breadth":   "市场宽度",
				"GET /api/market/sentiment": "市场情绪",
			},
			// ---- 选股分析 (Module 4) ----
			"analysis": gin.H{
				"GET /api/candidates":              "AI 候选股 (?limit=50&strategy=xxx)",
				"GET /api/screener":                "选股器 (?min_score=70&limit=50)",
				"GET /api/analyze":                 "8维度综合分析 (?code=600176)",
				"GET /api/score-history/:code":     "评分历史",
				"GET /api/pattern-scanner":         "形态扫描 (?code=600176)",
				"GET /api/multi-trend":             "多周期趋势 (?code=600176)",
				"GET /api/correlation":             "相关性分析 (?code=600176)",
				"GET /api/stockinfo":               "个股详情 (含资金流向)",
			},
			// ---- 股票对比 (Module 5) ----
			"compare": gin.H{
				"GET /api/compare/sector":   "板块对比",
				"GET /api/compare/backtest": "回测对比 (?codes=600176,000001&days=30)",
			},
			// ---- 投资组合 (Module 6) ----
			"portfolio": gin.H{
				"GET /api/portfolio":          "持仓列表",
				"POST /api/portfolio":         "添加持仓",
				"PUT /api/portfolio/:id":      "更新持仓",
				"DELETE /api/portfolio/:id":   "删除持仓",
				"GET /api/portfolio/analytics": "组合分析",
				"GET /api/portfolio/risk":      "风险分析",
			},
			// ---- 交易日志 (Module 7) ----
			"trading_journal": gin.H{
				"GET /api/trading-journal":          "交易记录",
				"POST /api/trading-journal":         "添加交易",
				"DELETE /api/trading-journal/:id":   "删除交易",
				"GET /api/trading-journal/stats":    "交易统计",
				"GET /api/trading-journal/calendar": "交易日历",
				"GET /api/trade-strategy-eval":      "策略评估",
			},
			// ---- 自定义策略 (Module 8) ----
			"strategies": gin.H{
				"GET /api/strategies":                "策略列表",
				"POST /api/strategies":               "创建策略",
				"PUT /api/strategies/:id":            "更新策略",
				"DELETE /api/strategies/:id":         "删除策略",
				"POST /api/strategies/:id/activate":  "激活策略",
				"POST /api/strategies/:id/deactivate": "停用策略",
				"GET /api/candidates?strategy=xxx":    "策略选股",
			},
			// ---- 自定义预警 (Module 9) ----
			"custom_alerts": gin.H{
				"GET /api/custom-alerts":       "预警列表",
				"POST /api/custom-alerts":      "创建预警",
				"DELETE /api/custom-alerts/:id": "删除预警",
				"GET /api/custom-alerts/check":  "检查预警",
			},
			// ---- 股票笔记 (Module 10) ----
			"stock_notes": gin.H{
				"GET /api/stock-notes/:code":        "获取笔记",
				"POST /api/stock-notes":             "添加笔记",
				"PUT /api/stock-notes/:id":          "更新笔记",
				"DELETE /api/stock-notes/:id":       "删除笔记",
				"GET /api/stock-notes/tags/all":     "所有标签",
			},
			// ---- 龙虎榜/机构 (Module 11) ----
			"dragon_tiger": gin.H{
				"GET /api/dragon-tiger":          "龙虎榜",
				"GET /api/dragon-tiger-history":  "龙虎榜历史",
				"GET /api/institution-tracker":   "机构追踪",
			},
			// ---- 热门概念 (Module 12) ----
			"hot_concepts": gin.H{
				"GET /api/market/hot-concepts":             "热门概念",
				"GET /api/market/hot-concepts/:code/stocks": "概念成分股",
				"GET /api/watchlist-concept-overlap":        "自选股概念重叠",
			},
			// ---- 资金流向 (Module 13) ----
			"fund_flow": gin.H{
				"GET /api/fund-flow/flow": "资金流向 (?code=600176&days=5)",
				"GET /flow":               "资金流向 (兼容旧路径)",
				"GET /api/stockinfo":      "个股详情 (含资金流向)",
			},
			// ---- 公告数据 (Module 14) ----
			"announcements": gin.H{
				"GET /api/announcements": "公告列表 (?code=600176&limit=10)",
			},
			// ---- 板块轮动 (Module 15) ----
			"sector_rotation": gin.H{
				"GET /api/sector-rotation":         "板块轮动",
				"GET /api/sector-rotation/history":  "轮动历史",
			},
			// ---- 报告系统 (Module 16) ----
			"reports": gin.H{
				"GET /reports":                   "报告列表 (重定向到API)",
				"GET /api/reports":               "报告列表",
				"GET /api/reports/:filename":     "查看报告",
				"POST /api/daily-report/generate": "生成日报",
				"GET /api/daily-report/latest":    "最新日报",
				"GET /api/daily-report/list":      "日报列表",
				"GET /api/daily-brief":            "每日简报",
			},
			// ---- 投资计划 (Module 17) ----
			"investment_plans": gin.H{
				"GET /api/investment-plans":       "计划列表",
				"POST /api/investment-plans":      "创建/更新计划",
				"DELETE /api/investment-plans/:code": "删除计划",
			},
			// ---- 信号系统 (Module 18) ----
			"signals": gin.H{
				"GET /api/signal-calendar":  "信号日历",
				"GET /api/signal-history":   "信号历史",
				"GET /api/anomalies":        "异常检测",
				"GET /api/alerts":           "系统告警",
				"GET /api/activity-log":     "活动日志",
			},
			// ---- 自选股分析 (Module 19) ----
			"watchlist_analysis": gin.H{
				"GET /api/watchlist-heatmap":           "热力图数据",
				"GET /api/watchlist-sectors":            "板块分布",
				"GET /api/watchlist-ranking":            "排名",
				"GET /api/watchlist-groups":             "分组列表",
				"POST /api/watchlist-groups":            "创建分组",
				"PUT /api/watchlist-groups/:id":         "更新分组",
				"DELETE /api/watchlist-groups/:id":      "删除分组",
				"POST /api/watchlist-groups/assign":     "分配分组",
			},
			// ---- 系统管理 (Module 21) ----
			"system": gin.H{
				"GET /health":                "健康检查",
				"GET /api/system/info":       "系统信息",
				"GET /api/system/datasources": "数据源状态",
				"GET /api/system-status":     "系统状态",
				"GET /api/status":            "服务状态",
				"GET /api/docs":              "API 文档 (本页)",
				"GET /api/slow-queries":      "慢查询",
				"GET /api/performance-stats": "性能统计",
				"POST /api/cache/clear":      "清除缓存",
				"GET /api/activity-log":      "活动日志",
			},
			// ---- 认证 ----
			"auth": gin.H{
				"POST /api/auth/register": "用户注册",
				"POST /api/auth/login":    "用户登录",
				"GET /api/auth/verify":    "验证 Token",
			},
			// ---- 管理 ----
			"admin": gin.H{
				"POST /api/admin/invite-codes":      "创建邀请码",
				"GET /api/admin/invite-codes":        "邀请码列表",
				"DELETE /api/admin/invite-codes/:id": "删除邀请码",
			},
		},
		"cache_policy_seconds": gin.H{
			"quote":    5,
			"klines":   60,
			"flow":     60,
			"sectors":  300,
			"news":     120,
			"candidates": 300,
		},
		"analysis_dimensions": []string{
			"order_flow",
			"volume_price",
			"valuation",
			"volatility",
			"money_flow",
			"technical",
			"sector",
			"sentiment",
			"summary",
		},
	})
}

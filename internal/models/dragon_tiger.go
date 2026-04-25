package models

type DragonTigerItem struct {
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Close       float64            `json:"close"`
	ChangePct   float64            `json:"change_pct"`
	NetBuy      float64            `json:"net_buy"`
	BuyTotal    float64            `json:"buy_total"`
	SellTotal   float64            `json:"sell_total"`
	Reason      string             `json:"reason"`
	TradeDate   string             `json:"trade_date"`
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
	OK           bool              `json:"ok"`
	Items        []DragonTigerItem `json:"items"`
	Count        int               `json:"count"`
	TotalNetBuy  float64           `json:"total_net_buy"`
	TotalNetSell float64           `json:"total_net_sell"`
	Cached       bool              `json:"cached"`
}

type DragonTigerHistoryResponse struct {
	OK               bool              `json:"ok"`
	Dates            []string          `json:"dates"`
	DailySummary     []DailySummary    `json:"daily_summary"`
	InstitutionStats []InstitutionStat `json:"institution_stats"`
	RecurringStocks  []RecurringStock  `json:"recurring_stocks"`
	Cached           bool              `json:"cached"`
}

type DailySummary struct {
	Date         string       `json:"date"`
	Count        int          `json:"count"`
	TotalNetBuy  float64      `json:"total_net_buy"`
	TotalNetSell float64      `json:"total_net_sell"`
	TopBuyers    []StockBrief `json:"top_buyers"`
	TopSellers   []StockBrief `json:"top_sellers"`
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
	OK           bool              `json:"ok"`
	Institutions []InstitutionStat `json:"institutions"`
	Period       string            `json:"period"`
	Cached       bool              `json:"cached"`
}

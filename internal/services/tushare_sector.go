package services

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"alphapulse/internal/models"
)

// TushareSectorService provides sector rotation and market breadth data from Tushare tables.
type TushareSectorService struct {
	pool *pgxpool.Pool
}

// NewTushareSectorService creates a new TushareSectorService.
func NewTushareSectorService(pool *pgxpool.Pool) *TushareSectorService {
	return &TushareSectorService{pool: pool}
}

// GetSectorRotation computes sector rotation from tushare_daily + tushare_stock_basic.
func (s *TushareSectorService) GetSectorRotation(ctx context.Context) ([]models.SectorRotationItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.industry,
		       AVG(d.pct_chg) as avg_pct,
		       COUNT(*) as total,
		       COUNT(*) FILTER (WHERE d.pct_chg > 0) as up_count,
		       COUNT(*) FILTER (WHERE d.pct_chg < 0) as down_count,
		       SUM(d.amount) as total_amount,
		       AVG(d.close) as avg_price
		FROM tushare_daily d
		JOIN tushare_stock_basic b ON d.ts_code = b.ts_code
		WHERE d.trade_date = (SELECT MAX(trade_date) FROM tushare_daily)
		  AND b.industry IS NOT NULL AND b.industry != ''
		GROUP BY b.industry
		ORDER BY avg_pct DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.SectorRotationItem
	for rows.Next() {
		var (
			industry   string
			avgPct     float64
			total      int
			upCount    int
			downCount  int
			totalAmt   float64
			avgPrice   float64
		)
		if err := rows.Scan(&industry, &avgPct, &total, &upCount, &downCount, &totalAmt, &avgPrice); err != nil {
			return nil, err
		}

		breadthRatio := 0.0
		if downCount > 0 {
			breadthRatio = float64(upCount) / float64(downCount)
		} else if upCount > 0 {
			breadthRatio = float64(upCount)
		}

		// Simple strength score: combination of avg pct change and breadth
		strengthScore := avgPct * (1 + breadthRatio*0.1)

		item := models.SectorRotationItem{
			Code:          industry,
			Name:          industry,
			ChangePct:     avgPct,
			Price:         avgPrice,
			RisingCount:   upCount,
			FallingCount:  downCount,
			BreadthRatio:  breadthRatio,
			NetFlow:       totalAmt,
			StrengthScore: strengthScore,
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

// GetSectors returns sectors with average change percent.
func (s *TushareSectorService) GetSectors(ctx context.Context) ([]models.Sector, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT b.industry,
		       AVG(d.pct_chg) as avg_pct
		FROM tushare_daily d
		JOIN tushare_stock_basic b ON d.ts_code = b.ts_code
		WHERE d.trade_date = (SELECT MAX(trade_date) FROM tushare_daily)
		  AND b.industry IS NOT NULL AND b.industry != ''
		GROUP BY b.industry
		ORDER BY avg_pct DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sectors []models.Sector
	for rows.Next() {
		var sec models.Sector
		if err := rows.Scan(&sec.Code, &sec.ChangePercent); err != nil {
			return nil, err
		}
		sec.Name = sec.Code
		sectors = append(sectors, sec)
	}
	return sectors, rows.Err()
}

// GetMarketBreadth returns advance/decline/flat counts for the latest trading day.
func (s *TushareSectorService) GetMarketBreadth(ctx context.Context) (models.MarketBreadth, error) {
	var b models.MarketBreadth
	err := s.pool.QueryRow(ctx, `
		SELECT 
		  COUNT(*) FILTER (WHERE pct_chg > 0),
		  COUNT(*) FILTER (WHERE pct_chg < 0),
		  COUNT(*) FILTER (WHERE pct_chg = 0)
		FROM tushare_daily
		WHERE trade_date = (SELECT MAX(trade_date) FROM tushare_daily)
	`).Scan(&b.UpCount, &b.DownCount, &b.FlatCount)
	if err != nil {
		return b, err
	}

	// Compute sentiment ratio and label
	total := b.UpCount + b.DownCount + b.FlatCount
	if total > 0 {
		b.SentimentRatio = float64(b.UpCount) / float64(total) * 100
	}
	switch {
	case b.SentimentRatio >= 70:
		b.Sentiment = "bullish"
	case b.SentimentRatio >= 55:
		b.Sentiment = "slightly_bullish"
	case b.SentimentRatio >= 45:
		b.Sentiment = "neutral"
	case b.SentimentRatio >= 30:
		b.Sentiment = "slightly_bearish"
	default:
		b.Sentiment = "bearish"
	}

	return b, nil
}

// GetTopMovers returns top gainers or losers. sort should be "desc" (gainers) or "asc" (losers).
func (s *TushareSectorService) GetTopMovers(ctx context.Context, sort string, limit int) ([]models.TopMover, error) {
	if limit <= 0 {
		limit = 10
	}
	order := "DESC"
	if strings.EqualFold(sort, "asc") {
		order = "ASC"
	}

	rows, err := s.pool.Query(ctx, `
		SELECT d.ts_code, b.name, d.close, d.change, d.pct_chg, d.vol, d.amount
		FROM tushare_daily d
		JOIN tushare_stock_basic b ON d.ts_code = b.ts_code
		WHERE d.trade_date = (SELECT MAX(trade_date) FROM tushare_daily)
		ORDER BY d.pct_chg `+order+`
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movers []models.TopMover
	for rows.Next() {
		var m models.TopMover
		if err := rows.Scan(&m.Code, &m.Name, &m.Price, &m.Change, &m.ChangePercent, &m.Volume, &m.Amount); err != nil {
			return nil, err
		}
		movers = append(movers, m)
	}
	return movers, rows.Err()
}

// GetOverview returns market overview with major index data and breadth.
func (s *TushareSectorService) GetOverview(ctx context.Context) (models.MarketOverview, error) {
	overview := models.MarketOverview{
		UpdatedAt: time.Now(),
	}

	// Try to get major index data from tushare_index_daily
	indexCodes := []string{"000001.SH", "399001.SZ", "399006.SZ"}
	rows, err := s.pool.Query(ctx, `
		SELECT ts_code, close, change, pct_chg
		FROM tushare_index_daily
		WHERE trade_date = (SELECT MAX(trade_date) FROM tushare_index_daily)
		  AND ts_code = ANY($1)
	`, indexCodes)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var idx models.OverviewIndex
			if err := rows.Scan(&idx.Code, &idx.Price, &idx.Change, &idx.ChangePercent); err != nil {
				return overview, err
			}
			// Map code to friendly name
			switch idx.Code {
			case "000001.SH":
				idx.Name = "上证指数"
			case "399001.SZ":
				idx.Name = "深证成指"
			case "399006.SZ":
				idx.Name = "创业板指"
			default:
				idx.Name = idx.Code
			}
			overview.Indices = append(overview.Indices, idx)
		}
		if err := rows.Err(); err != nil {
			return overview, err
		}
	}
	// If tushare_index_daily doesn't exist, indices will be nil (JSON: null)

	// Add breadth counts
	err = s.pool.QueryRow(ctx, `
		SELECT 
		  COUNT(*) FILTER (WHERE pct_chg > 0),
		  COUNT(*) FILTER (WHERE pct_chg < 0),
		  COUNT(*) FILTER (WHERE pct_chg = 0)
		FROM tushare_daily
		WHERE trade_date = (SELECT MAX(trade_date) FROM tushare_daily)
	`).Scan(&overview.AdvanceCount, &overview.DeclineCount, &overview.FlatCount)
	if err != nil {
		return overview, err
	}

	return overview, nil
}

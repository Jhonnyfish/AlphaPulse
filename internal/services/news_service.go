package services

import (
	"context"
	"time"

	"alphapulse/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// NewsService provides news data from DB with EastMoney as fallback.
type NewsService struct {
	pool      *pgxpool.Pool
	eastMoney *EastMoneyService
	logger    *zap.Logger
}

// NewNewsService creates a new NewsService.
func NewNewsService(pool *pgxpool.Pool, eastMoney *EastMoneyService, logger *zap.Logger) *NewsService {
	return &NewsService{
		pool:      pool,
		eastMoney: eastMoney,
		logger:    logger,
	}
}

// GetStockNews returns news for a stock. Reads from DB first, falls back to EastMoney.
func (s *NewsService) GetStockNews(ctx context.Context, code string, limit int) ([]models.NewsItem, error) {
	if limit <= 0 {
		limit = 10
	}

	// Try DB first
	rows, err := s.pool.Query(ctx, `
		SELECT code, title, summary, source, url, published_at
		FROM stock_news
		WHERE code = $1
		ORDER BY published_at DESC
		LIMIT $2
	`, code, limit)
	if err == nil {
		defer rows.Close()
		var items []models.NewsItem
		for rows.Next() {
			var item models.NewsItem
			if err := rows.Scan(&item.Code, &item.Title, &item.Summary, &item.Source, &item.URL, &item.PublishedAt); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
	}

	// Fallback: fetch from EastMoney and store
	items, err := s.eastMoney.FetchStockNews(ctx, code, limit)
	if err != nil {
		return nil, err
	}

	// Store in background
	go s.storeNews(items)

	return items, nil
}

// GetStockNewsDBOnly reads news from DB only, no external API fallback.
// Returns empty slice (not error) if no data in DB.
func (s *NewsService) GetStockNewsDBOnly(ctx context.Context, code string, limit int) ([]models.NewsItem, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT code, title, summary, source, url, published_at
		FROM stock_news WHERE code = $1
		ORDER BY published_at DESC LIMIT $2
	`, code, limit)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	var items []models.NewsItem
	for rows.Next() {
		var item models.NewsItem
		if err := rows.Scan(&item.Code, &item.Title, &item.Summary, &item.Source, &item.URL, &item.PublishedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// GetStockAnnouncements returns announcements for a stock.
func (s *NewsService) GetStockAnnouncements(ctx context.Context, code string, limit int) ([]models.Announcement, error) {
	if limit <= 0 {
		limit = 10
	}

	// Try DB first
	rows, err := s.pool.Query(ctx, `
		SELECT title, url, published_at
		FROM stock_announcements
		WHERE code = $1
		ORDER BY published_at DESC
		LIMIT $2
	`, code, limit)
	if err == nil {
		defer rows.Close()
		var items []models.Announcement
		for rows.Next() {
			var item models.Announcement
			if err := rows.Scan(&item.Title, &item.URL, &item.PublishedAt); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return items, nil
		}
	}

	// Fallback: fetch from EastMoney and store
	items, err := s.eastMoney.FetchStockAnnouncements(ctx, code, limit)
	if err != nil {
		return nil, err
	}

	// Store in background
	go s.storeAnnouncements(items)

	return items, nil
}

// GetStockAnnouncementsDBOnly reads announcements from DB only, no external API fallback.
func (s *NewsService) GetStockAnnouncementsDBOnly(ctx context.Context, code string, limit int) ([]models.Announcement, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT title, url, published_at
		FROM stock_announcements WHERE code = $1
		ORDER BY published_at DESC LIMIT $2
	`, code, limit)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()
	var items []models.Announcement
	for rows.Next() {
		var item models.Announcement
		if err := rows.Scan(&item.Title, &item.URL, &item.PublishedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// storeNews inserts news items into DB (upsert, ignore duplicates).
func (s *NewsService) storeNews(items []models.NewsItem) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, item := range items {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO stock_news (code, title, summary, source, url, published_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (code, title, published_at) DO NOTHING
		`, item.Code, item.Title, item.Summary, item.Source, item.URL, item.PublishedAt)
		if err != nil {
			s.logger.Warn("failed to store news", zap.String("code", item.Code), zap.Error(err))
		}
	}
}

// storeAnnouncements inserts announcements into DB (upsert, ignore duplicates).
func (s *NewsService) storeAnnouncements(items []models.Announcement) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, item := range items {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO stock_announcements (code, title, url, published_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code, title, published_at) DO NOTHING
		`, "", item.Title, item.URL, item.PublishedAt)
		if err != nil {
			s.logger.Warn("failed to store announcement", zap.Error(err))
		}
	}
}

// CleanupOldData deletes news and announcements older than 6 months.
func (s *NewsService) CleanupOldData(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, -6, 0)

	tag1, err := s.pool.Exec(ctx, `DELETE FROM stock_news WHERE published_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}

	tag2, err := s.pool.Exec(ctx, `DELETE FROM stock_announcements WHERE published_at < $1`, cutoff)
	if err != nil {
		return tag1.RowsAffected(), err
	}

	total := tag1.RowsAffected() + tag2.RowsAffected()
	s.logger.Info("cleaned up old news data",
		zap.Int64("news_deleted", tag1.RowsAffected()),
		zap.Int64("announcements_deleted", tag2.RowsAffected()),
		zap.Time("cutoff", cutoff),
	)
	return total, nil
}

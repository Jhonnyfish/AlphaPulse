package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alphapulse/internal/models"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type TencentService struct {
	client *http.Client
}

func NewTencentService(timeout time.Duration) *TencentService {
	return &TencentService{
		client: &http.Client{Timeout: timeout},
	}
}

func (s *TencentService) FetchQuote(ctx context.Context, code string) (models.Quote, error) {
	symbol := TencentSymbol(code)
	requestURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return models.Quote{}, err
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return models.Quote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return models.Quote{}, fmt.Errorf("tencent request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Quote{}, err
	}

	// Decode GBK → UTF-8
	decoded, _, _ := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), body)

	quote, err := parseTencentQuote(code, string(decoded))
	if err != nil {
		return models.Quote{}, err
	}
	if err := quote.Validate(); err != nil {
		return models.Quote{}, fmt.Errorf("tencent data validation failed: %w", err)
	}
	return quote, nil
}

func parseTencentQuote(code, payload string) (models.Quote, error) {
	start := strings.Index(payload, "\"")
	end := strings.LastIndex(payload, "\"")
	if start < 0 || end <= start {
		return models.Quote{}, fmt.Errorf("unexpected quote payload")
	}

	fields := strings.Split(payload[start+1:end], "~")
	if len(fields) < 35 {
		fields = strings.Split(payload[start+1:end], "|")
	}
	if len(fields) < 10 {
		return models.Quote{}, fmt.Errorf("quote payload missing fields")
	}

	price, _ := strconv.ParseFloat(fieldValue(fields, 3), 64)
	prevClose, _ := strconv.ParseFloat(fieldValue(fields, 4), 64)
	open, _ := strconv.ParseFloat(fieldValue(fields, 5), 64)
	change, _ := strconv.ParseFloat(fieldValue(fields, 31), 64)
	changePercent, _ := strconv.ParseFloat(fieldValue(fields, 32), 64)
	high, _ := strconv.ParseFloat(fieldValue(fields, 33), 64)
	low, _ := strconv.ParseFloat(fieldValue(fields, 34), 64)

	return models.Quote{
		Code:          code,
		Name:          fieldValue(fields, 1),
		Price:         price,
		Open:          open,
		PrevClose:     prevClose,
		High:          high,
		Low:           low,
		Change:        change,
		ChangePercent: changePercent,
		UpdatedAt:     fieldValue(fields, 30),
	}, nil
}

func (s *TencentService) HealthCheck(ctx context.Context) error {
	// Lightweight check: fetch Shanghai Composite Index quote
	symbol := "sh000001"
	requestURL := fmt.Sprintf("https://qt.gtimg.cn/q=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "AlphaPulse/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("tencent health check failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return fmt.Errorf("tencent returned empty response")
	}
	return nil
}

func fieldValue(fields []string, index int) string {
	if index < 0 || index >= len(fields) {
		return ""
	}
	return fields[index]
}

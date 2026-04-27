package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestParseKlinePoint(t *testing.T) {
	// Format: date,open,close,high,low,volume,amount
	parts := []string{"2026-04-25", "1845.00", "1850.00", "1860.00", "1835.00", "12345", "228000000"}
	point, err := parseKlinePoint(parts)
	if err != nil {
		t.Fatalf("parseKlinePoint failed: %v", err)
	}

	if point.Date != "2026-04-25" {
		t.Errorf("expected date 2026-04-25, got %s", point.Date)
	}
	if point.Open != 1845.00 {
		t.Errorf("expected open 1845.00, got %.2f", point.Open)
	}
	if point.Close != 1850.00 {
		t.Errorf("expected close 1850.00, got %.2f", point.Close)
	}
	if point.High != 1860.00 {
		t.Errorf("expected high 1860.00, got %.2f", point.High)
	}
	if point.Low != 1835.00 {
		t.Errorf("expected low 1835.00, got %.2f", point.Low)
	}
	if point.Volume != 12345 {
		t.Errorf("expected volume 12345, got %.0f", point.Volume)
	}
	if point.Amount != 228000000 {
		t.Errorf("expected amount 228000000, got %.0f", point.Amount)
	}

	// Validate the parsed data
	if err := point.Validate(); err != nil {
		t.Errorf("parsed kline point failed validation: %v", err)
	}
}

func TestParseKlinePointInvalidFloat(t *testing.T) {
	parts := []string{"2026-04-25", "abc", "1850.00", "1860.00", "1835.00", "12345", "228000000"}
	_, err := parseKlinePoint(parts)
	if err == nil {
		t.Error("expected error for invalid float in open")
	}
}

func TestParseKlinePointNegativeVolume(t *testing.T) {
	parts := []string{"2026-04-25", "100", "100", "100", "100", "-100", "100"}
	point, err := parseKlinePoint(parts)
	if err != nil {
		t.Fatalf("parseKlinePoint failed: %v", err)
	}
	// parseKlinePoint doesn't validate — it just parses.
	// Validation should happen at the caller level via Validate()
	if err := point.Validate(); err == nil {
		t.Error("expected validation error for negative volume")
	}
}

func TestParseEastMoneyTime(t *testing.T) {
	tests := []struct {
		input    string
		hasError bool
	}{
		{"2026-04-25 15:00:03", false},
		{"2026-04-25 15:00", false},
		{"2026-04-25T15:00:03Z", false},
		{"", true},  // No valid layout, returns zero time
		{"invalid", true},
	}

	for _, tt := range tests {
		got := parseEastMoneyTime(tt.input)
		if tt.hasError {
			if !got.IsZero() {
				t.Errorf("parseEastMoneyTime(%q) expected zero time, got %v", tt.input, got)
			}
		} else {
			if got.IsZero() {
				t.Errorf("parseEastMoneyTime(%q) returned zero time unexpectedly", tt.input)
			}
		}
	}
}

func TestParseEastMoneyTimeSpecific(t *testing.T) {
	got := parseEastMoneyTime("2026-04-25 15:00:03")
	expected := time.Date(2026, 4, 25, 15, 0, 3, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("parseEastMoneyTime = %v, want %v", got, expected)
	}
}

// ---------------------------------------------------------------------------
// FetchMoneyFlow fallback logic tests
// ---------------------------------------------------------------------------

// eastMoneyTestTransport is a custom http.RoundTripper that redirects requests
// matching eastmoney push2his/push2 URLs to a local test server.
type eastMoneyTestTransport struct {
	serverURL string
	// handler overrides per-host; if nil, all requests go to serverURL.
	primaryHandler   http.HandlerFunc // handles push2his requests
	fallbackHandler  http.HandlerFunc // handles push2 requests
}

func (t *eastMoneyTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite any eastmoney URL to point at our test server
	host := req.URL.Host
	var handler http.HandlerFunc

	switch {
	case strings.Contains(host, "push2his"):
		if t.primaryHandler != nil {
			handler = t.primaryHandler
		}
	case strings.Contains(host, "push2"):
		if t.fallbackHandler != nil {
			handler = t.fallbackHandler
		}
	}

	if handler != nil {
		// Use a ResponseRecorder-style approach: call the handler directly
		rec := &testResponseRecorder{header: http.Header{}, statusCode: 200}
		// Build a request that the handler can process
		handlerReq := req.Clone(context.Background())
		handlerReq.URL, _ = url.Parse(t.serverURL + req.URL.Path + "?" + req.URL.RawQuery)
		handler(rec, handlerReq)
		return rec.toResponse(), nil
	}

	// Default: redirect to test server
	req.URL, _ = url.Parse(t.serverURL + req.URL.Path + "?" + req.URL.RawQuery)
	return http.DefaultTransport.RoundTrip(req)
}

// testResponseRecorder captures an HTTP handler's response for use with a
// custom RoundTripper.
type testResponseRecorder struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *testResponseRecorder) Header() http.Header         { return r.header }
func (r *testResponseRecorder) Write(b []byte) (int, error)  { return r.body.Write(b) }
func (r *testResponseRecorder) WriteHeader(code int)         { r.statusCode = code }

func (r *testResponseRecorder) toResponse() *http.Response {
	return &http.Response{
		StatusCode: r.statusCode,
		Header:     r.header,
		Body:       io.NopCloser(bytes.NewReader(r.body.Bytes())),
	}
}

// newTestEastMoneyService creates a fresh EastMoneyService with a custom
// transport that routes requests to the given handlers.  A fresh service is
// needed per test to avoid cache pollution.
func newTestEastMoneyService(transport http.RoundTripper) *EastMoneyService {
	svc := NewEastMoneyService(5 * time.Second)
	svc.client = &http.Client{Timeout: 5 * time.Second, Transport: transport}
	return svc
}

// Sample kline response JSON helpers
func makeKlinesJSON(klines []string) []byte {
	resp := struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}{}
	resp.Data.Klines = klines
	b, _ := json.Marshal(resp)
	return b
}

const validKline = "2026-04-25,100.5,200.5,-50.5,150.5,300.5,1.23,2.34,3.45,4.56,5.67"

func TestFetchMoneyFlow_PrimarySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{validKline}))
	}))
	defer server.Close()

	transport := &eastMoneyTestTransport{
		serverURL:      server.URL,
		primaryHandler: nil, // will use default redirect
	}
	svc := newTestEastMoneyService(transport)

	flows, err := svc.FetchMoneyFlow(context.Background(), "600519", 5)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow day, got %d", len(flows))
	}

	f := flows[0]
	if f.Date != "2026-04-25" {
		t.Errorf("expected date 2026-04-25, got %s", f.Date)
	}
	// parseMoneyFlowValue divides by 10000: 100.5/10000 = 0.01005
	if f.MainNet != 100.5/10000 {
		t.Errorf("expected MainNet %f, got %f", 100.5/10000, f.MainNet)
	}
	if f.SmallNet != 200.5/10000 {
		t.Errorf("expected SmallNet %f, got %f", 200.5/10000, f.SmallNet)
	}
	if f.MiddleNet != -50.5/10000 {
		t.Errorf("expected MiddleNet %f, got %f", -50.5/10000, f.MiddleNet)
	}
	if f.BigNet != 150.5/10000 {
		t.Errorf("expected BigNet %f, got %f", 150.5/10000, f.BigNet)
	}
	if f.HugeNet != 300.5/10000 {
		t.Errorf("expected HugeNet %f, got %f", 300.5/10000, f.HugeNet)
	}
	if f.MainNetPct != 1.23 {
		t.Errorf("expected MainNetPct 1.23, got %f", f.MainNetPct)
	}
	if f.SmallNetPct != 2.34 {
		t.Errorf("expected SmallNetPct 2.34, got %f", f.SmallNetPct)
	}
	if f.MiddleNetPct != 3.45 {
		t.Errorf("expected MiddleNetPct 3.45, got %f", f.MiddleNetPct)
	}
	if f.BigNetPct != 4.56 {
		t.Errorf("expected BigNetPct 4.56, got %f", f.BigNetPct)
	}
	if f.HugeNetPct != 5.67 {
		t.Errorf("expected HugeNetPct 5.67, got %f", f.HugeNetPct)
	}
}

func TestFetchMoneyFlow_PrimaryFailsFallbackSucceeds(t *testing.T) {
	// Primary returns 400 (non-retryable), fallback returns valid data
	primaryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	})
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{validKline}))
	})

	server := httptest.NewServer(fallbackHandler)
	defer server.Close()

	transport := &eastMoneyTestTransport{
		serverURL:       server.URL,
		primaryHandler:  primaryHandler,
		fallbackHandler: fallbackHandler,
	}
	svc := newTestEastMoneyService(transport)

	flows, err := svc.FetchMoneyFlow(context.Background(), "600519", 5)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow day from fallback, got %d", len(flows))
	}
	if flows[0].Date != "2026-04-25" {
		t.Errorf("expected date 2026-04-25, got %s", flows[0].Date)
	}
}

func TestFetchMoneyFlow_BothFailGracefulDegradation(t *testing.T) {
	// Both endpoints return 400 — should return empty slice, nil error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	transport := &eastMoneyTestTransport{
		serverURL:       server.URL,
		primaryHandler:  handler,
		fallbackHandler: handler,
	}
	svc := newTestEastMoneyService(transport)

	flows, err := svc.FetchMoneyFlow(context.Background(), "600519", 5)
	if err != nil {
		t.Fatalf("expected nil error for graceful degradation, got: %v", err)
	}
	if len(flows) != 0 {
		t.Errorf("expected empty slice, got %d flows", len(flows))
	}
}

func TestFetchMoneyFlow_PrimaryEmptyFallbackReturnsData(t *testing.T) {
	callCount := 0
	primaryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Primary returns success but with empty klines
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{}))
	})
	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{validKline}))
	})

	server := httptest.NewServer(fallbackHandler)
	defer server.Close()

	transport := &eastMoneyTestTransport{
		serverURL:       server.URL,
		primaryHandler:  primaryHandler,
		fallbackHandler: fallbackHandler,
	}
	svc := newTestEastMoneyService(transport)

	flows, err := svc.FetchMoneyFlow(context.Background(), "600519", 5)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error: %v", err)
	}
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow day from fallback, got %d", len(flows))
	}
	// Primary should have been called at least once
	if callCount < 1 {
		t.Error("expected primary endpoint to be called")
	}
}

func TestFetchMoneyFlow_MalformedKlineSkipped(t *testing.T) {
	// Kline with < 11 fields should be skipped; valid one should still parse
	malformedKline := "2026-04-25,100.5,200.5" // only 3 fields
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{malformedKline, validKline}))
	}))
	defer server.Close()

	transport := &eastMoneyTestTransport{serverURL: server.URL}
	svc := newTestEastMoneyService(transport)

	flows, err := svc.FetchMoneyFlow(context.Background(), "600519", 5)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error: %v", err)
	}
	// Only the valid kline should be parsed; malformed one skipped
	if len(flows) != 1 {
		t.Fatalf("expected 1 flow day (malformed skipped), got %d", len(flows))
	}
	if flows[0].Date != "2026-04-25" {
		t.Errorf("expected date 2026-04-25, got %s", flows[0].Date)
	}
}

func TestFetchMoneyFlow_DaysZeroDefaultsTo10(t *testing.T) {
	// When days <= 0, the function should default to 10 and pass lmt=10 in the request
	var capturedLmt string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLmt = r.URL.Query().Get("lmt")
		w.Header().Set("Content-Type", "application/json")
		w.Write(makeKlinesJSON([]string{validKline}))
	}))
	defer server.Close()

	transport := &eastMoneyTestTransport{serverURL: server.URL}
	svc := newTestEastMoneyService(transport)

	_, err := svc.FetchMoneyFlow(context.Background(), "600519", 0)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error: %v", err)
	}
	if capturedLmt != "10" {
		t.Errorf("expected lmt=10 for days=0, got lmt=%s", capturedLmt)
	}

	// Also test negative days
	svc2 := newTestEastMoneyService(&eastMoneyTestTransport{serverURL: server.URL})
	_, err = svc2.FetchMoneyFlow(context.Background(), "600519", -5)
	if err != nil {
		t.Fatalf("FetchMoneyFlow returned error for negative days: %v", err)
	}
	if capturedLmt != "10" {
		t.Errorf("expected lmt=10 for negative days, got lmt=%s", capturedLmt)
	}
}

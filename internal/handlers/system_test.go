package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDataSourceHealthAllOK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sh := &SystemHandler{}
	r.GET("/api/system/datasources",
		sh.DataSourceHealth(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil },
		),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system/datasources", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}

	em := resp["eastmoney"].(map[string]interface{})
	if em["status"] != "ok" {
		t.Errorf("eastmoney status should be ok, got %v", em["status"])
	}
	if _, ok := em["latency"]; !ok {
		t.Error("eastmoney should have latency field")
	}

	tc := resp["tencent"].(map[string]interface{})
	if tc["status"] != "ok" {
		t.Errorf("tencent status should be ok, got %v", tc["status"])
	}
}

func TestDataSourceHealthEastMoneyFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sh := &SystemHandler{}
	r.GET("/api/system/datasources",
		sh.DataSourceHealth(
			func(ctx context.Context) error { return errors.New("connection refused") },
			func(ctx context.Context) error { return nil },
		),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system/datasources", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", resp["status"])
	}

	em := resp["eastmoney"].(map[string]interface{})
	if em["status"] != "error" {
		t.Errorf("eastmoney status should be error, got %v", em["status"])
	}
	if em["error"] != "connection refused" {
		t.Errorf("expected error message, got %v", em["error"])
	}
}

func TestDataSourceHealthTencentFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sh := &SystemHandler{}
	r.GET("/api/system/datasources",
		sh.DataSourceHealth(
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return errors.New("timeout") },
		),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system/datasources", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", resp["status"])
	}

	tc := resp["tencent"].(map[string]interface{})
	if tc["status"] != "error" {
		t.Errorf("tencent status should be error, got %v", tc["status"])
	}
}

func TestDataSourceHealthBothFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sh := &SystemHandler{}
	r.GET("/api/system/datasources",
		sh.DataSourceHealth(
			func(ctx context.Context) error { return errors.New("eastmoney down") },
			func(ctx context.Context) error { return errors.New("tencent down") },
		),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/system/datasources", nil)
	r.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "degraded" {
		t.Errorf("expected status degraded, got %v", resp["status"])
	}

	em := resp["eastmoney"].(map[string]interface{})
	tc := resp["tencent"].(map[string]interface{})
	if em["status"] != "error" || tc["status"] != "error" {
		t.Error("both sources should be in error state")
	}
}

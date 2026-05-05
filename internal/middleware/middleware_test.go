package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alphapulse/internal/models"

	"github.com/gin-gonic/gin"
)

const testSecret = "test-secret-key"

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── GenerateToken + ParseToken ─────────────────────────────────────

func TestTokenRoundTrip(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}

	tokenStr, err := GenerateToken(testSecret, user, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ParseToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if claims.Subject != "u1" {
		t.Errorf("expected subject u1, got %s", claims.Subject)
	}
	if claims.Username != "alice" {
		t.Errorf("expected username alice, got %s", claims.Username)
	}
	if claims.Role != "user" {
		t.Errorf("expected role user, got %s", claims.Role)
	}
}

func TestTokenAdminRole(t *testing.T) {
	user := models.AuthUser{ID: "u2", Username: "bob", Role: "admin"}

	tokenStr, err := GenerateToken(testSecret, user, 30*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := ParseToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if claims.Role != "admin" {
		t.Errorf("expected role admin, got %s", claims.Role)
	}
}

func TestTokenWrongSecret(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}

	tokenStr, err := GenerateToken(testSecret, user, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = ParseToken("wrong-secret", tokenStr)
	if err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestTokenExpired(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}

	// Token that expired 1 second ago
	tokenStr, err := GenerateToken(testSecret, user, -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = ParseToken(testSecret, tokenStr)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestTokenMalformed(t *testing.T) {
	_, err := ParseToken(testSecret, "not-a-valid-token")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

// ─── AuthRequired middleware ─────────────────────────────────────────

func setupAuthRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(AuthRequired(secret))
	r.GET("/protected", func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no user"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": user.ID, "username": user.Username, "role": user.Role})
	})
	return r
}

func TestAuthRequiredMissingHeader(t *testing.T) {
	r := setupAuthRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "MISSING_AUTH_HEADER" {
		t.Errorf("expected MISSING_AUTH_HEADER, got %v", resp["code"])
	}
}

func TestAuthRequiredInvalidHeaderFormat(t *testing.T) {
	r := setupAuthRouter(testSecret)

	tests := []string{
		"Token abc123",
		"Basic dXNlcjpwYXNz",
		"Bearer",
		"Bearer ",
	}
	for _, header := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", header)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: expected 401, got %d", header, w.Code)
		}
	}
}

func TestAuthRequiredInvalidToken(t *testing.T) {
	r := setupAuthRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalidtokenvalue")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "INVALID_TOKEN" {
		t.Errorf("expected INVALID_TOKEN, got %v", resp["code"])
	}
}

func TestAuthRequiredValidToken(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}
	tokenStr, err := GenerateToken(testSecret, user, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	r := setupAuthRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["user_id"] != "u1" {
		t.Errorf("expected user_id u1, got %v", resp["user_id"])
	}
	if resp["username"] != "alice" {
		t.Errorf("expected username alice, got %v", resp["username"])
	}
}

func TestAuthRequiredExpiredToken(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}
	tokenStr, err := GenerateToken(testSecret, user, -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	r := setupAuthRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ─── RequireAdmin middleware ─────────────────────────────────────────

func setupAdminRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(AuthRequired(secret))
	r.Use(RequireAdmin())
	r.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRequireAdminAsAdmin(t *testing.T) {
	user := models.AuthUser{ID: "u1", Username: "admin", Role: "admin"}
	tokenStr, _ := GenerateToken(testSecret, user, 1*time.Hour)

	r := setupAdminRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", w.Code)
	}
}

func TestRequireAdminAsUser(t *testing.T) {
	user := models.AuthUser{ID: "u2", Username: "regular", Role: "user"}
	tokenStr, _ := GenerateToken(testSecret, user, 1*time.Hour)

	r := setupAdminRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "ADMIN_REQUIRED" {
		t.Errorf("expected ADMIN_REQUIRED, got %v", resp["code"])
	}
}

func TestRequireAdminUnauthenticated(t *testing.T) {
	r := setupAdminRouter(testSecret)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated, got %d", w.Code)
	}
}

// ─── CORS middleware ─────────────────────────────────────────────────

func TestCORSHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected ACAO=*, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected non-empty ACAM")
	}
}

func TestCORSOptions(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

// ─── RequestLogger middleware ────────────────────────────────────────

func TestRequestLoggerStatusCode(t *testing.T) {
	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/notfound", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/test", 200},
		{"/notfound", 404},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tt.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != tt.wantStatus {
			t.Errorf("path %s: expected %d, got %d", tt.path, tt.wantStatus, w.Code)
		}
	}
}

// ─── CurrentUser helper ──────────────────────────────────────────────

func TestCurrentUserNotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	_, ok := CurrentUser(c)
	if ok {
		t.Error("expected false when user not set in context")
	}
}

func TestCurrentUserSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	expected := models.AuthUser{ID: "u1", Username: "alice", Role: "user"}
	c.Set("auth_user", expected)

	user, ok := CurrentUser(c)
	if !ok {
		t.Fatal("expected true when user set in context")
	}
	if user.ID != "u1" || user.Username != "alice" {
		t.Errorf("expected u1/alice, got %s/%s", user.ID, user.Username)
	}
}

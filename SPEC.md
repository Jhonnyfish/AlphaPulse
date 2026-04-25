# AlphaPulse Go Backend — Phase 1 Spec

## Overview
Build a Go backend API server for AlphaPulse, a Chinese A-share stock analysis platform.
The backend replaces an existing Python FastAPI server (server.py, 9700 lines).

## Tech Stack
- **Language**: Go 1.24
- **Web Framework**: Gin (github.com/gin-gonic/gin)
- **Database**: PostgreSQL 16 via pgx (github.com/jackc/pgx/v5)
- **Auth**: JWT (github.com/golang-jwt/jwt/v5) + bcrypt
- **Config**: godotenv (.env file)
- **Playwright**: github.com/playwright-community/playwright-go (for JS-rendered pages)

## Project Structure
```
alphapulse/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/config.go         # Config from .env
│   ├── database/db.go           # PostgreSQL connection pool
│   ├── models/                  # Data models
│   │   ├── user.go
│   │   ├── invite_code.go
│   │   └── watchlist.go
│   ├── handlers/                # HTTP handlers
│   │   ├── auth.go              # Login/Register/Invite
│   │   ├── watchlist.go         # Watchlist CRUD
│   │   ├── market.go            # Market data endpoints
│   │   └── system.go            # Health/status
│   ├── services/                # Business logic
│   │   ├── eastmoney.go         # EastMoney API fetcher (HTTP)
│   │   └── tencent.go           # Tencent quote fetcher (HTTP)
│   ├── middleware/
│   │   ├── auth.go              # JWT auth middleware
│   │   └── cors.go              # CORS middleware
│   └── cache/cache.go           # In-memory TTL cache
├── migrations/
│   └── 001_initial.sql          # Schema
├── .env.example
├── go.mod
└── Makefile
```

## Database Schema (PostgreSQL)

```sql
-- Users
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    role          VARCHAR(16) NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Invite codes
CREATE TABLE invite_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(32) UNIQUE NOT NULL,
    created_by  UUID REFERENCES users(id),
    max_uses    INT NOT NULL DEFAULT 1,
    uses        INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Watchlist (自选股)
CREATE TABLE watchlist (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(16) NOT NULL,           -- e.g. "600519"
    name       VARCHAR(64),
    group_name VARCHAR(64) DEFAULT 'default',
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(code)
);

-- Refresh tokens (for JWT rotation)
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## API Endpoints

### Auth (public)
```
POST /api/auth/register   { username, password, invite_code }  → { token, user }
POST /api/auth/login      { username, password }               → { token, user }
GET  /api/auth/verify     (check if token is valid)
```

### Auth (admin)
```
POST /api/admin/invite-codes   { max_uses?, expires_in? }  → { code }
GET  /api/admin/invite-codes                               → [ codes ]
DELETE /api/admin/invite-codes/{id}                        → { ok }
```

### Watchlist (authenticated)
```
GET    /api/watchlist              → [ stocks ]
POST   /api/watchlist              { code }  → { stock }
DELETE /api/watchlist/{code}        → { ok }
POST   /api/watchlist/batch        { codes: [...] }  → { added }
```

### Market Data (authenticated, cached)
```
GET /api/market/quote?code=600519         → real-time quote
GET /api/market/kline?code=600519&days=60 → K-line data
GET /api/market/sectors                   → sector ranking
GET /api/market/overview                  → market overview (up/down counts)
GET /api/market/news?limit=20             → news feed
```

### System
```
GET /health           → { status, version, uptime }
GET /api/system/info  → { db, cache, version }
```

## Auth Flow

1. Admin generates invite code: `POST /api/admin/invite-codes`
2. User registers with invite code: `POST /api/auth/register`
   - Validate invite code (exists, not expired, uses < max_uses)
   - Hash password with bcrypt
   - Create user, increment invite code uses
   - Return JWT access token (24h) + refresh token (7d)
3. User logs in: `POST /api/auth/login`
   - Verify password
   - Return JWT + refresh token
4. All authenticated endpoints check `Authorization: Bearer <token>`
5. JWT contains: user_id, username, role, exp

## JWT Structure
```json
{
  "sub": "user-uuid",
  "username": "finn",
  "role": "admin",
  "exp": 1714200000,
  "iat": 1714113600
}
```

## Data Fetching — EastMoney API

The existing Python code fetches from these EastMoney endpoints. Port to Go HTTP client:

### Real-time Quote (腾讯)
```
URL: https://qt.gtimg.cn/q=sh600519
Parse: pipe-delimited text response
```

### K-line (东方财富)
```
URL: https://push2his.eastmoney.com/api/qt/stock/kline/get
Params: secid=1.600519, klt=101, fqt=1, lmt=60, end=20500101
```

### Sector Ranking (东方财富)
```
URL: https://push2.eastmoney.com/api/qt/clist/get
Params: pn=1, pz=50, fs=m:90, fields=f2,f3,f4,f12,f14
```

### Market Overview (东方财富)
```
URL: https://push2.eastmoney.com/api/qt/ulist.np/get
Params: fltt=2, fields=f1,f2,f3,f4,f12,f14,f104,f105,f106
```

## In-Memory Cache

Simple TTL cache (no Redis dependency):
- Quote data: 3s TTL
- K-line: 30s TTL
- Sectors: 30s TTL
- Market overview: 10s TTL
- News: 60s TTL

Implementation: sync.RWMutex + map with expiration timestamps.

## Config (.env)

```env
PORT=8080
DATABASE_URL=postgres://localhost:5432/alphapulse?sslmode=disable
JWT_SECRET=change-me-to-a-random-string
JWT_EXPIRY=24h
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me
```

On first startup, if admin user doesn't exist, create it from ADMIN_USERNAME/ADMIN_PASSWORD.

## Error Response Format

```json
{
  "error": "invalid invite code",
  "code": "INVALID_INVITE_CODE"
}
```

HTTP status codes: 400 (bad request), 401 (unauthorized), 403 (forbidden), 404 (not found), 500 (internal).

## Implementation Order

1. `internal/config/config.go` — load .env
2. `internal/database/db.go` — pgx pool + migrations
3. `internal/models/*.go` — structs
4. `internal/cache/cache.go` — TTL cache
5. `internal/middleware/auth.go` — JWT middleware
6. `internal/middleware/cors.go` — CORS
7. `internal/handlers/auth.go` — register/login/invite
8. `internal/handlers/watchlist.go` — CRUD
9. `internal/services/eastmoney.go` — HTTP fetcher
10. `internal/services/tencent.go` — quote fetcher
11. `internal/handlers/market.go` — market endpoints
12. `internal/handlers/system.go` — health/info
13. `cmd/server/main.go` — wire everything together
14. `Makefile` — build/run/migrate commands
15. `.env.example`

## Constraints
- All code in Go, no CGO
- Use pgx directly (no ORM)
- JSON tags on all structs
- Context propagation (ctx context.Context) on all DB and HTTP calls
- Graceful shutdown on SIGINT/SIGTERM
- Request logging middleware
- All endpoints return JSON (except health which can return plain text)
- Keep each file under 500 lines — split if needed

## First Run Behavior
1. Connect to PostgreSQL
2. Run migrations (create tables if not exist)
3. Create admin user if not exists
4. Start HTTP server on PORT
5. Log: "AlphaPulse server running on :8080"

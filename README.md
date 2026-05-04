# AlphaPulse

A-share quantitative stock analysis platform with AI-powered insights.

## Overview

AlphaPulse is a full-stack web application for Chinese A-share stock market analysis. It combines real-time market data, technical indicators, AI analysis (via DeepSeek), and quantitative ranking (Alpha300) to help investors make informed decisions.

## Tech Stack

**Backend**
- Go 1.24 + Gin (HTTP framework)
- PostgreSQL 16 (data storage)
- EastMoney + Tencent Finance APIs (market data)
- DeepSeek API (AI analysis via CLI Proxy)
- Swagger (API documentation)

**Frontend**
- React 18 + TypeScript + Vite
- Tailwind CSS v4
- ECharts (charts & visualizations)
- Capacitor (Android APK build)

## Features

### Market Analysis
- Real-time stock quotes and K-line charts
- Sector overview and hot concept tracking
- Market breadth and sentiment indicators
- Dragon-tiger list (龙虎榜) analysis
- Top movers (gainers/losers)

### Quantitative Tools
- Alpha300 stock ranking system (quant factors: momentum 45%, volatility 30%, trend 25%)
- Multi-period trend analysis (daily/weekly/monthly)
- Pairwise correlation matrix
- Stock screener with customizable filters
- Backtest comparison

### Portfolio Management
- Watchlist with drag-and-drop sorting
- Portfolio tracking with P&L
- Risk analysis (VaR, drawdown)
- Trading journal
- Investment plans

### AI Integration
- Daily automated reports with AI commentary
- 8-dimension stock analysis (order flow, volume-price, valuation, volatility, money flow, technical, sector, sentiment)
- DeepSeek-powered market insights

### Automation
- Built-in scheduler for daily tasks
- Alpha300 top 10 auto-sync to watchlist (9:00 AM)
- Daily report generation (3:30 PM)
- Health checks and auto-restart

## Architecture

```
┌─────────────────────────────────────────────┐
│                   Frontend                   │
│         React + Vite (port 5173)            │
├─────────────────────────────────────────────┤
│                   Backend                    │
│          Go + Gin (port 8899)               │
├──────────┬──────────┬───────────────────────┤
│ EastMoney│ Tencent  │  DeepSeek (via CPA)   │
│   API    │ Finance  │  localhost:8317        │
└──────────┴──────────┴───────────────────────┘
```

## Quick Start

### Prerequisites
- Go 1.24+
- Node.js 20+
- PostgreSQL 16+

### Backend

```bash
# Clone
git clone https://github.com/Jhonnyfish/AlphaPulse.git
cd AlphaPulse

# Configure
cp .env.example .env
# Edit .env with your database URL, JWT secret, etc.

# Database setup
sudo -u postgres psql -d alphapulse -c "$(cat migrations/001_initial.sql)"

# Build & run
export PATH=$HOME/.local/go/bin:$PATH
go build -o bin/server ./cmd/server
export $(grep -v '^#' .env | grep -v '^$' | xargs) && ./bin/server
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173

### Android APK

```bash
cd frontend
npx cap sync
npx cap open android
```

## API Endpoints

| Category | Endpoints |
|----------|-----------|
| Auth | `POST /login`, `POST /register`, `GET /verify` |
| Watchlist | `GET/POST/DELETE /watchlist`, `POST /watchlist/sync` |
| Market | `GET /quote`, `GET /kline`, `GET /sectors`, `GET /news`, `GET /trends` |
| Analysis | `GET /analyze`, `GET /multi-trend`, `GET /correlation`, `GET /screener` |
| Ranking | `GET /candidates`, `GET /watchlist-ranking` |
| Reports | `GET /daily-report/latest`, `POST /daily-report/generate`, `GET /daily-brief` |
| Portfolio | `GET /portfolio`, `GET /risk`, `GET /trading-journal` |
| System | `GET /health`, `GET /system/info`, `GET /system/scheduler` |

Full API docs: http://localhost:8899/swagger/index.html

## Default Credentials

```
Username: admin
Password: admin123
```

## Project Structure

```
alphapulse/
├── cmd/server/          # Go entry point
├── internal/
│   ├── handlers/        # HTTP handlers (53 files)
│   ├── services/        # Business logic (EastMoney, Tencent, DeepSeek)
│   ├── models/          # Data models
│   ├── middleware/       # Auth, CORS, logging
│   ├── cache/           # In-memory TTL cache
│   └── database/        # PostgreSQL connection
├── migrations/          # SQL schema
├── frontend/
│   ├── src/
│   │   ├── pages/       # 40 React pages
│   │   ├── components/  # Shared components
│   │   ├── lib/         # API client, utils
│   │   └── assets/      # Static files
│   └── android/         # Capacitor Android project
└── docs/                # Swagger generated docs
```

## License

MIT

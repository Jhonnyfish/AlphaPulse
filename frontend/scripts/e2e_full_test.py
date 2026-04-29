#!/usr/bin/env python3
"""AlphaPulse E2E Full Test - tests all 39 views via Playwright."""
import json, time, sys, argparse
from pathlib import Path
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
SCREENSHOT_DIR = Path("/tmp/e2e_screenshots")
SCREENSHOT_DIR.mkdir(exist_ok=True)

ALL_VIEWS = [
    "dashboard", "watchlist", "market", "kline",
    "analyze", "sectors", "compare", "flow", "trends", "breadth", "sentiment",
    "multi-trend", "correlation",
    "candidates", "screener", "ranking", "hot-concepts", "dragon-tiger", "pattern-scanner",
    "portfolio", "journal", "strategies", "backtest", "strategy-eval",
    "trade-calendar", "signals", "portfolio-risk", "investment-plans",
    "watchlist-analysis", "news", "daily-brief", "daily-report",
    "institutions", "anomalies", "diag", "vitals", "perf-stats", "settings", "quick-actions",
]

VIEW_TO_LABEL = {
    "dashboard": "总览", "watchlist": "自选股", "market": "行情", "kline": "K线",
    "analyze": "个股分析", "sectors": "板块", "compare": "对比", "flow": "资金流向",
    "trends": "趋势", "breadth": "市场广度", "sentiment": "市场情绪",
    "multi-trend": "多周期趋势", "correlation": "相关性",
    "candidates": "候选股", "screener": "选股器", "ranking": "综合排名",
    "hot-concepts": "热门概念", "dragon-tiger": "龙虎榜", "pattern-scanner": "形态扫描",
    "portfolio": "持仓", "journal": "交易日志", "strategies": "策略",
    "backtest": "策略回测", "strategy-eval": "策略评估", "trade-calendar": "交易日历",
    "signals": "信号", "portfolio-risk": "组合风险", "investment-plans": "投资计划",
    "watchlist-analysis": "自选分析", "news": "资讯", "daily-brief": "每日简报",
    "daily-report": "每日报告", "institutions": "机构动向", "anomalies": "异常检测",
    "diag": "系统诊断", "vitals": "性能监控", "perf-stats": "绩效统计",
    "settings": "设置", "quick-actions": "快捷操作",
}

# Pages that call known-broken endpoints (for leaked endpoint filtering)
_PAGE_API_ENDPOINTS = {
    "dashboard": {"/api/dashboard-summary", "/api/market/overview"},
    "market": {"/api/market/overview"},
}

_LEAKED_ENDPOINTS = {"/api/market/overview"}


def test_all_pages():
    results = {}
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1920, "height": 1080})

        # Login
        page.goto(BASE_URL, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
        try:
            inp = page.locator('input[type="text"]').first
            if inp.is_visible(timeout=2000):
                inp.fill("admin")
                page.locator('input[type="password"]').first.fill("admin123")
                page.locator('button[type="submit"]').first.click()
                page.wait_for_load_state("networkidle", timeout=10000)
                time.sleep(1)
        except:
            pass

        # Drain pending API responses from login (axios retry backoff ~8s)
        _drain_errs = []
        def _drain(r):
            try:
                if "/api/" in r.url and r.status >= 400:
                    _drain_errs.append(f"{r.status} {r.url.split('?')[0]}")
            except:
                pass
        page.on("response", _drain)
        page.wait_for_timeout(10000)
        page.remove_listener("response", _drain)
        if _drain_errs:
            print(f"DRAIN: flushed {len(_drain_errs)} pending API errors from login", file=sys.stderr)

        # Expand sidebar groups
        for g in ["核心", "分析", "选股", "交易", "工具"]:
            try:
                page.locator(f'aside >> text="{g}"').first.click(timeout=500)
                page.wait_for_timeout(100)
            except:
                pass

        # Collect nav buttons
        nav_buttons = {}
        for btn in page.locator('aside button').all():
            try:
                t = btn.inner_text().strip().split("\n")[0].strip()
                if t and len(t) < 15 and t not in nav_buttons:
                    nav_buttons[t] = btn
            except:
                pass

        print(f"Found {len(nav_buttons)} nav buttons: {list(nav_buttons.keys())}", file=sys.stderr)

        # Test each page
        for i, view in enumerate(ALL_VIEWS):
            label = VIEW_TO_LABEL.get(view, view)
            btn = nav_buttons.get(label)
            if not btn:
                results[view] = {"status": "NO_NAV", "label": label}
                continue

            # Set up listeners
            page_errs, api_errs, js_errors, console_msgs = [], [], [], []

            def on_pe(e):
                page_errs.append(str(e)[:600])

            def on_resp(r):
                try:
                    if "/api/" in r.url and r.status >= 400:
                        api_errs.append(f"{r.status} {r.url.split('?')[0]}")
                except:
                    pass

            def on_console(msg):
                if msg.type in ('error', 'warning'):
                    console_msgs.append(f'{msg.type}: {msg.text[:300]}')

            page.on("pageerror", on_pe)
            page.on("response", on_resp)
            page.on("console", on_console)

            try:
                btn.click()
                page.wait_for_timeout(2500)

                crash = False
                try:
                    crash = page.locator('text="页面出错了"').is_visible(timeout=300)
                except:
                    pass

                # Check for JS errors in console
                js_err = any("map is not a function" in m or "is not a function" in m for m in console_msgs + page_errs)

                # Filter leaked endpoints
                own_endpoints = _PAGE_API_ENDPOINTS.get(view, set())
                filtered_api_errs = api_errs
                if api_errs and not own_endpoints.intersection(_LEAKED_ENDPOINTS):
                    filtered_api_errs = [e for e in api_errs if not any(ep in e for ep in _LEAKED_ENDPOINTS)]

                # Determine status
                if crash:
                    status = "CRASH"
                elif js_err:
                    status = "JS_ERROR"
                elif page_errs:
                    status = "PAGE_ERR"
                elif filtered_api_errs:
                    status = "API_ERR"
                else:
                    status = "OK"

                body = page.inner_text("body").strip()
                h1 = page.locator("h1, h2, h3").count()
                tables = page.locator("table").count()
                charts = page.locator("canvas, svg").count()

                ss = SCREENSHOT_DIR / f"{i+1:02d}_{view}.png"
                page.screenshot(path=str(ss))

                results[view] = {
                    "status": status,
                    "body_len": len(body),
                    "h1_count": h1, "table_count": tables, "chart_count": charts,
                    "api_errs": filtered_api_errs[:10],
                    "page_errs": page_errs[:5],
                    "console_msgs": console_msgs[:5],
                    "screenshot": str(ss),
                }
            except Exception as e:
                results[view] = {"status": "ERROR", "error": str(e)[:200]}
            finally:
                page.remove_listener("pageerror", on_pe)
                page.remove_listener("response", on_resp)
                page.remove_listener("console", on_console)

        browser.close()

    # Build report
    status_counts = {}
    failed_pages = []
    failed_details = {}
    for v, r in results.items():
        s = r.get("status", "UNKNOWN")
        status_counts[s] = status_counts.get(s, 0) + 1
        if s != "OK":
            failed_pages.append(v)
            failed_details[v] = r

    report = {
        "total": len(ALL_VIEWS),
        "status_counts": status_counts,
        "failed_pages": failed_pages,
        "failed_details": failed_details,
        "results": results,
    }

    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-o", "--output", help="Output JSON file path")
    args = parser.parse_args()

    report = test_all_pages()

    if args.output:
        with open(args.output, "w") as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
        print(f"Report saved to {args.output}", file=sys.stderr)

    # Print summary
    print(f"\n=== E2E Test Summary ===")
    print(f"Total: {report['total']}")
    for status, count in sorted(report['status_counts'].items()):
        print(f"  {status}: {count}")
    if report['failed_pages']:
        print(f"\nFailed pages ({len(report['failed_pages'])}):")
        for p in report['failed_pages']:
            d = report['failed_details'][p]
            print(f"  {p}: {d.get('status')} - {d.get('api_errs', d.get('error', ''))}")
    print(f"\nPass rate: {report['status_counts'].get('OK', 0)}/{report['total']}")

    sys.exit(0 if not report['failed_pages'] else 1)

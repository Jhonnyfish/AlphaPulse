#!/usr/bin/env python3
"""AlphaPulse Frontend E2E Test — visits every page, captures errors & screenshots."""

import json, os, time
from datetime import datetime
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
SCREENSHOT_DIR = "/home/finn/alphapulse/test_screenshots"
REPORT_PATH = "/home/finn/alphapulse/TEST_REPORT.md"

os.makedirs(SCREENSHOT_DIR, exist_ok=True)

# All menu items: (view_name, label, needs_param)
PAGES = [
    ("dashboard", "仪表盘", False),
    ("watchlist", "自选股", False),
    ("market", "行情", False),
    ("kline", "K线", False),
    ("analyze", "个股分析", False),
    ("sectors", "板块", False),
    ("compare", "对比", False),
    ("news", "新闻", False),
    ("portfolio", "持仓", False),
    ("journal", "交易日志", False),
    ("candidates", "候选股", False),
    ("screener", "选股器", False),
    ("dragon-tiger", "龙虎榜", False),
    ("hot-concepts", "热门概念", False),
    ("strategies", "策略", False),
    ("signals", "信号", False),
    ("watchlist-analysis", "自选分析", False),
    ("flow", "资金流向", False),
    ("trends", "趋势", False),
    ("breadth", "市场广度", False),
    ("sentiment", "市场情绪", False),
    ("daily-brief", "每日简报", False),
    ("diag", "系统诊断", False),
    ("anomalies", "异常检测", False),
    ("institutions", "机构追踪", False),
    ("ranking", "自选排名", False),
    ("daily-report", "每日报告", False),
    ("perf-stats", "绩效统计", False),
    ("multi-trend", "多周期趋势", False),
    ("correlation", "相关性", False),
    ("investment-plans", "投资计划", False),
    ("backtest", "回测", False),
    ("strategy-eval", "策略评估", False),
    ("trade-calendar", "交易日历", False),
    ("pattern-scanner", "形态扫描", False),
    ("portfolio-risk", "持仓风险", False),
    ("quick-actions", "快捷操作", False),
    ("settings", "设置", False),
]

def run_tests():
    results = []

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(
            viewport={"width": 1280, "height": 800},
            locale="zh-CN",
        )
        page = context.new_page()

        # Collect console errors
        console_errors = []
        page.on("console", lambda msg: console_errors.append(msg.text) if msg.type == "error" else None)
        page.on("pageerror", lambda err: console_errors.append(f"PAGE_ERROR: {err}"))

        # 1. Login
        print("🔐 Logging in...")
        page.goto(BASE_URL, wait_until="networkidle", timeout=15000)
        time.sleep(1)

        # Check if we need to login
        if page.locator("input[type='text'], input[placeholder*='用户']").count() > 0:
            page.fill("input[type='text'], input[placeholder*='用户']", "admin")
            page.fill("input[type='password']", "admin123")
            page.click("button[type='submit']")
            time.sleep(2)
            print("✅ Logged in")
        else:
            print("✅ Already logged in (token in localStorage)")

        # Wait for dashboard to load
        time.sleep(2)

        # 2. Test each page
        for view, label, _ in PAGES:
            console_errors.clear()
            print(f"\n📄 Testing: {label} ({view})")

            try:
                # Navigate by clicking sidebar or using JS
                page.evaluate(f"""
                    // Try to find and click the sidebar item
                    const items = document.querySelectorAll('button, a, [role="button"]');
                    for (const item of items) {{
                        if (item.textContent.includes('{label}') && item.offsetParent !== null) {{
                            item.click();
                            break;
                        }}
                    }}
                """)
                time.sleep(1.5)

                # Check for ErrorBoundary
                error_boundary = page.locator("text=页面加载出错").count() > 0

                # Check for empty data indicators
                has_loading = page.locator("text=加载中").count() > 0
                has_skeleton = page.locator(".animate-pulse").count() > 0

                # Wait a bit more for data to load
                time.sleep(2)

                # Recheck after waiting
                error_boundary = page.locator("text=页面加载出错").count() > 0
                has_error_text = page.locator("text=失败").count() > 0 or page.locator("text=出错").count() > 0

                # Check if page has any content
                body_text = page.inner_text("body")
                is_blank = len(body_text.strip()) < 50

                # Check for visible data tables or cards
                has_tables = page.locator("table").count() > 0
                has_data_cards = page.locator(".glass-panel").count() > 1

                # Take screenshot
                screenshot_path = f"{SCREENSHOT_DIR}/{view}.png"
                page.screenshot(path=screenshot_path, full_page=False)

                # Determine status
                if error_boundary:
                    status = "❌ CRASH"
                    detail = "ErrorBoundary triggered — 页面加载出错"
                elif is_blank:
                    status = "⚠️ BLANK"
                    detail = "Page is blank/empty"
                elif has_error_text and not has_data_cards:
                    status = "⚠️ ERROR"
                    detail = "Shows error message, no data"
                elif console_errors:
                    status = "⚠️ CONSOLE_ERR"
                    detail = f"Console errors: {'; '.join(console_errors[:3])}"
                elif has_tables or has_data_cards:
                    status = "✅ OK"
                    detail = f"Has content (tables:{has_tables}, cards:{has_data_cards})"
                else:
                    status = "✅ OK"
                    detail = "Page loaded"

                results.append({
                    "view": view,
                    "label": label,
                    "status": status,
                    "detail": detail,
                    "screenshot": screenshot_path,
                    "console_errors": list(console_errors),
                })

                print(f"  {status}: {detail}")

            except Exception as e:
                results.append({
                    "view": view,
                    "label": label,
                    "status": "❌ EXCEPTION",
                    "detail": str(e)[:200],
                    "screenshot": None,
                    "console_errors": [],
                })
                print(f"  ❌ EXCEPTION: {e}")

        browser.close()

    # 3. Generate report
    print("\n\n📊 Generating report...")
    with open(REPORT_PATH, "w") as f:
        f.write(f"# AlphaPulse Frontend Test Report\n")
        f.write(f"**Date:** {datetime.now().strftime('%Y-%m-%d %H:%M')}\n")
        f.write(f"**Total pages:** {len(results)}\n")

        ok = sum(1 for r in results if r["status"].startswith("✅"))
        warn = sum(1 for r in results if "⚠️" in r["status"])
        fail = sum(1 for r in results if "❌" in r["status"])
        f.write(f"**Results:** ✅ {ok} OK | ⚠️ {warn} Warnings | ❌ {fail} Failed\n\n")

        # Failed pages
        if fail:
            f.write("## ❌ Failed Pages (need immediate fix)\n\n")
            for r in results:
                if "❌" in r["status"]:
                    f.write(f"### {r['label']} (`{r['view']}`)\n")
                    f.write(f"- **Status:** {r['status']}\n")
                    f.write(f"- **Detail:** {r['detail']}\n")
                    if r['console_errors']:
                        f.write(f"- **Console errors:**\n")
                        for err in r['console_errors'][:5]:
                            f.write(f"  - `{err[:200]}`\n")
                    if r['screenshot']:
                        f.write(f"- **Screenshot:** `{r['screenshot']}`\n")
                    f.write("\n")

        # Warning pages
        if warn:
            f.write("## ⚠️ Warning Pages\n\n")
            for r in results:
                if "⚠️" in r["status"]:
                    f.write(f"### {r['label']} (`{r['view']}`)\n")
                    f.write(f"- **Status:** {r['status']}\n")
                    f.write(f"- **Detail:** {r['detail']}\n")
                    if r['console_errors']:
                        f.write(f"- **Console errors:**\n")
                        for err in r['console_errors'][:5]:
                            f.write(f"  - `{err[:200]}`\n")
                    f.write("\n")

        # Full table
        f.write("## Full Results\n\n")
        f.write("| Page | View | Status | Detail |\n")
        f.write("|------|------|--------|--------|\n")
        for r in results:
            f.write(f"| {r['label']} | `{r['view']}` | {r['status']} | {r['detail'][:80]} |\n")

    print(f"\n📝 Report saved to: {REPORT_PATH}")
    print(f"📸 Screenshots saved to: {SCREENSHOT_DIR}/")
    return results

if __name__ == "__main__":
    run_tests()

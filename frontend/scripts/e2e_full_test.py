#!/usr/bin/env python3
"""
AlphaPulse 全量 E2E 测试脚本
每30分钟由 cron 调度运行，检测所有页面是否正常
输出 JSON 报告到 /tmp/e2e_report.json
"""

import json
import time
import sys
from pathlib import Path
from datetime import datetime
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
SCREENSHOT_DIR = Path("/tmp/e2e_screenshots")
SCREENSHOT_DIR.mkdir(exist_ok=True)

# 从 Layout.tsx navItems 提取的完整映射
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

# 侧边栏分组
SIDEBAR_GROUPS = ["核心", "分析", "选股", "交易", "工具"]

# 已知会失败的端点（后端数据问题，不是前端 bug）
KNOWN_BROKEN_ENDPOINTS = {"/api/market/overview"}


def test_all_pages():
    """测试所有页面，返回结果报告"""
    results = {}
    start_time = datetime.now()

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1920, "height": 1080})

        # 登录
        print("🔐 登录中...")
        page.goto(BASE_URL, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
        try:
            inp = page.locator('input[type="text"]').first
            if inp.is_visible(timeout=2000):
                inp.fill("admin")
                page.locator('input[type="password"]').first.fill("admin123")
                page.locator('button[type="submit"]').first.click()
                page.wait_for_load_state("networkidle", timeout=10000)
                time.sleep(2)
        except Exception as e:
            print(f"❌ 登录失败: {e}")
            browser.close()
            return {"status": "LOGIN_FAILED", "error": str(e)}

        # 排除登录页面的初始 API 请求
        print("⏳ 排除初始请求...")
        page.wait_for_timeout(10000)

        # 展开侧边栏分组
        print("📂 展开侧边栏...")
        for g in SIDEBAR_GROUPS:
            try:
                page.locator(f'aside >> text="{g}"').first.click(timeout=500)
                page.wait_for_timeout(100)
            except:
                pass

        # 收集导航按钮
        nav_buttons = {}
        for btn in page.locator('aside button').all():
            try:
                t = btn.inner_text().strip().split("\n")[0].strip()
                if t and len(t) < 15 and t not in nav_buttons:
                    nav_buttons[t] = btn
            except:
                pass

        print(f"📋 找到 {len(nav_buttons)} 个导航按钮")

        # 测试每个页面
        for i, (view, label) in enumerate(VIEW_TO_LABEL.items()):
            btn = nav_buttons.get(label)
            if not btn:
                results[view] = {"status": "NO_NAV", "label": label}
                print(f"  [{i+1}/{len(VIEW_TO_LABEL)}] {view} ({label}) - ❌ 未找到导航")
                continue

            # 设置事件监听器
            page_errs, api_errs = [], []

            def on_pe(e):
                page_errs.append(str(e)[:400])

            def on_resp(r):
                try:
                    if "/api/" in r.url and r.status >= 400:
                        endpoint = r.url.split("?")[0].replace("http://localhost:8899", "")
                        # 过滤已知后端问题
                        if endpoint not in KNOWN_BROKEN_ENDPOINTS:
                            api_errs.append(f"{r.status} {endpoint}")
                except:
                    pass

            page.on("pageerror", on_pe)
            page.on("response", on_resp)

            try:
                btn.click()
                page.wait_for_timeout(2500)

                # 检查是否崩溃
                crash = False
                try:
                    crash = page.locator('text="页面出错了"').is_visible(timeout=300)
                except:
                    pass

                # 检查 JS 错误
                js_error = any("is not a function" in e or "Cannot read" in e for e in page_errs)

                # 统计页面内容
                body = page.inner_text("body").strip()
                h1 = page.locator("h1, h2, h3").count()
                tables = page.locator("table").count()
                charts = page.locator("canvas, svg").count()

                # 截图
                ss = SCREENSHOT_DIR / f"{i+1:02d}_{view}.png"
                page.screenshot(path=str(ss))

                # 确定状态
                if crash:
                    status = "CRASH"
                elif js_error:
                    status = "JS_ERROR"
                elif page_errs:
                    status = "PAGE_ERR"
                elif api_errs:
                    status = "API_ERR"
                else:
                    status = "OK"

                results[view] = {
                    "status": status,
                    "label": label,
                    "body_len": len(body),
                    "h1_count": h1,
                    "table_count": tables,
                    "chart_count": charts,
                    "api_errs": api_errs[:10],
                    "page_errs": page_errs[:5],
                    "screenshot": str(ss),
                }

                status_icon = "✅" if status == "OK" else "❌"
                print(f"  [{i+1}/{len(VIEW_TO_LABEL)}] {view} ({label}) - {status_icon} {status}")
                if api_errs:
                    print(f"    API 错误: {api_errs[:3]}")
                if page_errs:
                    print(f"    页面错误: {page_errs[:2]}")

            except Exception as e:
                results[view] = {"status": "ERROR", "label": label, "error": str(e)[:200]}
                print(f"  [{i+1}/{len(VIEW_TO_LABEL)}] {view} ({label}) - ❌ ERROR: {e}")
            finally:
                page.remove_listener("pageerror", on_pe)
                page.remove_listener("response", on_resp)

        browser.close()

    # 生成报告
    end_time = datetime.now()
    duration = (end_time - start_time).total_seconds()

    # 统计
    status_counts = {}
    failed_pages = []
    for view, r in results.items():
        status = r.get("status", "UNKNOWN")
        status_counts[status] = status_counts.get(status, 0) + 1
        if status != "OK":
            failed_pages.append({"view": view, "label": r.get("label"), "status": status, "errors": r.get("api_errs", []) + r.get("page_errs", [])})

    report = {
        "timestamp": start_time.isoformat(),
        "duration_seconds": round(duration, 1),
        "total_pages": len(results),
        "status_counts": status_counts,
        "pass_rate": f"{status_counts.get('OK', 0)}/{len(results)}",
        "failed_pages": failed_pages,
        "results": results,
    }

    # 保存报告
    report_path = Path("/tmp/e2e_report.json")
    with open(report_path, "w") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)

    # 输出摘要
    print("\n" + "=" * 60)
    print(f"📊 测试完成 - {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"⏱️  耗时: {duration:.1f} 秒")
    print(f"📄 总页面: {len(results)}")
    print(f"✅ 通过: {status_counts.get('OK', 0)}")
    print(f"❌ 失败: {len(failed_pages)}")
    print(f"📈 通过率: {status_counts.get('OK', 0)}/{len(results)}")
    print("=" * 60)

    if failed_pages:
        print("\n❌ 失败页面:")
        for fp in failed_pages:
            print(f"  - {fp['view']} ({fp['label']}): {fp['status']}")
            if fp['errors']:
                for err in fp['errors'][:3]:
                    print(f"    • {err}")

    return report


if __name__ == "__main__":
    report = test_all_pages()
    # 退出码: 0=全部通过, 1=有失败
    sys.exit(0 if report.get("status_counts", {}).get("OK", 0) == report.get("total_pages", 0) else 1)

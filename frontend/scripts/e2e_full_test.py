#!/usr/bin/env python3
"""AlphaPulse 全量 E2E 测试 — 覆盖 39 个页面

输出 JSON 报告到 stdout，包含每个页面的状态、错误、截图路径。

用法:
  python3 e2e_full_test.py              # 默认输出到 /tmp/e2e_report.json
  python3 e2e_full_test.py --output /path/to/report.json
"""

import json, time, sys, argparse
from pathlib import Path
from datetime import datetime
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
SCREENSHOT_DIR = Path("/tmp/e2e_screenshots")

# 完整页面映射 (view -> 中文标签)
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

ALL_VIEWS = list(VIEW_TO_LABEL.keys())

# 按导航栏分组（用于展开折叠）
NAV_GROUPS = ["核心", "分析", "选股", "交易", "工具"]


def expand_sidebar_groups(page):
    """展开所有侧边栏分组"""
    for g in NAV_GROUPS:
        try:
            btn = page.locator(f'aside >> text="{g}"').first
            if btn.is_visible(timeout=300):
                btn.click()
                page.wait_for_timeout(150)
        except:
            pass


def collect_nav_buttons(page) -> dict:
    """收集所有侧边栏导航按钮 {label: element}"""
    nav = {}
    for btn in page.locator("aside button").all():
        try:
            t = btn.inner_text().strip().split("\n")[0].strip()
            if t and len(t) < 20 and t not in nav:
                nav[t] = btn
        except:
            pass
    return nav


def login(page) -> bool:
    """登录，返回成功与否"""
    try:
        page.goto(BASE_URL, timeout=15000)
        page.wait_for_load_state("networkidle", timeout=10000)
        time.sleep(1)

        # 检查是否已登录（有侧边栏）
        if page.locator("aside").count() > 0:
            return True

        # 尝试登录
        inp = page.locator('input[type="text"]').first
        if inp.is_visible(timeout=3000):
            inp.fill("admin")
            page.locator('input[type="password"]').first.fill("admin123")
            page.locator('button[type="submit"]').first.click()
            page.wait_for_load_state("networkidle", timeout=10000)
            time.sleep(1)
        return True
    except Exception as e:
        print(f"LOGIN_FAILED: {e}", file=sys.stderr)
        return False


def test_single_page(page, nav_buttons, view, index) -> dict:
    """测试单个页面，返回结果 dict"""
    label = VIEW_TO_LABEL.get(view, view)
    btn = nav_buttons.get(label)

    if not btn:
        return {"status": "NO_NAV", "label": label, "error": f"导航按钮 '{label}' 未找到"}

    page_errs = []
    api_errs = []

    def on_page_error(e):
        page_errs.append(str(e)[:500])

    def on_response(r):
        try:
            url = r.url.split("?")[0]
            if "/api/" in url and r.status >= 400:
                api_errs.append(f"{r.status} {url}")
        except:
            pass

    page.on("pageerror", on_page_error)
    page.on("response", on_response)

    result = {"view": view, "label": label, "status": "UNKNOWN"}

    try:
        btn.click()
        page.wait_for_timeout(2500)

        # 检测崩溃页面
        crash = False
        try:
            crash = page.locator('text="页面出错了"').is_visible(timeout=300)
        except:
            pass
        try:
            if not crash:
                crash = page.locator('text="加载失败"').is_visible(timeout=300)
        except:
            pass

        # 检测 map is not a function 等常见错误
        body = page.inner_text("body").strip()
        has_map_error = "is not a function" in body or "Cannot read properties" in body

        # DOM 统计
        h1_count = page.locator("h1, h2, h3").count()
        table_count = page.locator("table").count()
        chart_count = page.locator("canvas, svg").count()

        # 截图
        ss_path = SCREENSHOT_DIR / f"{index+1:02d}_{view}.png"
        page.screenshot(path=str(ss_path), full_page=False)

        # 判断状态
        if crash:
            status = "CRASH"
        elif has_map_error:
            status = "JS_ERROR"
        elif page_errs:
            status = "PAGE_ERR"
        elif api_errs:
            status = "API_ERR"
        else:
            status = "OK"

        result.update({
            "status": status,
            "body_len": len(body),
            "h1_count": h1_count,
            "table_count": table_count,
            "chart_count": chart_count,
            "api_errors": api_errs[:10],
            "page_errors": page_errs[:5],
            "screenshot": str(ss_path),
        })

    except Exception as e:
        result = {
            "status": "ERROR",
            "error": str(e)[:500],
        }
    finally:
        page.remove_listener("pageerror", on_page_error)
        page.remove_listener("response", on_response)

    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", "-o", default="/tmp/e2e_report.json")
    args = parser.parse_args()

    SCREENSHOT_DIR.mkdir(parents=True, exist_ok=True)
    results = {}
    start_time = datetime.now()

    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1920, "height": 1080})

        if not login(page):
            print(json.dumps({"error": "LOGIN_FAILED"}))
            browser.close()
            sys.exit(1)

        # 等待 auth 稳定
        page.wait_for_timeout(2000)

        # 展开侧边栏
        expand_sidebar_groups(page)
        page.wait_for_timeout(500)

        # 收集导航按钮
        nav_buttons = collect_nav_buttons(page)

        # 测试每个页面
        for i, view in enumerate(ALL_VIEWS):
            results[view] = test_single_page(page, nav_buttons, view, i)

        browser.close()

    end_time = datetime.now()

    # 统计
    status_counts = {}
    for r in results.values():
        s = r.get("status", "UNKNOWN")
        status_counts[s] = status_counts.get(s, 0) + 1

    failed = {v: r for v, r in results.items() if r.get("status") not in ("OK", "NO_NAV")}

    report = {
        "timestamp": start_time.isoformat(),
        "duration_sec": (end_time - start_time).total_seconds(),
        "total": len(ALL_VIEWS),
        "status_counts": status_counts,
        "failed_pages": list(failed.keys()),
        "failed_details": failed,
        "results": results,
    }

    # 输出到文件
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)

    # 输出到 stdout
    print(json.dumps(report, ensure_ascii=False, indent=2))

    # 退出码
    if failed:
        sys.exit(1)
    sys.exit(0)


if __name__ == "__main__":
    main()

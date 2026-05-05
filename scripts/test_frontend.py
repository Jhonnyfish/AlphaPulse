#!/usr/bin/env python3
"""AlphaPulse Frontend Smoke Test — Playwright-based page health checker."""
import asyncio
import json
import sys
import os
from datetime import datetime

# Pages to test: sidebar_label -> expected_content_keyword
PAGES = {
    "策略": ["策略列表", "Alpha300", "添加策略"],
    "策略回测": ["开始回测", "股票代码"],
    "持仓": ["持仓", "添加持仓"],
    "交易日志": ["交易记录", "添加交易"],
    "组合风险": ["组合风险", "风险评级"],
    "自选股": ["自选股", "添加"],
    "设置": ["设置"],
    "每日简报": ["简报", "市场"],
    "异常检测": ["异常", "检测"],
    "板块": ["板块"],
    "对比": ["对比"],
    "资金流向": ["资金"],
}

async def run_test():
    from playwright.async_api import async_playwright
    
    results = []
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        page = await browser.new_page()
        
        # Login
        try:
            await page.goto("http://localhost:5173", timeout=15000)
            await page.wait_for_timeout(1500)
            await page.fill('input[type="text"]', "admin")
            await page.fill('input[type="password"]', "admin123")
            await page.click('button:has-text("登录")')
            await page.wait_for_timeout(2000)
        except Exception as e:
            print(json.dumps({"error": f"Login failed: {str(e)}", "status": "FAIL"}))
            await browser.close()
            return 1
        
        for name, keywords in PAGES.items():
            js_errors = []
            page.on("pageerror", lambda err: js_errors.append(str(err)))
            
            try:
                items = await page.query_selector_all('button, a')
                target = None
                for item in items:
                    if (await item.inner_text()).strip() == name:
                        target = item
                        break
                
                if not target:
                    results.append({"page": name, "status": "NOT_FOUND", "error": "Sidebar item not found"})
                    continue
                
                await target.click()
                await page.wait_for_timeout(2500)
                
                body = await page.inner_text("body")
                has_crash = "页面加载出错" in body
                real_errors = [e for e in js_errors if "WebSocket" not in str(e)]
                
                if has_crash:
                    results.append({"page": name, "status": "CRASH", "error": "Page error boundary triggered"})
                elif real_errors:
                    results.append({"page": name, "status": "JS_ERROR", "error": real_errors[0][:200]})
                else:
                    has_content = any(kw in body for kw in keywords)
                    if has_content:
                        results.append({"page": name, "status": "OK"})
                    else:
                        results.append({"page": name, "status": "EMPTY", "error": f"Expected keywords not found: {keywords}"})
            except Exception as e:
                results.append({"page": name, "status": "ERROR", "error": str(e)[:200]})
        
        await browser.close()
    
    # Output
    ok = sum(1 for r in results if r["status"] == "OK")
    total = len(results)
    failed = [r for r in results if r["status"] != "OK"]
    
    report = {
        "timestamp": datetime.now().isoformat(),
        "summary": f"{ok}/{total} pages OK",
        "ok": ok,
        "total": total,
        "failed": failed,
        "all_ok": ok == total,
    }
    
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if ok == total else 1

if __name__ == "__main__":
    sys.exit(asyncio.run(run_test()))

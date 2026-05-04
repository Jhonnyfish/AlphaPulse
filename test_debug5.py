from playwright.sync_api import sync_playwright
import time

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    ctx = browser.new_context(viewport={"width": 1280, "height": 800})
    page = ctx.new_page()

    api_calls = []
    page.on("response", lambda r: api_calls.append(f"{r.status} {r.url}") if "/api/" in r.url else None)

    page.goto("http://localhost:5173", timeout=10000)
    time.sleep(1)
    if page.locator("input[type='text']").count() > 0:
        page.fill("input[type='text']", "admin")
        page.fill("input[type='password']", "admin123")
        page.click("button[type='submit']")
        time.sleep(2)

    # Navigate to 候选股
    api_calls.clear()
    page.evaluate("""() => {
        const items = document.querySelectorAll('button, a, [role="button"]');
        for (const item of items) {
            if (item.textContent.includes('候选股') && item.offsetParent !== null) {
                item.click();
                return 'clicked';
            }
        }
        return 'not found';
    }""")
    time.sleep(5)

    print("=== API calls for 候选股 ===")
    for call in api_calls:
        if "candidates" in call or "screener" in call:
            print(f"  {call}")

    # Check page content
    body = page.inner_text("body")
    has_alpha300 = "Alpha300" in body
    has_loading = "加载" in body
    has_error = "失败" in body or "出错" in body
    print(f"\nHas Alpha300: {has_alpha300}")
    print(f"Has loading: {has_loading}")
    print(f"Has error: {has_error}")
    print(f"Body contains 候选: {'候选' in body}")

    # Navigate to 选股器
    api_calls.clear()
    page.evaluate("""() => {
        const items = document.querySelectorAll('button, a, [role="button"]');
        for (const item of items) {
            if (item.textContent.includes('选股器') && item.offsetParent !== null) {
                item.click();
                return 'clicked';
            }
        }
        return 'not found';
    }""")
    time.sleep(5)

    print("\n=== API calls for 选股器 ===")
    for call in api_calls:
        if "screener" in call:
            print(f"  {call}")

    body = page.inner_text("body")
    has_screener = "筛选" in body or "选股" in body
    print(f"\nHas screener content: {has_screener}")

    browser.close()

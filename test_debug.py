from playwright.sync_api import sync_playwright
import time

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    ctx = browser.new_context(viewport={"width": 1280, "height": 800})
    page = ctx.new_page()

    errors = []
    page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
    page.on("pageerror", lambda e: errors.append(f"PAGE_ERROR: {e}"))

    page.goto("http://localhost:5173", timeout=10000)
    time.sleep(1)
    if page.locator("input[type='text']").count() > 0:
        page.fill("input[type='text']", "admin")
        page.fill("input[type='password']", "admin123")
        page.click("button[type='submit']")
        time.sleep(2)

    for label in ["持仓", "交易日志", "候选池", "筛选器"]:
        errors.clear()
        page.evaluate("""(label) => {
            const items = document.querySelectorAll('button, a, [role="button"]');
            for (const item of items) {
                if (item.textContent.includes(label) && item.offsetParent !== null) {
                    item.click();
                    break;
                }
            }
        }""", label)
        time.sleep(3)

        err_boundary = page.locator("text=页面加载出错").count() > 0
        print(f"\n=== {label} ===")
        print(f"ErrorBoundary: {err_boundary}")
        if errors:
            for e in errors[:5]:
                print(f"Error: {e[:300]}")
        else:
            print("No console errors")

    browser.close()

from playwright.sync_api import sync_playwright
import time

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    ctx = browser.new_context(viewport={"width": 1280, "height": 800})
    page = ctx.new_page()

    # Capture network requests
    api_calls = []
    page.on("response", lambda r: api_calls.append(f"{r.status} {r.url}") if "/api/" in r.url else None)

    page.goto("http://localhost:5173", timeout=10000)
    time.sleep(1)
    if page.locator("input[type='text']").count() > 0:
        page.fill("input[type='text']", "admin")
        page.fill("input[type='password']", "admin123")
        page.click("button[type='submit']")
        time.sleep(2)

    # Navigate to CandidatesPage
    api_calls.clear()
    page.evaluate("""() => {
        const items = document.querySelectorAll('button, a, [role="button"]');
        for (const item of items) {
            if (item.textContent.includes('候选池') && item.offsetParent !== null) {
                item.click();
                break;
            }
        }
    }""")
    time.sleep(4)

    print("=== API calls for 候选池 ===")
    for call in api_calls:
        print(f"  {call}")

    # Check page content
    body = page.inner_text("body")
    print(f"\nBody text length: {len(body)}")
    print(f"Body preview: {body[:300]}")

    # Check if there's a loading state
    loading = page.locator("text=加载").count()
    error = page.locator("text=失败").count()
    empty = page.locator("text=暂无").count()
    print(f"\nLoading elements: {loading}")
    print(f"Error elements: {error}")
    print(f"Empty elements: {empty}")

    browser.close()

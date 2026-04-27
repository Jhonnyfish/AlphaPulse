from playwright.sync_api import sync_playwright
import time

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    ctx = browser.new_context(viewport={"width": 1280, "height": 800})
    page = ctx.new_page()

    page.goto("http://localhost:5173", timeout=10000)
    time.sleep(1)
    if page.locator("input[type='text']").count() > 0:
        page.fill("input[type='text']", "admin")
        page.fill("input[type='password']", "admin123")
        page.click("button[type='submit']")
        time.sleep(2)

    # Find all sidebar items
    items = page.evaluate("""() => {
        const items = document.querySelectorAll('button, a, [role="button"]');
        const result = [];
        for (const item of items) {
            const text = item.textContent.trim();
            if (text && text.length < 20 && item.offsetParent !== null) {
                result.push({ tag: item.tagName, text: text, className: item.className.substring(0, 50) });
            }
        }
        return result;
    }""")

    print("=== Sidebar items ===")
    for item in items:
        print(f"  {item['tag']}: '{item['text']}'")

    # Try clicking on 候选股 (not 候选池)
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
    time.sleep(3)

    # Check if we're on the candidates page
    body = page.inner_text("body")
    has_candidates = "Alpha300" in body or "候选" in body or "排行榜" in body
    print(f"\nOn candidates page: {has_candidates}")
    print(f"Body preview: {body[:200]}")

    browser.close()

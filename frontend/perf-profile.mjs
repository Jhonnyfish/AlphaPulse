import { chromium } from 'playwright';
import { writeFileSync } from 'fs';

const BASE_URL = 'http://localhost:5173';
const API_URL = 'http://localhost:8899';

const PAGES = [
  { path: '/', name: 'Dashboard' },
  { path: '/watchlist', name: 'Watchlist' },
  { path: '/market', name: 'Market' },
  { path: '/analyze', name: 'Analyze' },
  { path: '/hot-concepts', name: 'Hot Concepts' },
  { path: '/sectors', name: 'Sectors' },
  { path: '/signals', name: 'Signals' },
  { path: '/candidates', name: 'Candidates' },
  { path: '/screener', name: 'Screener' },
  { path: '/portfolio', name: 'Portfolio' },
  { path: '/trading-journal', name: 'Trading Journal' },
  { path: '/backtest', name: 'Backtest' },
  { path: '/flow', name: 'Flow' },
  { path: '/kline', name: 'Kline' },
  { path: '/compare', name: 'Compare' },
  { path: '/dragon-tiger', name: 'Dragon Tiger' },
  { path: '/watchlist-analysis', name: 'Watchlist Analysis' },
  { path: '/breadth', name: 'Breadth' },
  { path: '/trends', name: 'Trends' },
  { path: '/alerts', name: 'Alerts' },
  { path: '/news', name: 'News' },
  { path: '/trade-calendar', name: 'Trade Calendar' },
  { path: '/portfolio-risk', name: 'Portfolio Risk' },
  { path: '/investment-plans', name: 'Investment Plans' },
  { path: '/reports', name: 'Reports' },
  { path: '/settings', name: 'Settings' },
];

async function login() {
  const resp = await fetch(`${API_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: 'admin', password: 'admin123' }),
  });
  if (!resp.ok) {
    throw new Error(`Login failed: ${resp.status} ${await resp.text()}`);
  }
  const data = await resp.json();
  return data.token || data.access_token || '';
}

async function measurePage(context, pageDef, token) {
  const result = {
    name: pageDef.name,
    path: pageDef.path,
    url: `${BASE_URL}${pageDef.path}`,
    lcp: null,
    cls: null,
    fid: null,
    inp: null,
    ttfb: null,
    domContentLoaded: null,
    jsHeapSizeMB: null,
    networkRequests: 0,
    errors: [],
    status: 'ok',
    httpStatus: null,
  };

  // Create a fresh page for each test to isolate performance entries and request counts
  const page = await context.newPage();
  let requestCount = 0;
  const requestHandler = () => { requestCount++; };
  page.on('request', requestHandler);

  try {
    // Register PerformanceObserver for LCP BEFORE navigation using addInitScript
    await page.addInitScript(() => {
      // Buffer LCP entries globally
      window.__lcpEntries = [];
      window.__clsValue = 0;
      window.__longestFrame = 0;

      try {
        const lcpObserver = new PerformanceObserver((list) => {
          const entries = list.getEntries();
          for (const entry of entries) {
            window.__lcpEntries.push(entry.startTime);
          }
        });
        lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
      } catch (e) {}

      try {
        const clsObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (!entry.hadRecentInput) {
              window.__clsValue += entry.value;
            }
          }
        });
        clsObserver.observe({ type: 'layout-shift', buffered: true });
      } catch (e) {}

      try {
        const longFrameObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (entry.duration > window.__longestFrame) {
              window.__longestFrame = entry.duration;
            }
          }
        });
        longFrameObserver.observe({ type: 'long-animation-frame', buffered: true });
      } catch (e) {}

      // Inject token before React boots
      try {
        // We'll set this later via a second addInitScript
      } catch (e) {}
    });

    // Also inject the token before the app loads
    await page.addInitScript((t) => {
      try {
        localStorage.setItem('token', t);
        localStorage.setItem('user', JSON.stringify({ id: '1', username: 'admin', role: 'admin' }));
      } catch (e) {}
    }, token);

    // Navigate to the page
    const startTime = Date.now();
    const response = await page.goto(`${BASE_URL}${pageDef.path}`, {
      waitUntil: 'networkidle',
      timeout: 30000,
    });
    const loadTime = Date.now() - startTime;

    result.networkRequests = requestCount;

    // Wait for page to fully settle (LCP fires within ~2-3s after load)
    await page.waitForTimeout(3000);

    // Collect metrics from the page
    const metrics = await page.evaluate(() => {
      const perf = performance;

      // Navigation timing
      const navEntries = perf.getEntriesByType('navigation');
      const navEntry = navEntries.length > 0 ? navEntries[0] : null;

      const ttfb = navEntry ? Math.round(navEntry.responseStart - navEntry.requestStart) : null;
      const domContentLoaded = navEntry
        ? Math.round(navEntry.domContentLoadedEventEnd - navEntry.startTime)
        : null;

      // JS Heap
      let jsHeapSizeMB = null;
      if (perf.memory) {
        jsHeapSizeMB = Math.round((perf.memory.usedJSHeapSize / 1024 / 1024) * 100) / 100;
      }

      // LCP from our custom observer buffer
      let lcp = null;
      if (window.__lcpEntries && window.__lcpEntries.length > 0) {
        lcp = Math.round(window.__lcpEntries[window.__lcpEntries.length - 1]);
      }

      // Fallback: try native API
      if (lcp === null) {
        try {
          const lcpEntries = perf.getEntriesByType('largest-contentful-paint');
          if (lcpEntries && lcpEntries.length > 0) {
            lcp = Math.round(lcpEntries[lcpEntries.length - 1].startTime);
          }
        } catch (e) {}
      }

      // CLS
      let cls = Math.round((window.__clsValue || 0) * 10000) / 10000;
      if (cls === 0) {
        try {
          const layoutShifts = perf.getEntriesByType('layout-shift');
          if (layoutShifts) {
            for (const entry of layoutShifts) {
              if (!entry.hadRecentInput) {
                cls += entry.value;
              }
            }
            cls = Math.round(cls * 10000) / 10000;
          }
        } catch (e) {}
      }

      // INP proxy
      let longestFrame = Math.round(window.__longestFrame || 0);
      if (longestFrame === 0) {
        try {
          const longFrames = perf.getEntriesByType('long-animation-frame');
          if (longFrames && longFrames.length > 0) {
            longestFrame = Math.round(Math.max(...longFrames.map((f) => f.duration)));
          }
        } catch (e) {}
      }

      return { lcp, ttfb, domContentLoaded, jsHeapSizeMB, cls, longestFrame };
    });

    result.lcp = metrics.lcp;
    result.ttfb = metrics.ttfb;
    result.domContentLoaded = metrics.domContentLoaded;
    result.jsHeapSizeMB = metrics.jsHeapSizeMB;
    result.cls = metrics.cls;
    result.inp = metrics.longestFrame > 0 ? metrics.longestFrame : null;

    if (response) {
      result.httpStatus = response.status();
    }

    // If we still don't have LCP, use load timing as fallback estimate
    if (result.lcp === null) {
      // Try one more time with a fresh observer
      const lcpRetry = await page.evaluate(() => {
        return new Promise((resolve) => {
          let val = null;
          try {
            const obs = new PerformanceObserver((list) => {
              const entries = list.getEntries();
              if (entries.length > 0) {
                val = Math.round(entries[entries.length - 1].startTime);
              }
            });
            obs.observe({ type: 'largest-contentful-paint', buffered: true });
            setTimeout(() => {
              obs.disconnect();
              resolve(val);
            }, 500);
          } catch (e) {
            resolve(null);
          }
        });
      });
      if (lcpRetry !== null) {
        result.lcp = lcpRetry;
      }
    }

    // Last resort: use Paint Timing API for FCP as LCP proxy
    if (result.lcp === null) {
      const paintTimings = await page.evaluate(() => {
        const entries = performance.getEntriesByType('paint');
        return entries.map((e) => ({ name: e.name, startTime: Math.round(e.startTime) }));
      });
      const fcp = paintTimings.find((p) => p.name === 'first-contentful-paint');
      if (fcp) {
        result.lcp = fcp.startTime;
        result.lcpNote = 'FCP used as LCP proxy (no LCP entry recorded)';
      }
    }

  } catch (err) {
    result.status = 'error';
    result.errors.push(err.message);
  }

  page.off('request', requestHandler);
  await page.close();
  return result;
}

async function main() {
  console.log('=== AlphaPulse Frontend Performance Profiling ===');
  console.log(`    ${new Date().toISOString()}\n`);

  // Login
  let token;
  try {
    token = await login();
    if (!token) throw new Error('No token returned from login');
    console.log('✅ Login successful, token obtained.\n');
  } catch (err) {
    console.error('❌ Failed to login:', err.message);
    process.exit(1);
  }

  // Launch browser
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage'],
  });

  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
  });

  const results = [];

  for (let i = 0; i < PAGES.length; i++) {
    const p = PAGES[i];
    const progress = `[${String(i + 1).padStart(2)}/${PAGES.length}]`;
    process.stdout.write(`${progress} ${p.name.padEnd(23)} `);

    const result = await measurePage(context, p, token);
    results.push(result);

    const lcpStr = result.lcp !== null ? `${result.lcp}ms` : 'N/A';
    const heapStr = result.jsHeapSizeMB !== null ? `${result.jsHeapSizeMB}MB` : 'N/A';
    const status = result.status === 'error' ? '❌' : '✅';
    console.log(`${status} LCP:${String(lcpStr).padStart(8)} | Heap:${String(heapStr).padStart(10)} | Req:${String(result.networkRequests).padStart(3)}`);
  }

  await browser.close();

  // Generate report
  const report = {
    timestamp: new Date().toISOString(),
    baseUrl: BASE_URL,
    apiUrl: API_URL,
    browser: 'Chromium (Playwright headless)',
    viewport: '1280x720',
    methodology: 'Each page tested with a fresh browser page. LCP via PerformanceObserver (buffered). Token injected via addInitScript before app boot.',
    pages: results,
  };

  const reportPath = '/home/finn/alphapulse/frontend/performance-report.json';
  writeFileSync(reportPath, JSON.stringify(report, null, 2));
  console.log(`\n📄 Report saved to: ${reportPath}`);

  // Print summary table sorted by LCP (worst first)
  const sorted = [...results]
    .filter((r) => r.lcp !== null)
    .sort((a, b) => (b.lcp || 0) - (a.lcp || 0));

  const noLcp = results.filter((r) => r.lcp === null);

  console.log('\n' + '='.repeat(120));
  console.log('  PERFORMANCE SUMMARY — All Pages Ranked by LCP (worst first)');
  console.log('='.repeat(120));
  console.log('');
  console.log(
    ' Rank | Page                    |  LCP (ms) | TTFB (ms) | DCL (ms) | Heap (MB) | Reqs |  CLS   | Status'
  );
  console.log(
    '------|-------------------------|-----------|-----------|----------|-----------|------|--------|----------'
  );

  sorted.forEach((r, i) => {
    const lcp = String(r.lcp).padStart(9);
    const ttfb = r.ttfb !== null ? String(r.ttfb).padStart(9) : '      N/A';
    const dcl = r.domContentLoaded !== null ? String(r.domContentLoaded).padStart(8) : '     N/A';
    const heap = r.jsHeapSizeMB !== null ? String(r.jsHeapSizeMB).padStart(9) : '      N/A';
    const reqs = String(r.networkRequests).padStart(4);
    const clsVal = String(r.cls).padStart(6);
    const flag = r.lcp > 2500 ? '⚠️  SLOW' : r.lcp > 1500 ? '🟡 FAIR' : '✅ GOOD';
    const name = r.name.padEnd(23);
    const note = r.lcpNote ? ' *' : '';
    console.log(
      `  ${String(i + 1).padStart(2)}  | ${name} | ${lcp} | ${ttfb} | ${dcl} | ${heap} | ${reqs} | ${clsVal} | ${flag}${note}`
    );
  });

  if (noLcp.length > 0) {
    console.log(`\n  ⚠️  Pages where LCP could not be measured: ${noLcp.length}`);
    noLcp.forEach((r) => {
      console.log(`    - ${r.name} (${r.path}): ${r.errors.join(', ') || 'no LCP entries'}`);
    });
  }

  console.log('\n  * = FCP used as LCP proxy');

  // Summary stats
  const lcpValues = sorted.map((r) => r.lcp);
  if (lcpValues.length > 0) {
    const avg = Math.round(lcpValues.reduce((a, b) => a + b, 0) / lcpValues.length);
    const sorted2 = [...lcpValues].sort((a, b) => a - b);
    const median = sorted2[Math.floor(sorted2.length / 2)];
    const worst = sorted[0];
    const best = sorted[sorted.length - 1];
    const needsOpt = sorted.filter((r) => r.lcp > 2500);

    console.log('\n' + '='.repeat(120));
    console.log('  SUMMARY STATISTICS');
    console.log('='.repeat(120));
    console.log(`  Total pages tested:          ${results.length}`);
    console.log(`  Pages with LCP measured:     ${lcpValues.length}`);
    console.log(`  LCP Average:                 ${avg}ms`);
    console.log(`  LCP Median:                  ${median}ms`);
    console.log(`  LCP Best:                    ${best.lcp}ms  (${best.name})`);
    console.log(`  LCP Worst:                   ${worst.lcp}ms  (${worst.name})`);
    console.log(`  Pages LCP > 2500ms:          ${needsOpt.length} (NEED OPTIMIZATION)`);

    if (needsOpt.length > 0) {
      console.log(`\n  ⚠️  PAGES NEEDING OPTIMIZATION (LCP > 2500ms):`);
      needsOpt.forEach((r) => {
        console.log(`    ❌ ${r.name.padEnd(23)} ${r.lcp}ms  (${r.path})`);
      });
    }

    const fair = sorted.filter((r) => r.lcp > 1500 && r.lcp <= 2500);
    if (fair.length > 0) {
      console.log(`\n  🟡 PAGES WITH FAIR PERFORMANCE (LCP 1500-2500ms):`);
      fair.forEach((r) => {
        console.log(`    🟡 ${r.name.padEnd(23)} ${r.lcp}ms  (${r.path})`);
      });
    }

    const good = sorted.filter((r) => r.lcp <= 1500);
    console.log(`\n  ✅ Pages with good LCP (≤1500ms): ${good.length}`);

    const errorPages = results.filter((r) => r.status === 'error');
    if (errorPages.length > 0) {
      console.log(`\n  ❌ Pages with errors: ${errorPages.length}`);
      errorPages.forEach((r) => console.log(`    - ${r.name}: ${r.errors.join(', ')}`));
    }

    // TTFB stats
    const ttfbValues = results.map((r) => r.ttfb).filter((v) => v !== null);
    if (ttfbValues.length > 0) {
      const avgTtfb = Math.round(ttfbValues.reduce((a, b) => a + b, 0) / ttfbValues.length);
      console.log(`\n  TTFB Average:                ${avgTtfb}ms`);
    }

    // Heap stats
    const heapValues = results.map((r) => r.jsHeapSizeMB).filter((v) => v !== null);
    if (heapValues.length > 0) {
      const avgHeap = Math.round(heapValues.reduce((a, b) => a + b, 0) / heapValues.length * 100) / 100;
      const maxHeap = Math.max(...heapValues);
      console.log(`  Avg JS Heap:                 ${avgHeap}MB`);
      console.log(`  Max JS Heap:                 ${maxHeap}MB`);
    }

    // Request stats
    const reqValues = results.map((r) => r.networkRequests);
    if (reqValues.length > 0) {
      const avgReqs = Math.round(reqValues.reduce((a, b) => a + b, 0) / reqValues.length);
      console.log(`  Avg Network Requests:        ${avgReqs}`);
    }
  }
}

main().catch((err) => {
  console.error('Fatal error:', err);
  process.exit(1);
});

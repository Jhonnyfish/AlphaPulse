// AlphaPulse Service Worker v2 — segmented caching strategies
const STATIC_CACHE = 'alphapulse-static-v2';
const PAGES_CACHE = 'alphapulse-pages-v2';
const IMAGES_CACHE = 'alphapulse-images-v2';
const ALL_CACHES = [STATIC_CACHE, PAGES_CACHE, IMAGES_CACHE];
const PRECACHE = ['/', '/index.html'];
const LIMITS = { static: 60, pages: 20, images: 30 };

// Trim oldest entries when cache exceeds maxEntries
async function trimCache(name, max) {
  const c = await caches.open(name);
  const keys = await c.keys();
  if (keys.length > max) await Promise.all(keys.slice(0, keys.length - max).map((k) => c.delete(k)));
}

// Install: precache app shell
self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(STATIC_CACHE).then((c) => c.addAll(PRECACHE)));
  self.skipWaiting();
});

// Activate: claim clients, purge old caches, trim size limits
self.addEventListener('activate', (e) => {
  e.waitUntil(caches.keys().then((ks) =>
    Promise.all([...ks.filter((k) => !ALL_CACHES.includes(k)).map((k) => caches.delete(k)),
      trimCache(STATIC_CACHE, LIMITS.static), trimCache(PAGES_CACHE, LIMITS.pages), trimCache(IMAGES_CACHE, LIMITS.images)])
  ));
  self.clients.claim();
});

self.addEventListener('fetch', (e) => {
  if (e.request.method !== 'GET') return;
  const url = new URL(e.request.url);

  // API: network only
  if (url.pathname.startsWith('/api') || url.pathname.startsWith('/health')) return;

  // Static assets (hashed bundles): cache-first
  if (url.pathname.startsWith('/assets/') || /\.(?:js|css)$/.test(url.pathname)) {
    e.respondWith((async () => {
      const hit = await caches.match(e.request);
      if (hit) return hit;
      const resp = await fetch(e.request);
      const clone = resp.clone();
      caches.open(STATIC_CACHE).then((c) => c.put(e.request, clone));
      return resp;
    })());
    return;
  }

  // Navigation/HTML: stale-while-revalidate with offline fallback
  if (e.request.mode === 'navigate' || e.request.headers.get('Accept')?.includes('text/html')) {
    e.respondWith((async () => {
      const cached = await caches.match(e.request);
      const net = fetch(e.request).then((r) => {
        caches.open(PAGES_CACHE).then((c) => c.put(e.request, r.clone()));
        return r;
      }).catch(() => null);
      if (cached) return cached;
      return (await net) || caches.match('/index.html');
    })());
    return;
  }

  // Images: cache-first with size cap
  if (/\.(?:png|svg|ico|jpe?g|gif|webp)$/.test(url.pathname)) {
    e.respondWith((async () => {
      const hit = await caches.match(e.request);
      if (hit) return hit;
      const resp = await fetch(e.request);
      const clone = resp.clone();
      caches.open(IMAGES_CACHE).then((c) => c.put(e.request, clone));
      trimCache(IMAGES_CACHE, LIMITS.images);
      return resp;
    })());
    return;
  }

  // Other: network-first with cache fallback
  e.respondWith((async () => {
    try {
      const resp = await fetch(e.request);
      const clone = resp.clone();
      caches.open(STATIC_CACHE).then((c) => c.put(e.request, clone));
      return resp;
    } catch (_) {
      return caches.match(e.request);
    }
  })());
});

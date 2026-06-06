'use strict';

const { chromium } = require('playwright');
const express = require('express');
const os = require('os');
const path = require('path');

const PORT = parseInt(process.env.OGCODE_SEARCH_BRIDGE_PORT || '7331', 10);
const USE_REAL_PROFILE = process.env.OGCODE_SEARCH_USE_REAL_PROFILE === 'true';

// Max pages open at once. Capping concurrency prevents Chrome from thrashing
// under a burst of parallel fetch_page calls — fewer simultaneous tabs finish
// faster overall than many contending for the browser's main thread.
// 8 is a good balance: enough parallelism to keep network utilization high,
// but low enough to avoid Chrome Memory/JS-heap pressure.
const MAX_CONCURRENCY = parseInt(process.env.OGCODE_SEARCH_MAX_CONCURRENCY || '15', 10);

// Real Chrome profile — user's actual Chrome data directory (cookies, logins).
// Chrome must be fully closed before ogcode starts when this is enabled.
const REAL_CHROME_PROFILE = process.env.OGCODE_SEARCH_CHROME_PROFILE ||
  path.join(os.homedir(), 'Library', 'Application Support', 'Google', 'Chrome');

// Isolated profile — safe default, no login state shared.
const ISOLATED_PROFILE = path.join(os.homedir(), '.local', 'share', 'ogcode', 'search-profile');

const userDataDir = USE_REAL_PROFILE ? REAL_CHROME_PROFILE : ISOLATED_PROFILE;
console.log(`search bridge: profile=${USE_REAL_PROFILE ? 'real Chrome' : 'isolated'} (${userDataDir})`);

const app = express();
app.use(express.json());

// ── Browser singleton (race-safe) ───────────────────────────────────────────
// Concurrent first-callers must not each launch a browser. We memoise the
// launch promise itself, so every caller awaits the same in-flight launch.
let browserPromise = null;

function getBrowser() {
  if (!browserPromise) {
    browserPromise = chromium.launchPersistentContext(userDataDir, {
      headless: true,
      args: [
        '--no-sandbox',
        '--disable-blink-features=AutomationControlled',
        '--disable-dev-shm-usage',
        // New headless mode (Chrome 112+) is far harder for bot-detection
        // scripts to fingerprint than the legacy headless mode.
        '--headless=new',
      ],
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36',
    }).then(async (ctx) => {
      // Block images, media, fonts, and stylesheets for every page in this
      // context. We only extract text, so these bytes are pure waste — skipping
      // them is the single biggest latency win (often 2–5× faster page loads).
      // Scripts and XHR/fetch are kept so JS-rendered content still appears.
      await ctx.route('**/*', (route) => {
        const type = route.request().resourceType();
        if (type === 'image' || type === 'media' || type === 'font' || type === 'stylesheet') {
          return route.abort();
        }
        return route.continue();
      });
      return ctx;
    });
  }
  return browserPromise;
}

// Anti-detection: applied to every new page to make the headless browser
// harder to fingerprint. Overrides navigator.webdriver (the most common
// headless check) and removes Chrome automation property markers.
const stealthInitScript = () => {
  Object.defineProperty(navigator, 'webdriver', { get: () => false });
};

// ── Concurrency limiter ──────────────────────────────────────────────────────
// JavaScript is single-threaded at the event-loop level: no two callbacks run
// at the same time. acquire/release mutate `active` and `waiters` safely
// without a mutex — a concurrent reader in another language would need one.
let active = 0;
const waiters = [];

function acquire() {
  if (active < MAX_CONCURRENCY) {
    active++;
    return Promise.resolve();
  }
  return new Promise((resolve) => waiters.push(resolve));
}

function release() {
  active--;
  const next = waiters.shift();
  if (next) {
    active++;
    next();
  }
}

// Run fn with a fresh page, respecting the concurrency cap.
// Creates a new page per request and closes it when done. Page creation/closure
// takes ~50-100ms but avoids state leakage between requests.
async function withPage(fn) {
  await acquire();
  const ctx = await getBrowser();
  const page = await ctx.newPage();
  try {
    // Apply anti-detection script before any navigation.
    await page.addInitScript(stealthInitScript);
    return await fn(page);
  } finally {
    await page.close().catch(() => {});
    release();
  }
}

function cleanText(raw) {
  return raw
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 14000);
}

// ── Result cache ─────────────────────────────────────────────────────────────
// Avoids re-loading Google or re-fetching a page that was already visited
// within the same session. TTL of 10 minutes: search results and page content
// rarely change minute-to-minute, and the latency savings from cache hits
// (microseconds vs seconds) are dramatic during a multi-query search session.
const CACHE_TTL = 10 * 60 * 1000;
const searchCache = new Map();
const fetchCache = new Map();

function cacheGet(map, key) {
  const entry = map.get(key);
  if (!entry || Date.now() > entry.expiresAt) { map.delete(key); return null; }
  return entry.value;
}

function cacheSet(map, key, value) {
  map.set(key, { value, expiresAt: Date.now() + CACHE_TTL });
}

// POST /search  { query, limit? }
app.post('/search', async (req, res) => {
  const { query, limit = 8 } = req.body || {};
  if (!query) return res.status(400).json({ error: 'query is required' });

  try {
    const cacheKey = `${query}::${limit}`;
    const cached = cacheGet(searchCache, cacheKey);
    if (cached) return res.json({ results: cached });

    // Primary: Google Search
    let results = await searchGoogle(query, limit);

    // Fallback: Bing Search when Google returns 0 results (typically means
    // CAPTCHA or rate-limit — Google serves a bot-check page with no <h3>
    // elements). Bing is less aggressive about headless-browser detection
    // and returns organic results reliably even for automated requests.
    if (results.length === 0) {
      console.log('search bridge: Google returned 0 results, falling back to Bing');
      results = await searchBing(query, limit);
    }

    cacheSet(searchCache, cacheKey, results);
    res.json({ results });
  } catch (err) {
    console.error('search error:', err.message);
    res.status(500).json({ error: err.message });
  }
});

// searchGoogle scrapes Google Search results using the headless browser.
async function searchGoogle(query, limit) {
  return withPage(async (page) => {
    const url = `https://www.google.com/search?q=${encodeURIComponent(query)}&hl=en&num=${limit}`;
    // 10s timeout is generous for Google which typically loads in 1-3s.
    // Slow loads usually indicate Google serving a CAPTCHA or throttling.
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 10000 });

    // Wait for the results container instead of a fixed sleep. Falls through
    // quickly if results are already present; bails after 3s if Google stalls.
    await page.waitForSelector('#search h3, #rso h3', { timeout: 3000 }).catch(() => {});

    return page.evaluate((maxResults) => {
      // Google rotates result-block class names (.g is long gone), so we anchor
      // on the stable bits: every organic result has an <h3> title inside a
      // link. Walk from each h3 up to its result container, pull the URL and a
      // nearby snippet. Dedup by URL and skip Google's own links.
      const root = document.querySelector('#rso') || document.querySelector('#search') || document.body;
      const h3s = Array.from(root.querySelectorAll('h3'));
      const seen = new Set();
      const results = [];

      for (const h3 of h3s) {
        // Find the result's anchor: the h3 is usually wrapped in <a>, else the
        // closest ancestor containing an http link.
        let anchor = h3.closest('a[href]');
        let container = h3;
        for (let i = 0; i < 6 && container; i++) {
          if (!anchor) anchor = container.querySelector('a[href^="http"]');
          // A result block is the first ancestor with both the title and a snippet.
          if (container.querySelector('h3') && container.innerText.length > h3.innerText.length + 40) break;
          container = container.parentElement;
        }
        const url = anchor ? anchor.href : '';
        if (!url || !url.startsWith('http')) continue;
        if (url.includes('google.com') || url.includes('/search?')) continue;
        if (seen.has(url)) continue;
        seen.add(url);

        // Snippet: longest text node in the container that isn't the title.
        let snippet = '';
        if (container) {
          const title = h3.innerText.trim();
          const full = container.innerText.replace(/\s+/g, ' ').trim();
          snippet = full.replace(title, '').trim().slice(0, 300);
        }

        results.push({ title: h3.innerText.trim(), url, snippet });
        if (results.length >= maxResults) break;
      }
      return results;
    }, limit);
  });
}

// searchBing scrapes Bing Search results as a fallback when Google returns
// empty (typically CAPTCHA or rate-limit). Bing is far less aggressive about
// headless-browser detection and reliably returns organic results.
async function searchBing(query, limit) {
  return withPage(async (page) => {
    const url = `https://www.bing.com/search?q=${encodeURIComponent(query)}&count=${limit}`;
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 10000 }).catch(() => {});
    // Bing renders quickly; 2s is enough for JS-rendered results to appear.
    await page.waitForTimeout(2000).catch(() => {});

    return page.evaluate((maxResults) => {
      const results = [];
      const seen = new Set();
      // Bing organic results live in <li class="b_algo"> elements.
      const items = document.querySelectorAll('li.b_algo');

      for (const item of items) {
        const anchor = item.querySelector('h2 a');
        if (!anchor) continue;

        // Extract the actual URL. Bing wraps some links through a redirect
        // (bing.com/ck/a?...&u=a1BASE64). The real URL is base64-encoded
        // in the 'u' parameter after the 'a1' prefix.
        let href = anchor.getAttribute('href') || '';
        if (href.includes('bing.com/ck/')) {
          try {
            const match = href.match(/u=a1([a-zA-Z0-9+/=_-]+)/);
            if (match) href = atob(match[1]);
          } catch (_) {}
        }
        // Second attempt: Bing sometimes uses <cite> to display the
        // readable URL even when the href is a redirect.
        if (href.includes('bing.com')) {
          const cite = item.querySelector('cite');
          if (cite) {
            const citeText = cite.textContent.trim();
            if (citeText && citeText.includes('.')) {
              href = citeText.startsWith('http') ? citeText : 'https://' + citeText;
            }
          }
        }
        if (!href || !href.startsWith('http')) continue;
        if (href.includes('bing.com')) continue;
        if (seen.has(href)) continue;
        seen.add(href);

        const title = anchor.textContent.trim().replace(/\s+/g, ' ');
        const snippetEl = item.querySelector('.b_caption p, p');
        const snippet = snippetEl ? snippetEl.textContent.trim().slice(0, 300) : '';

        results.push({ title, url: href, snippet });
        if (results.length >= maxResults) break;
      }
      return results;
    }, limit);
  });
}

// POST /fetch  { url }
app.post('/fetch', async (req, res) => {
  const { url } = req.body || {};
  if (!url) return res.status(400).json({ error: 'url is required' });

  try {
    const cached = cacheGet(fetchCache, url);
    if (cached) return res.json(cached);

    const result = await withPage(async (page) => {
      // Reduced from 12s to 8s: research pages that take >8s to reach
      // domcontentloaded are usually bloated with ads/trackers and return
      // poor text anyway. The 8s cap prevents one slow site from blocking
      // the entire parallel fetch batch.
      await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 8000 });

      // Give dynamic content a short window to render (SPAs, React, etc.)
      // 300ms catches most JS-rendered content without adding significant latency.
      await page.waitForTimeout(300).catch(() => {});

      return page.evaluate(() => {
        document.querySelectorAll('nav, footer, header, script, style, aside, iframe, .ads, [role="banner"], [role="navigation"]')
          .forEach(el => el.remove());

        const title = document.title || '';
        // Prefer focused content containers over the entire body — less noise,
        // less likely to hit the 14K truncation cap on content-heavy pages.
        const el = document.querySelector('main, article, [role="main"], #main, .main-content')
                   || document.body;
        const raw = el ? el.innerText : '';
        return { title, raw };
      });
    });

    const text = cleanText(result.raw);
    const payload = { url, title: result.title, text, truncated: result.raw.length > 14000 };
    cacheSet(fetchCache, url, payload);
    res.json(payload);
  } catch (err) {
    console.error('fetch error:', err.message);
    res.status(500).json({ error: err.message });
  }
});

// GET /health
app.get('/health', (_, res) => res.json({ ok: true }));

async function main() {
  // Pre-warm browser
  try {
    await getBrowser();
    console.log('search bridge: browser ready');
  } catch (err) {
    console.error('search bridge: browser init failed:', err.message);
    process.exit(1);
  }

  app.listen(PORT, '127.0.0.1', () => {
    console.log(`search bridge: listening on http://127.0.0.1:${PORT} (max concurrency ${MAX_CONCURRENCY})`);
  });
}

async function shutdown() {
  if (browserPromise) {
    const ctx = await browserPromise.catch(() => null);
    if (ctx) await ctx.close().catch(() => {});
  }
  process.exit(0);
}
process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

main();

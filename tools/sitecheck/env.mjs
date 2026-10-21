// Shared run state of the site checklist (ticket 102): the verdict sink, the browser and the page set that every
// section in sections/ works on. run.mjs owns the command line; a section reads from this object only what it needs.
import { LIVE_BASE, launch, serveTree, helperPage } from './lib.mjs';

export async function createEnv({ MODE, OUT, want }) {
  const R = [];
  const add = (box, group, status, note, data) => { R.push({ box, group, status, note, data }); console.log(`[${status}] ${box} ${group}: ${note}`); };

  let BASE, server;
  const browser = await launch();
  if (MODE === 'tree') { const s = await serveTree(); BASE = s.base; server = s.server; } else BASE = LIVE_BASE;
  const helper = await helperPage(browser);

  // ---------- the page set ----------
  const sm = await (await browser.newContext()).request.get(BASE + 'sitemap.xml');
  const smText = await sm.text();
  const smLocs = [...smText.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
  const toFile = (u) => { const p = new URL(u).pathname.replace(/^\/doc-html-translate\/?/, ''); return !p ? 'index.html' : p.endsWith('/') ? p + 'index.html' : p; };
  const FILES = [...new Set(smLocs.map(toFile))];
  const url = (f) => BASE + (f === 'index.html' ? '' : f.replace(/index\.html$/, ''));
  const LOCALES = FILES.filter((f) => /^[a-z]{2}\/index\.html$/.test(f));
  const ROOT3 = ['index.html', 'extension.html', 'privacy.html', 'install-trust.html', 'extension-privacy.html', 'support.html'];
  const DOCS = ['docs.html', 'docs.ru.html', 'docs.uk.html'];
  console.log(`mode=${MODE} base=${BASE} pages=${FILES.length}`);

  const newCtx = (o = {}) => browser.newContext({ viewport: { width: 1280, height: 900 }, ...o });
  async function open(ctx, u, extra) {
    const page = await ctx.newPage();
    const reqs = [];
    page.on('request', (r) => reqs.push(r.url()));
    const resp = await page.goto(u, { waitUntil: 'load' });
    await page.evaluate(() => document.fonts && document.fonts.ready);
    page.__reqs = reqs; page.__resp = resp;
    return page;
  }
  const themeCtx = (theme, o = {}) => newCtx({ colorScheme: theme, ...o }).then(async (c) => { await c.addInitScript((t) => { try { localStorage.setItem('sza-theme', t); } catch {} }, theme); return c; });

  const readRaw = async (ctx, f) => (await ctx.request.get(url(f))).text();

  return { MODE, OUT, want, R, add, BASE, server, browser, helper, smLocs, toFile, FILES, url, LOCALES, ROOT3, DOCS, newCtx, open, themeCtx, readRaw, crawlLinks: [] };
}

// Shared helpers of the UI-contract checks (ticket 119): the real doc-html-ui.exe under an isolated profile, a
// headless Chromium, geometry and colour readers that look at what the engine actually laid out, and a verdict
// that speaks CHECK-VERDICT. Nothing here reads or writes the developer's own settings.
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

export const REPO = path.resolve(import.meta.dirname, '..', '..');
const helpgen = await import(pathToFileURL(path.join(REPO, 'tools', 'helpgen', 'app.mjs')).href);
export const { loadPlaywright, launchBrowser, demoProfile, startApp } = helpgen;

// ---- profiles of the matrix --------------------------------------------------------------------------
// The declared scale range of UI-SIZING (docs: ticket 119 INDEX, decision D5) and the declared smallest targets.
export const SCALE = { compact: 0.875, ordinary: 1, large: 1.25, xlarge: 1.5, huge: 2 };
export const TEXT = { ordinary: 1, large: 1.25, xlarge: 1.5, huge: 2 };
export const TARGET = { pointer: 32, touch: 48 }; // ordinary baselines; compact scales them (28 / 42)

// ---- the report ----------------------------------------------------------------------------------------
export class Report {
  constructor(name) { this.name = name; this.rows = []; this.notes = []; }
  check(id, ok, detail = '') { this.rows.push({ id, ok: !!ok, detail }); return !!ok; }
  advisory(id, detail) { this.rows.push({ id, ok: true, advisory: true, detail }); }
  note(text) { this.notes.push(text); }
  get failed() { return this.rows.filter((r) => !r.ok); }
  get advisories() { return this.rows.filter((r) => r.advisory); }
  verdict() {
    if (this.failed.length) return { code: 1, line: `${this.name}: FAIL (${this.failed.length} of ${this.rows.length} checks)` };
    if (this.advisories.length) return { code: 3, line: `${this.name}: PASS WITH ADVISORIES (${this.rows.length} checks, ${this.advisories.length} advisories)` };
    return { code: 0, line: `${this.name}: PASS (${this.rows.length} checks)` };
  }
}

// ---- an app session ------------------------------------------------------------------------------------
export async function openSession({ exe, seed = {}, root }) {
  const seeded = { theme: 'light', lang: 'en', ...seed };
  const profile = demoProfile(root, seeded);
  const settingsFile = path.join(profile, 'doc-html-translate', 'ui-settings.json');
  const server = await startApp(exe, profile);
  const { chromium } = await loadPlaywright();
  const browser = await launchBrowser(chromium);
  return {
    server, browser,
    async page({ q = {}, w = 1366, h = 768, dsf = 1, scheme = 'light', forced = 'none', init, keepState = false } = {}) {
      // The window remembers its page and its open groups and flushes them when a page closes (APP-BEHAVIOUR
      // rule 10), so a page opened after another would start from that page's state. Every check here assumes
      // the seed, so each page starts from it unless it asks to keep the state; the server reads the file per request.
      if (!keepState) fs.writeFileSync(settingsFile, JSON.stringify(seeded));
      const ctx = await browser.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: dsf, colorScheme: scheme, forcedColors: forced });
      if (init) await ctx.addInitScript(init);
      const page = await ctx.newPage();
      page.consoleErrors = [];
      page.on('console', (m) => { if (m.type() === 'error') page.consoleErrors.push(m.text()); });
      page.on('pageerror', (e) => page.consoleErrors.push('PAGEERROR ' + e.message));
      page.requests = [];
      page.on('request', (r) => { const u = new URL(r.url()); if (u.pathname.startsWith('/api/')) page.requests.push(u.pathname + ' ' + r.method()); });
      await page.goto(server.base + '/?' + new URLSearchParams(q).toString());
      await page.waitForFunction(() => !document.documentElement.hasAttribute('data-boot'), null, { timeout: 8000 });
      // A short wait after closing lets the pagehide flush of this page reach the server before the next page resets the file.
      return { page, ctx, async close() { await ctx.close(); await new Promise((r) => setTimeout(r, 250)); } };
    },
    async close() { await browser.close(); server.stop(); },
  };
}

// ---- readers (run inside the page) ------------------------------------------------------------------------
// effective colour of a property over the composited background of the element's ancestors
export const readerSource = `
window.__uic = {
  rgba(s) { const m = s.match(/rgba?\\(([^)]+)\\)/); if (!m) return null; const p = m[1].split(/[ ,\\/]+/).filter(Boolean).map(Number); return [p[0], p[1], p[2], p[3] === undefined ? 1 : p[3]]; },
  over(top, bottom) { const a = top[3]; return [top[0] * a + bottom[0] * (1 - a), top[1] * a + bottom[1] * (1 - a), top[2] * a + bottom[2] * (1 - a), 1]; },
  bgOf(el) {
    const stack = [];
    for (let e = el; e; e = e.parentElement) { const c = this.rgba(getComputedStyle(e).backgroundColor); if (c && c[3] > 0) { stack.push(c); if (c[3] >= 1) break; } }
    let base = [255, 255, 255, 1];
    for (let i = stack.length - 1; i >= 0; i--) base = this.over(stack[i], base);
    return base;
  },
  fgOf(el, prop) { const c = this.rgba(getComputedStyle(el)[prop || 'color']); return c; },
  lum(c) { const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); }; return 0.2126 * f(c[0]) + 0.7152 * f(c[1]) + 0.0722 * f(c[2]); },
  contrast(a, b) { const l1 = this.lum(a), l2 = this.lum(b); return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05); },
  // contrast of an element's text ink against the face it is drawn on
  textContrast(el) { const bg = this.bgOf(el); const fg = this.over(this.fgOf(el), bg); return this.contrast(fg, bg); },
  rect(el) { const r = el.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height }; },
  // checkVisibility also answers false inside a collapsed <details> (its body is content-visibility: hidden), which is what a reader sees
  visible(el) { return el.checkVisibility({ checkVisibilityCSS: true, contentVisibilityAuto: true }) && getComputedStyle(el).visibility !== 'hidden'; },
};`;

export function nearestRankP95(samples) {
  const s = [...samples].sort((a, b) => a - b);
  return s[Math.ceil(0.95 * s.length) - 1];
}
export function writeJSON(file, data) { fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, JSON.stringify(data, null, 2)); }

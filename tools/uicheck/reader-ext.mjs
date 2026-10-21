// The extension viewer's toolbar, search panel and contents panel against the same rules as the generated reader
// (ticket 119, slice R1; reader.mjs and reader-rules.mjs hold them). The viewer only exists under a browser extension's
// own origin, where chrome.i18n and chrome.storage live, so the real extension/ folder is served over http on a loopback
// port and the page gets a stub of the few chrome.* calls viewer.js makes at load (storage in memory, the language from
// the case, messages read from the real _locales files). The viewer's own scripts run unmodified and render a real
// Markdown document, so what is measured is the toolbar the engine laid out with its real, translated, state-bearing
// controls - not a copy of its markup.
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { REPO, readerSource, loadPlaywright, launchBrowser } from './lib.mjs';
import { WORK, MODES, caseTag, geometry, capture, contrast, selectedState, keyboard, panel } from './reader.mjs';
import { writePicturesHtml, missingImageExt } from './reader-missing.mjs';

export const EXT = {
  name: 'ext', root: '#toolbar', bar: '#toolbar', controls: 'button, a, select, input', items: '#toolbar .title, #toolbar .group > span', panels: '#search-panel',
  scaleVar: '--ui', version: null, progress: null,
  search: { button: '#btn-search', panel: '#search-panel', input: '#search-input', close: '#search-close', row: 'button, select, input', items: 'label' },
  contents: { button: '#btn-toc', panel: '#toc', row: '.toc-toggle', items: ':not(*)' },
  toggles: ['#btn-night', '#btn-ocr'],
};

const MIME = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.mjs': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.wasm': 'application/wasm', '.png': 'image/png', '.svg': 'image/svg+xml', '.md': 'text/markdown; charset=utf-8', '.txt': 'text/plain; charset=utf-8' };

// A nested contents tree (parts holding chapters) so the TOC panel has real disclosure buttons to measure; the Markdown
// parser makes a flat list.
function writeBookFb2() {
  const para = (c, n) => `<p>Paragraph ${n} of chapter ${c}: the quick brown fox jumps over the lazy dog while the reader scrolls through a sufficiently long page of text.</p>`;
  const chapter = (p, c) => `<section><title><p>Chapter ${p}.${c}</p></title>${[1, 2, 3, 4, 5, 6].map((n) => para(c, n)).join('')}</section>`;
  const part = (p) => `<section><title><p>Part ${p}</p></title>${[1, 2, 3].map((c) => chapter(p, c)).join('')}</section>`;
  const xml = `<?xml version="1.0" encoding="UTF-8"?><FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"><description><title-info><book-title>The Long Walk</book-title><lang>en</lang></title-info></description><body>${[1, 2, 3].map(part).join('')}</body></FictionBook>`;
  const file = path.join(WORK, 'src', 'book.fb2');
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, xml);
  return file;
}

function serve() {
  const extDir = path.join(REPO, 'extension');
  const book = writeBookFb2();
  const pictures = writePicturesHtml();
  const fixtures = { '/__fixtures__/book.fb2': book, '/__fixtures__/pictures.html': pictures };
  const server = http.createServer((req, res) => {
    const u = decodeURIComponent(new URL(req.url, 'http://x').pathname);
    const file = fixtures[u] || path.join(extDir, u);
    if (!(file === book || file === pictures || file.startsWith(extDir)) || !fs.existsSync(file) || !fs.statSync(file).isFile()) { res.writeHead(404); res.end(); return; }
    res.writeHead(200, { 'content-type': MIME[path.extname(file)] || 'application/octet-stream', 'access-control-allow-origin': '*' });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve({ server, base: `http://127.0.0.1:${server.address().port}` })));
}

// the few chrome.* calls viewer.js and its modules make: an in-memory storage, the language of the case, the real
// message files (i18n.js fetches _locales/<code>/messages.json through getURL)
const STUB = ([lang, prefs]) => {
  const stored = { uiLang: lang, viewerPrefs: prefs };
  const pick = (keys) => {
    if (keys == null) return { ...stored };
    const list = Array.isArray(keys) ? keys : typeof keys === 'string' ? [keys] : Object.keys(keys);
    const out = {};
    for (const k of list) if (k in stored) out[k] = stored[k];
    return out;
  };
  const noop = { addListener() {}, removeListener() {} };
  window.chrome = {
    runtime: { getURL: (p) => location.origin + '/' + String(p).replace(/^\//, ''), sendMessage: async () => ({}), onMessage: noop, lastError: undefined, id: 'uicheck' },
    storage: { local: { get: async (k) => pick(k), set: async (o) => { Object.assign(stored, o); }, remove: async () => {} }, onChanged: noop },
    i18n: { getMessage: () => '', getUILanguage: () => lang },
    tabs: { getCurrent: async () => ({ id: 1 }) },
  };
};

export async function extensionChecks(report, { shotsDir = null } = {}) {
  const { server, base } = await serve();
  const { chromium } = await loadPlaywright();
  const browser = await launchBrowser(chromium);
  const cases = [];
  cases.push({ mode: 'fine', deep: true }, { mode: 'coarse', deep: true }, { mode: 'narrow', deep: true }, { mode: 'fine', deep: true, reveal: true }, { mode: 'coarse', reveal: true }, { mode: 'narrow', reveal: true });
  for (const theme of ['sepia', 'dark', 'night']) cases.push({ mode: 'fine', theme, deep: true });
  cases.push({ mode: 'fine', forced: true, deep: true }, { mode: 'fine', theme: 'dark', forced: true, deep: true }, { mode: 'fine', contrastMore: true });
  for (const mode of ['fine', 'narrow']) cases.push({ mode, lang: 'ar', deep: mode === 'fine' });
  for (const lang of ['de', 'ru', 'hi']) for (const mode of ['fine', 'narrow', 'coarse']) cases.push({ mode, lang });
  for (const scale of [0.875, 2, 0, 100]) cases.push({ mode: 'fine', scale });
  cases.push({ mode: 'coarse', scale: 0.875 }, { mode: 'coarse', scale: 2 });
  try {
    for (const c of cases) {
      c.page = 'viewer';
      const dark = c.theme === 'dark' || c.theme === 'night';
      const ctx = await browser.newContext({ ...MODES[c.mode], colorScheme: dark ? 'dark' : 'light', forcedColors: c.forced ? 'active' : 'none', ...(c.contrastMore ? { contrast: 'more' } : {}) });
      await ctx.addInitScript(STUB, [c.lang || 'en', c.theme ? { theme: c.theme } : {}]);
      const page = await ctx.newPage();
      page.errors = [];
      page.on('pageerror', (e) => page.errors.push(e.message));
      try {
        await page.goto(`${base}/src/viewer.html?file=${encodeURIComponent(base + '/__fixtures__/book.fb2')}`);
        await page.waitForSelector('#content p', { timeout: 20000 });
        // the OCR layer toggle belongs to documents with plates; its group is revealed here to measure the button too
        if (c.reveal) await page.evaluate(() => { document.getElementById('grp-ocr').hidden = false; });
        await page.evaluate(readerSource);
        if (c.scale !== undefined) await page.evaluate(([v, s]) => document.documentElement.style.setProperty(v, String(s)), [EXT.scaleVar, c.scale]);
        await page.waitForTimeout(250);
        const m = await geometry(report, page, EXT, c, c.lang === 'ar');
        if (!c.deep) continue;
        await capture(page, c, EXT);
        await contrast(report, page, EXT, c);
        for (const t of EXT.toggles) if (m.controls.some((x) => x.key === t)) await selectedState(report, page, EXT, c, t, 'aria-pressed');
        await keyboard(report, page, EXT, c, m);
        await panel(report, page, EXT, c, 'search', true);
        await panel(report, page, EXT, c, 'contents', true);
        if (c.mode === 'fine' && !c.theme && !c.lang && !c.forced && !c.contrastMore && c.scale === undefined) {
          const tip = await page.evaluate(() => document.getElementById('btn-search').getAttribute('title'));
          report.check('reader.search-tooltip-names-chord/ext', /Ctrl\+F|Cmd\+F/.test(tip || ''), `title ${tip}`);
          const ext = await page.evaluate(() => [...document.querySelectorAll('#toolbar a.proj-link')].map((a) => ({ marker: !!a.querySelector('svg.glyph'), blank: a.target })));
          report.check('reader.external-marker/ext', ext.length === 2 && ext.every((e) => e.marker), `${ext.length} outside links, markers ${ext.map((e) => e.marker).join(',')}`);
          // the controls that exist only for some documents are measured where they exist
          const hidden = await page.evaluate(() => ['grp-ocr', 'btn-original', 'btn-save-src', 'btn-save-html'].filter((id) => { const e = document.getElementById(id); return e && e.getBoundingClientRect().width === 0; }));
          if (!c.reveal) report.advisory('reader.ext-controls-not-on-this-document', `not shown for an FB2 document, so not measured in the base case: ${hidden.join(', ')}; the OCR group is measured in the revealed case, Original PDF needs a PDF opened from a URL`);
        }
      } finally { await ctx.close(); }
    }
    await missingImageExt(report, browser, base, STUB, shotsDir);
  } finally { await browser.close(); server.close(); }
}

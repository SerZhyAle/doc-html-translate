// The browser extension's own pages against BUTTONS 2.0, BOOLEAN-OPTION 2.0, COLLAPSIBLE-GROUP 1.1, SETTINGS-NAVIGATION 2.0
// and UI-SIZING 1.1 (ticket 119, slice X1): the options page, the toolbar popup, the image-OCR page and the in-page OCR
// bar, measured in a headless Chromium.
//
//   node tools/uicheck/run.mjs --only extension          the whole matrix (a couple of minutes)
//
// The real extension/ folder is served over http on a loopback port and each page gets a stub of the few chrome.* calls it
// makes (storage in memory, the language of the case, the manifest, tabs, runtime messages), so the pages' own scripts run
// unmodified: what is measured is what the engine laid out for the real, translated, state-bearing controls. What this
// cannot prove: a real installed extension in Chrome or Edge, a physical display, a screen reader (docs/UI_PROFILE.md).
import fs from 'node:fs';
import http from 'node:http';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { REPO, readerSource, loadPlaywright, launchBrowser } from './lib.mjs';
import * as R from './extension-rules.mjs';
import { navigationChecks, dialogChecks, timingChecks } from './extension-nav.mjs';
import { barChecks } from './extension-bar.mjs';
import { negativeControls } from './extension-negative.mjs';
import { modeChecks, fileChecks } from './extension-modes.mjs';
import { noticeChecks } from './extension-notices.mjs';

const { DEFAULT_OPTIONS } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'defaults.js')).href);
const { OPTIONS_PAGES } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'ui-settings.js')).href);

const SCALE = { compact: 0.875, ordinary: 1, large: 1.25, xlarge: 1.5, huge: 2 };
const TEXT = { ordinary: 1, large: 1.25, xlarge: 1.5, huge: 2 };
const TARGET = { pointer: 32, touch: 48 };
export const FIXTURES = path.join(REPO, 'temp', 'uicheck', 'ext-fixtures');
const MIME = { '.md': 'text/markdown; charset=utf-8', '.cbz': 'application/vnd.comicbook+zip', '.pdf': 'application/pdf', '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.mjs': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.wasm': 'application/wasm', '.png': 'image/png', '.svg': 'image/svg+xml', '.txt': 'text/plain; charset=utf-8' };

// ---- the server and the chrome.* stub -------------------------------------------------------------------------
export function serveExtension() {
  const dir = path.join(REPO, 'extension');
  const server = http.createServer((req, res) => {
    const u = decodeURIComponent(new URL(req.url, 'http://x').pathname);
    if (u === '/__page__') {
      const hostile = new URL(req.url, 'http://x').searchParams.get('hostile') === '1';
      res.writeHead(200, { 'content-type': MIME['.html'] });
      res.end(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Third party</title><style>body{margin:24px;font:16px Georgia,serif;color:#222;background:#fff}h1{font-size:28px;margin:0 0 12px}p{margin:0 0 12px}${hostile ? 'div,span,button,a{font:700 22px/2 serif !important;padding:30px !important;margin:12px !important;letter-spacing:3px !important;text-transform:uppercase !important;border:5px dashed red !important;color:red !important;background:yellow !important} :root{--dht-tgt:1px !important;--dht-ui:9 !important;--s:9 !important}' : ''}</style></head><body><h1>A page</h1><p>Some text of a page the reader is on, with a picture below.</p><img alt="" width="300" height="200" src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='300' height='200'%3E%3Crect width='300' height='200' fill='%23ddd'/%3E%3C/svg%3E"></body></html>`);
      return;
    }
    // Documents the viewer is pointed at (?file=), written by extension-notices.mjs
    const file = u.startsWith('/__fixtures__/') ? path.join(FIXTURES, u.slice('/__fixtures__/'.length)) : path.join(dir, u);
    if (!(file.startsWith(dir) || file.startsWith(FIXTURES)) || !fs.existsSync(file) || !fs.statSync(file).isFile()) { res.writeHead(404); res.end(); return; }
    res.writeHead(200, { 'content-type': MIME[path.extname(file)] || 'application/octet-stream' });
    fs.createReadStream(file).pipe(res);
  });
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve({ server, base: `http://127.0.0.1:${server.address().port}` })));
}

// The calls the pages make at load and in use. Everything a page writes is logged on window.__log, so a check can say what
// an interaction did: which keys were stored, whether a message was sent, whether a tab was opened.
const STUB = (cfg) => {
  const stored = JSON.parse(JSON.stringify(cfg.stored || {}));
  const log = (window.__log = { sets: [], sends: [], created: [], opened: 0 });
  window.__stored = stored;
  window.__listeners = [];
  const changed = [];
  const pick = (keys) => {
    if (keys == null) return JSON.parse(JSON.stringify(stored));
    const list = Array.isArray(keys) ? keys : typeof keys === 'string' ? [keys] : Object.keys(keys);
    const out = {};
    for (const k of list) if (k in stored) out[k] = JSON.parse(JSON.stringify(stored[k]));
    return out;
  };
  const noop = { addListener() {}, removeListener() {} };
  window.chrome = {
    runtime: {
      getURL: (p) => location.origin + '/' + String(p).replace(/^\//, ''),
      getManifest: () => ({ version: '26.929.150', name: 'doc-html-translate' }),
      sendMessage: async (m) => { log.sends.push(m); return {}; },
      onMessage: { addListener(f) { window.__listeners.push(f); }, removeListener() {} },
      openOptionsPage: () => { log.opened++; }, lastError: undefined, id: 'uicheck',
    },
    storage: {
      local: {
        get: async (k) => pick(k),
        set: async (o) => {
          log.sets.push(Object.keys(o));
          const change = {};
          for (const [k, v] of Object.entries(o)) { change[k] = { oldValue: stored[k], newValue: JSON.parse(JSON.stringify(v)) }; stored[k] = JSON.parse(JSON.stringify(v)); }
          for (const f of changed) f(change, 'local');
        },
        remove: async () => {},
      },
      onChanged: { addListener: (f) => changed.push(f), removeListener() {} },
    },
    // The browser-language messages, as chrome.i18n serves them (the real _locales file of the case's language).
    i18n: { getMessage: (k) => (cfg.messages && cfg.messages[k]) || '', getUILanguage: () => cfg.lang },
    tabs: { query: async () => [{ url: cfg.tabUrl, id: 1 }], create: (a) => { log.created.push(a); }, getCurrent: async () => ({ id: 1 }) },
    extension: { isAllowedFileSchemeAccess: (cb) => cb(cfg.fileAccess !== false) },
    scripting: noop,
  };
};

// ---- one page in one configuration -----------------------------------------------------------------------------
export function storedFor(c) {
  return {
    options: { ...DEFAULT_OPTIONS, enabledByDefault: c.enabled ?? false, ocrImages: c.ocr ?? true, uiSize: c.size || 'ordinary', textSize: c.text || 'ordinary', inputProfile: c.input || 'pointer', chromeTheme: c.chrome || 'system', ...(c.options || {}) },
    uiLang: c.lang && c.lang !== 'en' ? c.lang : '',
    ...(c.uiState ? { uiState: c.uiState } : {}),
  };
}

const messageCache = {};
export function messagesOf(lang) {
  const dir = lang === 'zh' ? 'zh_CN' : lang;
  if (!messageCache[dir]) {
    const raw = JSON.parse(fs.readFileSync(path.join(REPO, 'extension', '_locales', dir, 'messages.json'), 'utf8'));
    messageCache[dir] = Object.fromEntries(Object.entries(raw).map(([k, v]) => [k, v.message]));
  }
  return messageCache[dir];
}

export async function openPage(browser, base, c) {
  const dark = c.scheme === 'dark';
  const ctx = await browser.newContext({
    viewport: { width: c.w || 1280, height: c.h || 900 }, colorScheme: dark ? 'dark' : 'light', forcedColors: c.forced ? 'active' : 'none',
    ...(c.autoTouch ? { hasTouch: true, isMobile: true, deviceScaleFactor: 2 } : {}),
    ...(c.contrastMore ? { contrast: 'more' } : {}),
    ...(c.permissions ? { permissions: c.permissions } : {}),
  });
  const cfg = { stored: c.stored || storedFor(c), lang: c.lang || 'en', messages: messagesOf(c.lang || 'en'), tabUrl: c.tabUrl === undefined ? 'https://www.books.test/shelf/' : c.tabUrl, fileAccess: c.fileAccess };
  await ctx.addInitScript(STUB, cfg);
  const page = await ctx.newPage();
  page.errors = []; page.remote = []; page.requests = [];
  page.on('pageerror', (e) => page.errors.push(e.message));
  page.on('console', (m) => { if (m.type() === 'error' && !/Failed to load resource/.test(m.text())) page.errors.push('console: ' + m.text()); });
  page.on('response', (r) => { if (r.status() >= 400 && !/favicon/.test(r.url())) page.errors.push('HTTP ' + r.status() + ' ' + r.url()); });
  page.on('request', (r) => { page.requests.push(r.url()); if (!r.url().startsWith(base) && !/^(data|blob):/.test(r.url())) page.remote.push(r.url()); });
  await page.goto(base + c.url);
  if (!c.noWait) await page.waitForFunction(() => !document.documentElement.hasAttribute('data-boot'), null, { timeout: 15000 });
  // A negative control breaks the page on purpose (extension-negative.mjs); the rules must then fail.
  if (c.breakCss) await page.addStyleTag({ content: c.breakCss });
  if (c.breakJs) await page.evaluate(c.breakJs);
  await page.evaluate(readerSource);
  await page.waitForTimeout(c.settle ?? 150);
  return { page, ctx, async close() { await ctx.close(); } };
}

const tokens = (c) => {
  const s = SCALE[c.size || 'ordinary'], t = TEXT[c.text || 'ordinary'], gm = 1 + (t - 1) * 0.5, input = c.input || 'pointer';
  return { s, tgt: TARGET[input] * s, box: (input === 'touch' ? 24 : 20) * s, glyph: (input === 'touch' ? 20 : 16) * s * gm };
};
const tagOf = (c, extra = '') => [c.name, c.size || 'ordinary', c.input || 'pointer', c.text && c.text !== 'ordinary' ? 'text-' + c.text : null, c.lang || 'en', c.w || 1280, c.scheme === 'dark' ? 'dark' : null, c.forced ? 'forced' : null, extra].filter(Boolean).join('/');

// ---- the options page: every destination, geometry and contrast ------------------------------------------------
export async function optionsCase(browser, base, report, c, shots) {
  const T = tokens(c);
  const { page, close } = await openPage(browser, base, { ...c, url: '/src/options.html' });
  try {
    const rtl = c.lang === 'ar';
    for (const id of OPTIONS_PAGES) {
      await page.evaluate((i) => document.getElementById('tab-' + i).click(), id);
      await page.waitForTimeout(60);
      const tag = tagOf(c, id);
      await R.buttonsGeometry(page, report, tag, { ...T, expectButtons: id !== 'interface' });
      if (c.deep) { await R.contrastRules(page, report, tag); await R.outlineContrast(page, report, tag); }
      if (c.deep && (id === 'data' || id === 'about')) await R.hoverContrast(page, report, tag);
      if (id === 'ocr') await R.stateWidth(page, report, tag);
      if (shots && c.shot) await page.screenshot({ path: path.join(shots, `options-${id}-${c.shot}.png`) });
    }
    report.check(`console.clean/${tagOf(c)}`, page.errors.length === 0, page.errors.slice(0, 3).join(' | ') || 'no script or console error');
    report.check(`about.nothing-remote/${tagOf(c)}`, page.remote.length === 0, page.remote.join(', ') || 'no request left the extension origin while every page opened');
    // every rail row: a target, a whole caption, a glyph before it
    const rail = await page.evaluate(() => { const U = window.__uic; return [...document.querySelectorAll('.rail-item')].map((b) => ({ id: b.dataset.page, vis: U.visible(b), h: U.rect(b).h, clipped: b.scrollWidth > b.clientWidth + 1, glyph: !!b.querySelector('svg'), name: b.textContent.trim().length })); });
    if (rail.some((r) => r.vis)) {
      report.check(`rail.rows-target/${tagOf(c)}`, rail.every((r) => r.h >= T.tgt - 1), `rows ${rail.map((r) => r.h.toFixed(0)).join('/')} px, floor ${T.tgt.toFixed(1)}`);
      report.check(`rail.glyph-and-whole-name/${tagOf(c)}`, rail.every((r) => r.glyph && r.name > 0 && !r.clipped), 'every destination has a glyph and a whole, unclipped name');
    }
  } finally { await close(); }
}

// ---- the options page's first page: the option rows and the About groups --------------------------------------------
export async function optionRowCases(browser, base, report, c) {
  const T = tokens(c);
  const { page, close } = await openPage(browser, base, { ...c, url: '/src/options.html' });
  try {
    const tag = tagOf(c);
    for (const [pageId, id, named] of [['general', 'enabled', 'Reflow'], ['ocr', 'ocr-images', 'OCR'], ['sites', 'allow-remote', 'remote']]) {
      await page.evaluate((i) => document.getElementById('tab-' + i).click(), pageId);
      await page.waitForTimeout(60);
      await R.optionGeometry(page, report, `${tag}/${id}`, { id, tgt: T.tgt, box: T.box, s: T.s, glyph: id === 'ocr-images' ? T.glyph : null });
      if (c.interact) await R.optionInteraction(page, report, `${tag}/${id}`, { id, tgt: T.tgt, named: c.lang && c.lang !== 'en' ? '' : named });
    }
  } finally { await close(); }
}

export async function groupCases(browser, base, report, c) {
  for (const dir of ['ltr', 'rtl']) {
    const cc = { ...c, lang: dir === 'rtl' ? 'ar' : 'en', uiState: { page: 'about', groups: {} } };
    const { page, close } = await openPage(browser, base, { ...cc, url: '/src/options.html', h: 640 });
    try {
      const T = tokens(cc);
      await R.groupRules(page, report, `${tagOf(cc)}/aboutResources`, { id: 'aboutResources', rtl: dir === 'rtl', tgt: T.tgt });
      if (dir === 'ltr') await R.groupRules(page, report, `${tagOf(cc)}/aboutSupport`, { id: 'aboutSupport', rtl: false, tgt: T.tgt });
      // COLLAPSIBLE-GROUP 4 rule 2: the activated header is the viewport anchor - opening a group below it moves what follows,
      // not the header the reader just pressed
      await page.setViewportSize({ width: 1280, height: 330 });
      const anchor = await page.evaluate(() => {
        for (const id of ['aboutSupport', 'aboutResources']) document.getElementById(id).open = true;
        document.getElementById('aboutRelated').open = false;
        const s = document.querySelector('#aboutRelated > summary');
        s.scrollIntoView({ block: 'center' });
        return { before: s.getBoundingClientRect().top, room: document.getElementById('pane').scrollHeight - document.getElementById('pane').clientHeight };
      });
      await page.waitForTimeout(80);
      const top0 = await page.evaluate(() => document.querySelector('#aboutRelated > summary').getBoundingClientRect().top);
      await page.evaluate(() => document.querySelector('#aboutRelated > summary').click());
      await page.waitForTimeout(80);
      const top1 = await page.evaluate(() => document.querySelector('#aboutRelated > summary').getBoundingClientRect().top);
      report.check(`group.viewport-anchor/${tagOf(cc)}`, anchor.room > 40 && R.approx(top0, top1, 3), `header top ${top0.toFixed(1)} -> ${top1.toFixed(1)} px when the body opened (the pane scrolls ${anchor.room.toFixed(0)} px)`);
    } finally { await close(); }
  }
}

// ---- the popup ------------------------------------------------------------------------------------------------
export async function popupCase(browser, base, report, c, shots) {
  const T = tokens(c);
  const { page, close } = await openPage(browser, base, { ...c, url: '/src/popup.html', w: 800, h: 1400 });
  try {
    const tag = tagOf({ ...c, name: 'popup' });
    await R.buttonsGeometry(page, report, tag, T);
    if (c.deep) { await R.contrastRules(page, report, tag); await R.outlineContrast(page, report, tag); }
    const p = await page.evaluate(() => {
      const U = window.__uic, body = document.body, cta = document.getElementById('open-pdf');
      return {
        sliders: document.querySelectorAll('.slider, .switch').length, boxes: [...document.querySelectorAll('input[type=checkbox]')].map((b) => b.classList.contains('opt-box')),
        width: body.getBoundingClientRect().width, cta: U.rect(cta), ctaRadius: parseFloat(getComputedStyle(cta).borderTopLeftRadius), primary: document.querySelectorAll('button.primary').length,
        settingsIsButton: document.getElementById('opts').tagName === 'BUTTON', deadLinks: document.querySelectorAll('a[href="#"]').length,
        ext: [...document.querySelectorAll('a[target=_blank], a[href^=mailto]')].map((a) => !!a.querySelector('svg')),
        group: (() => { const s = document.querySelector('#popupOcrLangs > summary'); const cs = getComputedStyle(s, '::before'); return { mask: cs.maskImage, h: U.rect(s).h, weight: getComputedStyle(s).fontWeight }; })(),
        scrollW: document.documentElement.scrollWidth, boxW: body.scrollWidth,
      };
    });
    report.check(`popup.no-sliding-switch/${tag}`, p.sliders === 0 && p.boxes.length === 3 && p.boxes.every(Boolean), `${p.sliders} sliding switches, ${p.boxes.length} checkboxes (all .opt-box: ${p.boxes.every(Boolean)})`);
    report.check(`popup.one-primary/${tag}`, p.primary === 1, `${p.primary} primary button`);
    report.check(`popup.cta-radius/${tag}`, R.approx(p.ctaRadius, 4 * T.s, 0.6), `radius ${p.ctaRadius}`);
    report.check(`popup.settings-is-a-button/${tag}`, p.settingsIsButton && p.deadLinks === 0, `Settings is a <button>; dead href="#" links: ${p.deadLinks}`);
    report.check(`popup.external-links-marked/${tag}`, p.ext.length === 3 && p.ext.every(Boolean), `${p.ext.length} outside links, markers ${p.ext.join(',')}`);
    report.check(`popup.group-marker/${tag}`, !!p.group.mask && p.group.mask !== 'none' && Number(p.group.weight) >= 600 && p.group.h >= T.tgt - 1, `group header ${p.group.h.toFixed(0)} px high, weight ${p.group.weight}, vector marker`);
    report.check(`popup.fits-the-popup/${tag}`, p.scrollW <= 800, `content ${p.scrollW} px of the 800 px a browser allows a popup`);
    report.check(`console.clean/${tag}`, page.errors.length === 0, page.errors.slice(0, 3).join(' | ') || 'no script or console error');
    if (shots && c.shot) await page.screenshot({ path: path.join(shots, `popup-${c.shot}.png`) });
  } finally { await close(); }
}

async function popupStates(browser, base, report) {
  // the disabled site switch says why and how to enable it; enabling the switch it names removes the reason
  const off = await openPage(browser, base, { name: 'popup', url: '/src/popup.html', w: 800, h: 1400, enabled: false });
  try {
    const d = await off.page.evaluate(() => ({ disabled: document.getElementById('site').disabled, reason: document.getElementById('site-reason').textContent, described: document.getElementById('site').getAttribute('aria-describedby'), inside: document.getElementById('site-expl').contains(document.getElementById('site-reason')) }));
    report.check('popup.disabled-switch-says-why', d.disabled && d.reason.length > 20 && d.described === 'site-expl' && d.inside, `disabled, reason "${d.reason}"`);
    await off.page.click('label[for="global"]');
    await off.page.waitForTimeout(80);
    const e = await off.page.evaluate(() => ({ disabled: document.getElementById('site').disabled, reason: document.getElementById('site-reason').textContent, stored: window.__stored.options.enabledByDefault }));
    report.check('popup.enabling-global-enables-site', !e.disabled && e.reason === '' && e.stored === true, `site switch enabled ${!e.disabled}, reason cleared ${e.reason === ''}, stored ${e.stored}`);
    // the label of the site option toggles once and the host stays inside it
    await off.page.click('label[for="site"] > span');
    await off.page.waitForTimeout(80);
    const s = await off.page.evaluate(() => ({ list: window.__stored.options.disabledHosts, checked: document.getElementById('site').checked }));
    report.check('popup.site-label-toggles-once', JSON.stringify(s.list) === '["www.books.test"]' && s.checked === false, `disabled hosts ${JSON.stringify(s.list)}`);
    await off.page.click('#opts');
    report.check('popup.settings-opens-options', (await off.page.evaluate(() => window.__log.opened)) === 1, 'the Settings button opened the options page once');
    // the group opens when OCR is turned on, and the choice is remembered in its own key (not in `options`)
    await off.page.evaluate(() => { document.getElementById('popupOcrLangs').open = true; });
    await off.page.waitForTimeout(80);
    await off.page.evaluate(() => { document.getElementById('popupOcrLangs').open = false; });
    await off.page.waitForTimeout(80);
    const g = await off.page.evaluate(() => window.__stored.uiState && window.__stored.uiState.groups);
    report.check('popup.group-state-remembered', g && g.popupOcrLangs === false, `remembered groups ${JSON.stringify(g)}`);
  } finally { await off.close(); }
  const sub = await openPage(browser, base, { name: 'popup', url: '/src/popup.html', w: 800, h: 1400, enabled: true, tabUrl: 'https://www.books.test/a.pdf' });
  try {
    const cap = await sub.page.evaluate(() => ({ text: document.getElementById('open-pdf').textContent.trim(), note: document.getElementById('tab-state').hidden }));
    report.check('popup.document-tab-offers-open-in-reader', cap.text.length > 0 && cap.note === false, `CTA "${cap.text}", note shown ${!cap.note}`);
  } finally { await sub.close(); }
}

// ---- the image-OCR page ----------------------------------------------------------------------------------------
export async function ocrCase(browser, base, report, c) {
  const T = tokens(c);
  const { page, close } = await openPage(browser, base, { ...c, name: 'ocr', url: '/src/ocr.html?src=' + encodeURIComponent('file:///C:/x.png'), fileAccess: false });
  try {
    await page.waitForSelector('.ocr-hint button', { timeout: 8000 });
    const tag = tagOf({ ...c, name: 'ocr' });
    await R.buttonsGeometry(page, report, tag, T);
    const b = await page.evaluate(() => ({ primary: document.querySelectorAll('button.primary').length, lang: document.documentElement.hasAttribute('lang') }));
    report.check(`ocr.one-primary-action/${tag}`, b.primary === 1, `${b.primary} primary button`);
    report.check(`console.clean/${tag}`, page.errors.length === 0, page.errors.slice(0, 3).join(' | ') || 'no script or console error');
  } finally { await close(); }
}

// A scenario that throws (a selector that is not there, a timeout) is a failed check naming it, not the end of the run.
async function guard(report, name, fn) {
  try { await fn(); } catch (e) { report.check(`harness.scenario-ran/${name}`, false, String(e.message).split('\n')[0]); }
}

// ---- the matrix --------------------------------------------------------------------------------------------------
const OPTIONS_CASES = [
  { size: 'ordinary', input: 'pointer', deep: true, shot: 'light' },
  { size: 'ordinary', input: 'touch', deep: true },
  { size: 'compact', input: 'pointer', deep: true },
  { size: 'compact', input: 'touch' },
  { size: 'xlarge', input: 'pointer', deep: true },
  { size: 'xlarge', input: 'touch' },
  { size: 'huge', input: 'pointer', w: 1500 },
  { size: 'ordinary', input: 'pointer', lang: 'ru', deep: true },
  { size: 'ordinary', input: 'pointer', lang: 'de', w: 1000 },
  { size: 'ordinary', input: 'pointer', lang: 'ar', deep: true, shot: 'rtl-ar' },
  { size: 'ordinary', input: 'pointer', lang: 'hi' },
  { size: 'large', input: 'pointer', deep: true },
  { size: 'compact', input: 'touch', lang: 'ru', w: 1100 },
  { size: 'xlarge', input: 'touch', lang: 'de', w: 1400 },
  { size: 'compact', input: 'pointer', lang: 'ar' },
  { size: 'ordinary', input: 'touch', lang: 'ar' },
  { size: 'ordinary', input: 'pointer', text: 'huge', lang: 'de', w: 1200 },
  { size: 'ordinary', input: 'pointer', w: 700, shot: 'narrow' },
  { size: 'ordinary', input: 'pointer', w: 420 },
  { size: 'ordinary', input: 'pointer', scheme: 'dark', deep: true, shot: 'dark' },
  { size: 'ordinary', input: 'pointer', scheme: 'dark', chrome: 'dark', deep: true },
  { size: 'ordinary', input: 'pointer', forced: true, shot: 'forced' },
];
const POPUP_CASES = [
  { size: 'ordinary', input: 'pointer', deep: true, shot: 'light', enabled: true },
  { size: 'ordinary', input: 'touch', deep: true, enabled: true },
  { size: 'compact', input: 'pointer' }, { size: 'compact', input: 'touch' },
  { size: 'xlarge', input: 'pointer' }, { size: 'xlarge', input: 'touch' },
  { size: 'huge', input: 'pointer', text: 'xlarge' },
  { size: 'ordinary', input: 'pointer', lang: 'ru', deep: true },
  { size: 'ordinary', input: 'pointer', lang: 'de' },
  { size: 'large', input: 'pointer', enabled: true },
  { size: 'compact', input: 'touch', lang: 'ru' },
  { size: 'xlarge', input: 'touch', lang: 'de' },
  { size: 'compact', input: 'pointer', lang: 'ar' },
  { size: 'ordinary', input: 'touch', lang: 'ar', deep: true },
  { size: 'ordinary', input: 'pointer', lang: 'ar', deep: true, shot: 'rtl-ar' },
  { size: 'ordinary', input: 'pointer', scheme: 'dark', deep: true, shot: 'dark' },
  { size: 'ordinary', input: 'pointer', forced: true, shot: 'forced' },
];

export async function extensionChecks(report, { shotsDir, outDir } = {}) {
  const { server, base } = await serveExtension();
  const { chromium } = await loadPlaywright();
  const browser = await launchBrowser(chromium);
  try {
    if (shotsDir) fs.mkdirSync(shotsDir, { recursive: true });
    for (const c of OPTIONS_CASES) await guard(report, 'options ' + JSON.stringify(c), () => optionsCase(browser, base, report, { ...c, name: 'options' }, shotsDir));
    for (const c of [{ size: 'ordinary', input: 'pointer', interact: true }, { size: 'ordinary', input: 'touch' }, { size: 'compact', input: 'pointer' }, { size: 'xlarge', input: 'touch' }, { size: 'ordinary', input: 'pointer', lang: 'ru' }, { size: 'ordinary', input: 'pointer', lang: 'ar' }]) {
      await guard(report, 'option rows ' + JSON.stringify(c), () => optionRowCases(browser, base, report, { ...c, name: 'options' }));
    }
    await guard(report, 'groups', () => groupCases(browser, base, report, { name: 'options', size: 'ordinary', input: 'pointer' }));
    await guard(report, 'navigation', () => navigationChecks(browser, base, report, { openPage, storedFor }));
    await guard(report, 'dialogs', () => dialogChecks(browser, base, report, { openPage, storedFor }));
    if (!process.argv.includes('--quick')) await guard(report, 'timing', () => timingChecks(browser, base, report, { openPage, outDir }));
    for (const c of POPUP_CASES) await guard(report, 'popup ' + JSON.stringify(c), () => popupCase(browser, base, report, { ...c, name: 'popup' }, shotsDir));
    await guard(report, 'popup states', () => popupStates(browser, base, report));
    for (const c of [{ size: 'ordinary', input: 'pointer' }, { size: 'ordinary', input: 'touch' }, { size: 'compact', input: 'pointer' }, { size: 'xlarge', input: 'pointer' }, { size: 'ordinary', input: 'pointer', scheme: 'dark' }]) {
      await guard(report, 'ocr ' + JSON.stringify(c), () => ocrCase(browser, base, report, c));
    }
    await guard(report, 'bar', () => barChecks(browser, base, report, { openPage, storedFor, shotsDir }));
    await guard(report, 'viewer notices', () => noticeChecks(browser, base, report, { openPage, messagesOf, fixtures: FIXTURES, shotsDir }));
    await guard(report, 'display modes', () => modeChecks(browser, base, report, { openPage }));
    await guard(report, 'files', () => fileChecks(browser, base, report, { openPage }));
    await guard(report, 'negative controls', () => negativeControls(browser, base, report, { openPage, storedFor, runners: { optionsCase, optionRowCases, groupCases, popupCase } }));
    // the automatic profile resolves once from the primary pointer: a touch device opens in the touch profile
    const auto = await openPage(browser, base, { name: 'options', url: '/src/options.html', input: 'auto', autoTouch: true, w: 390, h: 844 });
    try {
      const d = await auto.page.evaluate(() => ({ input: document.documentElement.dataset.input, auto: document.documentElement.dataset.inputAuto, coarse: matchMedia('(pointer: coarse)').matches }));
      report.check('profile.auto-touch-on-a-coarse-pointer', d.coarse && d.input === 'touch' && d.auto === 'touch', `(pointer: coarse) ${d.coarse}, data-input ${d.input}`);
    } finally { await auto.close(); }
    const fine = await openPage(browser, base, { name: 'options', url: '/src/options.html', input: 'auto' });
    try {
      const d = await fine.page.evaluate(() => ({ input: document.documentElement.dataset.input }));
      report.check('profile.auto-pointer-on-a-fine-pointer', d.input === 'pointer', `data-input ${d.input}`);
      // never on an event: a touch event and a wide mouse move leave the profile as it was
      await fine.page.mouse.move(300, 300);
      await fine.page.touchscreen?.tap(200, 200).catch(() => {});
      report.check('profile.never-changes-on-an-event', (await fine.page.evaluate(() => document.documentElement.dataset.input)) === 'pointer', 'pointer and touch events left the profile as it was');
    } finally { await fine.close(); }
  } finally { await browser.close(); server.close(); }
}

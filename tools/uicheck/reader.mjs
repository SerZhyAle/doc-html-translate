// The generated reader chrome against BUTTONS 2.0 and UI-SIZING 1.1 (ticket 119, slice R1), measured on pages the real
// CLI converts: a text book in multipage mode (chapter page and index toolbar) and in the default single-page mode, an
// image-page book (a comic: page jump, fit width) and the same with an OCR layer, each opened from disk in a headless
// Chromium with a fine pointer, a coarse pointer and a narrow window, in the four reading themes, with forced colours,
// right-to-left and the longest translations. Nothing here reads a stylesheet: every number is what the engine laid out.
//
//   node tools/uicheck/run.mjs --only reader            the generated reader and the extension viewer
//   node tools/uicheck/run.mjs --only reader --no-build reuse temp/uicheck/bin
//
// The extension viewer twin is measured by reader-ext.mjs with the same rules (reader-rules.mjs).
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import { REPO, readerSource, loadPlaywright, launchBrowser } from './lib.mjs';
import * as M from './reader-measure.mjs';
import { geometryRules, tokens } from './reader-rules.mjs';

export const WORK = path.join(REPO, 'temp', 'uicheck', 'reader');
const BIN = path.join(REPO, 'temp', 'uicheck', 'bin');
export const MODES = {
  fine: { viewport: { width: 1280, height: 900 } },
  narrow: { viewport: { width: 420, height: 800 } },
  coarse: { viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, deviceScaleFactor: 2 },
};

// The profile of the generated reader: where its bar, controls, panels and state controls are.
export const GO = {
  name: 'go', root: '.dht-navbar, .dht-toolbar', bar: '.dht-navbar', controls: 'button, a, select', items: '.nav-file, .nav-title, .nav-info, .dht-sel > svg', panels: '.dht-search-panel',
  scaleVar: '--dht-ui', themeKey: 'dht_theme', version: '.nav-version', progress: '#dht-progress',
  search: { button: '#dht-search-button', panel: '.dht-search-panel', input: '#dht-search-input', close: '#dht-search-close', row: 'button, select, input', items: 'label' },
  contents: { button: '#dht-contents-button', panel: '.dht-contents', close: '#dht-contents-close', row: '.dht-contents-head button', items: 'strong' },
  toggles: ['#dht-night-toggle', '#dht-ocr-toggle', '#dht-fitw-toggle'],
};

// ---- fixtures ---------------------------------------------------------------------------------------------
const LANGS = ['en', 'de', 'ru', 'hi', 'ar'];
const out = (...p) => path.join(WORK, 'out', ...p);

function crc32(buf) { let crc = 0xffffffff; for (const b of buf) { let c = (crc ^ b) & 0xff; for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1; crc = (crc >>> 8) ^ c; } return (crc ^ 0xffffffff) >>> 0; }
function pngChunk(type, data) { const len = Buffer.alloc(4); len.writeUInt32BE(data.length); const td = Buffer.concat([Buffer.from(type), data]); const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(td)); return Buffer.concat([len, td, crc]); }
function png(w, h, shade) {
  const raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) { const o = y * (w * 3 + 1) + 1 + x * 3; raw[o] = (shade + x) % 256; raw[o + 1] = (shade + y) % 256; raw[o + 2] = 128; }
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(w, 0); ihdr.writeUInt32BE(h, 4); ihdr[8] = 8; ihdr[9] = 2;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), pngChunk('IHDR', ihdr), pngChunk('IDAT', zlib.deflateSync(raw)), pngChunk('IEND', Buffer.alloc(0))]);
}

// the book's text: seven sections, enough to scroll and to give the contents panel a tree
export function writeBookSource() {
  const dir = path.join(WORK, 'src');
  fs.mkdirSync(dir, { recursive: true });
  const para = (c, n) => `Paragraph ${n} of chapter ${c}: the quick brown fox jumps over the lazy dog while the reader scrolls through a sufficiently long page of text so that the progress bar has something to measure.`;
  const md = '# The Long Walk\n\n' + [1, 2, 3, 4, 5, 6].map((c) => `## Chapter ${c}\n\n` + [1, 2, 3, 4, 5, 6, 7, 8].map((n) => para(c, n)).join('\n\n')).join('\n\n') + '\n';
  fs.writeFileSync(path.join(dir, 'book.md'), md);
  return path.join(dir, 'book.md');
}
async function writeComicSource() {
  const dir = path.join(WORK, 'src');
  const entry = path.join(REPO, 'extension', 'node_modules', 'fflate', 'esm', 'browser.js');
  const { zipSync } = await import(pathToFileURL(entry).href);
  const files = {};
  for (let i = 1; i <= 4; i++) files[`page${i}.png`] = new Uint8Array(png(300, 420, i * 40));
  fs.writeFileSync(path.join(dir, 'comic.cbz'), Buffer.from(zipSync(files, { level: 0 })));
  return path.join(dir, 'comic.cbz');
}

export function buildCli() {
  fs.mkdirSync(BIN, { recursive: true });
  execFileSync('go', ['build', '-o', path.join(BIN, 'doc-html-translate.exe'), './cmd/doc-html-translate'], { cwd: REPO, env: { ...process.env, GOFLAGS: '-tags=nodynamic' }, stdio: 'inherit' });
}

export async function buildFixtures() {
  const exe = path.join(BIN, 'doc-html-translate.exe');
  fs.rmSync(path.join(WORK, 'out'), { recursive: true, force: true });
  const book = writeBookSource(), comic = await writeComicSource();
  const conv = (args) => execFileSync(exe, ['-noopen', '-notranslate', '-force', ...args], { cwd: REPO, stdio: 'ignore' });
  for (const lang of LANGS) {
    conv(['-ui-lang', lang, '-folder', out(lang + '-single'), book]);
    conv(['-ui-lang', lang, '-multipage', '-folder', out(lang + '-multi'), book]);
  }
  conv(['-ui-lang', 'en', '-folder', out('en-comic'), comic]);
  // The OCR toggle shows itself only on a page that carries plates; the converter finds no text in a synthetic picture,
  // so the plate container the reader script looks for is added to a copy.
  const src = fs.readFileSync(out('en-comic', 'comic', 'index.html'), 'utf8');
  fs.mkdirSync(out('en-comic-ocr', 'comic'), { recursive: true });
  fs.writeFileSync(out('en-comic-ocr', 'comic', 'index.html'), src.replace('<main class="dht-single">', '<main class="dht-single"><div class="ocr-fig" hidden></div>'));
  return {
    chapter: (l) => out(l + '-multi', 'book', 'page_002.html'), index: (l) => out(l + '-multi', 'book', 'index.html'),
    first: () => out('en-multi', 'book', 'page_001.html'), single: (l) => out(l + '-single', 'book', 'index.html'), paged: () => out('en-comic', 'comic', 'index.html'), ocr: () => out('en-comic-ocr', 'comic', 'index.html'),
  };
}

// ---- the case matrix: the baseline plus risk-selected combinations (UI-SIZING section 4) ---------------------------
export function matrix() {
  const cases = [];
  for (const page of ['chapter', 'first', 'index', 'single', 'paged', 'ocr']) cases.push({ page, mode: 'fine', deep: true });
  for (const page of ['chapter', 'index', 'single', 'paged']) cases.push({ page, mode: 'coarse', deep: page !== 'paged' });
  for (const page of ['chapter', 'single', 'paged']) cases.push({ page, mode: 'narrow', deep: page !== 'paged' });
  for (const theme of ['sepia', 'dark', 'night']) for (const page of ['chapter', 'single', 'index']) cases.push({ page, mode: 'fine', theme, deep: true });
  for (const page of ['chapter', 'single', 'index']) cases.push({ page, mode: 'fine', forced: true, deep: true });
  cases.push({ page: 'single', mode: 'fine', theme: 'dark', forced: true, deep: true }, { page: 'single', mode: 'fine', contrastMore: true });
  for (const page of ['chapter', 'single', 'index']) for (const mode of ['fine', 'narrow']) cases.push({ page, mode, lang: 'ar', deep: mode === 'fine' });
  for (const lang of ['de', 'ru', 'hi']) for (const page of ['chapter', 'single']) for (const mode of ['fine', 'narrow', 'coarse']) cases.push({ page, mode, lang });
  for (const scale of [0.875, 2, 0, 100]) cases.push({ page: 'chapter', mode: 'fine', scale });
  cases.push({ page: 'single', mode: 'coarse', scale: 0.875 }, { page: 'single', mode: 'coarse', scale: 2 });
  cases.push({ page: 'chapter', mode: 'fine', zoom: 50 });
  return cases;
}
export const caseTag = (c) => [c.page, c.mode, c.lang || 'en', c.theme || 'light', c.forced && 'forced', c.contrastMore && 'contrast', c.scale !== undefined && 'x' + c.scale, c.zoom && 'zoom' + c.zoom, c.reveal && 'ocr-revealed'].filter(Boolean).join('/');

// ---- helpers ------------------------------------------------------------------------------------------------
const lum = (c) => { const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(c[0]) + 0.7152 * f(c[1]) + 0.0722 * f(c[2]); };
export const ratio = (a, b) => { const x = lum(a), y = lum(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05); };
export const rgb = (s) => { const m = String(s).match(/rgba?\(([^)]+)\)/); return m ? m[1].split(/[ ,/]+/).filter(Boolean).slice(0, 3).map(Number) : null; };
const f1 = (n) => (Math.round(n * 10) / 10).toString();

// ---- one case -------------------------------------------------------------------------------------------------
export async function openCase(browser, P, url, c) {
  const dark = c.theme === 'dark' || c.theme === 'night';
  const ctx = await browser.newContext({ ...MODES[c.mode], colorScheme: dark ? 'dark' : 'light', forcedColors: c.forced ? 'active' : 'none', ...(c.contrastMore ? { contrast: 'more' } : {}) });
  if (c.theme && P.themeKey) await ctx.addInitScript(([k, t]) => { try { localStorage.setItem(k, t); } catch { /* storage is optional */ } }, [P.themeKey, c.theme]);
  const page = await ctx.newPage();
  page.errors = [];
  page.on('pageerror', (e) => page.errors.push(e.message));
  await page.goto(url);
  await page.evaluate(readerSource);
  if (c.scale !== undefined) await page.evaluate(([v, s]) => document.documentElement.style.setProperty(v, String(s)), [P.scaleVar, c.scale]);
  await page.waitForTimeout(200);
  c.pageOverflow = await page.evaluate(M.overflowWithoutChrome, P);
  // The converter's own page CSS (body width 95 percent plus 1em of padding, at the reader's 175 percent type) makes a
  // book page wider than a phone, which widens the layout viewport and moves every fixed panel with it. That is the
  // book's layout, not the chrome (reported as an advisory in geometry()); the rest is measured with it taken out so
  // that what is judged is the chrome.
  await page.addStyleTag({ content: 'html body, main.dht-single { box-sizing: border-box !important; }' });
  return { ctx, page };
}

// the captures a person looks at beside the measurements (a screenshot proves none of the rules)
export async function capture(page, c, P) {
  if (!SHOTS.dir || !c.deep) return;
  fs.mkdirSync(SHOTS.dir, { recursive: true });
  await page.screenshot({ path: path.join(SHOTS.dir, 'reader-' + P.name + '-' + caseTag(c).replaceAll('/', '-') + '.png'), clip: { x: 0, y: 0, width: MODES[c.mode].viewport.width, height: Math.min(MODES[c.mode].viewport.height, 460) } });
}
const SHOTS = { dir: null };

export async function geometry(report, page, P, c, rtl) {
  const tag = `${P.name}/${caseTag(c)}`;
  const m = await page.evaluate(M.collect, P);
  const coarse = c.mode === 'coarse';
  for (const r of geometryRules(m, { coarse, scale: c.scale ?? 1, rtl })) report.check(`reader.${r.id}/${tag}`, r.ok, r.detail);
  // the chrome adds no horizontal scroll of its own: the page with the chrome out scrolls exactly as wide
  const ov = await page.evaluate(M.overflowWithoutChrome, P);
  report.check(`reader.no-horizontal-scroll-from-chrome/${tag}`, ov.withChrome <= ov.without + 1, `scrollWidth ${ov.withChrome} with the chrome, ${ov.without} without (viewport ${m.iw})`);
  const baseline = !c.theme && !c.forced && !c.lang && c.scale === undefined && !c.contrastMore;
  if (baseline && c.pageOverflow && c.pageOverflow.without > m.iw + 1) report.advisory(`reader.book-page-wider-than-window/${tag}`, `the book's own page scrolls horizontally with the chrome removed too (scrollWidth ${c.pageOverflow.without} in a ${m.iw} px window): converter CSS, not the chrome; the chrome adds ${c.pageOverflow.withChrome - c.pageOverflow.without} px`);
  if (c.contrastMore) {
    const plain = m.controls.filter((x) => !/nav-version|proj-link/.test(x.key));
    const off = plain.filter((x) => x.borderColor !== x.color);
    report.check(`reader.contrast-more-borders/${tag}`, plain.length > 0 && off.length === 0, off.length ? off.map((x) => `${x.key} border ${x.borderColor} ink ${x.color}`).join('; ') : `prefers-contrast: more draws all ${plain.length} borders in the ink colour`);
  }
  report.check(`reader.page-error-free/${tag}`, page.errors.length === 0, page.errors.join('; ') || 'no script error');
  return m;
}

// contrast of caption and glyph on the actual face, idle and hovered (BUTTONS 3 rule 6)
export async function contrast(report, page, P, c) {
  const tag = `${P.name}/${caseTag(c)}`;
  const at = await page.evaluate(M.centers, P.controls);
  const idle = [], hover = [];
  for (let i = 0; i < at.length; i++) {
    const a = await page.evaluate(M.contrastOf, [P.controls, i]);
    if (a && !a.disabled) idle.push(a);
    if (at[i].disabled) continue;
    await page.mouse.move(at[i].x, at[i].y);
    const h = await page.evaluate(M.contrastOf, [P.controls, i]);
    if (h) hover.push(h);
  }
  await page.mouse.move(0, 0);
  if (c.mode === 'fine' && !c.lang && !c.forced && !c.contrastMore && c.scale === undefined && (c.page === 'chapter' || c.page === 'viewer')) {
    const b = idle.filter((x) => x.border !== null).map((x) => x.border);
    if (b.length) report.advisory(`reader.button-border-contrast/${tag}`, `the quietest button border reads ${f1(Math.min(...b))}:1 against the bar: the theme's own --border (internal/appearance) is a quiet line, below the 3:1 that WCAG 1.4.11 asks of a component boundary; BUTTONS 3 rule 6 names the caption and glyph contrast, which are measured and pass`);
  }
  for (const [name, list] of [['idle', idle], ['hover', hover]]) {
    const low = list.filter((x) => (x.text !== null && x.text < 4.5) || (x.glyph !== null && x.glyph < 3));
    const minT = Math.min(...list.filter((x) => x.text !== null).map((x) => x.text)), minG = Math.min(...list.filter((x) => x.glyph !== null).map((x) => x.glyph));
    report.check(`reader.contrast-${name}/${tag}`, list.length > 0 && low.length === 0, low.length ? low.map((x) => `${x.key} text ${f1(x.text)} glyph ${f1(x.glyph)}`).join('; ') : `${list.length} controls, lowest caption ${f1(minT)}:1 (need 4.5), lowest glyph ${f1(minG)}:1 (need 3)`);
  }
}

// a selected toggle is a distinct face plus a state that is not colour alone; a disclosure that is open says the same
export async function selectedState(report, page, P, c, sel, attr) {
  const tag = `${P.name}/${caseTag(c)} [${sel}]`;
  const p = await page.evaluate(M.pressedProbe, [sel, attr]);
  if (!p) return;
  const face = rgb(p.after.bg), ink = rgb(p.after.color), was = rgb(p.before.bg);
  if (c.forced) {
    const d = ratio(face, was);
    report.check(`reader.selected-cue/${tag}`, p.after.bg !== p.before.bg && d >= 3, `forced colours: selected face ${p.after.bg} vs idle ${p.before.bg}, ${f1(d)}:1 between them`);
  } else {
    const inset = /inset/.test(p.after.shadow) && p.after.shadow !== p.before.shadow;
    const bar = rgb(p.after.shadow);
    report.check(`reader.selected-cue/${tag}`, inset && bar && ratio(bar, face) >= 3 && p.after.bg !== p.before.bg, `selected face ${p.after.bg}, inset bar ${p.after.shadow.slice(0, 40)} (${bar ? f1(ratio(bar, face)) : '?'}:1 on the face; idle shadow ${p.before.shadow.slice(0, 20)})`);
  }
  report.check(`reader.selected-contrast/${tag}`, ratio(ink, face) >= 4.5, `ink on the selected face ${f1(ratio(ink, face))}:1`);
}

// keyboard: every control takes a visible ring, in document order
export async function keyboard(report, page, P, c, m) {
  const tag = `${P.name}/${caseTag(c)}`;
  await page.evaluate(() => document.activeElement && document.activeElement.blur());
  const stops = [];
  for (let i = 0; i < m.controls.length + 4; i++) {
    await page.keyboard.press('Tab');
    const f = await page.evaluate(M.focusInfo);
    if (f && f.chrome) stops.push(f);
  }
  const weak = stops.filter((s) => s.style === 'none' || s.width < 2 || s.contrast === null || s.contrast < 3);
  report.check(`reader.focus-visible/${tag}`, stops.length >= m.controls.length - 2 && weak.length === 0, weak.length ? weak.map((s) => `${s.key} ${s.style} ${s.width}px ${s.contrast && f1(s.contrast)}:1`).join('; ') : `${stops.length} stops, ring >= 2 px at >= ${f1(Math.min(...stops.map((s) => s.contrast)))}:1 on the bar`);
  const want = m.controls.filter((x) => x.disabledAria !== 'true').map((x) => x.key);
  const got = stops.filter((s) => s.key !== P.progress).map((s) => s.key);
  report.check(`reader.tab-order-is-document-order/${tag}`, JSON.stringify(want) === JSON.stringify(got.slice(0, want.length)), want.join(' ') === got.slice(0, want.length).join(' ') ? `${want.length} controls in document order` : `want ${want.join(' ')} got ${got.join(' ')}`);
}

// a disclosure panel hangs under the bar at its real height and never under it (BUTTONS 4 rule 2)
export async function panel(report, page, P, c, which, hasBar) {
  const D = P[which];
  if (!D || !(await page.$(D.button))) return;
  const tag = `${P.name}/${caseTag(c)} [${which}]`;
  await page.click(D.button);
  await page.waitForTimeout(120);
  const expanded = await page.getAttribute(D.button, 'aria-expanded');
  const box = await page.evaluate(M.panelBox, [D.panel, P.bar]);
  report.check(`reader.disclosure-state/${tag}`, expanded === 'true' && box && box.visible, `aria-expanded ${expanded}, panel visible ${box && box.visible}`);
  if (box && hasBar) {
    report.check(`reader.panel-not-covered/${tag}`, box.top >= box.barBottom - 0.5, `panel top ${f1(box.top)}, bar bottom ${f1(box.barBottom)}`);
  }
  if (box) report.check(`reader.panel-in-viewport/${tag}`, box.left >= -1 && box.right <= box.iw + 1 && box.top >= -1, `panel x ${f1(box.left)}..${f1(box.right)} of ${f1(box.iw)}`);
  const pm = await page.evaluate(M.collect, { root: D.panel, controls: D.row, items: D.items });
  const T = tokens(c.mode === 'coarse', c.scale ?? 1);
  const bad = pm.controls.filter((x) => x.h < T.tgt - 0.5 || x.w < T.tgt - 0.5 || x.radius > T.radius + 0.5);
  // under the reader's own page zoom the rects are in zoomed pixels, so the target size is the advisory above, not a rule here
  if (!c.zoom) report.check(`reader.panel-controls/${tag}`, pm.controls.length > 0 && bad.length === 0, bad.length ? bad.map((x) => `${x.key} ${f1(x.w)}x${f1(x.h)} r${x.radius}`).join('; ') : `${pm.controls.length} control(s) >= ${f1(T.tgt)}, radius <= ${f1(T.radius)}`);
  await selectedState(report, page, P, c, D.button, 'aria-expanded');
  await page.keyboard.press('Escape');
  await page.waitForTimeout(80);
  const after = await page.evaluate(([b]) => ({ expanded: document.querySelector(b).getAttribute('aria-expanded'), active: document.activeElement && document.activeElement.id }), [D.button]);
  report.check(`reader.disclosure-escape-returns-focus/${tag}`, after.expanded === 'false' && '#' + after.active === D.button, `aria-expanded ${after.expanded}, focus on ${after.active}`);
}

// forced colours: what a colour carried must survive in system colours
export async function forcedColours(report, page, P, c) {
  const tag = `${P.name}/${caseTag(c)}`;
  if (P.progress && (await page.$(P.progress))) {
    const v = await page.evaluate((s) => ({ bg: getComputedStyle(document.querySelector(s)).backgroundColor, page: getComputedStyle(document.body).backgroundColor }), P.progress);
    report.check(`reader.forced-progress-visible/${tag}`, v.bg !== v.page, `progress fill ${v.bg} on a ${v.page} page`);
  }
  if (P.version && (await page.$(P.version))) {
    const loc = page.locator(P.version).first();
    const a = await loc.screenshot();
    await page.addStyleTag({ content: `${P.version}::after{visibility:hidden !important}` });
    const b = await loc.screenshot();
    report.check(`reader.forced-marker-paints/${tag}`, !a.equals(b), 'the external-link marker changes the pixels of the link');
  }
}

export async function runCase(browser, report, P, urls, c) {
  const rtl = c.lang === 'ar';
  const { ctx, page } = await openCase(browser, P, urls[c.page](c.lang || 'en'), c);
  try {
    const m = await geometry(report, page, P, c, rtl);
    if (!c.deep) return;
    await capture(page, c, P);
    await contrast(report, page, P, c);
    for (const t of P.toggles) if (m.controls.some((x) => x.key === t)) await selectedState(report, page, P, c, t, 'aria-pressed');
    await keyboard(report, page, P, c, m);
    const hasBar = c.page !== 'index';
    await panel(report, page, P, c, 'search', hasBar);
    if (c.page === 'single') await panel(report, page, P, c, 'contents', true);
    if (c.forced) await forcedColours(report, page, P, c);
    if (c.page === 'chapter' && !c.forced && c.mode === 'fine' && !c.theme && !c.lang) {
      const mk = await page.evaluate(M.markerOf, P.version);
      report.check(`reader.external-marker/${P.name}/${caseTag(c)}`, mk && mk.content !== 'none' && mk.w > 8 && mk.mask !== 'none', JSON.stringify(mk));
      const search = await page.evaluate((s) => document.querySelector(s).getAttribute('title'), P.search.button);
      report.check(`reader.search-tooltip-names-chord/${P.name}`, /Ctrl\+F|Cmd\+F/.test(search || ''), `title ${search}`);
    }
    if (c.page === 'first' && !c.theme && !c.lang && !c.forced) {
      const disabled = await page.evaluate(() => [...document.querySelectorAll('a.disabled')].map((a) => a.getAttribute('aria-disabled') + (a.hasAttribute('href') ? ' with href' : '')));
      report.check(`reader.disabled-paging-announced/${P.name}`, disabled.length === 1 && disabled.every((v) => v === 'true'), `first page: ${disabled.length} disabled end(s), aria-disabled ${disabled.join(',')}`);
    }
  } finally { await ctx.close(); }
}

// the reader's own page zoom (Ctrl+wheel, 50 - 300 percent) scales the chrome with the page: a known open point, reported;
// what must hold at any zoom is that a panel still hangs under the bar, which uses layout pixels for both
export async function zoomAdvisory(report, browser, P, urls, c) {
  const { ctx, page } = await openCase(browser, P, urls[c.page]('en') + '?z=' + c.zoom, c);
  try {
    const box = await page.locator('#dht-font-dec').boundingBox();
    report.advisory(`reader.page-zoom-scales-targets/${P.name}`, `at the reader's own page zoom ${c.zoom} percent the A- target measures ${f1(box.width)} x ${f1(box.height)} screen px: the custom zoom scales the chrome with the page and is not changed here (docs/UI_PROFILE.md, Reader chrome)`);
  } finally { await ctx.close(); }
  for (const z of [50, 200]) for (const which of ['search', 'contents']) {
    const k = { ...c, page: which === 'contents' ? 'single' : 'chapter', zoom: z };
    const o = await openCase(browser, P, urls[k.page]('en') + '?z=' + z, k);
    try {
      await panel(report, o.page, P, k, which, true);
    } finally { await o.ctx.close(); }
  }
}

// Negative cases (BUTTONS 5: geometry checks must detect stretching, clipping, wrong targets, gaps): each one breaks the
// page on purpose with an injected rule and expects the matching check to catch it, so a green run cannot be a blind one.
const BREAKS = [
  { name: 'rows-not-equalized', css: '.dht-navbar, .dht-group { align-items: center !important; } .dht-ico { height: 44px !important; }', rules: ['row-height-equal'] },
  { name: 'pill-shaped-buttons', css: '.dht-btn, .dht-navbar a { border-radius: 999px !important; }', rules: ['radius'] },
  { name: 'stretched-buttons', css: '.dht-group { flex: 1 1 100% !important; } .dht-btn { flex: 1 1 auto !important; }', rules: ['content-width'] },
  { name: 'group-pinned-to-the-far-end', css: '.dht-navbar .nav-turn { margin-inline-start: auto !important; }', rules: ['gap-8', 'flow-from-start'] },
  { name: 'targets-too-small', css: '.dht-btn, .dht-navbar select { min-height: 24px !important; min-width: 24px !important; height: 24px !important; }', rules: ['target'] },
  { name: 'gap-collapsed', css: '.dht-navbar, .dht-group { gap: 2px !important; }', rules: ['gap-8', 'row-gap-8'] },
  { name: 'caption-clipped', css: '.dht-btn { white-space: nowrap !important; overflow: hidden !important; max-width: 20px !important; }', rules: ['caption-not-clipped'] },
  { name: 'controls-overlap', css: '.dht-group { gap: 0 !important; } .dht-btn { margin-inline-end: -20px !important; }', rules: ['no-overlap'] },
];
async function negatives(report, browser, P, urls) {
  const c = { page: 'chapter', mode: 'fine' };
  for (const b of BREAKS) {
    const { ctx, page } = await openCase(browser, P, urls.chapter('en'), c);
    try {
      await page.addStyleTag({ content: b.css });
      const m = await page.evaluate(M.collect, P);
      const rows = geometryRules(m, { coarse: false, scale: 1 }).filter((r) => b.rules.includes(r.id));
      report.check(`reader.negative/${b.name}`, rows.some((r) => !r.ok), `the check ${b.rules.join(' or ')} ${rows.some((r) => !r.ok) ? 'caught it: ' + rows.find((r) => !r.ok).detail.slice(0, 80) : 'did NOT catch it'}`);
    } finally { await ctx.close(); }
  }
  // a panel left at a fixed offset under a bar that wraps is covered by it
  {
    const n = { page: 'chapter', mode: 'narrow' };
    const { ctx, page } = await openCase(browser, P, urls.chapter('en'), n);
    try {
      await page.addStyleTag({ content: '.dht-navbar .dht-search-panel { position: fixed !important; inset-block-start: 3.5em !important; }' });
      await page.click(P.search.button);
      const box = await page.evaluate(M.panelBox, [P.search.panel, P.bar]);
      report.check('reader.negative/panel-at-a-fixed-offset', box.top < box.barBottom - 0.5, `panel top ${f1(box.top)} against bar bottom ${f1(box.barBottom)}: the panel-not-covered check would fail`);
    } finally { await ctx.close(); }
  }
  // text that is too pale for its face
  {
    const { ctx, page } = await openCase(browser, P, urls.chapter('en'), c);
    try {
      await page.addStyleTag({ content: '.dht-btn { color: #bbb !important; }' });
      const x = await page.evaluate(M.contrastOf, [P.controls, 3]);
      report.check('reader.negative/pale-text', x && x.text < 4.5, `contrast ${x && f1(x.text)}:1 would fail the 4.5 check`);
    } finally { await ctx.close(); }
  }
  // no focus ring
  {
    const { ctx, page } = await openCase(browser, P, urls.chapter('en'), c);
    try {
      await page.addStyleTag({ content: ':focus-visible { outline: none !important; }' });
      await page.keyboard.press('Tab');
      const f = await page.evaluate(M.focusInfo);
      report.check('reader.negative/no-focus-ring', f && (f.style === 'none' || f.width < 2), `outline ${f && f.style} ${f && f.width}px would fail the focus check`);
    } finally { await ctx.close(); }
  }
}

export async function readerChecks(report, { build = true, onlyGo = false, shotsDir = null } = {}) {
  SHOTS.dir = shotsDir;
  if (build) buildCli();
  const urls = await buildFixtures();
  const { chromium } = await loadPlaywright();
  const browser = await launchBrowser(chromium);
  try {
    for (const c of matrix()) {
      if (c.zoom) { await zoomAdvisory(report, browser, GO, urls, c); continue; }
      const pageFile = urls[c.page](c.lang || 'en');
      if (!fs.existsSync(pageFile)) { report.check(`reader.fixture/${caseTag(c)}`, false, 'missing ' + pageFile); continue; }
      await runCase(browser, report, GO, urls, c);
    }
    await negatives(report, browser, GO, urls);
  } finally { await browser.close(); }
  if (!onlyGo) {
    const ext = await import('./reader-ext.mjs');
    await ext.extensionChecks(report);
  }
}

// The extension viewer's notices, offers and export card against BUTTONS 2.0 and UI-SIZING 1.1 (ticket 119, slice X1,
// second part): the export card (.export-dialog), the remote-images notice (.remote-notice), the resume offer
// (.resume-notice), the generic notices (.notice) and the status bar's Cancel (#status-stop). Each is opened the way a
// reader opens it - a real comic, book, page or broken PDF served over loopback into the real viewer - and measured in a
// headless Chromium: the order of the answers, how many are primary, that the faces are neutral, the targets and shape,
// the row, the glyph before the caption, contrast, the type taken from the chrome tokens (never the reader's own size),
// and - for the card - the default focus, Tab order, Escape and the focus return.
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';
import { pathToFileURL } from 'node:url';
import { REPO, readerSource } from './lib.mjs';
import { approx } from './extension-rules.mjs';

const { DEFAULT_OPTIONS } = await import(pathToFileURL(path.join(REPO, 'extension', 'src', 'defaults.js')).href);
const f1 = (n) => (Math.round(n * 10) / 10).toString();

// ---- fixtures ---------------------------------------------------------------------------------------------------
function crc32(buf) { let crc = 0xffffffff; for (const b of buf) { let c = (crc ^ b) & 0xff; for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1; crc = (crc >>> 8) ^ c; } return (crc ^ 0xffffffff) >>> 0; }
function chunk(type, data) { const len = Buffer.alloc(4); len.writeUInt32BE(data.length); const td = Buffer.concat([Buffer.from(type), data]); const crc = Buffer.alloc(4); crc.writeUInt32BE(crc32(td)); return Buffer.concat([len, td, crc]); }
function tinyPng(shade) {
  const w = 24, h = 32, raw = Buffer.alloc((w * 3 + 1) * h);
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) { const o = y * (w * 3 + 1) + 1 + x * 3; raw[o] = (shade + x * 4) % 256; raw[o + 1] = (shade + y * 4) % 256; raw[o + 2] = 140; }
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(w, 0); ihdr.writeUInt32BE(h, 4); ihdr[8] = 8; ihdr[9] = 2;
  return Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk('IHDR', ihdr), chunk('IDAT', zlib.deflateSync(raw)), chunk('IEND', Buffer.alloc(0))]);
}

async function writeFixtures(dir) {
  fs.mkdirSync(dir, { recursive: true });
  const { zipSync } = await import(pathToFileURL(path.join(REPO, 'extension', 'node_modules', 'fflate', 'esm', 'browser.js')).href);
  const comic = (n) => {
    const files = {};
    const base = new Uint8Array(tinyPng(60));
    for (let i = 1; i <= n; i++) files[`page${String(i).padStart(4, '0')}.png`] = base;
    return Buffer.from(zipSync(files, { level: 0 }));
  };
  // enough pages that the viewer has not inflated them all when Export is pressed: the card offers the complete export;
  // past the preparation limit (1000 pages still to go) it offers only the partial one
  fs.writeFileSync(path.join(dir, 'c60.cbz'), comic(60));
  fs.writeFileSync(path.join(dir, 'c1105.cbz'), comic(1105));
  const para = (c, n) => `Paragraph ${n} of chapter ${c}: the quick brown fox jumps over the lazy dog while the reader scrolls through a sufficiently long page of text.`;
  const chapters = [1, 2, 3, 4, 5, 6].map((c) => `## Chapter ${c}\n\n` + [1, 2, 3, 4, 5, 6, 7, 8].map((n) => para(c, n)).join('\n\n')).join('\n\n');
  fs.writeFileSync(path.join(dir, 'long.md'), `# The Long Walk\n\n${chapters}\n`);
  fs.writeFileSync(path.join(dir, 'remote.md'), `# A page with pictures\n\nThe first picture: ![one](https://example.invalid/one.png)\n\nThe second: ![two](https://example.invalid/two.png)\n\n${para(1, 1)}\n`);
  fs.writeFileSync(path.join(dir, 'bad.pdf'), '%PDF-1.4\nthis is not a pdf at all\n');
}

// ---- measuring ---------------------------------------------------------------------------------------------------
const msgOf = (messagesOf, lang) => { const m = messagesOf(lang); return (key, ...args) => (m[key] || '').replace(/\{(\d)\}/g, (x, i) => (args[i - 1] === undefined ? x : String(args[i - 1]))); };

const surfaceBoxes = (page, root) => page.evaluate((sel) => {
  const U = window.__uic, host = document.querySelector(sel);
  if (!host) return null;
  const probe = document.createElement('i'); probe.style.background = 'var(--accent)'; host.append(probe);
  const accent = getComputedStyle(probe).backgroundColor; probe.remove();
  const cs0 = getComputedStyle(host), r0 = U.rect(host);
  const buttons = [...host.querySelectorAll('button')].filter((b) => U.visible(b)).map((b) => {
    const r = U.rect(b), cs = getComputedStyle(b), svg = b.querySelector('svg');
    const clone = b.cloneNode(true);
    clone.style.cssText = 'position:absolute;visibility:hidden;width:max-content;max-width:none;min-width:0;flex:none;left:0;top:0';
    b.parentNode.appendChild(clone);
    const natural = clone.getBoundingClientRect().width;
    clone.remove();
    const face = U.bgOf(b), ink = U.over(U.fgOf(b), face);
    return {
      text: b.textContent.trim(), primary: b.classList.contains('primary'), r, natural, radius: parseFloat(cs.borderTopLeftRadius), fs: parseFloat(cs.fontSize),
      bg: cs.backgroundColor, bw: parseFloat(cs.borderTopWidth), border: cs.borderTopColor, color: cs.color,
      svgFirst: !!svg && (b.firstElementChild === svg || b.firstElementChild.contains(svg)), glyph: svg ? U.rect(svg).w : null,
      clipped: b.scrollWidth > b.clientWidth + 1 || b.scrollHeight > b.clientHeight + 1,
      text4: U.contrast(ink, face), glyphC: svg ? U.contrast(U.over(U.fgOf(svg, 'color'), face), face) : null,
    };
  });
  const lay = document.getElementById('layout');
  return { accent, layoutTop: lay ? U.rect(lay).y : null, fs: parseFloat(cs0.fontSize), root: r0, buttons, iw: window.innerWidth, ih: window.innerHeight, scrollW: document.documentElement.scrollWidth, active: document.activeElement && (document.activeElement.closest(sel) ? document.activeElement.textContent.trim() : (document.activeElement === document.body ? '<body>' : '<elsewhere>')), dir: getComputedStyle(host).direction };
}, root);

const rgbOf = (s) => (String(s).match(/rgba?\(([^)]+)\)/) || [])[1];
const transparent = (s) => /rgba\([^)]*,\s*0\)|transparent/.test(s);

// one surface in one configuration: the rules that hold in every state of the card
async function checkSurface(report, page, tag, spec, T, c) {
  const m = await surfaceBoxes(page, spec.root);
  if (!m) { report.check(`notice.present/${tag}`, false, `${spec.root} not found`); return null; }
  const B = m.buttons, want = spec.labels;
  const texts = B.map((b) => b.text);
  const same = texts.length === want.length && want.every((w, i) => (w instanceof RegExp ? w.test(texts[i]) : texts[i] === w));
  report.check(`notice.order/${tag}`, same, `answers read ${texts.join(' | ')}; expected ${want.map(String).join(' | ')}`);
  const prim = B.map((b, i) => (b.primary ? i : -1)).filter((i) => i >= 0);
  report.check(`notice.primary-count/${tag}`, JSON.stringify(prim) === JSON.stringify(spec.primary), `primary at ${JSON.stringify(prim)}, expected ${JSON.stringify(spec.primary)} (at most one accent face per surface)`);
  const loud = B.filter((b, i) => !b.primary && !transparent(b.bg));
  // forced colours replace the accent with the system's selection pair, so there the primary only has to have a face at all
  const accentFaces = B.filter((b) => b.primary && (c.forced ? !transparent(b.bg) : rgbOf(b.bg) === rgbOf(m.accent)));
  report.check(`notice.faces-neutral/${tag}`, loud.length === 0 && accentFaces.length === prim.length, loud.length ? `non-primary with a face: ${loud.map((b) => `${b.text} ${b.bg}`).join('; ')}` : `${B.length - prim.length} answers on the transparent neutral face, ${prim.length} on the accent`);
  if (spec.glyphs) report.check(`notice.glyph-before-caption/${tag}`, B.every((b, i) => b.svgFirst === spec.glyphs[i]), `glyph first: ${B.map((b) => b.svgFirst).join(',')}, expected ${spec.glyphs.join(',')}`);
  report.check(`notice.target-height/${tag}`, B.every((b) => b.r.h >= T.tgt - 1), `heights ${B.map((b) => f1(b.r.h)).join('/')} px, floor ${f1(T.tgt)}`);
  report.check(`notice.radius/${tag}`, B.every((b) => approx(b.radius, T.radius, 0.6)), `radius ${B.map((b) => f1(b.radius)).join('/')}, baseline ${f1(T.radius)}`);
  report.check(`notice.content-width/${tag}`, B.every((b) => b.r.w <= Math.max(b.natural, T.tgt) + 2), `widths ${B.map((b) => f1(b.r.w)).join('/')} vs content ${B.map((b) => f1(b.natural)).join('/')} (never stretched)`);
  report.check(`notice.no-clip/${tag}`, B.every((b) => !b.clipped), 'every caption whole');
  const lines = [];
  for (const b of B) { const l = lines.find((x) => Math.abs(x[0].r.y - b.r.y) < 2); if (l) l.push(b); else lines.push([b]); }
  for (const l of lines) {
    const hs = l.map((b) => b.r.h), sorted = [...l].sort((a, b) => a.r.x - b.r.x);
    report.check(`notice.row-height/${tag}`, Math.max(...hs) - Math.min(...hs) < 1.5, `heights ${hs.map(f1).join(', ')} in one line`);
    for (let i = 1; i < sorted.length; i++) report.check(`notice.gap/${tag}`, approx(sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w), T.gap, 1.5), `gap ${f1(sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w))}, token ${f1(T.gap)}`);
  }
  report.check(`notice.type-from-the-chrome-tokens/${tag}`, approx(m.fs, T.fs, 0.3) && B.every((b) => approx(b.fs, T.fs, 0.3)), `surface ${f1(m.fs)} px, buttons ${B.map((b) => f1(b.fs)).join('/')} px, token ${f1(T.fs)} px`);
  report.check(`notice.contrast/${tag}`, B.every((b) => b.text4 >= 4.5 && (b.glyphC === null || b.glyphC >= 3)), B.map((b) => `${b.text.slice(0, 12)} ${f1(b.text4)}${b.glyphC ? '/' + f1(b.glyphC) : ''}`).join('; '));
  report.check(`notice.no-horizontal-scroll/${tag}`, m.scrollW <= m.iw + 1, `scrollWidth ${m.scrollW} of ${m.iw} px`);
  if (spec.offer) report.check(`notice.offer-above-the-document/${tag}`, m.layoutTop !== null && m.root.y + m.root.h <= m.layoutTop + 1 && m.root.w >= Math.min(42 * T.fs, m.iw) * 0.8, `the offer ends at y ${f1(m.root.y + m.root.h)}, the contents and the text start at y ${f1(m.layoutTop)}; it is ${f1(m.root.w)} px wide (a bar across the page, not a column squeezed beside the text)`);
  if (spec.fixed) report.check(`notice.inside-the-viewport/${tag}`, m.root.x >= -1 && m.root.x + m.root.w <= m.iw + 1 && m.root.y >= -1, `card ${f1(m.root.x)}..${f1(m.root.x + m.root.w)} of ${m.iw} px, top ${f1(m.root.y)}`);
  if (c.contrastMore) report.check(`notice.contrast-more-borders/${tag}`, B.filter((b) => !b.primary).every((b) => b.border === b.color), 'prefers-contrast: more draws the borders in the ink');
  if (c.forced) report.check(`notice.forced-colours-rank-survives/${tag}`, !prim.length || B.filter((b) => !b.primary).every((b) => b.bg !== B[prim[0]].bg), `primary face ${B[prim[0]] && B[prim[0]].bg} against neutral ${B.filter((b) => !b.primary).map((b) => b.bg).join('/')}`);
  return m;
}

// the reader's own text size must never reach the chrome: scale it three steps up and read the type again
async function readerSizeLeak(report, page, tag, root, T) {
  const read = () => page.evaluate((sel) => { const h = document.querySelector(sel); const b = h && h.querySelector('button'); return h ? { host: parseFloat(getComputedStyle(h).fontSize), btn: b ? parseFloat(getComputedStyle(b).fontSize) : null, reader: parseFloat(getComputedStyle(document.body).fontSize) } : null; }, root);
  const before = await read();
  for (let i = 0; i < 3; i++) await page.click('#btn-font-inc');
  await page.evaluate(() => document.documentElement.style.setProperty('--reader-size', '60px'));
  const after = await read();
  await page.evaluate(() => document.documentElement.style.removeProperty('--reader-size'));
  report.check(`notice.no-leak-of-the-reader-size/${tag}`, before && after && after.reader > before.reader + 10 && approx(before.host, T.fs, 0.3) && approx(after.host, before.host, 0.1) && (after.btn === null || approx(after.btn, before.btn, 0.1)), `reader text ${f1(before && before.reader)} -> ${f1(after && after.reader)} px; the surface stayed ${f1(before && before.host)} -> ${f1(after && after.host)} px (token ${f1(T.fs)})`);
}

// ---- opening a surface the way a reader meets it -----------------------------------------------------------------------
export async function noticeChecks(browser, base, mainReport, { openPage, messagesOf, fixtures, shotsDir }) {
  await writeFixtures(fixtures);
  const stored = (c, extra = {}) => ({ options: { ...DEFAULT_OPTIONS, ocrImages: false }, uiLang: c.lang && c.lang !== 'en' ? c.lang : '', viewerPrefs: c.theme ? { theme: c.theme } : {}, ...extra });
  const view = async (c, file, extra) => {
    const o = await openPage(browser, base, {
      name: 'viewer', url: '/src/viewer.html' + (file ? '?file=' + encodeURIComponent(`${base}/__fixtures__/${file}`) : ''), stored: stored(c, extra),
      lang: c.lang || 'en', scheme: c.scheme, forced: c.forced, contrastMore: c.contrastMore, autoTouch: c.autoTouch, w: c.w || 1100, h: c.h || 800, settle: 300,
      breakCss: c.breakCss,
    });
    if (c.scale !== undefined) await o.page.evaluate((v) => document.documentElement.style.setProperty('--ui', String(v)), c.scale);
    return o;
  };
  const tokens = (c) => { const s = Math.min(2, Math.max(0.875, c.scale ?? 1)); return { s, tgt: (c.autoTouch ? 48 : 32) * s, radius: 4 * s, gap: 8 * s, fs: 14 * s }; };
  const tagOf = (kind, c) => [kind, c.lang || 'en', c.autoTouch ? 'coarse' : 'fine', 'x' + (c.scale ?? 1), c.theme || null, c.forced ? 'forced' : null, c.contrastMore ? 'contrast-more' : null, c.w || 1100].filter(Boolean).join('/');
  const shot = async (page, name, c) => { if (shotsDir && c.shot) await page.screenshot({ path: path.join(shotsDir, `notice-${name}-${c.shot}.png`) }); };

  // report defaults to the real one; a negative control passes a scratch report and a break to apply
  const run = async (kind, c, report = mainReport) => {
    const M = msgOf(messagesOf, c.lang || 'en'), T = tokens(c), tag = tagOf(kind, c);
    const cap = (re) => new RegExp('^' + re.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace(/\\\{(\d)\\\}/g, '\\d+') + '$');
    if (kind === 'export-offer' || kind === 'export-partial') {
      const o = await view(c, kind === 'export-offer' ? 'c60.cbz' : 'c1105.cbz');
      try {
        const { page } = o;
        await page.waitForSelector('#btn-save-html:not(.hidden)', { timeout: 20000 });
        await page.waitForTimeout(400);
        await page.click('#btn-save-html');
        await page.waitForSelector('.export-dialog', { timeout: 8000 });
        await page.evaluate(readerSource);
        if (c.afterJs) await page.evaluate(c.afterJs);
        const labels = kind === 'export-offer'
          ? [M('vExportCancel'), cap(M('vExportPartialBtn', '{1}')), cap(M('vExportPrepareBtn', '{1}'))]
          : [M('vExportCancel'), cap(M('vExportPartialBtn', '{1}'))];
        const m = await checkSurface(report, page, tag, { root: '.export-dialog', labels, primary: [labels.length - 1], glyphs: labels.map(() => true), fixed: true }, T, c);
        if (m) report.check(`notice.default-focus-is-the-safe-answer/${tag}`, m.active === M('vExportCancel'), `focus is on "${m.active}" when the card opens (Cancel is "${M('vExportCancel')}")`);
        if (c.deep) {
          const rtl = c.lang === 'ar';
          // the row wraps whole buttons, so reading order is line by line and, within a line, from the reading start
          const seq = m.buttons.map((b, i) => ({ i, y: b.r.y, x: b.r.x })).sort((a, b) => (Math.abs(a.y - b.y) > 2 ? a.y - b.y : (rtl ? b.x - a.x : a.x - b.x))).map((q) => q.i);
          report.check(`notice.reading-order-follows-the-direction/${tag}`, seq.every((v, i) => v === i), `read line by line from the reading start (${rtl ? 'right' : 'left'}) the answers come in order ${seq.map((v) => v + 1).join(', ')}; Cancel first at x ${f1(m.buttons[0].r.x)}`);
          const seen = [];
          for (let i = 0; i < labels.length - 1; i++) { await page.keyboard.press('Tab'); seen.push(await page.evaluate(() => document.activeElement.textContent.trim())); }
          report.check(`notice.tab-order-is-reading-order/${tag}`, seen.every((t, i) => (labels[i + 1] instanceof RegExp ? labels[i + 1].test(t) : labels[i + 1] === t)), `Tab from Cancel reaches ${seen.join(' | ')}`);
          await readerSizeLeak(report, page, tag, '.export-dialog', T);
          await shot(page, kind, c);
          // Escape is Cancel: nothing is decided, the card goes, focus returns to Export; nothing is downloaded
          let downloads = 0; page.on('download', () => { downloads++; });
          await page.focus('.export-dialog button');
          await page.keyboard.press('Escape');
          await page.waitForTimeout(150);
          const esc = await page.evaluate(() => ({ card: !!document.querySelector('.export-dialog'), focus: document.activeElement.id }));
          report.check(`notice.escape-closes-and-returns-focus/${tag}`, !esc.card && esc.focus === 'btn-save-html' && downloads === 0, `card open ${esc.card}, focus on #${esc.focus}, downloads ${downloads}`);
          await page.click('#btn-save-html');
          await page.waitForSelector('.export-dialog');
          await page.keyboard.press('Enter');
          await page.waitForTimeout(200);
          const ent = await page.evaluate(() => ({ card: !!document.querySelector('.export-dialog'), focus: document.activeElement.id }));
          report.check(`notice.enter-takes-the-safe-answer/${tag}`, !ent.card && ent.focus === 'btn-save-html' && downloads === 0, `Enter on the default focus closed the card (open ${ent.card}), focus on #${ent.focus}, downloads ${downloads}`);
        }
        report.check(`console.clean/${tag}`, o.page.errors.length === 0, o.page.errors.slice(0, 3).join(' | ') || 'no script or console error');
      } finally { await o.close(); }
      return;
    }
    if (kind === 'resume') {
      const first = await view(c, 'long.md');
      let positions;
      try {
        const { page } = first;
        await page.waitForSelector('#content section, #content p', { timeout: 20000 });
        await page.evaluate(() => { window.scrollTo(0, 1400); });
        await page.waitForTimeout(300);
        await page.evaluate(() => window.dispatchEvent(new Event('pagehide')));
        await page.waitForTimeout(300);
        positions = await page.evaluate(() => window.__stored.readingPositions);
      } finally { await first.close(); }
      if (!positions || !Object.keys(positions).length) { report.check(`notice.fixture-saved-a-position/${tag}`, false, 'the viewer saved no reading position, so there is no offer to measure'); return; }
      const o = await view(c, 'long.md', { readingPositions: positions });
      try {
        const { page } = o;
        await page.waitForSelector('.resume-notice', { timeout: 15000 });
        await page.evaluate(readerSource);
        if (c.afterJs) await page.evaluate(c.afterJs);
        const labels = [M('vResumeReset'), M('vResumeContinue')];
        const m = await checkSurface(report, page, tag, { root: '.resume-notice', labels, primary: [1], glyphs: [true, true], offer: true }, T, c);
        if (m) report.check(`notice.inline-offer-takes-no-focus/${tag}`, m.active === '<body>' || m.active === '<elsewhere>', `focus is on ${m.active}: the offer is a status in the flow, not a modal question`);
        if (c.deep) { await readerSizeLeak(report, page, tag, '.resume-notice', T); await shot(page, kind, c); }
      } finally { await o.close(); }
      return;
    }
    if (kind === 'remote') {
      const o = await view(c, 'remote.md');
      try {
        const { page } = o;
        await page.waitForSelector('.remote-notice', { timeout: 15000 });
        await page.evaluate(readerSource);
        if (c.afterJs) await page.evaluate(c.afterJs);
        const labels = [M('vRemoteLoad'), M('vRemoteAlways')];
        await checkSurface(report, page, tag, { root: '.remote-notice', labels, primary: [], glyphs: [true, true], offer: true }, T, c);
        if (c.deep) { await readerSizeLeak(report, page, tag, '.remote-notice', T); await shot(page, kind, c); }
      } finally { await o.close(); }
      return;
    }
    if (kind === 'empty' || kind === 'error') {
      const o = await view(c, kind === 'error' ? 'bad.pdf' : null);
      try {
        const { page } = o;
        await page.waitForSelector('.notice button', { timeout: 15000 });
        await page.evaluate(readerSource);
        if (c.afterJs) await page.evaluate(c.afterJs);
        const labels = kind === 'empty' ? [M('vOpenDocTitle')] : [M('vBtnOpenOriginalPdf')];
        const m = await checkSurface(report, page, tag, { root: '.notice', labels: labels.length ? labels : [/./], primary: [0], glyphs: kind === 'empty' ? [true] : [false] }, T, c);
        if (c.deep && m) { await readerSizeLeak(report, page, tag, '.notice', T); await shot(page, kind, c); }
      } finally { await o.close(); }
      return;
    }
    if (kind === 'status') {
      const o = await view(c, null);
      try {
        const { page } = o;
        // while no preparation runs the control is hidden - the shared button rule must not outrank [hidden]
        const idle = await page.evaluate(() => getComputedStyle(document.getElementById('status-stop')).display);
        report.check(`notice.status-cancel-is-hidden-until-needed/${tag}`, idle === 'none', `#status-stop computes display ${idle} while idle`);
        await page.evaluate(() => { document.getElementById('status').classList.remove('done'); document.getElementById('status-stop').hidden = false; });
        await page.evaluate(readerSource);
        if (c.afterJs) await page.evaluate(c.afterJs);
        const m = await checkSurface(report, page, tag, { root: '#status', labels: [M('vExportStop')], primary: [], glyphs: [true] }, T, c);
        if (c.deep && m) { await readerSizeLeak(report, page, tag, '#status', T); await shot(page, kind, c); }
      } finally { await o.close(); }
    }
  };

  const BASE = { lang: 'en' };
  const matrix = [
    { ...BASE, deep: true, shot: 'light', kinds: ['export-offer', 'export-partial', 'resume', 'remote', 'empty', 'error', 'status'] },
    { ...BASE, autoTouch: true, w: 390, h: 844, kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'], deep: true },
    { ...BASE, scale: 0.875, kinds: ['export-offer', 'resume', 'remote', 'status'] },
    { ...BASE, scale: 0.875, autoTouch: true, w: 390, h: 844, kinds: ['export-offer', 'remote', 'status'] },
    { ...BASE, scale: 1.5, kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'], deep: true },
    { ...BASE, scale: 2, kinds: ['export-offer', 'remote', 'status'] },
    { lang: 'ru', kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'], deep: true },
    { lang: 'de', w: 700, kinds: ['export-offer', 'resume', 'remote', 'status'] },
    { lang: 'hi', kinds: ['export-offer', 'resume', 'remote'] },
    { lang: 'ar', deep: true, shot: 'rtl-ar', kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'] },
    { ...BASE, w: 420, h: 800, shot: 'narrow', kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'] },
    { ...BASE, theme: 'dark', scheme: 'dark', deep: true, shot: 'dark', kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'] },
    { ...BASE, theme: 'night', scheme: 'dark', kinds: ['export-offer', 'remote'] },
    { ...BASE, theme: 'sepia', kinds: ['export-offer', 'remote'] },
    { ...BASE, forced: true, shot: 'forced', kinds: ['export-offer', 'resume', 'remote', 'empty', 'status'] },
    { ...BASE, contrastMore: true, kinds: ['export-offer', 'resume', 'remote', 'status'] },
  ];
  for (const c of matrix) {
    for (const kind of c.kinds) {
      try { await run(kind, c); } catch (e) { mainReport.check(`notice.scenario-ran/${tagOf(kind, c)}`, false, String(e.message).split('\n')[0]); }
    }
  }

  // Negative controls: each breaks a surface on purpose and runs the same rules, which must then fail. A rule list that
  // cannot fail proves nothing, so a break the rules do not catch is a failed check here.
  const { Report } = await import('./lib.mjs');
  const NEG = [
    { name: 'two accent answers on the export card', kind: 'export-offer', c: { afterJs: () => document.querySelectorAll('.export-dialog button').forEach((b) => b.classList.add('primary')) }, expect: 'notice.primary-count' },
    { name: 'Cancel last on the export card', kind: 'export-offer', c: { afterJs: () => { const row = document.querySelector('.export-actions'); row.append(row.firstElementChild); } }, expect: 'notice.order' },
    { name: 'the safe answer not focused', kind: 'export-offer', c: { afterJs: () => document.querySelector('.export-dialog button:last-child').focus() }, expect: 'notice.default-focus-is-the-safe-answer' },
    { name: 'a stretched answer', kind: 'export-offer', c: { breakCss: '.export-actions > button { flex: 1 1 100% !important; }' }, expect: 'notice.content-width' },
    { name: 'a tinted neutral answer', kind: 'remote', c: { breakCss: '.remote-notice button { background: #e8b4b4 !important; }' }, expect: 'notice.faces-neutral' },
    { name: 'the reader size reaching a notice', kind: 'remote', c: { deep: true, breakCss: '.remote-notice { font-size: var(--reader-size) !important; }' }, expect: 'notice.type-from-the-chrome-tokens' },
    { name: 'a pill corner', kind: 'resume', c: { breakCss: '.resume-notice button { border-radius: 20px !important; }' }, expect: 'notice.radius' },
    { name: 'a target below the floor', kind: 'status', c: { breakCss: '#status-stop { min-height: 14px !important; height: 14px !important; padding: 0 8px !important; }' }, expect: 'notice.target-height' },
    { name: 'an answer with no glyph', kind: 'resume', c: { afterJs: () => document.querySelectorAll('.resume-notice svg').forEach((s) => s.remove()) }, expect: 'notice.glyph-before-caption' },
    { name: 'unequal heights in one row', kind: 'resume', c: { breakCss: '.resume-notice .notice-actions { align-items: flex-start !important; } .resume-notice .notice-actions button:first-child { height: 80px !important; }' }, expect: 'notice.row-height' },
    { name: 'a wrong gap', kind: 'resume', c: { breakCss: '.notice-actions { gap: 30px !important; }' }, expect: 'notice.gap' },
    { name: 'an offer squeezed in beside the text', kind: 'resume', c: { afterJs: () => document.getElementById('content').before(document.querySelector('.resume-notice')) }, expect: 'notice.offer-above-the-document' },
    { name: 'a Cancel that is always visible', kind: 'status', c: { breakCss: '#status-stop[hidden] { display: inline-flex !important; }' }, expect: 'notice.status-cancel-is-hidden-until-needed' },
    { name: 'a pale caption', kind: 'remote', c: { breakCss: '.remote-notice button { color: #d4d4d4 !important; }' }, expect: 'notice.contrast' },
  ];
  for (const n of NEG) {
    const scratch = new Report('negative');
    try { await run(n.kind, { lang: 'en', deep: true, kinds: [n.kind], ...n.c }, scratch); } catch (e) { scratch.check('notice.scenario-ran/negative', false, String(e.message).split('\n')[0]); }
    const hit = scratch.failed.find((r) => r.id.startsWith(n.expect));
    mainReport.check(`notice.negative/${n.name}`, !!hit, hit ? `caught by ${hit.id.split('/')[0]}: ${hit.detail.slice(0, 100)}` : `the break went UNDETECTED (other failures: ${scratch.failed.slice(0, 2).map((r) => r.id.split('/')[0]).join(', ') || 'none'})`);
  }
}
// capture.mjs - the lab's screenshot scenes: the GUI launcher's real window, captured with its text geometry.
//
// Ticket E0121 phase C (OCR-OVERLAY evidence OC-1, owner decision 2026-10-30): a `screenshot` category whose scenes
// come from the product's own interface and whose truth is the rendered DOM - the box of every word the window
// draws, read from the page that drew it. That source is independent of OCR, which is the point: the engine is never
// graded against its own reading.
//
// It runs the real doc-html-ui.exe under a scratch profile with the help generator's machinery (tools/helpgen/app.mjs
// and the setups of tools/helpgen/shots.mjs), so a scene is a state the window really reaches. It writes the PNGs
// and one index.json into <root>/screens/; `go run ./tools/ocrlab screens` turns that index into manifest entries
// and annotations. Nothing here touches the manifest or the annotations itself.
//
//   node tools/ocrlab/screens/capture.mjs --exe <doc-html-ui.exe> [--root test_doc/ocrlab] [--date YYYY-MM-DD]
//
// Run from the repository root in PowerShell. Needs the Playwright Chromium of tools/sitecheck.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { demoBooks, demoProfile, launchBrowser, loadPlaywright, REPO, startApp } from '../../helpgen/app.mjs';
import { BOOKS, profileSettings, SETUPS } from '../../helpgen/shots.mjs';

const argv = process.argv.slice(2);
const opt = (n, d) => (argv.includes(n) ? argv[argv.indexOf(n) + 1] : d);

const WIDTH = 1160;
const MIN_HEIGHT = 760;
const MAX_HEIGHT = 2400;
const DEMO_EXE = 'C:\\Tools\\doc-html-translate.exe';
const DEMO_DIR = 'C:\\Books\\';

// The scenes: three authored locales of four window states, alternating the theme so both looks are in the set.
const STATES = [
  { setup: 'converter-ready', theme: 'light' },
  { setup: 'converter-states', theme: 'dark' },
  { setup: 'page-ocr', theme: 'light' },
  { setup: 'page-about', theme: 'dark' },
];
const LOCALES = [
  { locale: 'en', script: 'Latn' },
  { locale: 'ru', script: 'Cyrl' },
  { locale: 'uk', script: 'Cyrl' },
];

const exeArg = opt('--exe', null);
if (!exeArg) {
  console.error('capture: --exe <path to doc-html-ui.exe> is required (tools/helpgen/generate.mjs builds one under temp/helpgen/bin)');
  process.exit(2);
}
const exe = path.resolve(exeArg);
const root = path.resolve(opt('--root', path.join(REPO, 'test_doc', 'ocrlab')));
const outDir = path.join(root, 'screens');
const pad = (n) => String(n).padStart(2, '0');
const now = new Date();
const date = opt('--date', `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`);
const scratch = path.join(REPO, 'temp', 'ocrlab-screens');

// The window prints the machine's paths into the command line and the file field; the demo ones replace them, so a
// scene carries no path of the disk it was made on.
async function maskPaths(page, ctx) {
  await page.evaluate(({ books, dir, profile, demoProfileDir, demoExe }) => {
    const fix = (s) => {
      for (const b of books) s = s.split(b).join(dir + b.split(/[\\/]/).pop());
      return s.split(profile).join(demoProfileDir).replace(/&\s+\S*doc-html-translate\.exe/i, '& ' + demoExe);
    };
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    for (let n = walker.nextNode(); n; n = walker.nextNode()) {
      if (/[A-Za-z]:\\/.test(n.nodeValue)) n.nodeValue = fix(n.nodeValue);
    }
    for (const i of document.querySelectorAll('input')) if (/[A-Za-z]:\\/.test(i.value)) i.value = fix(i.value);
  }, { books: ctx.books, dir: DEMO_DIR, profile: path.join(ctx.root, 'profile'), demoProfileDir: 'C:\\Users\\demo\\AppData\\Local', demoExe: DEMO_EXE });
}

async function leakCheck(page, ctx) {
  const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const needles = [REPO, ctx.root, os.homedir()].filter(Boolean).map(esc)
    .concat([os.userInfo().username, process.env.COMPUTERNAME].filter((s) => s && s.length > 2).map((s) => '\\b' + esc(s) + '\\b'));
  return page.evaluate((ns) => {
    const text = document.body.innerText + ' ' + [...document.querySelectorAll('input')].map((i) => (i.type === 'password' || i.type === 'hidden' ? '' : i.value)).join(' ');
    for (const n of ns) {
      const m = new RegExp(n, 'i').exec(text);
      if (m) return m[0];
    }
    return null;
  }, needles);
}

// textGeometry runs in the page: every word the window draws, with its box, grouped into lines per block box.
// A word counts only when the page really paints it at that place - visible, inside every clipping ancestor, and the
// topmost element at its centre (so text under a dialog or an overlay is not truth). Form fields contribute the text
// they show (value, or placeholder when empty; the selected option of a select), measured with the field's own font.
function textGeometry() {
  const W = innerWidth, H = innerHeight;
  const SKIP = new Set(['SCRIPT', 'STYLE', 'NOSCRIPT', 'TEMPLATE', 'TITLE', 'OPTION', 'TEXTAREA', 'SELECT']);
  const clipOf = (el) => {
    let r = { x0: 0, y0: 0, x1: W, y1: H };
    for (let a = el.parentElement; a && a !== document.documentElement; a = a.parentElement) {
      const cs = getComputedStyle(a);
      if (cs.overflowX !== 'visible' || cs.overflowY !== 'visible') {
        const b = a.getBoundingClientRect();
        r = { x0: Math.max(r.x0, b.left), y0: Math.max(r.y0, b.top), x1: Math.min(r.x1, b.right), y1: Math.min(r.y1, b.bottom) };
      }
    }
    return r;
  };
  const blockOf = (el) => {
    for (let a = el; a; a = a.parentElement) {
      const d = getComputedStyle(a).display;
      if (d !== 'inline' && d !== 'contents') return a;
    }
    return document.body;
  };
  const painted = (el, cx, cy, clip) => {
    if (cx < clip.x0 || cx > clip.x1 || cy < clip.y0 || cy > clip.y1) return false;
    const hit = document.elementFromPoint(cx, cy);
    return !!hit && (hit === el || el.contains(hit) || hit.contains(el));
  };
  const transform = (t, tt) => (tt === 'uppercase' ? t.toUpperCase() : tt === 'lowercase' ? t.toLowerCase() : t);
  const blocks = new Map();
  const put = (blk, w) => {
    if (!blocks.has(blk)) blocks.set(blk, { tag: blk.tagName.toLowerCase(), heading: /^H[1-6]$/.test(blk.tagName) || blk.getAttribute('role') === 'heading', words: [] });
    blocks.get(blk).words.push(w);
  };

  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let n = walker.nextNode(); n; n = walker.nextNode()) {
    const el = n.parentElement;
    if (!el || SKIP.has(el.tagName) || !n.nodeValue.trim()) continue;
    if (el.closest('select, textarea, option')) continue;
    if (!el.checkVisibility({ opacityProperty: true, visibilityProperty: true })) continue;
    const cs = getComputedStyle(el);
    const fs = parseFloat(cs.fontSize);
    const clip = clipOf(el);
    const re = /\S+/g;
    const range = (a, b) => {
      const rg = document.createRange();
      rg.setStart(n, a);
      rg.setEnd(n, b);
      return rg;
    };
    for (let m = re.exec(n.nodeValue); m; m = re.exec(n.nodeValue)) {
      // A word the layout broke over two lines (a long path in the command line) has no one box: it is cut into
      // its pieces, character by character, one piece per line it is drawn on.
      const pieces = [];
      if (range(m.index, m.index + m[0].length).getClientRects().length > 1) {
        for (let i = 0; i < m[0].length; i++) {
          const r = range(m.index + i, m.index + i + 1).getBoundingClientRect();
          const last = pieces[pieces.length - 1];
          if (last && Math.abs(last.y0 - r.top) < 0.5 * r.height) {
            last.t += m[0][i];
            last.x1 = Math.max(last.x1, r.right);
            last.y1 = Math.max(last.y1, r.bottom);
          } else {
            pieces.push({ t: m[0][i], x0: r.left, y0: r.top, x1: r.right, y1: r.bottom });
          }
        }
      } else {
        const r = range(m.index, m.index + m[0].length).getBoundingClientRect();
        pieces.push({ t: m[0], x0: r.left, y0: r.top, x1: r.right, y1: r.bottom });
      }
      for (const p of pieces) {
        if (p.x1 - p.x0 < 1 || p.y1 - p.y0 < 1) continue;
        if (!painted(el, (p.x0 + p.x1) / 2, (p.y0 + p.y1) / 2, clip)) continue;
        put(blockOf(el), { ...p, t: transform(p.t, cs.textTransform), fs });
      }
    }
  }

  const ctx2d = document.createElement('canvas').getContext('2d');
  for (const el of document.querySelectorAll('input, select')) {
    if (el.tagName === 'INPUT' && !['text', 'search', 'number', 'email', 'url', 'tel', ''].includes(el.type)) continue;
    if (!el.checkVisibility({ opacityProperty: true, visibilityProperty: true })) continue;
    let text = el.tagName === 'SELECT' ? (el.selectedOptions[0] ? el.selectedOptions[0].textContent : '') : (el.value || el.placeholder || '');
    text = text.replace(/\s+/g, ' ').trim();
    if (!text) continue;
    const cs = getComputedStyle(el);
    const b = el.getBoundingClientRect();
    const left = b.left + parseFloat(cs.borderLeftWidth) + parseFloat(cs.paddingLeft);
    const right = b.right - parseFloat(cs.borderRightWidth) - parseFloat(cs.paddingRight) - (el.tagName === 'SELECT' ? 20 : 0);
    ctx2d.font = `${cs.fontStyle} ${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
    const fs = parseFloat(cs.fontSize);
    const clip = clipOf(el);
    let x = left;
    const words = [];
    const space = ctx2d.measureText(' ').width;
    for (const t of text.split(' ')) {
      const mt = ctx2d.measureText(t);
      const asc = mt.fontBoundingBoxAscent, desc = mt.fontBoundingBoxDescent;
      const cy = b.top + b.height / 2;
      const top = cy - (asc + desc) / 2;
      const w = { t, x0: x, y0: top, x1: Math.min(x + mt.width, right), y1: top + asc + desc, fs };
      x += mt.width + space;
      if (w.x1 - w.x0 < 1) break; // the rest of the text is scrolled out of the field
      if (!painted(el, (w.x0 + w.x1) / 2, (w.y0 + w.y1) / 2, clip)) continue;
      words.push(w);
    }
    for (const w of words) put(el, w);
  }

  const out = [];
  for (const g of blocks.values()) {
    const ws = g.words.slice().sort((a, b) => (a.y0 + a.y1) - (b.y0 + b.y1) || a.x0 - b.x0);
    const lines = [];
    for (const w of ws) {
      const cy = (w.y0 + w.y1) / 2;
      const line = lines.find((l) => Math.abs(l.cy - cy) < 0.5 * Math.min(l.h, w.y1 - w.y0));
      if (line) line.words.push(w);
      else lines.push({ cy, h: w.y1 - w.y0, words: [w] });
    }
    lines.sort((a, b) => a.cy - b.cy);
    const outLines = lines.map((l) => {
      l.words.sort((a, b) => a.x0 - b.x0);
      return {
        text: l.words.map((w) => w.t).join(' '),
        box: [Math.floor(Math.min(...l.words.map((w) => w.x0))), Math.floor(Math.min(...l.words.map((w) => w.y0))),
          Math.ceil(Math.max(...l.words.map((w) => w.x1))), Math.ceil(Math.max(...l.words.map((w) => w.y1)))],
        fontPx: Math.max(...l.words.map((w) => w.fs)),
      };
    });
    out.push({ kind: g.heading ? 'title' : outLines.length > 1 ? 'paragraph' : 'label', lines: outLines });
  }
  // reading order: top to bottom, then left to right, by each group's first line
  out.sort((a, b) => a.lines[0].box[1] - b.lines[0].box[1] || a.lines[0].box[0] - b.lines[0].box[0]);
  return out;
}

async function captureScene(browser, base, ctx, { setup, theme }, { locale, script }) {
  const page = await browser.newPage({ viewport: { width: WIDTH, height: MIN_HEIGHT }, deviceScaleFactor: 1, reducedMotion: 'reduce', colorScheme: theme });
  try {
    demoProfile(ctx.root, profileSettings({ lang: locale, theme, books: ctx.books }));
    await page.goto(`${base}/?lang=${locale}`, { waitUntil: 'load' });
    await page.waitForFunction(() => document.getElementById('aboutVer') && document.getElementById('aboutVer').textContent.trim() !== '');
    await page.waitForTimeout(700);
    await page.evaluate((th) => { applyTheme(th); }, theme);
    await SETUPS[setup](page, { ...ctx, lang: locale });
    await maskPaths(page, ctx);
    const leak = await leakCheck(page, ctx);
    if (leak) throw new Error(`${setup}/${locale}: the window shows "${leak}" - a scene carries demo content only`);
    // The window grows to the bottom of the page's content, as a capture of the whole surface (helpgen's window kind).
    const h = await page.evaluate(() => {
      const pg = document.querySelector('.page:not([hidden])');
      const body = (pg && (pg.querySelector('.convert-main') || pg)) || document.body;
      return Math.ceil(body.getBoundingClientRect().bottom + 24);
    });
    const height = Math.max(MIN_HEIGHT, Math.min(h, MAX_HEIGHT));
    await page.setViewportSize({ width: WIDTH, height });
    await page.evaluate(() => document.fonts.ready);
    await page.waitForTimeout(400);
    const groups = await page.evaluate(textGeometry);
    const version = await page.evaluate(() => document.getElementById('aboutVer').textContent.trim());
    const png = await page.screenshot({ type: 'png' });
    const id = `screen-gui-${setup}-${locale}-${theme}`;
    fs.writeFileSync(path.join(outDir, id + '.png'), png);
    return { id, file: `screens/${id}.png`, locale, script, theme, setup, width: WIDTH, height, version, groups };
  } finally {
    await page.close();
  }
}

fs.mkdirSync(outDir, { recursive: true });
const books = demoBooks(scratch, BOOKS).map((p) => { fs.writeFileSync(p, 'demo document\n'.repeat(300)); return p; });
const ctx = { root: scratch, books };
const app = await startApp(exe, path.join(scratch, 'profile'));
const { chromium } = await loadPlaywright();
const browser = await launchBrowser(chromium);
const scenes = [];
try {
  for (const l of LOCALES) {
    for (const s of STATES) {
      const sc = await captureScene(browser, app.base, ctx, s, l);
      const lines = sc.groups.reduce((n, g) => n + g.lines.length, 0);
      console.log(`captured ${sc.id} ${sc.width}x${sc.height}: ${sc.groups.length} groups, ${lines} lines`);
      scenes.push(sc);
    }
  }
} finally {
  await browser.close();
  app.stop();
}
const index = { producer: 'tools/ocrlab/screens/capture.mjs', generatedOn: date, scenes };
fs.writeFileSync(path.join(outDir, 'index.json'), JSON.stringify(index, null, 2) + '\n');
console.log(`capture: ${scenes.length} scenes and index.json written to ${outDir}`);

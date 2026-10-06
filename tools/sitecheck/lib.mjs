// Shared helpers of the site checklist run (ticket 102): the browser launch, a static server that serves the
// working tree the way GitHub Pages serves main, git hashing, and the pixel helpers that run in a helper page.
import { chromium } from 'playwright-core';
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';

export const REPO = path.resolve(import.meta.dirname, '..', '..');
export const LIVE_BASE = 'https://serzhyale.github.io/doc-html-translate/';
export const CATALOG = 'P:/Contracts';

export const sha256 = (buf) => crypto.createHash('sha256').update(buf).digest('hex');

export function gitShow(ref, file) {
  try {
    return execFileSync('git', ['show', `${ref}:${file}`], { cwd: REPO, maxBuffer: 64 * 1024 * 1024, stdio: ['ignore', 'pipe', 'ignore'] });
  } catch { return null; }
}
export const git = (...a) => execFileSync('git', a, { cwd: REPO, encoding: 'utf8' }).trim();

export async function launch() {
  const exe = process.env.SITECHECK_CHROME ||
    path.join(process.env.LOCALAPPDATA, 'ms-playwright', 'chromium-1223', 'chrome-win64', 'chrome.exe');
  return chromium.launch({ executablePath: exe, headless: true });
}

const MIME = { '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.js': 'text/javascript', '.json': 'application/json',
  '.png': 'image/png', '.ico': 'image/x-icon', '.xml': 'application/xml', '.txt': 'text/plain', '.svg': 'image/svg+xml', '.webp': 'image/webp' };

// Serves REPO under /doc-html-translate/ like Pages does: a folder answers its index.html, a missing path answers
// the root 404.html with status 404 (the host default text when there is none).
export function serveTree() {
  const prefix = '/doc-html-translate/';
  const server = http.createServer((req, res) => {
    const u = new URL(req.url, 'http://x');
    let rel = decodeURIComponent(u.pathname);
    const notFound = () => {
      const f = path.join(REPO, '404.html');
      if (fs.existsSync(f)) { res.writeHead(404, { 'content-type': MIME['.html'] }); res.end(fs.readFileSync(f)); }
      else { res.writeHead(404, { 'content-type': 'text/plain' }); res.end('There isn\'t a GitHub Pages site here.'); }
    };
    if (!rel.startsWith(prefix)) return notFound();
    rel = rel.slice(prefix.length);
    let f = path.join(REPO, rel);
    if (!f.startsWith(REPO)) return notFound();
    if (fs.existsSync(f) && fs.statSync(f).isDirectory()) f = path.join(f, 'index.html');
    if (!fs.existsSync(f) || !fs.statSync(f).isFile()) return notFound();
    res.writeHead(200, { 'content-type': MIME[path.extname(f)] || 'application/octet-stream' });
    res.end(fs.readFileSync(f));
  });
  return new Promise((resolve) => server.listen(0, '127.0.0.1', () => resolve({ server, base: `http://127.0.0.1:${server.address().port}/doc-html-translate/` })));
}

// The helper page decodes PNGs and does the pixel arithmetic the checks need.
export async function helperPage(browser) {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await page.setContent('<canvas id=c></canvas>');
  await page.evaluate(() => {
    window.load = (b64) => new Promise((res, rej) => { const i = new Image(); i.onload = () => res(i); i.onerror = rej; i.src = 'data:image/png;base64,' + b64; });
    window.pix = async (b64) => {
      const i = await window.load(b64); const c = document.getElementById('c'); c.width = i.width; c.height = i.height;
      const x = c.getContext('2d', { willReadFrequently: true }); x.drawImage(i, 0, 0); return { x, w: i.width, h: i.height };
    };
    window.lum = (r, g, b) => { const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
    window.ratio = (a, b) => { const l1 = window.lum(...a), l2 = window.lum(...b); return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05); };
  });
  return page;
}

export async function diffCount(helper, a, b) {
  return helper.evaluate(async ([a, b]) => {
    const A = await window.pix(a); const da = A.x.getImageData(0, 0, A.w, A.h).data;
    const B = await window.pix(b); const db = B.x.getImageData(0, 0, B.w, B.h).data;
    if (A.w !== B.w || A.h !== B.h) return -1;
    let n = 0; for (let i = 0; i < da.length; i += 4) if (Math.abs(da[i] - db[i]) + Math.abs(da[i + 1] - db[i + 1]) + Math.abs(da[i + 2] - db[i + 2]) > 24) n++;
    return n;
  }, [a.toString('base64'), b.toString('base64')]);
}

// items: [{id,x,y,w,h,fg:[r,g,b,a],threshold}] in screenshot pixels. The worst pixel of the box decides.
export async function contrastOf(helper, png, items) {
  return helper.evaluate(async ([b64, items]) => {
    const P = await window.pix(b64); const out = [];
    for (const it of items) {
      const x0 = Math.max(0, Math.floor(it.x)), y0 = Math.max(0, Math.floor(it.y));
      const w = Math.min(P.w - x0, Math.ceil(it.w)), h = Math.min(P.h - y0, Math.ceil(it.h));
      if (w < 1 || h < 1) { out.push({ id: it.id, skip: 'outside' }); continue; }
      const d = P.x.getImageData(x0, y0, w, h).data; const rs = [];
      const step = Math.max(1, Math.floor((w * h) / 400));
      for (let p = 0; p < w * h; p += step) {
        const o = p * 4, bg = [d[o], d[o + 1], d[o + 2]];
        const a = it.fg[3]; const fg = [0, 1, 2].map((k) => it.fg[k] * a + bg[k] * (1 - a));
        rs.push(window.ratio(fg, bg));
      }
      rs.sort((a, b) => a - b);
      out.push({ id: it.id, min: rs[0], p10: rs[Math.floor(rs.length * 0.1)], med: rs[Math.floor(rs.length / 2)] });
    }
    return out;
  }, [png.toString('base64'), items]);
}

export const ts = () => new Date().toISOString().replace(/[-:]/g, '').slice(0, 13).replace('T', '-');

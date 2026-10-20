// The in-page OCR bar (page-agent.js with page-overlay.css) injected into a page the extension does not own (ticket 119,
// slice X1): BUTTONS 2.0 and UI-SIZING 1.1 on the bar's own buttons, and the isolation the bar owes the page - nothing the
// page's own rules say changes the bar, nothing the bar's rules say changes the page. A stub of chrome.* lets the real
// script run in a plain page from the loopback server; the status message the broker would send is delivered directly.
import fs from 'node:fs';
import path from 'node:path';
import { REPO, readerSource } from './lib.mjs';
import { approx } from './extension-rules.mjs';

const f1 = (n) => (Math.round(n * 10) / 10).toString();
const SRC = path.join(REPO, 'extension', 'src');

async function injectBar(browser, base, openPage, c) {
  // A negative control's break is applied after the bar exists, so it wins the cascade it is meant to disturb.
  const { breakCss, breakJs, ...rest } = c;
  const o = await openPage(browser, base, { ...rest, url: '/__page__' + (c.hostile ? '?hostile=1' : ''), noWait: true, settle: 20 });
  const { page } = o;
  await page.addStyleTag({ content: fs.readFileSync(path.join(SRC, 'ocr-overlay.css'), 'utf8') });
  await page.addStyleTag({ content: fs.readFileSync(path.join(SRC, 'page-overlay.css'), 'utf8') });
  await page.addScriptTag({ content: fs.readFileSync(path.join(SRC, 'page-agent.js'), 'utf8') });
  await page.evaluate(() => window.__listeners.forEach((f) => f({ dht: 'page-ocr', t: 'status', running: true, done: 1, total: 3 }, {}, () => {})));
  await page.waitForFunction(() => { const b = document.getElementById('dht-ocr-bar'); return b && !b.hasAttribute('data-dht-pending'); }, null, { timeout: 8000 });
  if (breakCss) await page.addStyleTag({ content: breakCss });
  if (breakJs) await page.evaluate(breakJs);
  await page.evaluate(readerSource);
  await page.waitForTimeout(100);
  return o;
}

const boxes = (page) => page.evaluate(() => {
  const U = window.__uic;
  return [...document.querySelectorAll('#dht-ocr-bar button')].filter((b) => U.visible(b)).map((b) => {
    const r = U.rect(b), cs = getComputedStyle(b);
    const clone = b.cloneNode(true);
    clone.style.cssText = 'position:absolute;visibility:hidden;width:max-content;max-width:none;min-width:0;flex:none;left:0;top:0';
    b.parentNode.appendChild(clone);
    const natural = clone.getBoundingClientRect().width;
    clone.remove();
    return { id: b.textContent.trim(), r, natural, radius: parseFloat(cs.borderTopLeftRadius), fs: parseFloat(cs.fontSize), glyph: b.querySelector('svg') ? U.rect(b.querySelector('svg')) : null, clipped: b.scrollWidth > b.clientWidth + 1, pressed: b.getAttribute('aria-pressed') };
  });
});

const SCALES = { compact: 0.875, ordinary: 1, large: 1.25, xlarge: 1.5, huge: 2 };
const T = (size, input) => ({ s: SCALES[size], tgt: { pointer: 32, touch: 48 }[input] * SCALES[size], glyph: { pointer: 16, touch: 20 }[input] * SCALES[size] });

// One bar in one configuration, measured against BUTTONS 2.0 and UI-SIZING 1.1 (also run, broken on purpose, by the negative controls).
export async function barCase(browser, base, report, c, { openPage, storedFor, shotsDir }) {
    const t = T(c.size, c.input);
    const tag = `bar/${c.size}/${c.input}/${c.text || 'ordinary'}/${c.lang || 'en'}/${c.w || 1100}${c.forced ? '/forced' : ''}`;
    const cc = { ...c, name: 'bar', w: c.w || 1100, h: c.h || 700, stored: storedFor(c) };
    const o = await injectBar(browser, base, openPage, cc);
    try {
      const { page } = o;
      const bs = await boxes(page);
      report.check(`bar.buttons-found/${tag}`, bs.length >= 3, `${bs.length} buttons: ${bs.map((b) => b.id).join(' | ')}`);
      for (const b of bs) {
        const w = `${tag} [${b.id}]`;
        report.check(`bar.target-height/${w}`, b.r.h >= t.tgt - 1, `${f1(b.r.h)} px, floor ${f1(t.tgt)}`);
        report.check(`bar.radius/${w}`, approx(b.radius, 4 * t.s, 0.6), `radius ${f1(b.radius)}, baseline ${f1(4 * t.s)}`);
        report.check(`bar.content-width/${w}`, b.r.w <= Math.max(b.natural, t.tgt) + 2, `${f1(b.r.w)} px vs content ${f1(b.natural)} px`);
        report.check(`bar.no-clip/${w}`, !b.clipped, 'caption whole');
        if (b.glyph) report.check(`bar.glyph-tier/${w}`, approx(b.glyph.w, t.glyph * (c.text === 'huge' ? 1.5 : 1), 1), `glyph ${f1(b.glyph.w)} px, tier ${f1(t.glyph)}`);
      }
      const lines = [];
      for (const b of bs) { const l = lines.find((x) => Math.abs(x[0].r.y - b.r.y) < 2); if (l) l.push(b); else lines.push([b]); }
      for (const l of lines) {
        const hs = l.map((b) => b.r.h);
        report.check(`bar.row-height/${tag}`, Math.max(...hs) - Math.min(...hs) < 1.5, `heights ${hs.map(f1).join(', ')} in one line`);
        const sorted = [...l].sort((a, b) => a.r.x - b.r.x);
        for (let i = 1; i < sorted.length; i++) report.check(`bar.gap/${tag}`, approx(sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w), 8 * t.s, 1.5), `gap ${f1(sorted[i].r.x - (sorted[i - 1].r.x + sorted[i - 1].r.w))}, token ${f1(8 * t.s)}`);
      }
      const geo = await page.evaluate(() => { const U = window.__uic, bar = document.getElementById('dht-ocr-bar'), r = U.rect(bar); return { bar: r, iw: window.innerWidth, ih: window.innerHeight, dir: getComputedStyle(bar).direction, scroll: document.documentElement.scrollWidth <= window.innerWidth + 1 }; });
      report.check(`bar.inside-viewport-no-scroll/${tag}`, geo.bar.x >= -1 && geo.bar.x + geo.bar.w <= geo.iw + 1 && geo.bar.y >= -1 && geo.bar.y + geo.bar.h <= geo.ih + 1 && geo.scroll, `bar ${f1(geo.bar.x)}..${f1(geo.bar.x + geo.bar.w)} of ${geo.iw} px`);
      const rtl = c.lang === 'ar';
      report.check(`bar.sits-at-the-reading-end/${tag}`, rtl ? geo.dir === 'rtl' && geo.bar.x < geo.iw / 2 : geo.dir === 'ltr' && geo.bar.x + geo.bar.w > geo.iw / 2, `direction ${geo.dir}, bar x ${f1(geo.bar.x)} (inset-inline-end, no physical right)`);
      // contrast of caption, glyph and border on the face they are drawn on
      const k = await page.evaluate(() => {
        const U = window.__uic;
        return [...document.querySelectorAll('#dht-ocr-bar button')].filter((b) => U.visible(b)).map((b) => {
          const face = U.bgOf(b), ink = U.over(U.fgOf(b), face), g = b.querySelector('svg path');
          const bd = U.rgba(getComputedStyle(b).borderTopColor), side = U.bgOf(b.parentElement);
          return { id: b.textContent.trim(), text: U.contrast(ink, face), glyph: g ? U.contrast(U.over(U.rgba(getComputedStyle(g).fill), face), face) : null, border: bd ? U.contrast(U.over(bd, side), side) : null };
        });
      });
      if (!c.forced) {
        report.check(`bar.contrast/${tag}`, k.every((x) => x.text >= 4.5 && (x.glyph === null || x.glyph >= 3)), k.map((x) => `${x.id} ${f1(x.text)}${x.glyph ? '/' + f1(x.glyph) : ''}`).join('; '));
        report.check(`bar.border-contrast/${tag}`, k.every((x) => x.border === null || x.border >= 3), `lowest border ${f1(Math.min(...k.map((x) => x.border ?? 99)))}:1 against the bar`);
      }
      if (c.shot && shotsDir) await page.screenshot({ path: path.join(shotsDir, `bar-${c.shot}.png`) });
      if (c.states || (c.size === 'ordinary' && c.input === 'pointer' && !c.lang && !c.forced && !c.text && !c.w)) await barStates(page, report, tag, t);
      report.check(`console.clean/${tag}`, o.page.errors.length === 0, o.page.errors.slice(0, 3).join(' | ') || 'no script or console error');
    } finally { await o.close(); }
}

export async function barChecks(browser, base, report, deps) {
  for (const c of [
    { size: 'large', input: 'pointer' }, { size: 'ordinary', input: 'pointer', shot: 'light' }, { size: 'ordinary', input: 'touch' }, { size: 'compact', input: 'pointer' }, { size: 'compact', input: 'touch' },
    { size: 'xlarge', input: 'pointer' }, { size: 'huge', input: 'touch', w: 900 }, { size: 'ordinary', input: 'pointer', text: 'huge', w: 700 },
    { size: 'ordinary', input: 'pointer', lang: 'ru' }, { size: 'ordinary', input: 'pointer', lang: 'de', w: 520 }, { size: 'ordinary', input: 'pointer', lang: 'ar', shot: 'rtl-ar' },
    { size: 'ordinary', input: 'pointer', forced: true, shot: 'forced' }, { size: 'ordinary', input: 'pointer', w: 360, h: 640 },
  ]) await barCase(browser, base, report, c, deps);
  await barIsolation(browser, base, report, deps);
}

// the two state-bearing buttons: Hide text and Show text keep one width; Select text is a stable-name toggle with a cue
async function barStates(page, report, tag, t) {
  const hide = page.locator('#dht-ocr-bar button', { hasText: 'Hide text' });
  const w0 = (await hide.boundingBox()).width;
  // a real (trusted) click: the bar ignores any click a page script could make
  await page.click('#dht-ocr-bar button:has-text("Hide text")');
  const shown = page.locator('#dht-ocr-bar button', { hasText: 'Show text' });
  const w1 = (await shown.boundingBox()).width;
  report.check(`bar.state-width-reserved/${tag}`, approx(w0, w1, 0.6), `Hide text ${f1(w0)} px, Show text ${f1(w1)} px - the neighbours do not move`);
  const sel = page.locator('#dht-ocr-bar button', { hasText: 'Select text' });
  const before = await sel.evaluate((b) => { const cs = getComputedStyle(b); return { bg: cs.backgroundColor, shadow: cs.boxShadow, border: cs.borderTopColor, pressed: b.getAttribute('aria-pressed'), text: b.textContent.trim() }; });
  await page.click('#dht-ocr-bar button:has-text("Select text")');
  const after = await sel.evaluate((b) => { const cs = getComputedStyle(b); return { bg: cs.backgroundColor, shadow: cs.boxShadow, border: cs.borderTopColor, pressed: b.getAttribute('aria-pressed'), text: b.textContent.trim() }; });
  // page.click is a trusted event, so the toggle really flips
  report.check(`bar.toggle-state-is-not-colour-alone/${tag}`, before.pressed === 'false' && after.pressed === 'true' && before.text === after.text && /inset/.test(after.shadow) && after.shadow !== before.shadow && after.bg !== before.bg, `aria-pressed ${before.pressed} -> ${after.pressed}, caption "${after.text}" unchanged, face ${before.bg} -> ${after.bg}, inset bar ${/inset/.test(after.shadow)}`);
  const ring = await page.evaluate(() => { const b = [...document.querySelectorAll('#dht-ocr-bar button')].find((x) => /Select text/.test(x.textContent)); b.focus({ focusVisible: true }); const cs = getComputedStyle(b); return { style: cs.outlineStyle, width: parseFloat(cs.outlineWidth) }; });
  await page.keyboard.press('Tab');
  await page.keyboard.press('Shift+Tab');
  const ring2 = await page.evaluate(() => { const cs = getComputedStyle(document.activeElement); return { style: cs.outlineStyle, width: parseFloat(cs.outlineWidth), id: document.activeElement.textContent.trim() }; });
  report.check(`bar.focus-ring-visible/${tag}`, ring2.style !== 'none' && ring2.width >= 2, `keyboard focus on "${ring2.id}" draws a ${ring2.width}px ${ring2.style} ring`);
}

// nothing the page says reaches the bar, and nothing the bar says reaches the page
async function barIsolation(browser, base, report, { openPage, storedFor }) {
  const c = { name: 'bar', size: 'ordinary', input: 'pointer', w: 1100, h: 700 };
  const clean = await injectBar(browser, base, openPage, { ...c, stored: storedFor(c) });
  const hostile = await injectBar(browser, base, openPage, { ...c, hostile: true, stored: storedFor(c) });
  try {
    const a = await boxes(clean.page), b = await boxes(hostile.page);
    const same = a.length === b.length && a.every((x, i) => approx(x.r.w, b[i].r.w, 0.6) && approx(x.r.h, b[i].r.h, 0.6) && approx(x.radius, b[i].radius, 0.1) && approx(x.fs, b[i].fs, 0.1));
    report.check('bar.isolated-from-hostile-page-rules', same, `${a.length} buttons: ${a.map((x, i) => `${x.id} ${f1(x.r.w)}x${f1(x.r.h)} vs ${f1(b[i].r.w)}x${f1(b[i].r.h)}`).join('; ')} - the page's div/span/button rules, its custom properties and its own --dht-tgt changed none of them`);
    // the bar's rules leak nothing into the page: the same page without the bar's styles and script looks the same
    const plain = await openPage(browser, base, { ...c, url: '/__page__', noWait: true, settle: 20, stored: storedFor(c) });
    try {
      const probe = () => plain.page.evaluate(() => ['h1', 'p', 'img', 'body'].map((s) => { const cs = getComputedStyle(document.querySelector(s)); return [s, cs.fontSize, cs.color, cs.margin, cs.padding, cs.borderTopWidth, cs.display].join('|'); }).join(' ; '));
      const before = await probe();
      await plain.page.addStyleTag({ content: fs.readFileSync(path.join(SRC, 'ocr-overlay.css'), 'utf8') });
      await plain.page.addStyleTag({ content: fs.readFileSync(path.join(SRC, 'page-overlay.css'), 'utf8') });
      await plain.page.addScriptTag({ content: fs.readFileSync(path.join(SRC, 'page-agent.js'), 'utf8') });
      await plain.page.evaluate(() => window.__listeners.forEach((f) => f({ dht: 'page-ocr', t: 'status', running: false, done: 3, total: 3 }, {}, () => {})));
      await plain.page.waitForTimeout(250);
      const after = await probe();
      report.check('bar.leaks-no-style-into-the-page', before === after, before === after ? 'h1, p, img and body computed styles identical before and after the bar was injected' : `${before} != ${after}`);
      const rootVars = await plain.page.evaluate(() => ['--dht-tgt', '--dht-ui', '--tgt', '--s', '--ui'].map((v) => getComputedStyle(document.documentElement).getPropertyValue(v)).join(''));
      report.check('bar.declares-no-token-on-the-page-root', rootVars.trim() === '', `custom properties on the page's root: "${rootVars.trim()}"`);
    } finally { await plain.close(); }
  } finally { await clean.close(); await hostile.close(); }
}

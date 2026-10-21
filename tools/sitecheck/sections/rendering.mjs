// Rendered appearance and interaction: layout per width, keyboard walk, reduced motion, contrast and touch targets.
// Each exported function is one section of the site checklist run; run.mjs decides which of them run, and in which order.
import fs from 'node:fs';
import path from 'node:path';
import { diffCount, contrastOf } from '../lib.mjs';

// L - layout at 360 / 768 / 1280 / 1920 / 2560, both themes: overflow, full width, targets
export async function layout(env) {
  const { OUT, FILES, url, add, newCtx, open } = env;
  const widths = [360, 768, 1280, 1920, 2560];
  const res = {};
  const wrapperScan = [];
  for (const w of widths) {
    const ctx = await newCtx({ viewport: { width: w, height: w <= 768 ? 800 : 1000 } });
    for (const f of FILES) {
      const page = await open(ctx, url(f));
      const m = await page.evaluate(() => {
        const q = (s) => document.querySelector(s);
        const vw = document.documentElement.clientWidth;
        const rect = (e) => e ? e.getBoundingClientRect() : null;
        // the footer's left edge is its content edge: the 1.3 kit's .container spans the viewport and pads the content by the gutter
        const contentBox = (e) => { if (!e) return null; const r = e.getBoundingClientRect(), pl = parseFloat(getComputedStyle(e).paddingLeft) || 0, pr = parseFloat(getComputedStyle(e).paddingRight) || 0; return { left: r.left + pl, width: r.width - pl - pr }; };
        const main = rect(q('main')), hdr = rect(q('header')), h1 = rect([...document.querySelectorAll('h1')].find((h) => h.checkVisibility())), ft = contentBox(q('footer .container') || q('footer'));
        const over = [];
        for (const e of document.querySelectorAll('body *')) {
          if (!e.checkVisibility()) continue; const r = e.getBoundingClientRect();
          if (r.right > vw + 1 || r.left < -1) { let clipped = false; for (let a = e.parentElement; a && a !== document.body; a = a.parentElement) { const o = getComputedStyle(a).overflowX; if (o === 'auto' || o === 'scroll' || o === 'hidden') { clipped = true; break; } } if (!clipped) over.push(e.tagName + '.' + (e.className || '').toString().slice(0, 30) + ' ' + Math.round(r.left) + '..' + Math.round(r.right)); }
        }
        const wrappers = ['html', 'body', 'main', 'header', 'footer', '.container', '.site-footer > div', 'section', '.hero'].map((s) => {
          const e = q(s); if (!e) return null; const cs = getComputedStyle(e); const r = e.getBoundingClientRect();
          return { s, maxWidth: cs.maxWidth, ml: cs.marginLeft, mr: cs.marginRight, w: Math.round(r.width), left: Math.round(r.left) };
        }).filter(Boolean);
        return { vw, sw: document.documentElement.scrollWidth, main: main && [Math.round(main.left), Math.round(main.width)], hdr: hdr && [Math.round(hdr.left), Math.round(hdr.width)], h1: h1 && Math.round(h1.left), ft: ft && [Math.round(ft.left), Math.round(ft.width)], over: over.slice(0, 6), nOver: over.length, wrappers };
      });
      (res[w] ||= {})[f] = m;
      if (w === 360 && ['index.html', 'docs.html'].includes(f)) await page.screenshot({ path: path.join(OUT, 'shots', `${f.replace(/\W/g, '_')}-360.png`), fullPage: true });
      if (w === 2560 && ['index.html', 'docs.html', 'privacy.html'].includes(f)) await page.screenshot({ path: path.join(OUT, 'shots', `${f.replace(/\W/g, '_')}-2560.png`) });
      await page.close();
    }
    await ctx.close();
  }
  fs.writeFileSync(path.join(OUT, 'L-layout.json'), JSON.stringify(res, null, 1));
  // overflow
  for (const w of [360, 768, 1280]) {
    const bad = FILES.filter((f) => res[w][f].sw > res[w][f].vw + 1 || res[w][f].nOver);
    add(`L${w}`, 'all', bad.length ? 'fail' : 'pass', `${w} px: horizontal overflow or an element past the viewport edge on ${bad.length} of ${FILES.length} pages ${bad.map((f) => f + ':' + res[w][f].sw + '/' + res[w][f].vw + ' ' + res[w][f].over.slice(0, 2).join('|')).join('; ')}`, { bad });
  }
  // I0: full width
  const gr = {};
  for (const w of [1920, 2560]) for (const f of FILES) {
    const m = res[w][f]; const need = w - 80;
    const sidebar = false;
    const okMain = m.main && m.main[1] >= need;
    const okEdge = m.hdr && m.h1 !== null && m.ft && Math.abs(m.hdr[0] - m.h1) <= 2 && Math.abs(m.h1 - m.ft[0]) <= 2;
    (gr[f] ||= {})[w] = { mainW: m.main && m.main[1], need, okMain, edges: [m.hdr && m.hdr[0], m.h1, m.ft && m.ft[0]], okEdge };
  }
  const failMain = FILES.filter((f) => !gr[f][1920].okMain || !gr[f][2560].okMain);
  const failEdge = FILES.filter((f) => !gr[f][1920].okEdge || !gr[f][2560].okEdge);
  add('I0a', 'all', failMain.length ? 'fail' : 'pass', `main box >= viewport - 80 px at 1920 and 2560 on ${FILES.length - failMain.length} of ${FILES.length} pages; failing: ${failMain.map((f) => f + ' ' + gr[f][1920].mainW + '/' + gr[f][2560].mainW).join(', ') || 'none'}`, { gr });
  add('I0b', 'all', failEdge.length ? 'fail' : 'pass', `left edge of header, h1 and footer is one x position at 1920 and 2560 on ${FILES.length - failEdge.length} of ${FILES.length} pages; failing: ${failEdge.map((f) => f + ' ' + JSON.stringify(gr[f][1920].edges) + '@1920').join(', ') || 'none'}`, {});
  // wrapper max-width / auto margins in computed style at 2560, and in the stylesheet source
  const wr = [];
  for (const f of FILES) for (const x of res[2560][f].wrappers) {
    // max-width:100% is the 1.3 kit's own --wide, the whole container, not a cap
    if ((x.maxWidth !== 'none' && x.maxWidth !== '100%') || (x.ml === x.mr && parseFloat(x.ml) > 0 && x.w < 2560 - 2 * parseFloat(x.ml) + 2 && x.w < 2400)) wr.push(f + ' ' + x.s + ' max-width ' + x.maxWidth + ' margins ' + x.ml + '/' + x.mr + ' width ' + x.w);
  }
  const ctx = await newCtx({ viewport: { width: 2560, height: 1000 } });
  const page = await open(ctx, url('docs.html'));
  const rules = await page.evaluate(() => {
    const out = [];
    for (const sh of document.styleSheets) { let rs; try { rs = sh.cssRules; } catch { continue; } if ((sh.href || '').includes('fonts.googleapis')) continue;
      const walk = (list) => { for (const r of list) { if (r.cssRules && r.type !== CSSRule.STYLE_RULE) { walk(r.cssRules); continue; } const t = r.cssText || ''; if (r.style && (r.style.maxWidth || /margin(-inline)?\s*:\s*[^;]*\bauto\b/.test(t))) out.push((sh.href || 'inline').split('/').pop() + ' :: ' + r.selectorText + ' { max-width:' + r.style.maxWidth + (/margin[^;]*auto/.test(t) ? ' margin:auto' : '') + ' }'); } };
      walk(rs); }
    return out;
  });
  const wrUniq = {}; for (const x of wr) { const k = x.replace(/^\S+ /, ''); (wrUniq[k] ||= []).push(x.split(' ')[0]); }
  add('I0c', 'all', wr.length ? 'fail' : 'pass', `wrapper max-width or centred margin in computed style at 2560 px: ${wr.length ? Object.entries(wrUniq).map(([k, v]) => `${k} on ${v.length} pages`).join('; ') : 'none'}`, { wrapperPages: wrUniq, stylesheetRules: rules });
  add('I0d', 'info', 'info', `stylesheet rules that set max-width or an auto margin (review for wrappers of ordinary text): ${rules.length}`, { rules });
  await ctx.close();
}

// K - keyboard: tab walk, focus ring, skip link, per theme
export async function keyboard(env) {
  const { OUT, url, add, themeCtx, open, helper } = env;
  for (const theme of ['light', 'dark']) {
    for (const f of ['index.html', 'docs.html', 'privacy.html']) {
      const ctx = await themeCtx(theme);
      const page = await open(ctx, url(f));
      await page.mouse.move(0, 0);
      const stops = []; let cycled = false;
      for (let i = 0; i < 120; i++) {
        await page.keyboard.press('Tab');
        await page.waitForTimeout(260);
        // the kit scrolls smoothly: measure only once the scroll position stops moving
        for (let k = 0, prev = -1; k < 20; k++) { const y = await page.evaluate(() => scrollY); if (y === prev) break; prev = y; await page.waitForTimeout(120); }
        const info = await page.evaluate(() => {
          const e = document.activeElement; if (!e || e === document.body || e === document.documentElement) return null;
          const r = e.getBoundingClientRect(), cs = getComputedStyle(e);
          // a link that wraps over two lines has a bounding box whose centre lands on the text beside it: probe the
          // middle of each line fragment and call the stop obscured only when none of them shows the element (ticket 106)
          const frags = [...e.getClientRects()].filter((q) => q.width > 0 && q.height > 0);
          const hit = (q) => { const x = Math.min(innerWidth - 1, Math.max(0, q.left + q.width / 2)), y = Math.min(innerHeight - 1, Math.max(0, q.top + q.height / 2)); const t = document.elementFromPoint(x, y); return { t, ok: t === e || e.contains(t) }; };
          const hits = (frags.length ? frags : [r]).map(hit);
          const top = (hits.find((h) => h.ok) || hits[0]).t;
          return { tag: e.tagName.toLowerCase(), id: e.id, cls: (e.className || '').toString().slice(0, 24), name: (e.getAttribute('aria-label') || e.textContent || e.value || '').trim().replace(/\s+/g, ' ').slice(0, 36), href: e.getAttribute('href'), rect: { x: r.left, y: r.top, w: r.width, h: r.height }, outline: `${cs.outlineStyle} ${cs.outlineWidth} ${cs.outlineColor} off ${cs.outlineOffset}`, shadow: cs.boxShadow === 'none' ? '' : cs.boxShadow.slice(0, 40), fv: e.matches(':focus-visible'), obscured: !(top === e || e.contains(top)), inMain: !!e.closest('main'), inHeader: !!e.closest('header'), y: r.top + scrollY };
        });
        if (!info) { cycled = true; break; }
        // pixel evidence of a visible focus indicator
        const pad = 8; const vw = page.viewportSize();
        const clip = { x: Math.max(0, Math.floor(info.rect.x - pad)), y: Math.max(0, Math.floor(info.rect.y - pad)), width: 0, height: 0 };
        clip.width = Math.max(1, Math.min(vw.width - clip.x, Math.ceil(info.rect.w + 2 * pad))); clip.height = Math.max(1, Math.min(vw.height - clip.y, Math.ceil(info.rect.h + 2 * pad)));
        let changed = null;
        if (info.rect.y + info.rect.h > 0 && info.rect.y < vw.height) {
          const A = await page.screenshot({ clip });
          const h = await page.evaluateHandle(() => document.activeElement);
          await h.evaluate((e) => e.blur()); await page.waitForTimeout(260);
          const B = await page.screenshot({ clip });
          await h.evaluate((e) => e.focus()); await page.waitForTimeout(260);
          changed = await diffCount(helper, A, B);
          info.ringPx = changed;
        }
        stops.push(info);
      }
      const nFocusable = await page.evaluate(() => [...document.querySelectorAll('a[href], button, summary, input, select, textarea, [tabindex]')].filter((e) => e.checkVisibility() && !e.disabled && e.tabIndex >= 0).length);
      const noRing = stops.filter((s) => s.ringPx === 0 || (s.ringPx === null && !s.fv)).map((s) => s.tag + ':' + s.name);
      const obscured = stops.filter((s) => s.obscured).map((s) => s.tag + ':' + s.name);
      const jumps = []; for (let i = 1; i < stops.length; i++) if (stops[i].y < stops[i - 1].y - 6) jumps.push(`${stops[i - 1].name} -> ${stops[i].name}`);
      const first = stops[0] || {};
      const skipFirst = first.tag === 'a' && /^#/.test(first.href || '') && /skip|к содерж|до змісту/i.test(first.name);
      let skipWorks = null;
      if (skipFirst) {
        await page.reload({ waitUntil: 'load' }); await page.mouse.move(0, 0);
        await page.keyboard.press('Tab'); await page.waitForTimeout(150);
        await page.keyboard.press('Enter'); await page.waitForTimeout(150);
        await page.keyboard.press('Tab'); await page.waitForTimeout(150);
        skipWorks = await page.evaluate(() => { const e = document.activeElement; return { inMain: !!(e && e.closest('main')) && e.tagName !== 'MAIN', tag: e && e.tagName, hash: location.hash }; });
      }
      await page.screenshot({ path: path.join(OUT, 'shots', `focus-${f.replace(/\W/g, '_')}-${theme}.png`) });
      add(`K-${f}-${theme}`, f, !skipFirst || noRing.length || obscured.length ? 'fail' : 'pass', `Tab walk: ${stops.length} stops of ${nFocusable} focusable, wrapped to the browser ${cycled}; first stop "${first.name}" skip link ${skipFirst}, skip leads into main ${skipWorks && skipWorks.inMain}; stops with no visible ring: ${noRing.join(', ') || 'none'}; obscured: ${obscured.join(', ') || 'none'}; upward jumps: ${jumps.join(', ') || 'none'}`, { stops: stops.map((s) => [s.tag, s.name, s.outline, s.shadow, s.ringPx, s.obscured]), skipWorks });
      await ctx.close();
    }
  }
}

// M - reduced motion
export async function reducedMotion(env) {
  const { url, add, newCtx, open } = env;
  for (const f of ['index.html', 'docs.html', 'privacy.html', 'extension.html', 'install-trust.html']) {
    const probe = async (reduced) => {
      const ctx = await newCtx({ reducedMotion: reduced ? 'reduce' : 'no-preference' });
      const page = await open(ctx, url(f));
      await page.waitForTimeout(500);
      const m = await page.evaluate(() => {
        const sec = (v) => Math.max(0, ...String(v).split(',').map((x) => parseFloat(x) * (x.includes('ms') ? 0.001 : 1) || 0));
        const mov = [];
        for (const e of document.querySelectorAll('*')) for (const ps of [null, '::before', '::after']) {
          const cs = getComputedStyle(e, ps); const t = sec(cs.transitionDuration), a = sec(cs.animationDuration), an = cs.animationName;
          if ((t > 0.011 && cs.transitionProperty !== 'none') || (a > 0.011 && an !== 'none')) mov.push(e.tagName + '.' + (e.className || '').toString().slice(0, 20) + (ps || '') + ' t=' + t + ' a=' + a + ' ' + an);
        }
        return { mq: matchMedia('(prefers-reduced-motion: reduce)').matches, moving: mov.length, sample: mov.slice(0, 5), running: document.getAnimations().filter((a) => a.playState === 'running').length, scrollBehavior: getComputedStyle(document.documentElement).scrollBehavior, canvas: document.querySelectorAll('canvas').length };
      });
      // the back-to-top control scrolls with behaviour:'smooth'; does it still glide?
      let glide = null;
      const hasTop = await page.evaluate(() => !!document.getElementById('toTop'));
      if (hasTop) {
        await page.evaluate(() => window.scrollTo(0, 1500)); await page.waitForTimeout(200);
        await page.evaluate(() => { document.getElementById('toTop').click(); });
        await page.waitForTimeout(40);
        const y1 = await page.evaluate(() => scrollY); await page.waitForTimeout(900); const y2 = await page.evaluate(() => scrollY);
        glide = { afterClick40ms: Math.round(y1), settled: Math.round(y2), animated: y1 > 5 && y1 < 1495 };
      }
      await ctx.close();
      return { ...m, glide };
    };
    const n = await probe(false), r = await probe(true);
    const bad = r.mq !== true || r.moving > 0 || r.running > 0 || (r.glide && r.glide.animated) || r.canvas;
    add(`M-${f}`, f, bad ? 'fail' : 'pass', `reduced motion emulated ${r.mq}: elements with a transition/animation > 11 ms ${r.moving} (no preference: ${n.moving}); running animations ${r.running}; back-to-top glide ${JSON.stringify(r.glide)}; scroll-behavior ${r.scrollBehavior}; canvas ${r.canvas}; samples ${r.sample.join(' | ')}`, { normal: n, reduced: r });
  }
}

// T - contrast per theme, touch targets
export async function contrast(env) {
  const { OUT, url, add, newCtx, themeCtx, open, helper } = env;
  const PAGES = ['index.html', 'docs.html', 'docs.ru.html', 'extension.html', 'privacy.html', 'install-trust.html', 'extension-privacy.html', 'ar/index.html'];
  const all = {};
  for (const theme of ['light', 'dark']) {
    const ctx = await themeCtx(theme);
    for (const f of PAGES) {
      const page = await open(ctx, url(f));
      const th = await page.evaluate(() => document.documentElement.getAttribute('data-theme'));
      await page.evaluate(async () => { document.querySelectorAll('details').forEach((d) => (d.open = true)); document.querySelectorAll('img').forEach((i) => (i.loading = 'eager')); await Promise.all([...document.images].map((i) => i.decode().catch(() => 0))); });
      await page.waitForTimeout(300);
      const H = await page.evaluate(() => document.documentElement.scrollHeight), vh = page.viewportSize().height;
      const meas = []; const comps = []; const seen = new Set();
      for (let y = 0; y < H; y += vh - 100) {
        // the kit sets smooth scrolling: jump, then wait until the offset stops moving so rects and pixels agree
        await page.evaluate((y) => window.scrollTo({ top: y, left: 0, behavior: 'instant' }), y);
        for (let k = 0, last = -1; k < 20; k++) { const now = await page.evaluate(() => scrollY); if (now === last) break; last = now; await page.waitForTimeout(100); }
        const items = await page.evaluate(() => {
          const cv = document.createElement('canvas'); cv.width = cv.height = 1; const cx = cv.getContext('2d', { willReadFrequently: true });
          const rgba = (c) => { cx.clearRect(0, 0, 1, 1); cx.fillStyle = '#000'; cx.fillStyle = c; cx.fillRect(0, 0, 1, 1); const d = cx.getImageData(0, 0, 1, 1).data; return [d[0], d[1], d[2], d[3] / 255]; };
          let uid = window.__uid || 0; const out = []; const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n;
          while ((n = w.nextNode())) {
            const t = n.nodeValue.trim(); if (!t) continue; const el = n.parentElement; if (!el || /^(SCRIPT|STYLE|NOSCRIPT|OPTION|TEXTAREA)$/.test(el.tagName)) continue;
            if (!el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true }) || el.closest('[aria-hidden="true"]')) continue;
            const rg = document.createRange(); rg.selectNodeContents(n); const r = rg.getBoundingClientRect();
            if (r.width < 1 || r.height < 1 || r.top < 0 || r.bottom > innerHeight || r.left < 0 || r.right > innerWidth) continue;
            // text partly under the sticky header is measured in the scroll step where it is clear of it
            const hd = document.querySelector('header'); const hp = hd && getComputedStyle(hd).position;
            if (hd && (hp === 'sticky' || hp === 'fixed') && !el.closest('header')) { const hr = hd.getBoundingClientRect(); if (r.bottom > hr.top && r.top < hr.bottom) continue; }
            if (!n.__id) n.__id = ++uid; const cs = getComputedStyle(el);
            const topEl = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
            const obscured = !(topEl && (topEl === el || el.contains(topEl) || topEl.contains(el)));
            let o = 1; for (let a = el; a; a = a.parentElement) o *= parseFloat(getComputedStyle(a).opacity);
            const fg = rgba(cs.webkitTextFillColor || cs.color); fg[3] *= o;
            const fs = parseFloat(cs.fontSize), wt = +cs.fontWeight; const large = fs >= 24 || (fs >= 18.66 && wt >= 700);
            const glyph = !/[\p{L}\p{N}]/u.test(t);
            const cat = el.closest('.site-header') ? 'chrome' : el.closest('.note') ? 'callout' : el.closest('.pill, .badge, .tag') ? 'badge' : el.closest('a') ? 'link' : el.closest('footer') ? 'footer' : el.closest('button, .btn') ? 'button' : 'body';
            out.push({ uid: n.__id, id: n.__id, text: t.slice(0, 32), tag: el.tagName.toLowerCase() + (el.className ? '.' + el.className.toString().split(' ')[0] : ''), x: r.left, y: r.top, w: r.width, h: r.height, fg, threshold: glyph || large ? 3 : 4.5, cat, obscured, over: obscured && topEl ? topEl.tagName.toLowerCase() + '.' + (topEl.className || '').toString().split(' ')[0] + (topEl.id ? '#' + topEl.id : '') : '', bgclip: cs.backgroundClip === 'text' || cs.webkitBackgroundClip === 'text' });
          }
          window.__uid = uid;
          const compSel = ['.theme-btn', '.seg button', '.btn', '.copybox .copy', '.pill', '.note', '.card', 'summary', '.langs a'];
          const comps = [];
          for (const s of compSel) for (const e of document.querySelectorAll(s)) { if (!e.checkVisibility()) continue; const r = e.getBoundingClientRect(); if (r.top < 8 || r.bottom > innerHeight - 8 || r.left < 8 || r.right > innerWidth - 8 || r.width < 6) continue; if (!e.__cid) e.__cid = ++uid; comps.push({ uid: 'c' + e.__cid, s, x: r.left, y: r.top, w: r.width, h: r.height }); }
          window.__uid = uid;
          return { out, comps };
        });
        await page.addStyleTag({ content: '*,*::before,*::after{color:transparent!important;-webkit-text-fill-color:transparent!important;text-shadow:none!important;caret-color:transparent!important}' });
        await page.waitForTimeout(120);
        const png = await page.screenshot();
        const fresh = items.out.filter((i) => !seen.has(i.uid)); fresh.forEach((i) => seen.add(i.uid));
        const rs = await contrastOf(helper, png, fresh.map((i) => ({ id: i.uid, x: i.x, y: i.y, w: i.w, h: i.h, fg: i.fg })));
        const by = Object.fromEntries(rs.map((r) => [r.id, r]));
        for (const i of fresh) meas.push({ ...i, ...by[i.uid], pass: by[i.uid] && by[i.uid].p10 !== undefined ? by[i.uid].p10 >= i.threshold : null });
        const freshC = items.comps.filter((c) => !seen.has(c.uid)); freshC.forEach((c) => seen.add(c.uid));
        // boundary: pixel just inside the edge vs pixel 4 px outside
        const cr = await helper.evaluate(async ([b64, cs]) => {
          const P = await window.pix(b64); const out = [];
          const px = (x, y) => { x = Math.round(x); y = Math.round(y); if (x < 0 || y < 0 || x >= P.w || y >= P.h) return null; return P.x.getImageData(x, y, 1, 1).data.slice(0, 3); };
          for (const c of cs) { const mx = c.x + c.w / 2; const ins = px(mx, c.y + 1), outs = px(mx, c.y - 4), ins2 = px(c.x + 1, c.y + c.h / 2), outs2 = px(c.x - 4, c.y + c.h / 2); const r = [];
            if (ins && outs) r.push(window.ratio([...ins], [...outs])); if (ins2 && outs2) r.push(window.ratio([...ins2], [...outs2])); out.push({ uid: c.uid, s: c.s, edge: r.length ? Math.max(...r) : null }); }
          return out;
        }, [png.toString('base64'), freshC]);
        comps.push(...cr);
        await page.evaluate(() => { const s = [...document.querySelectorAll('style')].pop(); if (s && /transparent!important/.test(s.textContent)) s.remove(); });
      }
      (all[theme] ||= {})[f] = { th, meas, comps };
      await page.evaluate(() => window.scrollTo(0, 0));
      await page.close();
    }
    await ctx.close();
  }
  fs.writeFileSync(path.join(OUT, 'T-contrast.json'), JSON.stringify(all, null, 1));
  const obsc = {};
  for (const theme of ['light', 'dark']) {
    const lines = []; let nFail = 0, nAll = 0; const cats = {};
    for (const [f, v] of Object.entries(all[theme])) {
      const m = v.meas.filter((x) => x.pass !== null);
      const fails = m.filter((x) => !x.pass && !x.bgclip && !x.obscured);
      for (const x of m.filter((x) => x.obscured)) (obsc[theme] ||= []).push(`${f} "${x.text}" under ${x.over}`);
      nAll += m.length; nFail += fails.length;
      for (const x of fails) { (cats[x.cat] ||= []).push(`${f} "${x.text}" ${x.tag} ${x.p10.toFixed(2)}<${x.threshold}`); }
      lines.push(`${f}: ${m.length} text runs, ${fails.length} below, min ${m.length ? Math.min(...m.map((x) => x.p10)).toFixed(2) : '-'} (theme ${v.th})`);
    }
    add(`T-text-${theme}`, 'all', nFail ? 'fail' : 'pass', `${theme}: ${nFail} of ${nAll} text runs below 4.5 : 1 (3 : 1 for large text and glyphs), not counting ${(obsc[theme] || []).length} run(s) covered by another element (${[...new Set(obsc[theme] || [])].slice(0, 4).join('; ')}). ${lines.join('; ')}`, { obscured: obsc[theme], cats: Object.fromEntries(Object.entries(cats).map(([k, v]) => [k, v.slice(0, 25)])) });
    const edge = {}; for (const v of Object.values(all[theme])) for (const c of v.comps) { if (c.edge === null) continue; (edge[c.s] ||= []).push(c.edge); }
    add(`T-comp-${theme}`, 'all', 'info', `${theme}: component edge contrast vs the surround, min per component: ${Object.entries(edge).map(([k, v]) => `${k} ${Math.min(...v).toFixed(2)} (n=${v.length})`).join('; ')}`, { edge });
  }
  // touch targets, fine and coarse
  const tt = {};
  for (const mode of ['fine', 'coarse']) {
    const ctx = await newCtx(mode === 'coarse' ? { viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true, deviceScaleFactor: 2 } : { viewport: { width: 1280, height: 900 } });
    for (const f of ['index.html', 'docs.html', 'extension.html', 'privacy.html', 'install-trust.html', 'extension-privacy.html']) {
      const page = await open(ctx, url(f));
      const r = await page.evaluate(() => {
        const out = []; const sel = 'button, a.btn, .btn, summary, .copy, .to-top, .seg button, .theme-btn, .langs a, header a, .site-nav a, footer a, input, select';
        for (const e of document.querySelectorAll(sel)) { if (!e.checkVisibility()) continue; const r = e.getBoundingClientRect(); if (r.width < 1) continue; out.push({ k: (e.closest('header') ? 'header' : e.closest('footer') ? 'footer' : e.closest('main') ? 'main' : 'other') + ' ' + e.tagName.toLowerCase() + '.' + (e.className || '').toString().split(' ')[0], t: (e.getAttribute('aria-label') || e.textContent).trim().slice(0, 20), w: Math.round(r.width), h: Math.round(r.height) }); }
        return { coarse: matchMedia('(pointer: coarse)').matches, out };
      });
      (tt[mode] ||= {})[f] = { coarse: r.coarse, small: r.out.filter((x) => x.w < 44 || x.h < 44), n: r.out.length };
      await page.close();
    }
    await ctx.close();
  }
  for (const mode of ['fine', 'coarse']) {
    const small = Object.entries(tt[mode]).flatMap(([f, v]) => v.small.map((s) => `${f}: ${s.k} "${s.t}" ${s.w}x${s.h}`));
    const ctl = small.filter((s) => !/ footer a|header a\./.test(s));
    add(`T-target-${mode}`, 'all', ctl.length ? 'fail' : 'pass', `${mode} pointer (pointer:coarse ${tt[mode]['index.html'].coarse}): ${Object.values(tt[mode]).reduce((a, v) => a + v.n, 0)} chrome controls and links measured, ${small.length} under 44 px in a dimension. ${small.slice(0, 14).join('; ')}`, { small });
  }
}

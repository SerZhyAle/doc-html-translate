// Rendered structure of every page, the theme and language resolver, and the accessibility tree.
// Each exported function is one section of the site checklist run; run.mjs decides which of them run, and in which order.
import fs from 'node:fs';
import path from 'node:path';
import { CATALOG, sha256 } from '../lib.mjs';

const FONT_ORIGINS = ['fonts.googleapis.com', 'fonts.gstatic.com'];

// B - rendered structure of every page: landmarks, headings, lang/dir, kit, style attributes, origins, resolver
export async function renderedStructure(env) {
  const { OUT, BASE, FILES, url, add, newCtx, open, readRaw } = env;
  const kitCatalog = fs.readFileSync(path.join(CATALOG, 'product-web-pages/reference/sza-kit.css'));
  const ctx = await newCtx();
  const kitServed = Buffer.from(await (await ctx.request.get(BASE + 'assets/sza-kit.css')).body());
  add('B4a', 'site', sha256(kitServed) === sha256(kitCatalog) ? 'pass' : 'fail', `served kit ${sha256(kitServed).slice(0, 12)} vs catalog ${sha256(kitCatalog).slice(0, 12)}`, {});
  const pageSec7 = fs.readFileSync(path.join(CATALOG, 'product-web-pages/PAGE-STYLE.md'), 'utf8');
  const cat = pageSec7.match(/<!-- in <head>, before CSS -->\s*<script>([\s\S]*?)<\/script>/)[1].replace(/\s+/g, '');
  const per = {}; const resolvers = {}; const originsAll = {};
  for (const f of FILES) {
    const page = await open(ctx, url(f));
    const raw = await readRaw(ctx, f);
    const d = await page.evaluate(() => {
      const q = (s) => [...document.querySelectorAll(s)];
      const hs = q('h1,h2,h3,h4,h5,h6').filter((h) => h.checkVisibility()).map((h) => +h.tagName[1]);
      let skipped = false; for (let i = 1; i < hs.length; i++) if (hs[i] - hs[i - 1] > 1) skipped = true;
      const navs = q('nav').map((n) => ({ label: n.getAttribute('aria-label') || n.getAttribute('aria-labelledby') || '', vis: n.checkVisibility() }));
      const sheets = q('link[rel=stylesheet]').map((l) => l.getAttribute('href'));
      const styleAttr = q('[style]').map((e) => e.tagName + (e.id ? '#' + e.id : '') + ':' + e.getAttribute('style').slice(0, 60));
      const imgs = q('img').map((i) => ({ src: i.getAttribute('src'), alt: i.getAttribute('alt') }));
      return { lang: document.documentElement.lang, dir: document.documentElement.dir || getComputedStyle(document.documentElement).direction, dirAttr: document.documentElement.getAttribute('dir'), mains: q('main').length, h1: q('h1').filter((h) => h.checkVisibility()).length, skipped, hs, navs, sheets, styleAttr, imgs,
        skip: !!q('a[href^="#"]').find((a) => /skip|к содерж|до змісту/i.test(a.textContent)), canvas: q('canvas').length };
    });
    const heads = raw.slice(0, raw.indexOf('</head>'));
    const scripts = [...heads.matchAll(/<script>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
    const res = scripts.find((s) => s.includes('sza-theme'));
    const iRes = res ? heads.indexOf(res) : -1;
    const firstSheet = heads.search(/<link[^>]*rel="stylesheet"/);
    const kitIdx = heads.indexOf('assets/sza-kit.css');
    const layerIdx = heads.indexOf('assets/site.css');
    per[f] = { ...d, staticStyle: (raw.match(/\sstyle="/g) || []).length, resolverBeforeFirstSheet: iRes >= 0 && iRes < kitIdx, resolverBeforeKit: iRes >= 0 && iRes < kitIdx, kitBeforeLayer: kitIdx > 0 && kitIdx < layerIdx, resolverEqualsCatalog: res ? res.replace(/\s+/g, '') === cat : false };
    resolvers[f] = res ? sha256(Buffer.from(res.replace(/\s+/g, ''))).slice(0, 8) : 'none';
    const org = new Set(page.__reqs.map((u) => new URL(u).host).filter((h) => !h.startsWith('127.0.0.1')));
    originsAll[f] = [...org];
    await page.close();
  }
  const fl = Object.entries(per);
  const bad = (fn) => fl.filter(([, v]) => fn(v)).map(([f]) => f);
  const langOf = (f) => ({ 'docs.ru.html': 'ru', 'docs.uk.html': 'uk', 'ar/index.html': 'ar', 'ur/index.html': 'ur' }[f]);
  add('B6a', 'all', bad((v) => v.mains !== 1).length ? 'fail' : 'pass', `one main: ${bad((v) => v.mains !== 1).join(',') || '18 of 18'}`, {});
  add('B6b', 'all', bad((v) => v.h1 !== 1).length || bad((v) => v.skipped).length ? 'fail' : 'pass', `one visible h1 and no skipped level: h1 off ${bad((v) => v.h1 !== 1).join(',') || 'none'}; skipped ${bad((v) => v.skipped).join(',') || 'none'}`, {});
  const noNav = bad((v) => !v.navs.some((n) => n.vis));
  add('B6c', 'all', noNav.length ? 'fail' : 'pass', `a visible labelled nav landmark: missing on ${noNav.join(', ') || 'none'}; unlabelled nav on ${bad((v) => v.navs.some((n) => !n.label)).join(', ') || 'none'}`, { noNav });
  const rtlBad = ['ar/index.html', 'ur/index.html'].filter((f) => per[f].dirAttr !== 'rtl' && per[f].dir !== 'rtl');
  const ltrBad = FILES.filter((f) => !['ar/index.html', 'ur/index.html'].includes(f) && per[f].dir === 'rtl');
  add('B6d', 'all', rtlBad.length || ltrBad.length ? 'fail' : 'pass', `lang values ${[...new Set(fl.map(([f, v]) => f.split('/')[0].replace(/\.html|docs\.?/g, '') + ':' + v.lang))].join(' ')}; rtl on ar, ur ${!rtlBad.length}`, { lang: Object.fromEntries(fl.map(([f, v]) => [f, v.lang])) });
  add('B5', 'all', bad((v) => v.staticStyle || v.styleAttr.length).length ? 'fail' : 'pass', `style attributes in source/rendered DOM: ${fl.map(([f, v]) => v.staticStyle + '/' + v.styleAttr.length).join(' ')}`, { rendered: Object.fromEntries(fl.filter(([, v]) => v.styleAttr.length).map(([f, v]) => [f, v.styleAttr])) });
  add('B4b', 'all', bad((v) => !v.kitBeforeLayer).length ? 'fail' : 'pass', `kit stylesheet linked before the page layer on ${fl.length - bad((v) => !v.kitBeforeLayer).length} of ${fl.length}; sheets order ${[...new Set(fl.map(([, v]) => v.sheets.map((s) => s.replace(/^https?:\/\/[^/]+\//, '').slice(0, 24)).join(' > ')))].join(' | ')}`, {});
  add('B8a', 'all', bad((v) => !v.resolverBeforeFirstSheet).length ? 'fail' : 'pass', `theme resolver is in the head before the kit stylesheet (the Google Fonts link precedes it, as in the PAGE-STYLE section 0 skeleton): ${fl.length - bad((v) => !v.resolverBeforeFirstSheet).length} of ${fl.length}`, { notBeforeFirstSheet: bad((v) => !v.resolverBeforeFirstSheet) });
  add('B8b', 'all', bad((v) => !v.resolverEqualsCatalog).length ? 'fail' : 'pass', `the resolver text equals PAGE-STYLE section 7: ${fl.length - bad((v) => !v.resolverEqualsCatalog).length} of ${fl.length}; distinct texts on the site: ${new Set(Object.values(resolvers)).size}`, { resolvers });
  // origins
  const declared = new Set([...FONT_ORIGINS, 'api.github.com', 'serzhyale.github.io']);
  const undeclared = {}; for (const [f, o] of Object.entries(originsAll)) { const x = o.filter((h) => !declared.has(h) && !h.startsWith('127.')); if (x.length) undeclared[f] = x; }
  const ghOn = Object.entries(originsAll).filter(([, o]) => o.includes('api.github.com')).map(([f]) => f);
  add('B7', 'all', Object.keys(undeclared).length ? 'fail' : 'pass', `origins contacted by the 18 pages: ${[...new Set(Object.values(originsAll).flat())].join(', ')}; undeclared: ${JSON.stringify(undeclared)}; the release API is called by ${ghOn.length} page(s): ${ghOn.join(', ')}`, { originsAll });
  // images
  const imgs = fl.flatMap(([f, v]) => v.imgs.map((i) => ({ f, ...i })));
  const noAlt = imgs.filter((i) => i.alt === null);
  add('C8', 'all', noAlt.length ? 'fail' : 'pass', `images ${imgs.length}, without an alt attribute ${noAlt.length}; empty (decorative) alt ${imgs.filter((i) => i.alt === '').length}`, { imgs });
  add('I3b', 'all', fl.some(([, v]) => v.canvas) ? 'fail' : 'pass', `canvas elements (animated backdrop) on the site: ${fl.reduce((a, [, v]) => a + v.canvas, 0)}`, {});
  fs.writeFileSync(path.join(OUT, 'B-per-page.json'), JSON.stringify(per, null, 1));
  await ctx.close();
}

// B8c - theme resolver and the flash on reload; sza-lang and sza-theme values
export async function themeResolver(env) {
  const { url, add, newCtx, open } = env;
  const rows = [];
  for (const theme of ['light', 'dark']) {
    for (const f of ['index.html', 'docs.html', 'privacy.html']) {
      const ctx = await newCtx({ colorScheme: theme === 'light' ? 'dark' : 'light' });
      await ctx.addInitScript((t) => { try { if (!sessionStorage.getItem('__seeded')) { localStorage.setItem('sza-theme', t); sessionStorage.setItem('__seeded', '1'); } } catch {} window.__tl = { mut: [], paint: [] }; const t0 = performance.now();
        new MutationObserver((m) => { for (const x of m) window.__tl.mut.push([+(performance.now() - t0).toFixed(1), x.attributeName, x.target.getAttribute(x.attributeName)]); }).observe(document, { attributes: true, subtree: true, attributeFilter: ['data-theme', 'data-lang', 'lang'] });
        new PerformanceObserver((l) => { for (const e of l.getEntries()) window.__tl.paint.push([e.name, +e.startTime.toFixed(1)]); }).observe({ type: 'paint', buffered: true }); }, theme);
      const page = await ctx.newPage();
      await page.goto(url(f), { waitUntil: 'load' });
      await page.reload({ waitUntil: 'load' });
      await page.waitForTimeout(400);
      const tl = await page.evaluate(() => ({ ...window.__tl, theme: document.documentElement.getAttribute('data-theme'), lang: document.documentElement.getAttribute('data-lang'), bg: getComputedStyle(document.body).backgroundColor }));
      const fp = (tl.paint.find((p) => p[0] === 'first-paint') || tl.paint[0] || [0, Infinity])[1];
      const lateTheme = tl.mut.filter((m) => m[1] === 'data-theme' && m[0] > fp);
      const lateLang = tl.mut.filter((m) => (m[1] === 'data-lang' || m[1] === 'lang') && m[0] > fp);
      rows.push({ theme, f, final: tl.theme, firstPaintMs: fp, lateTheme, lateLang, mut: tl.mut });
      await ctx.close();
    }
  }
  const themeBad = rows.filter((r) => r.final !== r.theme || r.lateTheme.length);
  add('B8c', 'all', themeBad.length ? 'fail' : 'pass', `reload with a stored theme (light, dark; the OS scheme set to the opposite): theme set before first paint and equal to the stored one on ${rows.length - themeBad.length} of ${rows.length} runs; late theme mutations ${rows.reduce((a, r) => a + r.lateTheme.length, 0)}`, { rows });
  const langLate = rows.filter((r) => r.lateLang.length);
  add('B8d', 'all', langLate.length ? 'info' : 'pass', `language attributes changed after first paint on ${langLate.length} of ${rows.length} reloads (the visible-language flash, outside EX 7's theme text)`, { langLate: langLate.map((r) => [r.f, r.theme, r.lateLang]) });

  // allowed values after choosing
  const ctx = await newCtx(); const page = await open(ctx, url('index.html'));
  const seen = [];
  for (const l of ['ru', 'ua', 'en']) { await page.click(`[data-set-lang="${l}"]`); seen.push(await page.evaluate(() => localStorage.getItem('sza-lang'))); }
  const themes = [];
  for (let i = 0; i < 3; i++) { await page.click('#themeBtn'); themes.push(await page.evaluate(() => localStorage.getItem('sza-theme'))); }
  const keys = await page.evaluate(() => Object.keys(localStorage).sort());
  const okv = seen.every((v) => ['ru', 'en', 'ua'].includes(v)) && themes.every((v) => ['dark', 'light'].includes(v));
  add('B8e', 'landing', okv && keys.every((k) => ['sza-lang', 'sza-theme'].includes(k)) ? 'pass' : 'fail', `sza-lang after RU, UA, EN: ${seen.join(',')}; sza-theme after 3 toggles: ${themes.join(',')}; keys in storage: ${keys.join(',')}`, { seen, themes, keys });
  // a stale / foreign stored value is sanitised by the resolver
  await page.evaluate(() => { localStorage.setItem('sza-theme', 'sepia'); localStorage.setItem('sza-lang', 'xx'); });
  await page.reload({ waitUntil: 'load' });
  const th = await page.evaluate(() => [document.documentElement.getAttribute('data-theme'), document.documentElement.getAttribute('data-lang')]);
  add('B8f', 'landing', ['dark', 'light'].includes(th[0]) && ['ru', 'en', 'ua'].includes(th[1]) ? 'pass' : 'fail', `foreign stored values (sza-theme=sepia, sza-lang=xx) resolve to theme=${th[0]} lang=${th[1]}`, { th });
  const p2 = await open(ctx, url('docs.html'));
  await p2.evaluate(() => { localStorage.setItem('sza-theme', 'sepia'); }); await p2.reload({ waitUntil: 'load' });
  const th2 = await p2.evaluate(() => document.documentElement.getAttribute('data-theme'));
  add('B8g', 'manual', ['dark', 'light'].includes(th2) ? 'pass' : 'fail', `docs.html with sza-theme=sepia resolves to data-theme=${th2}`, { th2 });
  // the locale query (PAGE-STYLE 3.1 amendment 1): lang is canonical, l the legacy reader, a valid lang wins, only a
  // language the page ships is selected (the landing has no de), and reading the query stores nothing
  const qrows = [];
  for (const [q, want] of [['?lang=uk', 'ua'], ['?lang=ru', 'ru'], ['?l=uk', 'ua'], ['?l=ru', 'ru'], ['?l=ua', 'ua'], ['?lang=uk&l=ru', 'ua'], ['?l=ru&lang=uk', 'ua'], ['?lang=xx&l=ru', 'ru'], ['?lang=de', 'en'], ['?l=de', 'en']]) {
    const qc = await newCtx({ locale: 'en-US' }); const qp = await open(qc, url('index.html') + q);
    qrows.push({ q, want, ...(await qp.evaluate(() => ({ got: document.documentElement.getAttribute('data-lang'), stored: localStorage.getItem('sza-lang') }))) });
    await qc.close();
  }
  const qBad = qrows.filter((r) => r.got !== r.want || r.stored !== null);
  add('B8h', 'landing', qBad.length ? 'fail' : 'pass', `locale query on the landing: ${qrows.length - qBad.length} of ${qrows.length} rows resolve as the contract says (${qrows.map((r) => `${r.q} -> ${r.got}`).join('; ')})`, { qrows });
  await ctx.close();
}

// R - accessibility tree: names, landmarks, state reflected
export async function accessibilityTree(env) {
  const { OUT, FILES, url, add, newCtx, open } = env;
  const ctx = await newCtx(); const rows = {};
  for (const f of FILES) {
    const page = await open(ctx, url(f));
    const d = await page.evaluate(() => {
      const nameless = [...document.querySelectorAll('a[href], button, [role=button], summary, input, select')].filter((e) => e.checkVisibility()).filter((e) => !((e.getAttribute('aria-label') || e.getAttribute('aria-labelledby') || e.textContent || e.value || e.getAttribute('title') || '').trim())).map((e) => e.tagName + '.' + e.className);
      const genericLinks = [...document.querySelectorAll('a[href]')].filter((a) => a.checkVisibility() && /^(click here|here|read more|more|link)$/i.test(a.textContent.trim())).map((a) => a.textContent.trim());
      const themeBtn = document.getElementById('themeBtn');
      return { nameless, genericLinks, themeLabel: themeBtn && themeBtn.getAttribute('aria-label'), theme: document.documentElement.getAttribute('data-theme'), pressed: [...document.querySelectorAll('.seg button')].map((b) => b.getAttribute('aria-pressed')), segLabel: (document.querySelector('.seg') || {}).getAttribute && document.querySelector('.seg').getAttribute('aria-label'), segRole: (document.querySelector('.seg') || {}).getAttribute && document.querySelector('.seg').getAttribute('role'), brandLabel: (document.querySelector('a.brand') || {}).getAttribute && document.querySelector('a.brand').getAttribute('aria-label'), imgsNoAlt: [...document.images].filter((i) => !i.hasAttribute('alt')).length, extLinksNoRel: [...document.querySelectorAll('a[target=_blank]')].filter((a) => !/noopener/.test(a.rel)).length };
    });
    const snap = await page.locator('body').ariaSnapshot();
    rows[f] = { ...d, snapshotLines: snap.split('\n').length, landmarks: (snap.match(/^\s*- (banner|main|navigation|contentinfo|search)/gm) || []).map((s) => s.trim().replace('- ', '')) };
    if (f === 'index.html' || f === 'docs.html') fs.writeFileSync(path.join(OUT, `aria-${f.replace(/\W/g, '_')}.yml`), snap);
    await page.close();
  }
  const nl = Object.entries(rows).filter(([, v]) => v.nameless.length);
  add('R1', 'all', nl.length ? 'fail' : 'pass', `every visible link, button, summary and input has an accessible name: ${nl.length ? nl.map(([f, v]) => f + ' ' + v.nameless.join(',')).join('; ') : 'yes on 18 of 18'}; non-descriptive link text ${Object.values(rows).reduce((a, v) => a + v.genericLinks.length, 0)}`, { rows });
  const ls = Object.entries(rows).map(([f, v]) => f + ':' + v.landmarks.join('/')).join(' ');
  add('R2', 'all', 'info', `landmarks in the accessibility tree: ${ls}`, {});
  const seg = Object.entries(rows).filter(([, v]) => v.segLabel === 'Language' && v.segRole === 'group').length;
  add('R3', 'all', 'info', `language group named "Language" with aria-pressed state on ${seg} pages; theme button label states the target theme, e.g. "${rows['index.html'].themeLabel}" while data-theme=${rows['index.html'].theme}; target=_blank links without noopener: ${Object.values(rows).reduce((a, v) => a + v.extLinksNoRel, 0)}`, {});
  await ctx.close();
}

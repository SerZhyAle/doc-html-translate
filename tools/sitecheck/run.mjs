// SITE-CHECKLIST run with real browser input (ticket 102).
//   node run.mjs [--live | --tree] [--only A,B,E,H,I0,K,M,T,R,X] [--out <dir>]
// --live (default) reads the published site; --tree serves the working tree like Pages does. Every check writes
// verdict records {box, group, status, note, data}; status is pass | fail | na | info | manual | error.
import fs from 'node:fs';
import path from 'node:path';
import { REPO, LIVE_BASE, CATALOG, sha256, gitShow, git, launch, serveTree, helperPage, diffCount, contrastOf, ts } from './lib.mjs';

const args = process.argv.slice(2);
const flag = (n) => args.includes(n);
const opt = (n, d) => { const i = args.indexOf(n); return i >= 0 ? args[i + 1] : d; };
const MODE = flag('--tree') ? 'tree' : 'live';
const ONLY = (opt('--only', '') || '').split(',').filter(Boolean);
const want = (k) => !ONLY.length || ONLY.includes(k);
const OUT = path.resolve(opt('--out', path.join(REPO, 'temp', 'site-run', `${ts()}-${MODE}`)));
fs.mkdirSync(path.join(OUT, 'shots'), { recursive: true });

const R = [];
const add = (box, group, status, note, data) => { R.push({ box, group, status, note, data }); console.log(`[${status}] ${box} ${group}: ${note}`); };

let BASE, server;
const browser0 = await launch();
if (MODE === 'tree') { const s = await serveTree(); BASE = s.base; server = s.server; } else BASE = LIVE_BASE;
const browser = browser0;
const helper = await helperPage(browser);

// ---------- the page set ----------
const sm = await (await browser.newContext()).request.get(BASE + 'sitemap.xml');
const smText = await sm.text();
const smLocs = [...smText.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
const toFile = (u) => { const p = new URL(u).pathname.replace(/^\/doc-html-translate\/?/, ''); return !p ? 'index.html' : p.endsWith('/') ? p + 'index.html' : p; };
const FILES = [...new Set(smLocs.map(toFile))];
const url = (f) => BASE + (f === 'index.html' ? '' : f.replace(/index\.html$/, ''));
const groupOf = (f) => f === 'index.html' || /^[a-z]{2}\/index\.html$/.test(f) ? 'landing' : f.startsWith('docs') ? 'manual' : f === 'extension.html' ? 'extension' : 'trust-privacy';
const LOCALES = FILES.filter((f) => /^[a-z]{2}\/index\.html$/.test(f));
const ROOT3 = ['index.html', 'extension.html', 'privacy.html', 'install-trust.html', 'extension-privacy.html'];
const DOCS = ['docs.html', 'docs.ru.html', 'docs.uk.html'];
console.log(`mode=${MODE} base=${BASE} pages=${FILES.length}`);

const newCtx = (o = {}) => browser.newContext({ viewport: { width: 1280, height: 900 }, ...o });
async function open(ctx, u, extra) {
  const page = await ctx.newPage();
  const reqs = [];
  page.on('request', (r) => reqs.push(r.url()));
  const resp = await page.goto(u, { waitUntil: 'load' });
  await page.evaluate(() => document.fonts && document.fonts.ready);
  page.__reqs = reqs; page.__resp = resp;
  return page;
}
const themeCtx = (theme, o = {}) => newCtx({ colorScheme: theme, ...o }).then(async (c) => { await c.addInitScript((t) => { try { localStorage.setItem('sza-theme', t); } catch {} }, theme); return c; });

const readRaw = async (ctx, f) => (await ctx.request.get(url(f))).text();

// =====================================================================================================
// S0 - which deployment is this
// =====================================================================================================
if (want('S0')) {
  const ctx = await newCtx();
  const files = [...FILES, 'assets/site.js', 'assets/site.css', 'assets/sza-kit.css', 'robots.txt', 'sitemap.xml'];
  const ref = MODE === 'live' ? 'origin/main' : 'HEAD';
  let same = 0, diff = [], work = 0, workDiff = [];
  for (const f of files) {
    const r = await ctx.request.get(BASE + f);
    const b = Buffer.from(await r.body());
    const h = sha256(b);
    const g = gitShow(ref, f);
    if (g && sha256(g) === h) same++; else diff.push(f);
    const wf = path.join(REPO, f);
    if (fs.existsSync(wf) && sha256(fs.readFileSync(wf)) === h) work++; else workDiff.push(f);
  }
  const head = git('rev-parse', '--short', ref);
  add('S0', 'site', 'info', `${MODE}: ${files.length} files; ${same} equal to ${ref} ${head}; ${work} equal to the working tree`, { ref, head, notEqualToRef: diff, notEqualToWorkingTree: workDiff, headNow: git('rev-parse', '--short', 'HEAD') });
  await ctx.close();
}

// =====================================================================================================
// A - the site as a whole
// =====================================================================================================
const internal = (href) => { try { const u = new URL(href); return u.href.startsWith(BASE) ? u : null; } catch { return null; } };
let crawlLinks = [];
if (want('A') || want('E') || want('H')) {
  const ctx = await newCtx();
  const dist = new Map([['index.html', 0]]); const queue = ['index.html']; const links = [];
  while (queue.length) {
    const f = queue.shift();
    const page = await open(ctx, url(f));
    const hs = await page.evaluate(() => [
      ...[...document.querySelectorAll('a[href]')].map((a) => ({ href: a.href, text: a.textContent.trim().slice(0, 40), vis: a.checkVisibility() })),
      ...[...document.querySelectorAll('[data-href]')].map((b) => ({ href: new URL(b.dataset.href, location.href).href, text: b.textContent.trim(), vis: true, viaButton: true })),
    ]);
    await page.close();
    for (const h of hs) {
      const u = internal(h.href); if (!u) continue;
      links.push({ from: f, href: u.href, text: h.text });
      const t = toFile(u.href);
      if (!dist.has(t)) { dist.set(t, dist.get(f) + 1); queue.push(t); }
    }
  }
  crawlLinks = links;
  if (want('A')) {
    const reached = [...dist.entries()];
    const orphans = FILES.filter((f) => !dist.has(f));
    const outside = reached.map(([f]) => f).filter((f) => !FILES.includes(f));
    const maxd = Math.max(...reached.map(([, d]) => d));
    add('A2', 'site', maxd <= 3 && !orphans.length ? 'pass' : 'fail', `crawl from the landing by rendered links: ${reached.length} pages reached, max ${maxd} step(s), orphans ${orphans.length}`, { dist: Object.fromEntries(dist), orphans, outside });
    // A4 sitemap both ways + every sitemap URL answers
    const answers = [];
    for (const l of smLocs) { const r = await ctx.request.get(l.replace(LIVE_BASE, BASE)); answers.push([l, r.status()]); }
    const bad = answers.filter(([, s]) => s !== 200);
    add('A4', 'site', !orphans.length && !outside.length && !bad.length ? 'pass' : 'fail', `sitemap ${smLocs.length} entries -> ${FILES.length} files; crawl set equals sitemap files both ways; non-200 answers ${bad.length}`, { bad, outside, orphans });
    // tracked html files not in the sitemap (tree view)
    const tracked = git('ls-files', '*.html').split('\n').filter(Boolean);
    const unlisted = tracked.filter((f) => !FILES.includes(f) && !/^(extension|tools|msix|installer|winget|docs\/|DEV\/|temp\/|assets\/)/.test(f));
    add('A4b', 'site', unlisted.length ? 'info' : 'pass', `tracked html outside the sitemap (forwarders, 404 and the like): ${unlisted.join(', ') || 'none'}`, { unlisted });
  }
  await ctx.close();
}

if (want('A')) {
  // A6 the seven things of the landing
  const ctx = await newCtx(); const page = await open(ctx, url('index.html'));
  const d = await page.evaluate(() => {
    const q = (s) => [...document.querySelectorAll(s)];
    const img = document.querySelector('.identity img, .hero img, header img');
    const nm = document.querySelector('.identity .name');
    const hrefs = q('a[href]').map((a) => a.href);
    const has = (re) => hrefs.some((h) => re.test(h));
    return {
      icon: !!img && img.naturalWidth > 0, iconSize: img ? [img.width, img.height] : null,
      nameText: nm ? nm.textContent : null, nameFont: nm ? getComputedStyle(nm).fontSize : null,
      seg: q('.seg button').map((b) => b.textContent.trim()), otherLangs: q('.langs a').length,
      channels: { store: has(/apps\.microsoft\.com/), github: has(/github\.com\/SerZhyAle\/doc-html-translate\/releases/), chrome: has(/chromewebstore\.google\.com/), edge: has(/microsoftedge\.microsoft\.com\/addons/), winget: !!q('.copybox').find((e) => /winget/.test(e.textContent)), installer: !!q('a[href*="setup"], a[href*="installer"]').length || q('.channels a, .button-group a, .get a').some((a) => /setup/i.test(a.textContent + a.href)) },
      docs: has(/docs\.html/), portal: has(/sza\.od\.ua/), footerTools: q('.site-footer a').length, footerH: (document.querySelector('.site-footer h2') || {}).textContent,
      author: q('.site-footer a, .hero a').some((a) => /SerZhyAle|sza\.od\.ua|mailto:sza@ukr\.net/.test(a.href)),
    };
  });
  const ok = d.icon && d.nameFont && parseFloat(d.nameFont) >= 28 && d.seg.join() === 'RU,EN,UA' && d.otherLangs >= 10 && Object.values(d.channels).every(Boolean) && d.docs && d.portal && d.footerTools >= 8 && d.author;
  add('A6', 'landing', ok ? 'pass' : 'fail', `the seven things: icon ${d.icon}, name "${d.nameText}" at ${d.nameFont}, switchers ${d.seg.join('/')} + ${d.otherLangs} locales, channels ${JSON.stringify(d.channels)}, docs ${d.docs}, portal ${d.portal}, footer links ${d.footerTools}, author ${d.author}`, d);
  await ctx.close();
}

// =====================================================================================================
// B - rendered structure of every page: landmarks, headings, lang/dir, kit, style attributes, origins, resolver
// =====================================================================================================
const FONT_ORIGINS = ['fonts.googleapis.com', 'fonts.gstatic.com'];
if (want('B')) {
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

// =====================================================================================================
// B8c - theme resolver and the flash on reload; sza-lang and sza-theme values
// =====================================================================================================
if (want('B')) {
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
  await ctx.close();
}

// =====================================================================================================
// E - language control and locales
// =====================================================================================================
if (want('E')) {
  const ctx = await newCtx();
  // three-language pages: each button
  const rows = [];
  for (const f of ROOT3) {
    const page = await open(ctx, url(f));
    for (const [code, lang] of [['ru', 'ru'], ['en', 'en'], ['ua', 'uk']]) {
      await page.click(`.seg [data-set-lang="${code}"]`);
      await page.waitForTimeout(80);
      const s = await page.evaluate(() => {
        const h1 = [...document.querySelectorAll('h1')].find((h) => h.checkVisibility());
        const vis = document.body.innerText;
        return { lang: document.documentElement.lang, h1: h1 ? h1.textContent.trim().slice(0, 60) : null, title: document.title, cyr: (vis.match(/[Ѐ-ӿ]/g) || []).length, len: vis.length, pressed: [...document.querySelectorAll('.seg button')].map((b) => b.getAttribute('aria-pressed')).join(','), store: localStorage.getItem('sza-lang') };
      });
      rows.push({ f, code, ...s });
    }
    await page.close();
  }
  const trio = {};
  for (const f of ROOT3) { const r = rows.filter((x) => x.f === f); trio[f] = { langs: r.map((x) => x.lang).join(','), titles: new Set(r.map((x) => x.title)).size, cyrRu: r[0].cyr, cyrEn: r[1].cyr, cyrUk: r[2].cyr }; }
  const bad3 = Object.entries(trio).filter(([, v]) => v.langs !== 'ru,en,uk' || v.titles !== 3 || v.cyrRu < 100 || v.cyrUk < 100 || v.cyrEn > 40);
  add('E10', 'trust-privacy,extension,landing', bad3.length ? 'fail' : 'pass', `ru/en/uk render in page on the 5 three-language pages: ${JSON.stringify(trio)}`, { rows });
  // docs: buttons navigate to a sibling that answers
  const dr = [];
  for (const f of DOCS) {
    const page = await open(ctx, url(f));
    for (const code of ['ru', 'en', 'ua']) {
      await page.goto(url(f), { waitUntil: 'load' });
      await page.click(`.seg [data-set-lang="${code}"]`);
      await page.waitForLoadState('load');
      const [u, lang, st] = [page.url().replace(BASE, ''), await page.evaluate(() => document.documentElement.lang), page.__resp && 0];
      dr.push({ from: f, code, to: u, lang });
    }
    await page.close();
  }
  const exp = { ru: ['docs.ru.html', 'ru'], en: ['docs.html', 'en'], ua: ['docs.uk.html', 'uk'] };
  const drBad = dr.filter((r) => r.to.split('#')[0] !== exp[r.code][0] || r.lang !== exp[r.code][1]);
  add('E9a', 'manual', drBad.length ? 'fail' : 'pass', `manual language control: ${dr.length} targets, ${dr.length - drBad.length} answer with the right page and lang`, { dr, drBad });
  // landing other-languages row: every target answers, with the right lang, and each locale page links back
  const lr = [];
  const lp = await open(ctx, url('index.html'));
  const targets = await lp.evaluate(() => [...document.querySelectorAll('.langs a')].map((a) => ({ href: a.href, hl: a.getAttribute('hreflang') })));
  await lp.close();
  for (const t of targets) {
    const page = await open(ctx, t.href);
    const s = await page.evaluate(() => ({ lang: document.documentElement.lang, dir: document.documentElement.dir, back: [...document.querySelectorAll('.langs a, .seg button, a[href*="doc-html-translate/"]')].length, others: [...document.querySelectorAll('.langs a')].map((a) => a.href) }));
    lr.push({ t: t.href.replace(BASE, ''), hl: t.hl, status: page.__resp.status(), ...s });
    await page.close();
  }
  const lrBad = lr.filter((r) => r.status !== 200 || r.lang !== r.hl);
  add('E9b', 'landing', lrBad.length ? 'fail' : 'pass', `landing locale row: ${lr.length} targets answer 200 with lang = hreflang: ${lr.length - lrBad.length}; row of each locale page lists ${[...new Set(lr.map((r) => r.others.length))].join('/')} others`, { lr });
  // count pages per group per locale
  const lang13 = ['en', 'ru', 'uk', ...LOCALES.map((f) => f.split('/')[0])];
  add('E10b', 'landing', lang13.length === 13 ? 'pass' : 'fail', `landing: ${lang13.length} locales published (${lang13.join(' ')}), declared 13`, {});
  add('E10c', 'manual', 'pass', `manual: ${DOCS.length} files (en ru uk), declared 3`, {});
  // is there a search? (D)
  const sp = await open(ctx, url('index.html')); const hasSearch = await sp.evaluate(() => !!document.querySelector('[role=search], input[type=search], [aria-label*=search i], [data-search]')); await sp.close();
  add('D1', 'site', 'na', `no search control on the landing (${hasSearch ? 'FOUND' : 'none'}); the corpus is ${FILES.length} pages, under the 25-page threshold of SITE-STRUCTURE section 2, and the guide tier does not require search`, {});
  await ctx.close();
}

// =====================================================================================================
// H - held addresses, not-found
// =====================================================================================================
if (want('H')) {
  const ctx = await newCtx();
  const recs = fs.readFileSync(path.join(REPO, 'configs/site-held-addresses.jsonl'), 'utf8').split('\n').filter(Boolean).map((l) => JSON.parse(l)).filter((r) => r.kind === 'held');
  const addrs = [...new Set(recs.flatMap((r) => r.addresses))].filter((a) => a.startsWith(LIVE_BASE));
  const out = [];
  for (const a of addrs) {
    const u = a.replace(LIVE_BASE, BASE);
    const page = await ctx.newPage();
    const resp = await page.goto(u, { waitUntil: 'load' });
    const frag = new URL(u).hash.slice(1);
    const q = new URL(u).searchParams.get('l');
    const s = await page.evaluate(([frag, q]) => ({ lang: document.documentElement.lang, hasFrag: frag ? !!document.getElementById(frag) : null, title: document.title }), [frag, q]);
    out.push({ a, status: resp.status(), ...s, q, frag });
    await page.close();
  }
  const bad = out.filter((o) => o.status !== 200 || o.hasFrag === false || (o.q && o.lang !== (o.q === 'ua' ? 'uk' : o.q)));
  add('H1', 'site', bad.length ? 'fail' : 'pass', `${out.length} held addresses (${recs.length} holders) requested in the browser: ${out.length - bad.length} answer with the page they mean (status 200, section id present, ?l= language applied)`, { bad, all: out.map((o) => [o.a.replace(LIVE_BASE, ''), o.status, o.hasFrag, o.lang]) });
  const retired = fs.readFileSync(path.join(REPO, 'docs/SITE_ADDRESSES.md'), 'utf8').includes('No forwarder exists today');
  add('H2', 'site', retired ? 'na' : 'manual', retired ? 'no page has been retired or moved: docs/SITE_ADDRESSES.md records that no forwarder exists, so there is no old address to request' : 'a forwarder list exists; read it', {});
  // not found
  const page = await ctx.newPage();
  const r404 = await page.goto(BASE + 'no-such-page-ticket-102.html', { waitUntil: 'load' });
  const nf = await page.evaluate(() => ({ title: document.title, text: document.body.innerText.slice(0, 200), hasHeader: !!document.querySelector('header.site-header'), brand: !!document.querySelector('a.brand'), links: [...document.querySelectorAll('a[href]')].map((a) => a.getAttribute('href')).slice(0, 12), robots: (document.querySelector('meta[name=robots]') || {}).content, search: !!document.querySelector('[role=search], input[type=search]'), lang: document.documentElement.lang }));
  const nested = await page.goto(BASE + 'x/y/no-such.html', { waitUntil: 'load' });
  const nf2 = await page.evaluate(() => ({ cssLoaded: [...document.styleSheets].some((s) => (s.href || '').includes('sza-kit')), links: [...document.querySelectorAll('a[href]')].map((a) => a.href).slice(0, 6) }));
  const chrome = nf.hasHeader && nf.brand && nf.links.length >= 2;
  add('H3', 'site', r404.status() === 404 && chrome ? 'pass' : 'fail', `an unknown address answers ${r404.status()}; page "${nf.title}", product chrome ${chrome}, noindex "${nf.robots || ''}", search ${nf.search} (n/a at this tier), links ${nf.links.join(' ')}; a nested unknown address ${nested.status()} loads the kit ${nf2.cssLoaded}, links resolve to ${nf2.links.slice(0, 2).join(' ')}`, { nf, nf2 });
  await page.screenshot({ path: path.join(OUT, 'shots', '404.png') });
  await ctx.close();
}

// =====================================================================================================
// L - layout at 360 / 768 / 1280 / 1920 / 2560, both themes: overflow, full width, targets
// =====================================================================================================
if (want('I0') || want('L')) {
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

// =====================================================================================================
// K - keyboard: tab walk, focus ring, skip link, per theme
// =====================================================================================================
if (want('K')) {
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

// =====================================================================================================
// M - reduced motion
// =====================================================================================================
if (want('M')) {
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

// =====================================================================================================
// T - contrast per theme, touch targets
// =====================================================================================================
if (want('T')) {
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

// =====================================================================================================
// R - accessibility tree: names, landmarks, state reflected
// =====================================================================================================
if (want('R')) {
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

// =====================================================================================================
// F / G / C - privacy pages per language, editions as stated, the manual's structure
// =====================================================================================================
if (want('F')) {
  const ctx = await newCtx(); const rows = {};
  const NAMES = ['fonts.googleapis.com', 'fonts.gstatic.com', 'api.github.com'];
  for (const f of ['privacy.html', 'extension-privacy.html', 'install-trust.html']) {
    const page = await open(ctx, url(f));
    for (const [code, lang] of [['en', 'en'], ['ru', 'ru'], ['ua', 'uk']]) {
      await page.click(`.seg [data-set-lang="${code}"]`); await page.waitForTimeout(80);
      const t = await page.evaluate(() => document.body.innerText);
      rows[`${f}:${lang}`] = Object.fromEntries(NAMES.map((n) => [n, t.includes(n)]));
    }
    await page.close();
  }
  const missing = Object.entries(rows).flatMap(([k, v]) => Object.entries(v).filter(([, ok]) => !ok).map(([n]) => `${k} lacks ${n}`));
  add('F1', 'trust-privacy', missing.length ? 'fail' : 'pass', `each declared origin (${NAMES.join(', ')}) is named on privacy.html, extension-privacy.html and install-trust.html in en, ru and uk: ${missing.length ? missing.length + ' of ' + Object.keys(rows).length * NAMES.length + ' missing, e.g. ' + missing.slice(0, 4).join('; ') : 'all 27 present'}`, { rows });
  await ctx.close();
}

if (want('G')) {
  const ctx = await newCtx();
  const pos = JSON.parse(fs.readFileSync(path.join(REPO, 'docs/positioning.json'), 'utf8'));
  const rows = {};
  for (const [f, lang] of [['docs.html', 'en'], ['docs.ru.html', 'ru'], ['docs.uk.html', 'uk']]) {
    const page = await open(ctx, url(f));
    const names = await page.evaluate(() => [...document.querySelectorAll('#editions li > strong')].map((s) => s.textContent.trim()));
    const ids = await page.evaluate(() => ({ subject: !!document.getElementById('subject-index'), glossary: !!document.getElementById('glossary') }));
    rows[f] = { names, ids, declared: pos.editions.map((e) => e.names[lang]) };
    await page.close();
  }
  const bad = Object.entries(rows).filter(([, v]) => v.names.join('|') !== v.declared.join('|'));
  add('G3', 'manual', bad.length ? 'fail' : 'pass', `editions as the manual states them equal the declared public editions (3, one display name each, in order): ${bad.length ? JSON.stringify(bad) : 'en, ru, uk all equal'}`, { rows });
  const noIdx = Object.entries(rows).filter(([, v]) => !v.ids.subject || !v.ids.glossary).map(([f]) => f);
  add('A1', 'site', noIdx.length ? 'fail' : 'pass', `guide tier declared in the registry rows (owner D2); page types: landing, privacy, trust, edition page, manual with subject index and glossary on ${3 - noIdx.length} of 3 files; not-found page and release notes are conditional types the site now ships`, { rows });
  // pillar order on the landing cards
  const lp = await open(ctx, url('index.html'));
  const cards = await lp.evaluate(() => [...document.querySelectorAll('#use-cases .info-card h3, #use-cases h3')].map((h) => h.textContent.trim().slice(0, 50)));
  add('A5', 'landing', 'info', `landing use-case cards in rendered order: ${cards.join(' | ')}`, { cards });
  await lp.close();
  // chrome in the same place on the root pages
  const pos2 = {};
  for (const f of [...ROOT3, ...DOCS]) {
    const page = await open(ctx, url(f));
    pos2[f] = await page.evaluate(() => { const r = (s) => { const e = document.querySelector(s); if (!e) return null; const b = e.getBoundingClientRect(); return [Math.round(b.left), Math.round(b.top), Math.round(b.width), Math.round(b.height)]; }; return { brand: r('a.brand'), seg: r('.seg'), theme: r('#themeBtn'), header: r('header') }; });
    await page.close();
  }
  // same place = the brand at the same left edge, the controls on the same line, in the same order; the controls are
  // right-aligned, so their x moves with the width of the button text beside them
  const ref = pos2['index.html'];
  const offs = Object.entries(pos2).filter(([, v]) => !v.brand || !v.seg || !v.theme || Math.abs(v.brand[0] - ref.brand[0]) > 2 || ['brand', 'seg', 'theme'].some((k) => Math.abs(v[k][1] - ref[k][1]) > 2) || !(v.brand[0] < v.seg[0] && v.seg[0] < v.theme[0]));
  add('B1', 'root pages', offs.length ? 'fail' : 'pass', `brand link, language control and theme control sit in the same place (brand at the same left edge, controls on the same line and order) on the 8 root pages at 1280 px: ${offs.length ? offs.map(([f, v]) => f + ' ' + JSON.stringify(v)).join('; ') : 'yes'} (portal chrome of ST 5 is not bound at the guide tier)`, { pos2 });
  await ctx.close();
}

if (want('C')) {
  const ctx = await newCtx(); const out = {};
  for (const f of DOCS) {
    const page = await open(ctx, url(f));
    out[f] = await page.evaluate(() => {
      const txt = (e) => e.textContent.replace(/\s+/g, ' ').trim();
      const secs = [...document.querySelectorAll('main > section, main .docs-pair > article, main details.sec')].map((s) => ({
        id: s.id || null, tag: s.tagName.toLowerCase(), title: txt(s.querySelector('h2, h3, summary') || s).slice(0, 70),
        lead: !!s.querySelector(':scope > p'), steps: s.querySelectorAll('ol li').length, bullets: s.querySelectorAll(':scope > ul > li').length,
        figures: [...s.querySelectorAll('figure')].map((g) => ({ alt: (g.querySelector('img') || {}).alt, cap: txt(g.querySelector('figcaption') || g).slice(0, 60) })),
        callouts: [...s.querySelectorAll('.note')].map((n) => txt(n).slice(0, 50)), commands: s.querySelectorAll('.copybox').length,
        tables: s.querySelectorAll('table').length, related: s.querySelectorAll('a[href^="#"], a[href$=".html"]').length, ext: s.querySelectorAll('a[target=_blank]').length,
      }));
      const claims = [...document.querySelectorAll('main p, main li, main td')].map(txt).filter((t) => /offline|no api key|sends? nothing|nothing leaves|local(ly)?\b|stored in|localstorage|password|secret|without (an? )?(server|account)|no account|upload|telemetry|key\b/i.test(t)).map((t) => t.slice(0, 190));
      const defaults = [...document.querySelectorAll('main td:nth-child(2)')].map(txt);
      return { secs, claims, defaultsInFlags: defaults.length };
    });
    await page.close();
  }
  fs.writeFileSync(path.join(OUT, 'C-manual-structure.json'), JSON.stringify(out, null, 1));
  // RP 8: every bold or emphasised name in the task sections, looked up in the product's own strings of the page language
  const corpusFor = (lang) => {
    const parts = [fs.readFileSync(path.join(REPO, 'cmd/doc-html-ui/i18n.js'), 'utf8')];
    for (const f of fs.readdirSync(path.join(REPO, 'internal/i18n'))) if (f.endsWith('.go')) parts.push(fs.readFileSync(path.join(REPO, 'internal/i18n', f), 'utf8'));
    const mp = path.join(REPO, 'extension/_locales', lang, 'messages.json'); if (fs.existsSync(mp)) parts.push(fs.readFileSync(mp, 'utf8'));
    const en = path.join(REPO, 'extension/_locales/en/messages.json'); parts.push(fs.readFileSync(en, 'utf8'));
    return parts.join('\n').toLowerCase().replace(/\\"/g, '"').replace(/\s+/g, ' ');
  };
  const SKIP = /^(translate page|перевести страницу|перекласти сторінку|7-zip|calibre|tesseract|ffmpeg|pdftotext|doc-html-ui|filedo|environment\.txt|settings\.json|logs\/)$/i;
  // The one quoted control the product draws with a glyph: the Explorer verb (action.convert, internal/iconart). Every other quoted
  // control is a text button in the window and in the extension, so there is no glyph to show beside it (ICON-SET 8: never a picture chosen to decorate).
  const GLYPHED = /^convert to html$/i;
  const nameRows = {};
  for (const [f, lang] of [['docs.html', 'en'], ['docs.ru.html', 'ru'], ['docs.uk.html', 'uk']]) {
    const page = await open(ctx, url(f));
    const names = await page.evaluate(() => [...document.querySelectorAll(['workflow', 'quick-start', 'logs', 'secret-files'].map((id) => `#${id} strong:not(.claim), #${id} em`).concat('#flags > p strong').join(', '))].map((e) => ({ t: e.textContent.replace(/\s+/g, ' ').trim(), glyph: !!(e.previousElementSibling && e.previousElementSibling.matches('svg')) || !!e.querySelector('svg') })));
    const corpus = corpusFor(lang === 'uk' ? 'uk' : lang);
    const norm = (t) => t.toLowerCase().replace(/[«»"“”]/g, '').replace(/\.$/, '').trim();
    const uniq = [...new Map(names.map((n) => [n.t, n])).values()].filter((n) => !SKIP.test(n.t) && !/[.\\/%]/.test(n.t.replace(/\.$/, '')) && n.t.length > 2);
    nameRows[f] = { checked: uniq.length, found: uniq.filter((n) => corpus.includes(norm(n.t))).map((n) => n.t), missing: uniq.filter((n) => !corpus.includes(norm(n.t))).map((n) => n.t), glyphs: names.filter((n) => n.glyph).length, glyphMissing: uniq.filter((n) => GLYPHED.test(n.t) && !n.glyph).map((n) => n.t) };
    await page.close();
  }
  add('C4', 'manual', Object.values(nameRows).some((v) => v.missing.length || v.glyphMissing.length || v.glyphs === 0) ? 'fail' : 'pass', `names of controls and settings quoted in the manual, looked up in the product strings of the page language: ${Object.entries(nameRows).map(([f, v]) => `${f}: ${v.found.length}/${v.checked} found, glyph beside a name ${v.glyphs}, glyph missing: ${v.glyphMissing.join(' / ') || 'none'}; not found: ${v.missing.join(' / ') || 'none'}`).join(' || ')}`, { nameRows });

  // RP 7: the four task sections in the fixed order - title (a verb), lead, requirements with the availability marker, steps, outcome callout, related links
  const anat = {};
  for (const [f] of [['docs.html'], ['docs.ru.html'], ['docs.uk.html']]) {
    const page = await open(ctx, url(f));
    anat[f] = await page.evaluate(() => {
      const txt = (e) => e.textContent.replace(/\s+/g, ' ').trim();
      return ['workflow', 'quick-start', 'logs', 'secret-files'].map((id) => {
        const s = document.getElementById(id);
        const title = txt(s.querySelector('h2, summary')).replace(/^\d+/, '');
        const lead = s.querySelector(':scope > p');
        const req = s.querySelector(':scope > dl.req');
        const steps = s.querySelector(':scope > ol.steps');
        const note = s.querySelectorAll(':scope > .note');
        const rel = s.querySelector(':scope > p.related');
        const order = [lead, req, steps, note[0], rel];
        const inOrder = order.every((e, i) => e && (i === 0 || (order[i - 1].compareDocumentPosition(e) & Node.DOCUMENT_POSITION_FOLLOWING)));
        const liFigs = steps ? [...steps.querySelectorAll(':scope > li')].map((li) => li.querySelectorAll('figure').length) : [];
        return { id, title, inOrder, lead: !!lead, req: !!req, pills: req ? req.querySelectorAll('.pill').length : 0, dts: req ? req.querySelectorAll('dt').length : 0,
          steps: liFigs.length, stepsWithoutFigure: liFigs.filter((n) => n === 0).length, callouts: note.length, related: rel ? rel.querySelectorAll('a').length : 0 };
      });
    });
    await page.close();
  }
  const anatBad = Object.entries(anat).flatMap(([f, rows]) => rows.filter((r) => !(r.inOrder && r.lead && r.req && r.pills >= 1 && r.dts === 2 && r.steps >= 1 && r.callouts === 1 && r.related >= 1)).map((r) => f + '#' + r.id));
  add('C5', 'manual', anatBad.length ? 'fail' : 'pass', `the four task sections in the fixed order (lead, requirements with an edition marker, steps, one outcome callout, related links): ${anatBad.length ? 'missing or out of order in ' + anatBad.join(', ') : 'all 4 on en, ru and uk'}; steps without a figure (they happen in another vendor's interface: a terminal, Explorer, the browser menu, a mail program): ${Object.entries(anat).map(([f, rows]) => f + ' ' + rows.map((r) => r.id + ' ' + r.stepsWithoutFigure + '/' + r.steps).join(', ')).join(' | ')}`, { anat });
  const s = out['docs.html'].secs;
  add('C0', 'manual', 'info', `manual structure (en): ${s.map((x) => `${x.id || x.tag}:"${x.title}" steps ${x.steps}/${x.bullets} figs ${x.figures.length} callouts ${x.callouts.length}`).join(' | ')}`, { out });
  await ctx.close();
}

// =====================================================================================================
// X - extracted text for the manual reading (RP 5-10, RP 12) and the version strings (EX 13)
// =====================================================================================================
if (want('X')) {
  const ctx = await newCtx(); const out = {};
  for (const f of FILES) {
    const raw = await readRaw(ctx, f);
    const text = raw.replace(/<script[\s\S]*?<\/script>/g, ' ').replace(/<style[\s\S]*?<\/style>/g, ' ').replace(/<[^>]+>/g, ' ').replace(/&amp;/g, '&').replace(/\s+/g, ' ');
    out[f] = { versions: [...new Set(text.match(/\b\d{2}\.\d{3,4}\.\d{3,4}\b/g) || [])], tags: [...new Set(text.match(/\bv\d{2}\.\d{4}\.\d{4}\b/g) || [])] };
  }
  // The release-notes page is the dated history of the versions (SITE-STRUCTURE rule 14), not a claim about the current one.
  const withV = Object.entries(out).filter(([f, v]) => f !== 'release-notes.html' && (v.versions.length || v.tags.length));
  add('G4', 'all', withV.length ? 'fail' : 'pass', `a hard-coded version string on the pages: ${withV.length ? withV.map(([f, v]) => f + ' ' + [...v.versions, ...v.tags].join(',')).join('; ') : 'none (the release tag is fetched at run time into #releaseTag)'}`, { out });
  const lp = await open(ctx, url('index.html'));
  await lp.waitForFunction(() => (document.getElementById('releaseTag') || {}).textContent && document.getElementById('releaseTag').textContent.trim(), null, { timeout: 8000 }).catch(() => {});
  const tag = await lp.evaluate(() => (document.getElementById('releaseTag') || {}).textContent);
  add('G4b', 'landing', 'info', `release tag shown on the landing at run time: "${(tag || '').trim()}" (api.github.com contacted)`, { tag });
  await lp.close();
  await ctx.close();
}

fs.writeFileSync(path.join(OUT, 'results.json'), JSON.stringify({ mode: MODE, base: BASE, at: new Date().toISOString(), ref: git('rev-parse', '--short', 'origin/main'), head: git('rev-parse', '--short', 'HEAD'), results: R }, null, 1));
fs.writeFileSync(path.join(OUT, 'REPORT.md'), `# Site checklist run - ${MODE}\n\nBase ${BASE}; origin/main ${git('rev-parse', '--short', 'origin/main')}; HEAD ${git('rev-parse', '--short', 'HEAD')}; ${new Date().toISOString()}\n\n| Box | Group | Status | Note |\n| --- | --- | --- | --- |\n` + R.map((r) => `| ${r.box} | ${r.group} | ${r.status} | ${r.note.replace(/\|/g, '/').replace(/\n/g, ' ')} |`).join('\n') + '\n');
const cnt = R.reduce((a, r) => ((a[r.status] = (a[r.status] || 0) + 1), a), {});
console.log('\nDONE', JSON.stringify(cnt), '->', OUT);
await browser.close();
if (server) server.close();

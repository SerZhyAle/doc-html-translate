// Site-level checks: the served deployment, the link graph, the landing page, held addresses and not-found, version strings.
// Each exported function is one section of the site checklist run; run.mjs decides which of them run, and in which order.
import fs from 'node:fs';
import path from 'node:path';
import { REPO, LIVE_BASE, sha256, gitShow, git } from '../lib.mjs';

// S0 - which deployment is this
export async function deployment(env) {
  const { MODE, BASE, FILES, add, newCtx } = env;
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

// A - the site as a whole: the crawl from the landing by rendered links, the sitemap both ways
export async function siteCrawl(env) {
  const { BASE, FILES, smLocs, toFile, url, add, want, newCtx, open } = env;
  const internal = (href) => { try { const u = new URL(href); return u.href.startsWith(BASE) ? u : null; } catch { return null; } };
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
  env.crawlLinks = links;
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

// A6, A6b - the landing page: the seven things and the documentation link at the phone width
export async function landing(env) {
  const { LOCALES, url, add, newCtx, open } = env;
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
      docs: q('.site-header a[href]').some((a) => /docs(\.ru|\.uk)?\.html$/.test(a.getAttribute('href'))), portal: has(/sza\.od\.ua/), footerTools: q('.site-footer a').length, footerH: (document.querySelector('.site-footer h2') || {}).textContent,
      author: q('.site-footer a, .hero a').some((a) => /SerZhyAle|sza\.od\.ua|mailto:sza@ukr\.net/.test(a.href)),
    };
  });
  const ok = d.icon && d.nameFont && parseFloat(d.nameFont) >= 28 && d.seg.join() === 'RU,EN,UA' && d.otherLangs >= 10 && Object.values(d.channels).every(Boolean) && d.docs && d.portal && d.footerTools >= 8 && d.author;
  add('A6', 'landing', ok ? 'pass' : 'fail', `the seven things: icon ${d.icon}, name "${d.nameText}" at ${d.nameFont}, switchers ${d.seg.join('/')} + ${d.otherLangs} locales, channels ${JSON.stringify(d.channels)}, docs ${d.docs}, portal ${d.portal}, footer links ${d.footerTools}, author ${d.author}`, d);
  // A6b the documentation link of the header at the phone width (PAGE-CONTENT landing item 4): visible, a real target (44 fine, 48 coarse),
  // inside the header, and nothing scrolls sideways - on the landing in its three languages and on the ten locale landings
  const narrow = [];
  for (const f of ['index.html', ...LOCALES]) {
    for (const lang of f === 'index.html' ? ['en', 'ru', 'uk'] : ['en']) {
      for (const coarse of [false, true]) {
        const nc = await newCtx(coarse ? { viewport: { width: 360, height: 800 }, hasTouch: true, isMobile: true, deviceScaleFactor: 2 } : { viewport: { width: 360, height: 800 } });
        const np = await open(nc, url(f) + '?lang=' + lang);
        const m = await np.evaluate(() => {
          const h = document.querySelector('.site-header');
          const link = [...h.querySelectorAll('a[href]')].find((a) => /docs(\.ru|\.uk)?\.html$/.test(a.getAttribute('href')) && a.checkVisibility());
          const hb = h.getBoundingClientRect(), lb = link && link.getBoundingClientRect();
          return { link: !!link, w: lb && Math.round(lb.width), h: lb && Math.round(lb.height), inside: !!lb && lb.left >= hb.left - 0.5 && lb.right <= hb.right + 0.5, sideways: document.documentElement.scrollWidth - innerWidth };
        });
        narrow.push({ f, lang, coarse, ...m }); await nc.close();
      }
    }
  }
  const nbad = narrow.filter((r) => !r.link || r.h < (r.coarse ? 48 : 44) || r.w < (r.coarse ? 48 : 44) || !r.inside || r.sideways > 0);
  add('A6b', 'landing', nbad.length ? 'fail' : 'pass', `header documentation link at 360 px (fine 44 / coarse 48 target, inside the header, no sideways scroll): ${narrow.length - nbad.length} of ${narrow.length} landing runs (index in 3 languages, ${LOCALES.length} locale landings) ok${nbad.length ? '; failing ' + nbad.map((r) => `${r.f}?lang=${r.lang}${r.coarse ? ' coarse' : ''}`).join(', ') : ''}`, { narrow });
  await ctx.close();
}

// H - held addresses, not-found
export async function heldAddresses(env) {
  const { OUT, BASE, add, newCtx } = env;
  const ctx = await newCtx();
  const recs = fs.readFileSync(path.join(REPO, 'configs/site-held-addresses.jsonl'), 'utf8').split('\n').filter(Boolean).map((l) => JSON.parse(l)).filter((r) => r.kind === 'held');
  const addrs = [...new Set(recs.flatMap((r) => r.addresses))].filter((a) => a.startsWith(LIVE_BASE));
  const out = [];
  for (const a of addrs) {
    const u = a.replace(LIVE_BASE, BASE);
    const page = await ctx.newPage();
    const resp = await page.goto(u, { waitUntil: 'load' });
    const frag = new URL(u).hash.slice(1);
    const sp = new URL(u).searchParams;
    const q = sp.get('lang') || sp.get('l');
    const s = await page.evaluate(([frag, q]) => ({ lang: document.documentElement.lang, hasFrag: frag ? !!document.getElementById(frag) : null, title: document.title }), [frag, q]);
    out.push({ a, status: resp.status(), ...s, q, frag });
    await page.close();
  }
  const bad = out.filter((o) => o.status !== 200 || o.hasFrag === false || (o.q && o.lang !== (o.q === 'ua' ? 'uk' : o.q)));
  add('H1', 'site', bad.length ? 'fail' : 'pass', `${out.length} held addresses (${recs.length} holders) requested in the browser: ${out.length - bad.length} answer with the page they mean (status 200, section id present, ?lang= / legacy ?l= language applied)`, { bad, all: out.map((o) => [o.a.replace(LIVE_BASE, ''), o.status, o.hasFrag, o.lang]) });
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

// X - extracted text for the manual reading (RP 5-10, RP 12) and the version strings (EX 13)
export async function versionStrings(env) {
  const { FILES, url, add, newCtx, open, readRaw } = env;
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

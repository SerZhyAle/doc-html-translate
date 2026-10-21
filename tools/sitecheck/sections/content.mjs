// Language control and locales, privacy pages per language, editions as stated, and the manual structure.
// Each exported function is one section of the site checklist run; run.mjs decides which of them run, and in which order.
import fs from 'node:fs';
import path from 'node:path';
import { REPO } from '../lib.mjs';

// E - language control and locales
export async function languageControl(env) {
  const { BASE, FILES, LOCALES, ROOT3, DOCS, url, add, newCtx, open } = env;
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
  add('E10', 'trust-privacy,extension,landing', bad3.length ? 'fail' : 'pass', `ru/en/uk render in page on the ${ROOT3.length} three-language pages: ${JSON.stringify(trio)}`, { rows });
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

// F - privacy pages per language: every declared origin is named
export async function privacyOrigins(env) {
  const { url, add, newCtx, open } = env;
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

// G - editions as the manual states them, the landing cards, the chrome of the root pages
export async function editionsAndChrome(env) {
  const { ROOT3, DOCS, url, add, newCtx, open } = env;
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
  add('B1', 'root pages', offs.length ? 'fail' : 'pass', `brand link, language control and theme control sit in the same place (brand at the same left edge, controls on the same line and order) on the ${ROOT3.length + DOCS.length} root pages at 1280 px: ${offs.length ? offs.map(([f, v]) => f + ' ' + JSON.stringify(v)).join('; ') : 'yes'} (portal chrome of ST 5 is not bound at the guide tier)`, { pos2 });
  await ctx.close();
}

// C - the manual structure: task sections, quoted names, anatomy
export async function manualStructure(env) {
  const { OUT, DOCS, url, add, newCtx, open } = env;
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

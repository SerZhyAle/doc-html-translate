// Renders one help page (CONTEXT-HELP rule 8: title, what it is for, the screenshot, the order, the elements,
// related, footer) from a copy-deck page, the GUI dictionary, the chrome strings and the capture's pin positions.
// The output is plain HTML that reads completely with the script off; page.js only enhances it.
import { word } from './dict.mjs';

export const SITE = 'https://serzhyale.github.io/doc-html-translate/';
export const PORTAL = 'https://sza.od.ua';
export const SIBLING = { FileDO: 'https://serzhyale.github.io/FileDO/' };
export const SUPPORT = 'mailto:sza@ukr.net';

const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
// Dictionary labels end in a colon or two dots as field captions ("Output:", "Browse.."); a name in running text
// keeps the dots (they are part of the button) and loses the colon.
const bare = (s) => String(s).replace(/:\s*$/, '');

export class Pager {
  constructor({ dict, chrome, locale, source }) {
    this.dict = dict; this.chrome = chrome; this.locale = locale; this.source = source;
  }
  pick(v) {
    if (v == null) return '';
    if (typeof v === 'string') return v;
    const out = v[this.locale] ?? v[this.source];
    if (out == null) throw new Error(`no ${this.locale} or ${this.source} text in ${JSON.stringify(v).slice(0, 80)}`);
    return out;
  }
  key(k) { return bare(word(this.dict, this.locale, k)); }
  // The names of an element or an item: authored, one dictionary key, or several joined.
  nameOf(o) {
    if (o.nameKeys) return o.nameKeys.map((k) => esc(this.key(k))).join(' / ');
    if (o.nameKey) return esc(this.key(o.nameKey));
    return esc(this.pick(o.name));
  }
  // The one line: authored, or a dictionary hint (which may carry <code>, as the window renders it).
  oneOf(o) {
    if (o.oneKey) return this.expand(word(this.dict, this.locale, o.oneKey), {});
    return this.expand(this.pick(o.one), {});
  }
  // expand turns the copy deck's markers into markup: {k:key} a label of the window in bold, {kbd:A+B} keys,
  // {ref:id} the pin number of an element of this page.
  expand(s, refs) {
    return String(s)
      .replace(/\{k:([A-Za-z0-9_]+)\}/g, (_, k) => `<b>${esc(this.key(k))}</b>`)
      .replace(/\{kbd:([^}]+)\}/g, (_, keys) => keys.split('+').map((x) => `<kbd>${esc(x)}</kbd>`).join(' + '))
      .replace(/\{ref:([a-z0-9-]+)\}/g, (_, id) => {
        const n = refs[id];
        if (!n) throw new Error(`{ref:${id}} names no element of this page`);
        return `<a class="ref" href="#${id}">${n}</a>`;
      });
  }
  title(page) {
    if (page.titleKey) return this.key(page.titleKey);
    return this.pick(page.title);
  }
}

// resolveLink makes the href of a related entry: another help page (relative), the site's documentation in the
// page's language, the site, the portal or a sibling product.
function resolveLink(rel, locale, pages) {
  if (rel.to) {
    const [id, hash] = rel.to.split('#');
    if (!pages.has(id)) throw new Error(`related link names no page: ${rel.to}`);
    return { href: `${id}.html${hash ? '#' + hash : ''}`, external: false };
  }
  if (rel.site) {
    const [id, hash] = rel.site.split('#');
    if (id === 'home') return { href: SITE, external: true };
    const suffix = locale === 'en' ? '' : '.' + locale;
    return { href: `${SITE}${id}${suffix}.html${hash ? '#' + hash : ''}`, external: true };
  }
  if (rel.portal) return { href: PORTAL, external: true };
  if (rel.sibling) {
    if (!SIBLING[rel.sibling]) throw new Error(`unknown sibling product ${rel.sibling}`);
    return { href: SIBLING[rel.sibling], external: true };
  }
  throw new Error(`a related entry has no target: ${JSON.stringify(rel).slice(0, 80)}`);
}

function relatedItem(pg, rel, pages) {
  const c = pg.chrome;
  const { href, external } = resolveLink(rel, pg.locale, pages);
  const pill = { window: c.pillWindow, dialog: c.pillDialog, site: c.pillSite, portal: c.pillPortal, sibling: c.pillSibling }[rel.pill];
  const gold = rel.pill === 'site' || rel.pill === 'portal' || rel.pill === 'sibling' ? ' gold' : '';
  const attrs = external ? ` href="${esc(href)}" target="_blank" rel="noopener"` : ` href="${esc(href)}"`;
  return `    <li><a${attrs}><b>${esc(pg.pick(rel.name))}</b><span class="pill${gold}">${esc(pill)}</span><small>${pg.oneOf(rel)}</small></a></li>`;
}

function head(pg, titleText, depth) {
  const c = pg.chrome;
  return `<!doctype html>
<html lang="${c.lang}" dir="${c.dir}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer">
<meta name="robots" content="noindex">
<title>${esc(c.docTitle.replace('{name}', titleText))}</title>
<script src="${depth}page.js"></script>
<link rel="stylesheet" href="${depth}sza-kit.css">
<link rel="stylesheet" href="${depth}help.css">
</head>
`;
}

function footer(pg, version, date) {
  const c = pg.chrome;
  return `  <footer class="help-foot">
    <span>${esc(c.footCaptured.replace('{version}', version).replace('{date}', date))}</span>
    <a href="index.html">${esc(c.footIndex)}</a>
    <a href="${SUPPORT}">${esc(c.footSupport)}</a>
  </footer>
`;
}

// pins: { shotId: { w, h, image: {dark, light}, pins: { elementId: {x, y} } } } in percent of the image.
export function renderPage({ page, pg, pages, captures, version, date }) {
  const c = pg.chrome;
  const title = pg.title(page);
  const numbers = {};
  page.elements.forEach((e, i) => { numbers[e.id] = i + 1; });

  const figures = (page.shots || []).map((shot) => {
    const cap = captures[shot.id];
    if (!cap) throw new Error(`page ${page.id}: no capture for shot ${shot.id}`);
    const pins = page.elements.filter((e) => e.shot === shot.id).map((e) => {
      const p = cap.pins[e.id];
      if (!p) throw new Error(`page ${page.id}: element ${e.id} has no pin in shot ${shot.id}`);
      const name = pg.nameOf(e).replace(/<[^>]+>/g, '');
      return `      <a class="pin" style="--x:${p.x}%;--y:${p.y}%" href="#${e.id}" aria-label="${numbers[e.id]}, ${name}">${numbers[e.id]}</a>`;
    }).join('\n');
    const caption = page.kind === 'dialog' ? c.shotCaptionDialog : c.shotCaptionWindow;
    return `  <figure class="shot" id="shot-${shot.id}">
    <div class="frame">
      <img class="on-dark" src="${cap.image.dark}" width="${cap.w}" height="${cap.h}" alt="${esc(pg.pick(shot.alt))}">
      <img class="on-light" src="${cap.image.light}" width="${cap.w}" height="${cap.h}" alt="">
${pins}
    </div>
    <figcaption>${esc(caption)}${shot.sometimes ? ' ' + esc(c.figureOnlySometimes) : ''}</figcaption>
  </figure>
`;
  }).join('\n');

  const order = page.noOrder
    ? `  <h2>${esc(c.orderH)}</h2>\n  <p>${esc(c.orderNone)}</p>\n`
    : `  <h2>${esc(c.orderH)}</h2>\n  <ol class="order">\n${page.order.map((s) => `    <li>${pg.expand(pg.pick(s), numbers)}</li>`).join('\n')}\n  </ol>\n`;

  const factRows = (e) => [
    ['factWhy', e.why], ['factNext', e.next], ['factUndo', e.undo], ['factShortcut', e.shortcut],
  ].filter(([, v]) => v).map(([k, v]) => `          <dt>${esc(c[k])}</dt><dd>${pg.expand(pg.pick(v), numbers)}</dd>`).join('\n');

  const elements = page.elements.map((e) => {
    const body = [];
    if (e.text) body.push(`        <p>${pg.expand(pg.pick(e.text), numbers)}</p>`);
    if (e.items) {
      body.push('        <ul>\n' + e.items.map((it) => `          <li><b>${pg.nameOf(it)}</b> - ${pg.oneOf(it)}</li>`).join('\n') + '\n        </ul>');
    }
    const facts = factRows(e);
    if (facts) body.push(`        <dl class="facts">\n${facts}\n        </dl>`);
    if (e.note) {
      const tag = e.note.kind === 'care' ? c.noteCare : c.noteTip;
      body.push(`        <div class="note"><span class="tag">${esc(tag)}</span><p>${pg.expand(pg.pick(e.note.text), numbers)}</p></div>`);
    }
    return `    <details class="el" id="${e.id}">
      <summary><span class="mk">${numbers[e.id]}</span><span><span class="name">${pg.nameOf(e)}</span><span class="one">${pg.oneOf(e)}</span></span></summary>
      <div class="full">
${body.join('\n')}
      </div>
    </details>`;
  }).join('\n\n');

  const related = (page.related || []).map((r) => relatedItem(pg, r, pages)).join('\n');

  return head(pg, title, '../') + `<body class="help">
<main class="container">

  <nav class="help-crumbs" aria-label="${esc(c.crumbLabel)}">
    <a href="index.html">${esc(c.crumbIndex)}</a><span class="sep" aria-hidden="true">/</span><span>${esc(title)}</span>
  </nav>
  <header class="hero">
    <p class="kicker">${esc(c.kicker[page.kind])}</p>
    <h1>${esc(title)}</h1>
    <p class="lead">${pg.expand(pg.pick(page.lead), numbers)}</p>
  </header>

${figures}
${order}
  <h2>${esc(c.elementsH)}</h2>
  <div class="elements">

${elements}

  </div>

  <h2>${esc(c.relatedH)}</h2>
  <ul class="related">
${related}
  </ul>

${footer(pg, version, date)}
</main>
</body>
</html>
`;
}

export function renderIndex({ index, pg, pages, version, date, pageTitles }) {
  const c = pg.chrome;
  const title = pg.title(index);
  const windows = index.windows.map((w) => {
    const item = relatedItem(pg, w, pages);
    return item;
  }).join('\n');
  const elsewhere = index.elsewhere.map((r) => relatedItem(pg, r, pages)).join('\n');
  const keys = index.keys.map((k) => `    <dt>${k.keys.split('+').map((x) => `<kbd>${esc(x)}</kbd>`).join(' + ')}</dt><dd>${esc(pg.pick(k.what))}</dd>`).join('\n');
  return head(pg, title, '../') + `<body class="help">
<main class="container">

  <header class="hero">
    <p class="kicker">${esc(c.kicker.index)}</p>
    <h1>${esc(title)}</h1>
    <p class="lead">${esc(c.indexLead)}</p>
  </header>

  <h2>${esc(c.indexWindowsH)}</h2>
  <ul class="related">
${windows}
  </ul>

  <h2>${esc(c.indexKeysH)}</h2>
  <dl class="facts">
${keys}
  </dl>

  <h2>${esc(c.indexElsewhereH)}</h2>
  <ul class="related">
${elsewhere}
  </ul>

  <p>${esc(c.indexLanguagesNote)}</p>

  <footer class="help-foot">
    <span>${esc(c.footCaptured.replace('{version}', version).replace('{date}', date))}</span>
    <a href="${SUPPORT}">${esc(c.footSupport)}</a>
  </footer>
</main>
</body>
</html>
`;
}

// A document picture that cannot be shown, measured in a real engine (ICON-EXTERNAL section 13 F; ticket 122): the
// generated reader (a chapter page and the single page, converted from a real EPUB by the real CLI) and the extension
// viewer (the real extension/ folder over loopback) must draw the catalog's content.image and the picture's alt text in
// place of the browser's broken-image icon, change no valid picture, and ask for nothing. A unit test can only say the
// handler is wired; whether the engine delivers a non-bubbling error event to a capture listener on the document, where
// the failed picture ends up and what it looks like is only knowable by loading the page.
//
//   node tools/uicheck/run.mjs --only reader     (this runs inside it, after the chrome matrix)
//
// The EPUB holds: a cover in the single-image SVG wrapper idiom whose picture is not in the archive (alt from the
// wrapper's <title>), one picture that is there, one that is missing with an alt, one missing with an empty alt.
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { execFileSync } from 'node:child_process';
import { REPO, readerSource } from './lib.mjs';
import { WORK, png } from './reader.mjs';

const BIN = path.join(REPO, 'temp', 'uicheck', 'bin');
const vendored = () => fs.readFileSync(path.join(REPO, 'assets', 'glyphs', 'content.image.svg'), 'utf8').match(/\sd="([^"]+)"/)[1];
const f1 = (n) => (Math.round(n * 10) / 10).toString();

export const FIGURE = 'A missing figure';
const validPng = () => png(160, 100, 60);

// ---- fixtures ---------------------------------------------------------------------------------------------------
async function writeEpub() {
  const entry = path.join(REPO, 'extension', 'node_modules', 'fflate', 'esm', 'browser.js');
  const { zipSync } = await import(pathToFileURL(entry).href);
  const enc = (s) => new TextEncoder().encode(s);
  const xhtml = (body) => `<?xml version="1.0" encoding="UTF-8"?><html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="en"><head><title>T</title></head><body>${body}</body></html>`;
  const ch1 = xhtml(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 160 100"><title>Cover</title><image width="160" height="100" xlink:href="images/cover-gone.png"/></svg>` +
    `<h1>One</h1><p>Text before.</p><p><img src="images/ok.png" alt="A valid picture"/></p>` +
    `<p><img src="images/gone.png" alt="${FIGURE}"/></p><p><img src="images/blank.png" alt=""/></p><p>Text after.</p>`);
  const ch2 = xhtml(`<h1>Two</h1><p>Second chapter.</p>`);
  const files = {
    mimetype: [enc('application/epub+zip'), { level: 0 }],
    'META-INF/container.xml': enc('<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>'),
    'OEBPS/content.opf': enc('<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:identifier id="id">urn:uuid:122</dc:identifier><dc:title>Pictures</dc:title><dc:language>en</dc:language></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/><item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/><item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/><item id="ok" href="images/ok.png" media-type="image/png"/></manifest><spine><itemref idref="ch1"/><itemref idref="ch2"/></spine></package>'),
    'OEBPS/nav.xhtml': enc(xhtml('<nav epub:type="toc"><ol><li><a href="ch1.xhtml">One</a></li><li><a href="ch2.xhtml">Two</a></li></ol></nav>')),
    'OEBPS/ch1.xhtml': enc(ch1),
    'OEBPS/ch2.xhtml': enc(ch2),
    'OEBPS/images/ok.png': new Uint8Array(validPng()),
  };
  const dir = path.join(WORK, 'src');
  fs.mkdirSync(dir, { recursive: true });
  const file = path.join(dir, 'pictures.epub');
  fs.writeFileSync(file, Buffer.from(zipSync(files)));
  return file;
}

// Converts the EPUB twice with the real CLI (single page, then one page per chapter) and returns the pages to open.
export async function buildPictureFixtures() {
  const exe = path.join(BIN, 'doc-html-translate.exe');
  const epub = await writeEpub();
  const out = (...p) => path.join(WORK, 'out', ...p);
  const conv = (args) => execFileSync(exe, ['-noopen', '-notranslate', '-force', '-ui-lang', 'en', ...args], { cwd: REPO, stdio: 'ignore' });
  conv(['-folder', out('pictures-single'), epub]);
  conv(['-multipage', '-folder', out('pictures-multi'), epub]);
  const find = (root, pick) => {
    const hits = [];
    const walk = (d) => { for (const e of fs.readdirSync(d, { withFileTypes: true })) { const p = path.join(d, e.name); if (e.isDirectory()) walk(p); else if (pick(p)) hits.push(p); } };
    walk(root);
    return hits;
  };
  const html = (p) => /\.html$/i.test(p);
  const withFigure = (root) => find(root, (p) => html(p) && fs.readFileSync(p, 'utf8').includes(`alt="${FIGURE}"`));
  return { single: withFigure(out('pictures-single'))[0], chapter: withFigure(out('pictures-multi'))[0] };
}

// ---- the measurement (runs inside the page) --------------------------------------------------------------------
function probe(S) {
  const imgs = [...document.images];
  const boxes = [...document.querySelectorAll(S.box)].map((b) => {
    const svg = b.querySelector('svg'), p = svg && svg.querySelector('path'), r = b.getBoundingClientRect(), g = svg ? svg.getBoundingClientRect() : null;
    const label = [...b.children].find((c) => c.tagName.toLowerCase() === 'span');
    return {
      text: b.textContent.trim(), d: p ? p.getAttribute('d') : null, glyphHidden: svg ? svg.getAttribute('aria-hidden') : null, w: r.width, h: r.height, gw: g ? g.width : 0, gh: g ? g.height : 0,
      inside: r.left >= -1 && r.right <= document.documentElement.clientWidth + 1,
      textContrast: label ? window.__uic.textContrast(label) : null, glyphContrast: svg ? window.__uic.textContrast(svg) : null,
      // the label sits beside the glyph, not pushed away by an inherited indent or a stretched row
      gap: label && g ? Math.abs(label.getBoundingClientRect().left - g.right) : 0, fs: parseFloat(getComputedStyle(b).fontSize),
    };
  });
  return {
    broken: imgs.filter((i) => i.complete && i.naturalWidth === 0).map((i) => i.getAttribute('src')),
    valid: imgs.filter((i) => i.complete && i.naturalWidth > 0).map((i) => ({ src: i.getAttribute('src').slice(0, 40), w: i.naturalWidth })),
    boxes, scrollW: document.documentElement.scrollWidth,
  };
}

// the page's scroll width with every stand-in taken out of the layout: the converter's own page CSS can already be wider
// than a window (reported by the chrome matrix), so what is judged is what the stand-ins add
function scrollWidthWithoutStandIns(box) {
  const s = document.createElement('style');
  s.textContent = box + '{display:none !important}';
  document.head.appendChild(s);
  const w = document.documentElement.scrollWidth;
  s.remove();
  return w;
}

// What every surface must satisfy once its failing pictures have been seen. `want` lists the alt texts in document order.
function verdict(p, want, drawing) {
  const bad = [];
  if (p.broken.length) bad.push(`${p.broken.length} picture(s) still draw the browser's broken-image icon: ${p.broken.join(', ')}`);
  if (p.boxes.length !== want.length) bad.push(`${p.boxes.length} stand-in(s), expected ${want.length}`);
  p.boxes.forEach((b, i) => {
    if (b.d !== drawing) bad.push(`stand-in ${i} does not draw the vendored content.image`);
    if (b.glyphHidden !== 'true') bad.push(`stand-in ${i}: the glyph is not decorative (aria-hidden)`);
    if (want[i] !== undefined && b.text !== want[i]) bad.push(`stand-in ${i} reads ${JSON.stringify(b.text)}, expected ${JSON.stringify(want[i])}`);
    if (b.gw < 12 || b.gh < 12) bad.push(`stand-in ${i}: the glyph measures ${f1(b.gw)} x ${f1(b.gh)} px`);
    if (!b.inside) bad.push(`stand-in ${i} leaves the window`);
    if (b.text && b.gap > b.fs) bad.push(`stand-in ${i}: the label is ${f1(b.gap)} px from the glyph (type ${f1(b.fs)} px)`);
    if (b.text && b.textContrast !== null && b.textContrast < 4.5) bad.push(`stand-in ${i}: text contrast ${f1(b.textContrast)}:1 (need 4.5)`);
    if (b.glyphContrast !== null && b.glyphContrast < 3) bad.push(`stand-in ${i}: glyph contrast ${f1(b.glyphContrast)}:1 (need 3)`);
  });
  return { ok: bad.length === 0, detail: bad.join('; ') || `${p.boxes.length} stand-in(s) [${p.boxes.map((b) => JSON.stringify(b.text)).join(', ')}], glyph ${f1(p.boxes[0] ? p.boxes[0].gw : 0)} px, text contrast ${f1(Math.min(...p.boxes.filter((b) => b.textContrast).map((b) => b.textContrast)))}:1, glyph contrast ${f1(Math.min(...p.boxes.map((b) => b.glyphContrast)))}:1, no broken-image icon left` };
}

// Opens a page, lets every picture settle, and reports what is on it plus how often each failing address was requested.
async function open(browser, url, { init, route, mode = { viewport: { width: 1280, height: 900 } } } = {}) {
  const ctx = await browser.newContext(mode);
  if (init) await ctx.addInitScript(init[0], init[1]);
  const page = await ctx.newPage();
  page.errors = [];
  page.on('pageerror', (e) => page.errors.push(e.message));
  page.asked = [];
  page.on('request', (r) => page.asked.push(r.url()));
  if (route) await route(page);
  await page.goto(url);
  return { ctx, page };
}
const timesAsked = (page, tail) => page.asked.filter((u) => u.endsWith(tail)).length;

async function settle(page, ms = 600) { await page.waitForLoadState('load'); await page.waitForTimeout(ms); }

// ---- one surface -----------------------------------------------------------------------------------------------------
async function surface(report, browser, E) {
  const drawing = vendored();
  const { ctx, page } = await open(browser, E.url, E);
  try {
    await settle(page);
    await page.evaluate(readerSource);
    const S = { box: E.box };
    const p = await page.evaluate(probe, S);
    const v = verdict(p, E.want, drawing);
    report.check(`reader.missing-image.stand-in/${E.name}`, v.ok, v.detail);
    report.check(`reader.missing-image.valid-picture-kept/${E.name}`, p.valid.length === 1 && p.valid[0].w === E.validWidth, `${p.valid.length} picture(s) still drawn as pictures: ${JSON.stringify(p.valid)}; the valid one is ${E.validWidth} px wide`);
    const without = await page.evaluate(scrollWidthWithoutStandIns, E.box);
    report.check(`reader.missing-image.adds-no-horizontal-scroll/${E.name}`, p.scrollW <= without + 1, `scrollWidth ${p.scrollW} with the stand-ins, ${without} with them out of the layout`);
    report.check(`reader.missing-image.page-error-free/${E.name}`, page.errors.length === 0, page.errors.join('; ') || 'no script error');
    await shot(page, E.shotsDir, E.name);

    // The failing addresses were each asked for once, by the browser's own load; the handler asked for nothing more.
    const first = timesAsked(page, E.gone);
    const late = '/late-' + Date.now() + '.png';
    await page.evaluate(([sel, src]) => {
      const img = document.createElement('img');
      img.src = src.slice(1); img.alt = 'Late picture';
      (document.querySelector(sel) || document.body).appendChild(img);
    }, [E.host, late]);
    await settle(page, 500);
    const again = timesAsked(page, E.gone);
    report.check(`reader.missing-image.no-request-from-the-handler/${E.name}`, first >= 1 && again === first && timesAsked(page, late.slice(1)) === 1, `${E.gone} asked for ${first} time(s) by the load, ${again} after the handler ran again; the picture inserted later was asked for ${timesAsked(page, late.slice(1))} time(s) (a count of 0 would mean this instrument saw nothing)`);

    // A picture inserted after load is caught too: the listener is on the document, in the capture phase.
    const afterLate = await page.evaluate(probe, S);
    report.check(`reader.missing-image.late-picture/${E.name}`, afterLate.boxes.length === E.want.length + 1 && afterLate.boxes[afterLate.boxes.length - 1].text === 'Late picture' && afterLate.broken.length === 0, `${afterLate.boxes.length} stand-ins after a picture was inserted that cannot load; the last reads ${JSON.stringify(afterLate.boxes.length ? afterLate.boxes[afterLate.boxes.length - 1].text : null)}`);

    if (E.ocrGuard) {
      await page.evaluate(([sel, cls]) => {
        const box = document.createElement('div');
        box.className = cls;
        box.innerHTML = '<img src="plated-gone.png" alt="Under plates">';
        (document.querySelector(sel) || document.body).appendChild(box);
      }, [E.host, E.ocrGuard]);
      await settle(page, 500);
      const kept = await page.evaluate((cls) => ({ img: document.querySelectorAll('.' + cls + ' img').length, standIn: document.querySelectorAll('.' + cls + ' ' + '[class*="missing-img"]').length }), E.ocrGuard);
      report.check(`reader.missing-image.ocr-plates-left-alone/${E.name}`, kept.img === 1 && kept.standIn === 0, `${kept.img} picture(s) and ${kept.standIn} stand-in(s) inside the plate container`);
    }
  } finally { await ctx.close(); }
}

// The page the engine drew, for a person to look at beside the measurements.
async function shot(page, dir, name) {
  if (!dir) return;
  fs.mkdirSync(dir, { recursive: true });
  const first = page.locator('[class*="missing-img"]').first();
  await first.scrollIntoViewIfNeeded().catch(() => {});
  await page.screenshot({ path: path.join(dir, `reader-missing-image-${name}.png`) });
}

// ---- the generated reader ---------------------------------------------------------------------------------------------
export async function missingImageGo(report, browser, shotsDir) {
  const pages = await buildPictureFixtures();
  const want = ['Cover', FIGURE, ''];
  for (const [name, file] of [['go-single', pages.single], ['go-chapter', pages.chapter]]) {
    if (!file) { report.check(`reader.missing-image.fixture/${name}`, false, 'the converted page with the failing pictures was not found'); continue; }
    const url = pathToFileURL(file).href;
    const E = { name, url, box: '.dht-missing-img', want, validWidth: 160, gone: '/images/gone.png', host: 'main.dht-single, body', ocrGuard: 'ocr-fig', shotsDir };
    await surface(report, browser, E);

    // Negative control: the same page with the handler's script cut out must be caught by the same verdict.
    const bare = file.replace(/\.html$/i, '.no-handler.html');
    fs.writeFileSync(bare, fs.readFileSync(file, 'utf8').replace(/<script id="dht-missing-image">[\s\S]*?<\/script>/, ''));
    const neg = await open(browser, pathToFileURL(bare).href);
    try {
      await settle(neg.page);
      await neg.page.evaluate(readerSource);
      const v = verdict(await neg.page.evaluate(probe, { box: E.box }), want, vendored());
      report.check(`reader.negative/missing-image-handler-removed/${name}`, !v.ok, v.ok ? 'the verdict passed a page with the handler cut out' : `caught: ${v.detail.slice(0, 110)}`);
    } finally { await neg.ctx.close(); fs.rmSync(bare, { force: true }); }
  }

  // A valid picture is never touched: the comic's pages are all valid.
  const comic = path.join(WORK, 'out', 'en-comic', 'comic', 'index.html');
  if (fs.existsSync(comic)) {
    const { ctx, page } = await open(browser, pathToFileURL(comic).href);
    try {
      await settle(page);
      await page.evaluate(readerSource);
      const p = await page.evaluate(probe, { box: '.dht-missing-img' });
      report.check('reader.missing-image.valid-pictures-untouched/go-comic', p.boxes.length === 0 && p.broken.length === 0 && p.valid.length >= 4, `${p.valid.length} valid pictures, ${p.boxes.length} stand-ins, ${p.broken.length} broken`);
    } finally { await ctx.close(); }
  }
}

// ---- the extension viewer -----------------------------------------------------------------------------------------------
// The document is an HTML file the viewer opens by address: its relative pictures cannot be reached from the viewer's
// origin, so they fail exactly as a document's own missing pictures do; the valid one rides inside as a data: URI.
export function writePicturesHtml() {
  const file = path.join(WORK, 'src', 'pictures.html');
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Pictures</title></head><body><h1>One</h1><p>Text before.</p>` +
    `<p><img src="data:image/png;base64,${validPng().toString('base64')}" alt="A valid picture"></p>` +
    `<p><img src="images/gone.png" alt="${FIGURE}"></p><p><img src="images/blank.png" alt=""></p><p>Text after.</p></body></html>`);
  return file;
}

export async function missingImageExt(report, browser, base, stub, shotsDir) {
  const want = [FIGURE, ''];
  const url = `${base}/src/viewer.html?file=${encodeURIComponent(base + '/__fixtures__/pictures.html')}`;
  const E = { name: 'ext', url, box: '.missing-img', want, validWidth: 160, gone: '/images/gone.png', host: '#content', init: [stub, ['en', {}]], shotsDir };
  await surface(report, browser, E);

  // Negative control: the viewer with its listener switched off must be caught by the same verdict.
  const src = fs.readFileSync(path.join(REPO, 'extension', 'src', 'missing-image.js'), 'utf8');
  const off = src.replace('doc.addEventListener("error"', 'doc.addEventListener("error-switched-off"');
  const neg = await open(browser, url, {
    init: [stub, ['en', {}]],
    route: (pg) => pg.route('**/src/missing-image.js', (r) => r.fulfill({ status: 200, contentType: 'text/javascript', body: off })),
  });
  try {
    await settle(neg.page);
    await neg.page.evaluate(readerSource);
    const v = verdict(await neg.page.evaluate(probe, { box: E.box }), want, vendored());
    report.check('reader.negative/missing-image-listener-off/ext', off !== src && !v.ok, off === src ? 'the neutralising replacement did not apply' : (v.ok ? 'the verdict passed a viewer with its listener switched off' : `caught: ${v.detail.slice(0, 110)}`));
  } finally { await neg.ctx.close(); }
}

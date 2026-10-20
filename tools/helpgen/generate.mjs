// The in-app help generator (ticket 118, CONTEXT-HELP rules 8, 13 and 14). From the copy deck, the GUI dictionary and
// the real build of doc-html-ui.exe it writes the whole help set: the pages, the screenshots with their pins, the
// shared parts, captures.json and freshness.json - into cmd/doc-html-ui/help/ (embedded in the exe) and, byte for
// byte, into help/ at the repository root (the site's copy).
//
//   node tools/helpgen/generate.mjs                      build the exe, capture, render, write both copies
//   node tools/helpgen/generate.mjs --locales en         a subset (the rest are kept as they are)
//   node tools/helpgen/generate.mjs --check              regenerate into temp/ and compare with the committed set
//   node tools/helpgen/generate.mjs --exe <path>         use an exe instead of building one
//
// Nothing here is edited by hand afterwards: the output is a render target (canon invariant 16).
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { REPO, loadPlaywright, launchBrowser, demoBooks, startApp } from './app.mjs';
import { loadDictionary } from './dict.mjs';
import { Pager, renderPage, renderIndex } from './render.mjs';
import { captureShot, profileSettings, BOOKS } from './shots.mjs';

const HERE = import.meta.dirname;
const argv = process.argv.slice(2);
const flag = (n) => argv.includes(n);
const opt = (n, d) => (argv.includes(n) ? argv[argv.indexOf(n) + 1] : d);

const MAP_PATH = path.join(REPO, 'cmd', 'doc-html-ui', 'help', 'help-map.json');
const HELP = path.join(REPO, 'cmd', 'doc-html-ui', 'help');
const OUT = flag('--check') ? path.join(REPO, 'temp', 'helpgen', 'check-out') : HELP;
const MIRROR = path.join(REPO, 'help');
const THEMES = ['dark', 'light'];
const PAGE_IDS = ['converter', 'settings', 'cost-question', 'secret-file-password', 'run-failed'];

const readJSON = (p) => JSON.parse(fs.readFileSync(p, 'utf8'));
const sha = (b) => crypto.createHash('sha256').update(b).digest('hex');

// The inputs a screenshot or a page depends on. The Go gate (internal/helpcheck) computes the same digests, so the
// two agree on what "stale" means: ui.html counts without its script blocks (what the window looks like, not what it
// does), line endings normalized.
export function inputDigests() {
  const lf = (s) => s.replace(/\r\n/g, '\n');
  const ui = lf(fs.readFileSync(path.join(REPO, 'cmd', 'doc-html-ui', 'ui.html'), 'utf8')).replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, '');
  const parts = {
    'ui.html (markup and style)': sha(ui),
    'i18n.js': sha(lf(fs.readFileSync(path.join(REPO, 'cmd', 'doc-html-ui', 'i18n.js'), 'utf8'))),
    'help-map.json': sha(lf(fs.readFileSync(MAP_PATH, 'utf8'))),
    'sza-kit.css': sha(fs.readFileSync(path.join(REPO, 'assets', 'sza-kit.css'))),
    'help.css': sha(fs.readFileSync(path.join(HERE, 'vendor', 'help.css'))),
  };
  for (const f of fs.readdirSync(path.join(HERE, 'copy')).sort()) {
    parts['copy/' + f] = sha(lf(fs.readFileSync(path.join(HERE, 'copy', f), 'utf8')));
  }
  for (const f of ['generate.mjs', 'render.mjs', 'shots.mjs', 'dict.mjs', 'page.js']) {
    parts['generator/' + f] = sha(lf(fs.readFileSync(path.join(HERE, f), 'utf8')));
  }
  const names = Object.keys(parts).sort();
  const digest = sha(names.map((n) => `${n}\0${parts[n]}\n`).join(''));
  return { parts, digest };
}

function buildExe(version) {
  const bin = path.join(REPO, 'temp', 'helpgen', 'bin');
  fs.mkdirSync(bin, { recursive: true });
  const env = { ...process.env, GOFLAGS: '-tags=nodynamic' };
  // The converter goes beside the window, so the About group reads "found" the way an installed copy does.
  execFileSync('go', ['build', '-ldflags', `-X main.Version=${version}`, '-o', path.join(bin, 'doc-html-ui.exe'), './cmd/doc-html-ui'], { cwd: REPO, env, stdio: 'inherit' });
  execFileSync('go', ['build', '-ldflags', `-X main.Version=${version}`, '-o', path.join(bin, 'doc-html-translate.exe'), './cmd/doc-html-translate'], { cwd: REPO, env, stdio: 'inherit' });
  // The bundled English OCR data sits beside the exe in an installed copy; without it the OCR group would print
  // "English OCR data is not installed", which is true of this scratch folder and of no user's window.
  const tess = path.join(REPO, 'build', 'tessdata');
  if (fs.existsSync(tess)) fs.cpSync(tess, path.join(bin, 'tessdata'), { recursive: true });
  else console.warn('warning: build/tessdata is missing; the OCR group will show the missing-data line (run scripts/build.ps1 once)');
  return path.join(bin, 'doc-html-ui.exe');
}

function checkSources(map, pages, locales) {
  for (const p of pages) {
    const s = map.surfaces.find((x) => x.id === p.surface);
    if (!s) throw new Error(`page ${p.id}: the help map has no surface ${p.surface}`);
    if (s.help !== p.id) throw new Error(`page ${p.id}: the map gives surface ${s.id} the help id ${s.help}`);
    const ids = new Set(p.elements.map((e) => e.id));
    for (const row of s.pages || []) {
      if (!ids.has(row.anchor)) throw new Error(`page ${p.id}: the map's anchor ${row.anchor} (${row.id}) is no element of the page`);
    }
    const shots = new Set((p.shots || []).map((x) => x.id));
    for (const e of p.elements) {
      if (!shots.has(e.shot)) throw new Error(`page ${p.id}: element ${e.id} names no shot (${e.shot})`);
      if (!e.sel) throw new Error(`page ${p.id}: element ${e.id} has no selector`);
    }
  }
}

async function main() {
  const map = readJSON(MAP_PATH);
  const locales = (opt('--locales', map.locales.join(',')).split(',')).filter((l) => map.locales.includes(l));
  const only = opt('--only', '') ? opt('--only').split(',') : PAGE_IDS;
  const chrome = readJSON(path.join(HERE, 'copy', 'chrome.json')).locales;
  const pages = PAGE_IDS.map((id) => readJSON(path.join(HERE, 'copy', `${id}.json`)));
  const index = readJSON(path.join(HERE, 'copy', 'index.json'));
  const dict = loadDictionary();
  checkSources(map, pages, locales);

  const now = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  // --check reuses the committed set's version and date, so the comparison is of like with like
  const committed = fs.existsSync(path.join(HELP, 'captures.json')) ? readJSON(path.join(HELP, 'captures.json')) : {};
  const pinned = flag('--check') ? committed : {};
  const date = opt('--date', pinned.date || `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`);
  const version = opt('--version', pinned.version || `${String(now.getFullYear()).slice(2)}.${pad(now.getMonth() + 1)}${pad(now.getDate())}.${pad(now.getHours())}${pad(now.getMinutes())}`);

  const exe = opt('--exe', null) ? path.resolve(opt('--exe')) : buildExe(version);
  const root = path.join(REPO, 'temp', 'helpgen');
  const books = demoBooks(root, BOOKS).map((p) => { fs.writeFileSync(p, 'demo document\n'.repeat(300)); return p; });
  const ctx = { root, books };
  const settingsFor = (locale) => (theme) => profileSettings({ lang: locale, theme, books });

  fs.mkdirSync(OUT, { recursive: true });
  // Every file this run writes, so a page or image that no source produces any more is removed with the rest of its set.
  const written = new Set();
  const put = (rel, data) => {
    const f = path.join(OUT, rel);
    fs.mkdirSync(path.dirname(f), { recursive: true });
    fs.writeFileSync(f, data);
    written.add(rel.split(path.sep).join('/'));
  };

  const capturesAll = {};
  const app = await startApp(exe, path.join(root, 'profile'));
  const { chromium } = await loadPlaywright();
  const browser = await launchBrowser(chromium);
  try {
    for (const locale of locales) {
      const pg = new Pager({ dict, chrome: chrome[locale], locale, source: map.sourceLocale });
      const ids = new Set(PAGE_IDS);
      for (const page of pages) {
        if (!only.includes(page.id)) continue;
        const shotData = {};
        for (const shot of page.shots) {
          const elements = page.elements.filter((e) => e.shot === shot.id);
          const per = {};
          for (const theme of THEMES) {
            const ctxPage = await browser.newPage({ viewport: { width: 1160, height: 760 }, deviceScaleFactor: 1, reducedMotion: 'reduce', colorScheme: theme });
            try {
              per[theme] = await captureShot({
                page: ctxPage, base: app.base, locale, theme, page_: page, shot, elements, ctx, settingsFor: settingsFor(locale),
              });
            } finally { await ctxPage.close(); }
          }
          // the layout is the theme's business only in colour: both captures must agree on size and pin places
          const a = per.dark, b = per.light;
          if (a.w !== b.w || a.h !== b.h) throw new Error(`${locale}/${page.id}/${shot.id}: the two themes differ in size (${a.w}x${a.h} vs ${b.w}x${b.h})`);
          for (const e of elements) {
            if (Math.abs(a.pins[e.id].x - b.pins[e.id].x) > 0.3 || Math.abs(a.pins[e.id].y - b.pins[e.id].y) > 0.3) {
              throw new Error(`${locale}/${page.id}/${shot.id}: pin ${e.id} moves between themes`);
            }
          }
          const base = `${locale}/${page.id}-${shot.id}`;
          put(`${base}-dark.png`, a.png);
          put(`${base}-light.png`, b.png);
          shotData[shot.id] = { w: a.w, h: a.h, pins: a.pins, image: { dark: `${page.id}-${shot.id}-dark.png`, light: `${page.id}-${shot.id}-light.png` } };
          console.log(`captured ${locale}/${page.id}/${shot.id} ${a.w}x${a.h}`);
        }
        capturesAll[`${locale}/${page.id}`] = shotData;
        put(`${locale}/${page.id}.html`, renderPage({ page, pg, pages: ids, captures: shotData, version, date }));
      }
      put(`${locale}/index.html`, renderIndex({ index, pg, pages: new Set(PAGE_IDS), version, date }));
    }
  } finally {
    await browser.close();
    app.stop();
  }

  // The shared parts, and the records the gate and a reader use.
  put('sza-kit.css', fs.readFileSync(path.join(REPO, 'assets', 'sza-kit.css')));
  put('help.css', fs.readFileSync(path.join(HERE, 'vendor', 'help.css')));
  put('page.js', fs.readFileSync(path.join(HERE, 'page.js')));
  const prior = fs.existsSync(path.join(OUT, 'captures.json')) ? readJSON(path.join(OUT, 'captures.json')) : { captures: {} };
  put('captures.json', JSON.stringify({
    about: 'Written by tools/helpgen/generate.mjs. Per locale and page: the captures, their size and the pin place of every element in percent of the image, taken from the real window (CONTEXT-HELP rule 14).',
    version, date,
    captures: { ...prior.captures, ...capturesAll },
  }, null, 2) + '\n');
  const { parts, digest } = inputDigests();
  put('freshness.json', JSON.stringify({
    about: 'The digests of everything a help page or image is made from. The gate recomputes them; a difference means the set was not regenerated after an input changed (CONTEXT-HELP rule 18).',
    digest, inputs: parts,
  }, null, 2) + '\n');
  if (!flag('--check') && !argv.includes('--only') && locales.length === map.locales.length) {
    // the set is whole: nothing a source no longer makes may stay behind
    const keep = new Set([...written, 'help-map.json']);
    for (const f of walk(OUT)) {
      const rel = path.relative(OUT, f).split(path.sep).join('/');
      if (!keep.has(rel)) { fs.rmSync(f); console.log('removed stale', rel); }
    }
  }
  if (flag('--check')) {
    const diffs = [];
    for (const f of walk(OUT)) {
      const rel = path.relative(OUT, f);
      const other = path.join(HELP, rel);
      if (!fs.existsSync(other)) diffs.push(rel + ' (not in the committed set)');
      else if (!fs.readFileSync(f).equals(fs.readFileSync(other))) diffs.push(rel);
    }
    for (const f of walk(HELP)) {
      const rel = path.relative(HELP, f);
      if (rel !== 'help-map.json' && !fs.existsSync(path.join(OUT, rel)) && !(only.length < PAGE_IDS.length)) diffs.push(rel + ' (no source makes it any more)');
    }
    console.log(diffs.length ? 'check: ' + diffs.length + ' file(s) differ from the committed set:\n  ' + diffs.join('\n  ') : 'check: the committed set is what the generator makes now');
    process.exitCode = diffs.length ? 1 : 0;
  }
  if (!flag('--check')) {
    fs.rmSync(MIRROR, { recursive: true, force: true });
    copyTree(OUT, MIRROR);
  }
  console.log(`help set written: ${written.size} files, version ${version}, ${date}`);
}

function* walk(dir) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) yield* walk(p); else yield p;
  }
}
function copyTree(from, to) {
  for (const f of walk(from)) {
    const t = path.join(to, path.relative(from, f));
    fs.mkdirSync(path.dirname(t), { recursive: true });
    fs.copyFileSync(f, t);
  }
}

if (import.meta.url === new URL(`file:///${process.argv[1].replace(/\\/g, '/')}`).href) {
  main().catch((e) => { console.error(e); process.exitCode = 1; });
}

// The fast form of the in-app help generator (ticket E0125). It re-renders the pages and the indexes from the copy
// deck, the GUI dictionary and the captures that are already committed - no build of the exe, no browser - so a
// change of the help's wording costs a fraction of a second instead of the full generator's few minutes.
//
//   node tools/helpgen/repage.mjs             re-render, write what differs into both copies, refresh freshness.json
//   node tools/helpgen/repage.mjs --check     re-render in memory and report what differs from the committed set
//
// It never invents a screenshot. When an input of the window (ui.html markup or style, the dictionary, the capture
// code) has changed since the last capture, the pages it would write and the images beside them would disagree, so it
// writes nothing and exits 2: that case needs `node tools/helpgen/generate.mjs`. Exit 1 is a difference under --check or
// an error; exit 0 is a render that matches or was written.
//
// What this form cannot see: a change of a selector, a pin mode, a setup or a capture kind in the copy deck. Those
// fields decide where a pin sits, and only a capture can move one; the release runs `generate.mjs --check`, which
// compares this form's output with the full generator's byte for byte, so such an edit cannot ship unnoticed.
import fs from 'node:fs';
import path from 'node:path';
import { REPO } from './app.mjs';
import { loadDictionary } from './dict.mjs';
import { Pager, renderPage, renderIndex } from './render.mjs';
import { inputDigests } from './generate.mjs';

const HERE = import.meta.dirname;
const argv = process.argv.slice(2);
const check = argv.includes('--check');

const HELP = path.join(REPO, 'cmd', 'doc-html-ui', 'help');
const MIRROR = path.join(REPO, 'help');
const MAP_PATH = path.join(HELP, 'help-map.json');
// The same list as generate.mjs; the guard below refuses a copy deck or a capture set that disagrees with it.
const PAGE_IDS = ['converter', 'settings', 'cost-question', 'secret-file-password', 'run-failed'];
// The inputs a screenshot is made from: what the window looks like and the code that photographs it. Everything else
// in the digest list only shapes the text of a page.
const SCREEN_PARTS = ['ui.html (markup and style)', 'i18n.js', 'generator/generate.mjs', 'generator/shots.mjs'];

const readJSON = (p) => JSON.parse(fs.readFileSync(p, 'utf8'));
const same = (a, b) => fs.existsSync(b) && fs.readFileSync(b).equals(a);

class Refused extends Error {}

function checkSources(map, pages) {
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

function main() {
  const recorded = readJSON(path.join(HELP, 'freshness.json'));
  const { parts, digest } = inputDigests();
  const moved = SCREEN_PARTS.filter((n) => recorded.inputs[n] !== parts[n]);
  if (moved.length) {
    throw new Refused(`${moved.join(', ')} changed after the last capture, so new pages would disagree with the screenshots beside them; run node tools/helpgen/generate.mjs (it builds the GUI)`);
  }

  const map = readJSON(MAP_PATH);
  const committed = readJSON(path.join(HELP, 'captures.json'));
  const copyIds = fs.readdirSync(path.join(HERE, 'copy')).filter((f) => f.endsWith('.json')).map((f) => f.slice(0, -5))
    .filter((id) => id !== 'chrome' && id !== 'index').sort();
  if (copyIds.join() !== [...PAGE_IDS].sort().join()) {
    throw new Refused(`the copy deck holds the pages ${copyIds.join(', ')} and this form renders ${PAGE_IDS.join(', ')}; run node tools/helpgen/generate.mjs`);
  }
  const chrome = readJSON(path.join(HERE, 'copy', 'chrome.json')).locales;
  const pages = PAGE_IDS.map((id) => readJSON(path.join(HERE, 'copy', `${id}.json`)));
  const index = readJSON(path.join(HERE, 'copy', 'index.json'));
  const dict = loadDictionary();
  checkSources(map, pages);

  const { version, date } = committed;
  const out = new Map();
  const ids = new Set(PAGE_IDS);
  for (const locale of map.locales) {
    const pg = new Pager({ dict, chrome: chrome[locale], locale, source: map.sourceLocale });
    const held = Object.keys(committed.captures).filter((k) => k.startsWith(`${locale}/`)).map((k) => k.slice(locale.length + 1)).sort();
    if (held.join() !== [...PAGE_IDS].sort().join()) {
      throw new Refused(`the committed captures of ${locale} hold ${held.join(', ')}, not the pages ${PAGE_IDS.join(', ')}; run node tools/helpgen/generate.mjs`);
    }
    for (const page of pages) {
      const shotData = committed.captures[`${locale}/${page.id}`];
      for (const shot of page.shots) {
        if (!shotData[shot.id]) throw new Refused(`${locale}/${page.id}: no capture of the shot ${shot.id}; run node tools/helpgen/generate.mjs`);
      }
      for (const e of page.elements) {
        if (!shotData[e.shot].pins[e.id]) throw new Refused(`${locale}/${page.id}: element ${e.id} has no captured pin; run node tools/helpgen/generate.mjs`);
      }
      out.set(`${locale}/${page.id}.html`, Buffer.from(renderPage({ page, pg, pages: ids, captures: shotData, version, date })));
    }
    out.set(`${locale}/index.html`, Buffer.from(renderIndex({ index, pg, pages: ids, version, date })));
  }
  out.set('sza-kit.css', fs.readFileSync(path.join(REPO, 'assets', 'sza-kit.css')));
  out.set('help.css', fs.readFileSync(path.join(HERE, 'vendor', 'help.css')));
  out.set('page.js', fs.readFileSync(path.join(HERE, 'page.js')));
  out.set('freshness.json', Buffer.from(JSON.stringify({ about: recorded.about, digest, inputs: parts }, null, 2) + '\n'));

  const differ = [...out].filter(([rel, data]) => !same(data, path.join(HELP, rel)) || !same(data, path.join(MIRROR, rel))).map(([rel]) => rel);
  if (check) {
    console.log(differ.length ? `repage --check: ${differ.length} of ${out.size} file(s) differ from the committed set:\n  ${differ.join('\n  ')}` : `repage --check: the ${out.size} rendered files equal the committed set`);
    process.exitCode = differ.length ? 1 : 0;
    return;
  }
  for (const rel of differ) {
    for (const root of [HELP, MIRROR]) {
      const f = path.join(root, rel);
      fs.mkdirSync(path.dirname(f), { recursive: true });
      fs.writeFileSync(f, out.get(rel));
    }
  }
  console.log(`repage: ${differ.length} of ${out.size} file(s) written (version ${version}, ${date}); screenshots untouched`);
}

try {
  main();
} catch (e) {
  if (e instanceof Refused) { console.error('repage: COULD NOT RENDER - ' + e.message); process.exitCode = 2; }
  else { console.error(e); process.exitCode = 1; }
}

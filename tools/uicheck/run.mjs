// UI-contract checks of the GUI launcher (ticket 119): BUTTONS 2.0, BOOLEAN-OPTION 2.0, COLLAPSIBLE-GROUP 1.1,
// SETTINGS-NAVIGATION 2.0 and UI-SIZING 1.1 measured on the real build, in a headless Chromium, under an isolated
// profile (nothing of the developer's own settings is touched).
//
//   node tools/uicheck/run.mjs                 the whole matrix (a few minutes; the timings take most of it)
//   node tools/uicheck/run.mjs --quick         the baseline set, no timing samples
//   node tools/uicheck/run.mjs --only options  one scenario: buttons | dialogs | options | groups | nav | boot | timing | captures | site | reader | extension
//                                              (site measures the official site from the working tree, no exe needed; reader converts fixtures with the CLI and measures the generated reader chrome and the extension viewer twin; --go-only skips the extension; extension measures the browser extension's own pages - options, popup, image-OCR page and the in-page OCR bar - on the real extension/ folder, no exe needed)
//   node tools/uicheck/run.mjs --no-build      reuse temp/uicheck/bin
//
// Output: temp/uicheck/<stamp>/report.json (every check with its measured detail), timings.raw.json (every sample),
// screenshots/, and the last line is the CHECK-VERDICT line (exit 0 PASS, 1 FAIL, 2 COULD NOT VERIFY, 3 PASS WITH ADVISORIES).
// Needs the Go toolchain (it builds the two exes) and `npm ci` in tools/sitecheck (Playwright). Not part of the shipped
// product; the tree is what is measured, so a run says which tree it saw.
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { REPO, Report, openSession, writeJSON } from './lib.mjs';
import * as C from './checks.mjs';
import { siteChecks } from './site.mjs';
import { readerChecks } from './reader.mjs';
import { extensionChecks } from './extension.mjs';

const args = process.argv.slice(2);
const flag = (n) => args.includes('--' + n);
const only = args.includes('--only') ? args[args.indexOf('--only') + 1] : null;
const quick = flag('quick');
const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
const outDir = path.join(REPO, 'temp', 'uicheck', stamp);
const binDir = path.join(REPO, 'temp', 'uicheck', 'bin');
fs.mkdirSync(path.join(outDir, 'screenshots'), { recursive: true });

function build() {
  if (flag('no-build') && fs.existsSync(path.join(binDir, 'doc-html-ui.exe'))) return;
  fs.mkdirSync(binDir, { recursive: true });
  const env = { ...process.env, GOFLAGS: '-tags=nodynamic' };
  for (const [pkg, exe] of [['./cmd/doc-html-ui', 'doc-html-ui.exe'], ['./cmd/doc-html-translate', 'doc-html-translate.exe']]) {
    execFileSync('go', ['build', '-o', path.join(binDir, exe), pkg], { cwd: REPO, env, stdio: 'inherit' });
  }
}

const treeId = () => {
  try { return execFileSync('git', ['write-tree'], { cwd: REPO, encoding: 'utf8' }).trim(); } catch { return 'unknown'; }
};

// the baseline plus risk-selected combinations (UI-SIZING section 4: no full Cartesian product on every change)
const BUTTON_MATRIX = [
  { size: 'ordinary', input: 'pointer' },
  { size: 'ordinary', input: 'touch' },
  { size: 'compact', input: 'pointer' },
  { size: 'compact', input: 'touch' },
  { size: 'xlarge', input: 'pointer' },
  { size: 'xlarge', input: 'touch' },
  { size: 'ordinary', input: 'pointer', lang: 'ru' },
  { size: 'ordinary', input: 'pointer', lang: 'de', w: 1000 },
  { size: 'ordinary', input: 'pointer', lang: 'ar' },
  { size: 'ordinary', input: 'pointer', lang: 'hi', w: 900 },
  { size: 'ordinary', input: 'pointer', text: 'huge', lang: 'de', w: 1000 },
  { size: 'huge', input: 'pointer', w: 1366 },
  { size: 'huge', input: 'touch', text: 'huge', lang: 'ru', w: 1366 },
  { size: 'ordinary', input: 'pointer', w: 700 },
  { size: 'ordinary', input: 'pointer', page: 'about' },
  { size: 'ordinary', input: 'pointer', page: 'ocr' },
  { size: 'xlarge', input: 'touch', page: 'about', lang: 'de' },
];

const CAPTURES = [
  { name: 'convert-ordinary-light', q: { page: 'convert' } },
  { name: 'convert-ordinary-dark', q: { page: 'convert' }, scheme: 'dark', theme: 'dark' },
  { name: 'convert-compact-touch', q: { size: 'compact', input: 'touch' } },
  { name: 'convert-xlarge-pointer', q: { size: 'xlarge' } },
  { name: 'convert-huge-text-de', q: { text: 'huge', lang: 'de' }, w: 1100 },
  { name: 'convert-rtl-ar', q: { lang: 'ar' } },
  { name: 'convert-narrow-drawer-closed', q: {}, w: 700, h: 800 },
  { name: 'convert-narrow-drawer-open', q: {}, w: 700, h: 800, click: '#btnPages' },
  { name: 'about-ordinary-light', q: { page: 'about' } },
  { name: 'about-ordinary-dark-ru', q: { page: 'about', lang: 'ru' }, scheme: 'dark', theme: 'dark' },
  { name: 'ocr-option-help-open', q: { page: 'ocr' }, click: '#page-ocr .opt-help' },
  { name: 'interface-ordinary', q: { page: 'interface' } },
  { name: 'about-forced-colors', q: { page: 'about' }, forced: 'active' },
  { name: 'convert-forced-colors', q: {}, forced: 'active' },
];

async function captures(session) {
  for (const c of CAPTURES) {
    const { page, close } = await session.page({ q: c.q, w: c.w || 1366, h: c.h || 768, scheme: c.scheme || 'light', forced: c.forced || 'none' });
    try {
      if (c.theme) {
        // The theme select lives on the Interface page; go there, choose, and come back to the page under test.
        await page.click('#tab-interface');
        await page.selectOption('#themeSwitch', c.theme).catch(() => {});
        await page.click('#tab-' + (c.q.page || 'convert'));
      }
      if (c.click) await page.click(c.click);
      await page.waitForTimeout(150);
      await page.screenshot({ path: path.join(outDir, 'screenshots', c.name + '.png') });
    } finally { await close(); }
  }
}

const report = new Report('uicheck');
if (only === 'site' || only === 'reader' || only === 'extension') {
  if (only === 'site') await siteChecks(report);
  else if (only === 'extension') await extensionChecks(report, { shotsDir: path.join(outDir, 'screenshots'), outDir });
  else await readerChecks(report, { build: !flag('no-build'), onlyGo: flag('go-only'), shotsDir: path.join(outDir, 'screenshots') });
  const v = report.verdict();
  writeJSON(path.join(outDir, 'report.json'), { tree: treeId(), when: new Date().toISOString(), node: process.version, verdict: v.line, only, rows: report.rows });
  for (const r of report.failed) console.log('FAIL  ' + r.id + ': ' + r.detail);
  console.log('report: ' + path.join(outDir, 'report.json'));
  console.log(v.line);
  process.exit(v.code);
}
try { build(); } catch (e) { console.log('uicheck: COULD NOT VERIFY (build failed: ' + e.message.split('\n')[0] + ')'); process.exit(2); }
let session;
try {
  session = await openSession({ exe: path.join(binDir, 'doc-html-ui.exe'), root: path.join(REPO, 'temp', 'uicheck', 'work') });
} catch (e) { console.log('uicheck: COULD NOT VERIFY (' + e.message.split('\n')[0] + ')'); process.exit(2); }
const want = (n) => !only || only === n;
try {
  if (want('buttons')) for (const m of (quick ? BUTTON_MATRIX.slice(0, 8) : BUTTON_MATRIX)) await C.buttonsGeometry(session, report, m);
  if (want('dialogs')) await C.dialogs(session, report);
  if (want('options')) { await C.options(session, report); if (!quick) for (const m of [{ size: 'compact', input: 'pointer' }, { size: 'xlarge', input: 'touch' }, { size: 'ordinary', input: 'touch' }]) await C.options(session, report, m); }
  if (want('groups')) await C.groups(session, report);
  if (want('nav')) await C.navigation(session, report);
  if (want('boot')) await C.boot(session, report);
  if (want('timing') && !quick) await C.timings(session, report, outDir);
  if (want('captures')) await captures(session);
  if (!only) await siteChecks(report);
} finally {
  await session.close();
}
const v = report.verdict();
writeJSON(path.join(outDir, 'report.json'), { tree: treeId(), when: new Date().toISOString(), node: process.version, chromium: 'Playwright Chromium (headless)', verdict: v.line, quick, only, rows: report.rows });
for (const r of report.failed) console.log('FAIL  ' + r.id + ': ' + r.detail);
for (const r of report.advisories) console.log('ADVISORY  ' + r.id + ': ' + r.detail);
console.log('report: ' + path.join(outDir, 'report.json'));
console.log(v.line);
process.exit(v.code);

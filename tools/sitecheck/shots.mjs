// Captures of the doc-html-ui window for the manual's task sections (ticket 107): the "About the program" block
// (#logs) and the masked-password dialog of a FileDO secret file (#secret-files), one PNG per page language.
// The window is a local HTTP app, so a capture is the real page of the real build, as tools/store/make-gui-screenshot.ps1
// takes it. The password dialog is opened through the page's own openDialog() with the converter's own strings
// (internal/fdsec/credential.go, internal/i18n/i18n_fdsec.go); nothing is typed and no run is started.
//
//   go build -ldflags "-X main.Version=<release>" -o temp/docs-shots/doc-html-ui.exe ./cmd/doc-html-ui
//   node shots.mjs --ui ../../temp/docs-shots/doc-html-ui.exe     writes tools/store/docs-<kind>-<lang>.png, lang = en-us / ru / uk
import { chromium } from 'playwright-core';
import { spawn, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { launch, REPO } from './lib.mjs';

const argv = process.argv.slice(2);
const uiPath = argv.includes('--ui') ? argv[argv.indexOf('--ui') + 1] : path.join(REPO, 'build', 'doc-html-ui.exe');
const OUT = path.join(REPO, 'tools', 'store');
const LANGS = [['en', 'en-us'], ['ru', 'ru'], ['uk', 'uk']];
// Converter strings, as the secret-file question shows them (internal/i18n/i18n_fdsec.go).
const SECRET = {
  en: { title: 'FileDO secret file', message: 'Enter the password for diary.fd-sec.' },
  ru: { title: 'Секретный файл FileDO', message: 'Введите пароль для diary.fd-sec.' },
  uk: { title: 'Секретний файл FileDO', message: 'Введіть пароль для diary.fd-sec.' },
};

if (!fs.existsSync(uiPath)) throw new Error(`doc-html-ui.exe not found at ${uiPath}; build it first (scripts/build-ui.ps1) or pass --ui`);

// The app keeps its settings per user; put them back afterwards so a capture leaves the GUI as it was.
const settings = path.join(process.env.LOCALAPPDATA, 'doc-html-translate', 'ui-settings.json');
const backup = fs.existsSync(settings) ? fs.readFileSync(settings) : null;
const app = spawn(uiPath, [], { stdio: 'ignore', detached: false });
let browser;
try {
  // It never prints its port; the OS connection table is the honest source.
  let port = null;
  for (let i = 0; i < 100 && !port; i++) {
    await new Promise((r) => setTimeout(r, 200));
    const out = execFileSync('powershell', ['-NoProfile', '-Command',
      `(Get-NetTCPConnection -State Listen -OwningProcess ${app.pid} -ErrorAction SilentlyContinue | Where-Object { $_.LocalAddress -eq '127.0.0.1' } | Select-Object -First 1).LocalPort`], { encoding: 'utf8' }).trim();
    if (out) port = Number(out);
  }
  if (!port) throw new Error('doc-html-ui did not start listening on 127.0.0.1');
  browser = await launch();
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 768 }, deviceScaleFactor: 1, reducedMotion: 'reduce', colorScheme: 'dark' });
  for (const [lang, locale] of LANGS) {
    const page = await ctx.newPage();
    await page.goto(`http://127.0.0.1:${port}/?lang=${lang}`, { waitUntil: 'load' });
    await page.waitForFunction(() => document.getElementById('aboutVer') && document.getElementById('aboutVer').textContent.trim() !== '', null, { timeout: 15000 });
    // Dark, as the store captures are; set on the page only, the saved theme is not touched
    await page.evaluate(() => { document.documentElement.dataset.theme = 'dark'; });
    // About block, opened
    await page.evaluate(() => { document.getElementById('aboutSection').open = true; });
    await page.locator('#aboutSection').scrollIntoViewIfNeeded();
    const about = page.locator('#aboutSection');
    const file = path.join(OUT, `docs-about-${locale}.png`);
    await about.screenshot({ path: file });
    console.log(file, JSON.stringify(await about.boundingBox()));
    // Password dialog of a secret file
    await page.evaluate((s) => { openDialog({ title: s.title, message: s.message, secret: true,
      buttons: [{ id: 'open', key: 'btnOpenSecret', kind: 'primary' }, { id: 'cancel', key: 'btnCancel' }], cancel: 'cancel' }); }, SECRET[lang]);
    await page.waitForSelector('#dlg[open]');
    const dlg = page.locator('#dlg');
    const dfile = path.join(OUT, `docs-password-${locale}.png`);
    await dlg.screenshot({ path: dfile });
    console.log(dfile, JSON.stringify(await dlg.boundingBox()));
    await page.close();
  }
} finally {
  if (browser) await browser.close();
  try { execFileSync('taskkill', ['/PID', String(app.pid), '/T', '/F'], { stdio: 'ignore' }); } catch {}
  if (backup) fs.writeFileSync(settings, backup);
}

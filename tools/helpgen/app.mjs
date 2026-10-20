// The real doc-html-ui.exe under an isolated profile, for the help generator and the walk (ticket 118).
// The window is a local HTTP app, so a capture is the real page of the real build. Nothing here reads or
// writes the developer's own settings: LOCALAPPDATA points at a scratch profile, the app opens no window
// (DOCHT_GUI_NO_WINDOW), and the scratch profile holds only demo content.
import { spawn, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

export const REPO = path.resolve(import.meta.dirname, '..', '..');

// playwright-core is installed once for the site checklist (tools/sitecheck); the generator reuses it.
export async function loadPlaywright() {
  const entry = path.join(REPO, 'tools', 'sitecheck', 'node_modules', 'playwright-core', 'index.mjs');
  if (!fs.existsSync(entry)) throw new Error('playwright-core is missing: run `npm ci` in tools/sitecheck');
  return import(pathToFileURL(entry).href);
}

export async function launchBrowser(chromium) {
  const exe = process.env.SITECHECK_CHROME ||
    path.join(process.env.LOCALAPPDATA, 'ms-playwright', 'chromium-1223', 'chrome-win64', 'chrome.exe');
  return chromium.launch({ executablePath: exe, headless: true });
}

// demoProfile writes the scratch profile: the settings file of a person who has just installed the app
// and chosen nothing but the window's look. Recent entries point at the demo files, which exist so the
// list is not greyed out.
export function demoProfile(root, settings) {
  const profile = path.join(root, 'profile');
  fs.rmSync(profile, { recursive: true, force: true });
  const dir = path.join(profile, 'doc-html-translate');
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(dir, 'ui-settings.json'), JSON.stringify(settings));
  return profile;
}

export function demoBooks(root, names) {
  const dir = path.join(root, 'demo', 'Books');
  fs.mkdirSync(dir, { recursive: true });
  return names.map((n) => { const p = path.join(dir, n); fs.writeFileSync(p, ''); return p; });
}

// startApp launches the exe and returns its base URL. The app never prints its port; the OS connection
// table is the honest source (the way tools/sitecheck/shots.mjs finds it).
export async function startApp(exe, profile) {
  if (!fs.existsSync(exe)) throw new Error(`doc-html-ui.exe not found at ${exe}; build it first`);
  const env = { ...process.env, LOCALAPPDATA: profile, DOCHT_GUI_NO_WINDOW: '1' };
  const app = spawn(exe, [], { env, stdio: 'ignore', detached: false });
  let port = null;
  for (let i = 0; i < 100 && !port; i++) {
    await new Promise((r) => setTimeout(r, 200));
    const out = execFileSync('powershell', ['-NoProfile', '-Command',
      `(Get-NetTCPConnection -State Listen -OwningProcess ${app.pid} -ErrorAction SilentlyContinue | Where-Object { $_.LocalAddress -eq '127.0.0.1' } | Select-Object -First 1).LocalPort`], { encoding: 'utf8' }).trim();
    if (out) port = Number(out);
  }
  if (!port) { app.kill(); throw new Error('doc-html-ui did not start listening on 127.0.0.1'); }
  return {
    base: `http://127.0.0.1:${port}`,
    stop() { try { execFileSync('taskkill', ['/PID', String(app.pid), '/T', '/F'], { stdio: 'ignore' }); } catch { /* already gone */ } },
  };
}

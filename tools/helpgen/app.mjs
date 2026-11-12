// The real doc-html-ui.exe under an isolated profile, for the help generator and the walk (ticket 118).
// The window is a local HTTP app, so a capture is the real page of the real build. Nothing here reads or
// writes the developer's own settings: LOCALAPPDATA points at a scratch profile, the app opens no window
// (DOCHT_GUI_NO_WINDOW), and the scratch profile holds only demo content.
import { spawn, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import net from 'node:net';
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

// The listening port of one process, from the OS connection table; null when it lists none.
function tablePort(pid) {
  const out = execFileSync('powershell', ['-NoProfile', '-Command',
    `(Get-NetTCPConnection -State Listen -OwningProcess ${pid} -ErrorAction SilentlyContinue | Where-Object { $_.LocalAddress -eq '127.0.0.1' } | Select-Object -First 1).LocalPort`], { encoding: 'utf8' }).trim();
  return out ? Number(out) : null;
}

// The table is the honest source of the app's port, except where the shell running this tool sits in a
// network compartment the table does not list (measured 2026-11-12 in an agent's shell: its own listener
// was invisible to Get-NetTCPConnection and netstat while a client could still connect to it). So the table
// is first asked about a listener of this very process; only where it is blind does startApp fall back to
// the probe below.
async function tableSeesOwnListener() {
  const srv = net.createServer();
  await new Promise((resolve) => srv.listen(0, '127.0.0.1', resolve));
  try { return tablePort(process.pid) === srv.address().port; } finally { srv.close(); }
}

// The loopback ports of the dynamic range (what listen(0) hands out) that accept a connection now.
const DYNAMIC_PORTS = [49152, 65535];
function accepts(port, ms = 250) {
  return new Promise((resolve) => {
    const s = net.connect({ host: '127.0.0.1', port });
    const done = (ok) => { s.destroy(); resolve(ok); };
    s.setTimeout(ms, () => done(false));
    s.once('connect', () => done(true));
    s.once('error', () => done(false));
  });
}
async function openLoopbackPorts(concurrency = 512) {
  const open = [];
  let next = DYNAMIC_PORTS[0];
  const worker = async () => { for (;;) { const p = next++; if (p > DYNAMIC_PORTS[1]) return; if (await accepts(p)) open.push(p); } };
  await Promise.all(Array.from({ length: concurrency }, worker));
  return open;
}

// The window's own page names the per-launch token header; no other local server does, and the page needs no token.
async function servesApp(port) {
  try {
    const res = await fetch(`http://127.0.0.1:${port}/`, { signal: AbortSignal.timeout(1500) });
    return res.ok && (await res.text()).includes('X-DHT-Token');
  } catch { return false; }
}

// The one loopback port that opened after `before` was taken and serves the app. More than one is an
// error, never a guess: a second window of the app (the person's own) must not be driven by mistake.
async function newAppPort(before) {
  const hits = [];
  for (const p of (await openLoopbackPorts()).filter((x) => !before.has(x))) if (await servesApp(p)) hits.push(p);
  if (hits.length > 1) throw new Error(`more than one new loopback port serves the app page (${hits.join(', ')}); close other copies and retry`);
  return hits[0] ?? null;
}

// startApp launches the exe and returns its base URL. The app never prints its port; the OS connection
// table is the honest source (the way tools/sitecheck/shots.mjs finds it), with the probe above where the
// table cannot see this shell's own listeners.
export async function startApp(exe, profile) {
  if (!fs.existsSync(exe)) throw new Error(`doc-html-ui.exe not found at ${exe}; build it first`);
  const blind = !(await tableSeesOwnListener());
  const before = blind ? new Set(await openLoopbackPorts()) : null;
  const env = { ...process.env, LOCALAPPDATA: profile, DOCHT_GUI_NO_WINDOW: '1' };
  const app = spawn(exe, [], { env, stdio: 'ignore', detached: false });
  let port = null;
  const deadline = Date.now() + 45000;
  for (let i = 0; i < 100 && !port && (!blind || Date.now() < deadline); i++) {
    await new Promise((r) => setTimeout(r, 200));
    port = blind ? await newAppPort(before) : tablePort(app.pid);
  }
  if (!port) { app.kill(); throw new Error('doc-html-ui did not start listening on 127.0.0.1'); }
  return {
    base: `http://127.0.0.1:${port}`,
    stop() { try { execFileSync('taskkill', ['/PID', String(app.pid), '/T', '/F'], { stdio: 'ignore' }); } catch { /* already gone */ } },
  };
}

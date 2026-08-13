// make-screenshots.mjs - store-listing screenshots of the viewer in all 13 interface languages.
//
// Why a hand-rolled DevTools client: the viewer only exists under a real chrome-extension://
// origin (chrome.i18n and chrome.storage do not exist on a file:// page), so the capture has to
// drive a browser with the extension loaded. Puppeteer would do it in ten lines and add a
// ~150 MB dependency plus its own Chrome download to an extension that vendors everything and
// depends on Node builtins alone. Node 22 ships a global WebSocket, which is the only piece the
// Chrome DevTools Protocol actually needs - so this stays dependency-free like build.mjs.
//
// Output: extension/store/screenshots/shot1-<code>.png (a PDF reflowed) and shot2-<code>.png
// (an EPUB with the contents panel open), 1280x800, the size the Chrome Web Store and Edge
// Add-ons dashboards accept. Frames are the page viewport only - the four hand-made en/ru files
// this replaces were window captures that included the browser's own toolbar and tab strip.
//
// Usage (from extension/):
//   npm run screenshots               all 13 languages
//   node scripts/make-screenshots.mjs en ru      just these
//   CHROME=<path to chrome.exe> npm run screenshots

import { createServer } from "node:http";
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { die } from "./_lib.mjs";
import { samplePdf, sampleEpub, BOOK_TITLE } from "./_fixtures.mjs";

const HERE = dirname(fileURLToPath(import.meta.url));
const EXT_DIR = resolve(HERE, "..");
const REPO_ROOT = resolve(EXT_DIR, "..");
const OUT_DIR = join(EXT_DIR, "store", "screenshots");
// Run artifacts live in the repo's gitignored temp/, not in the system temp: a long system
// temp path has bitten this project before, and keeping them here makes a failed run
// inspectable.
const WORK_DIR = join(REPO_ROOT, "temp", "ext-screenshots");

// The language set, mirroring internal/i18n.Codes and UI_LANGUAGES in options.js.
const LANGUAGES = ["en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh"];
const RTL_LANGUAGES = ["ar", "ur"];

const WIDTH = 1280;
const HEIGHT = 800;
const RENDER_TIMEOUT_MS = 60_000;

// ---- Chrome ----------------------------------------------------------------

function findChrome() {
  if (process.env.CHROME) return process.env.CHROME;
  const candidates = [
    `${process.env.ProgramFiles}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env["ProgramFiles(x86)"]}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env.LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe`,
    `${process.env.ProgramFiles}\\Microsoft\\Edge\\Application\\msedge.exe`,
    `${process.env["ProgramFiles(x86)"]}\\Microsoft\\Edge\\Application\\msedge.exe`,
  ];
  const found = candidates.find((p) => p && existsSync(p));
  if (!found) die("chrome.exe not found - set CHROME=<path> (Edge works too)");
  return found;
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function waitFor(label, predicate, timeoutMs = 20_000, everyMs = 250) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const value = await predicate();
    if (value) return value;
    if (Date.now() > deadline) throw new Error(`timed out waiting for ${label}`);
    await sleep(everyMs);
  }
}

// ---- the DevTools client ---------------------------------------------------

class CDP {
  constructor(ws) {
    this.ws = ws;
    this.nextId = 1;
    this.pending = new Map();
    ws.addEventListener("message", (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.id && this.pending.has(msg.id)) {
        const { resolve: ok, reject } = this.pending.get(msg.id);
        this.pending.delete(msg.id);
        if (msg.error) reject(new Error(`${msg.error.message} (${JSON.stringify(msg.error.data ?? "")})`));
        else ok(msg.result);
      }
    });
  }

  static async connect(url) {
    const ws = new WebSocket(url);
    await new Promise((ok, reject) => {
      ws.addEventListener("open", ok, { once: true });
      ws.addEventListener("error", () => reject(new Error(`cannot connect to ${url}`)), { once: true });
    });
    return new CDP(ws);
  }

  send(method, params = {}, sessionId) {
    const id = this.nextId++;
    const payload = { id, method, params };
    if (sessionId) payload.sessionId = sessionId;
    return new Promise((ok, reject) => {
      this.pending.set(id, { resolve: ok, reject });
      this.ws.send(JSON.stringify(payload));
    });
  }

  close() {
    try { this.ws.close(); } catch { /* already gone */ }
  }
}

// evaluate runs an expression in the page and returns its value. Anything the page throws
// surfaces here rather than being swallowed into a blank frame.
async function evaluate(cdp, session, expression, awaitPromise = false) {
  const res = await cdp.send(
    "Runtime.evaluate",
    { expression, awaitPromise, returnByValue: true },
    session,
  );
  if (res.exceptionDetails) {
    throw new Error(`page threw: ${res.exceptionDetails.text} ${res.exceptionDetails.exception?.description ?? ""}`);
  }
  return res.result?.value;
}

// ---- the sample-document server -------------------------------------------

// The file names carry the book's real title because the viewer's toolbar takes a PDF's title
// from its file name (an EPUB's comes from its metadata). Serving "alice.pdf" put the word
// "alice" across the top of every PDF frame in the listing.
const PDF_NAME = `${BOOK_TITLE}.pdf`;
const EPUB_NAME = `${BOOK_TITLE}.epub`;

function startFixtureServer() {
  const pdf = samplePdf();
  const epub = sampleEpub();
  const routes = {
    [`/${PDF_NAME}`]: ["application/pdf", pdf],
    [`/${EPUB_NAME}`]: ["application/epub+zip", epub],
  };
  const server = createServer((req, res) => {
    const hit = routes[decodeURIComponent(req.url.split("?")[0])];
    if (!hit) {
      res.writeHead(404).end();
      return;
    }
    res.writeHead(200, { "Content-Type": hit[0], "Content-Length": hit[1].length });
    res.end(hit[1]);
  });
  return new Promise((ok) => {
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address();
      ok({ server, base: `http://127.0.0.1:${port}` });
    });
  });
}

// ---- capture ---------------------------------------------------------------

// localeDir mirrors src/i18n.js: Chrome names the Chinese directory zh_CN.
const localeDir = (code) => (code === "zh" ? "zh_CN" : code);

// expectedTocHeading reads the translation this run must actually show, from the same
// _locales file the extension serves it from.
function expectedTocHeading(lang) {
  const path = join(EXT_DIR, "_locales", localeDir(lang), "messages.json");
  const messages = JSON.parse(readFileSync(path, "utf8"));
  const value = messages.tocHeading?.message;
  if (!value) die(`_locales/${localeDir(lang)}/messages.json has no tocHeading`);
  return value;
}

// assertChromeLanguage guards the invariant the whole 13-language feature rests on, in the shape
// the extension implements it (applyViewerChromeI18n in src/viewer.js): the toolbar and the
// contents panel speak the interface language and mirror for Arabic and Urdu, while the document
// root keeps its own language and direction - that attribute is what makes the browser offer
// "Translate page", which is the product's entire free workflow.
//
// The heading text is compared against the locale file rather than checking a lang attribute:
// applyI18n deliberately sets no lang on the chrome, and a language that silently fell back to
// English would still carry the right attributes. A wrong frame must fail the run rather than
// reach a store listing.
async function assertChromeLanguage(cdp, session, lang) {
  const got = await evaluate(
    cdp,
    session,
    `JSON.stringify({
       rootDir: document.documentElement.getAttribute('dir') || '',
       toolbarDir: document.getElementById('toolbar')?.getAttribute('dir') || '',
       tocDir: document.getElementById('toc')?.getAttribute('dir') || '',
       heading: (document.querySelector('#toc h2')?.textContent || '').trim(),
     })`,
  );
  const { rootDir, toolbarDir, tocDir, heading } = JSON.parse(got);
  const wantDir = RTL_LANGUAGES.includes(lang) ? "rtl" : "ltr";

  if (rootDir === "rtl") {
    throw new Error("SCREENSHOT-RTL: the document root is mirrored - only the viewer chrome may mirror");
  }
  for (const [name, dir] of [["toolbar", toolbarDir], ["toc", tocDir]]) {
    if (dir !== wantDir) {
      throw new Error(`SCREENSHOT-RTL: ${name} dir="${dir}" but ${lang} expects "${wantDir}"`);
    }
  }
  const want = expectedTocHeading(lang);
  if (heading !== want) {
    throw new Error(`SCREENSHOT-LANG: contents heading is "${heading}", expected "${want}" for ${lang}`);
  }
}

// loadExtension installs the working tree as an unpacked extension and returns the id Chrome
// assigned it.
//
// It goes through the DevTools Extensions domain rather than the --load-extension command line
// because branded Google Chrome refuses that switch outright - it logs "--load-extension is not
// allowed in Google Chrome, ignoring" and starts without the extension, so every later step then
// waits on a page that does not exist. Edge still honours the switch, but one code path that
// works in both is worth more than a per-browser branch. The returned id is the browser's own
// answer, so nothing here has to derive it.
async function loadExtension(cdp) {
  const res = await cdp.send("Extensions.loadUnpacked", { path: EXT_DIR }).catch((e) => {
    die(
      `Extensions.loadUnpacked failed: ${e.message}\n` +
      "This browser is too old for the DevTools Extensions domain - use Chrome or Edge 12x or newer.",
    );
  });
  if (!res?.id) die("Extensions.loadUnpacked returned no extension id");
  return res.id;
}

// openDocument navigates to the viewer and waits until the render has actually finished - the
// status bar gains "done" and the content pane holds nodes. Polling the DOM rather than the load
// event is what keeps an empty frame from being captured for a slow PDF.
async function openDocument(cdp, session, extensionId, fileUrl) {
  const target = `chrome-extension://${extensionId}/src/viewer.html?file=${encodeURIComponent(fileUrl)}`;
  await cdp.send("Page.navigate", { url: target }, session);
  await waitFor(
    `${fileUrl} to render`,
    async () => {
      try {
        return await evaluate(
          cdp,
          session,
          `(() => { const s = document.getElementById('status'), c = document.getElementById('content');
             return !!(s && c && s.classList.contains('done') && c.children.length > 0); })()`,
        );
      } catch {
        return false; // navigation in flight - the old context is gone, try again
      }
    },
    RENDER_TIMEOUT_MS,
  );
}

async function capture(cdp, session, file) {
  const shot = await cdp.send("Page.captureScreenshot", { format: "png" }, session);
  writeFileSync(file, Buffer.from(shot.data, "base64"));
}

async function main() {
  const requested = process.argv.slice(2).filter((a) => !a.startsWith("-"));
  const languages = requested.length ? requested : LANGUAGES;
  for (const l of languages) {
    if (!LANGUAGES.includes(l)) die(`unknown language ${l} - one of: ${LANGUAGES.join(" ")}`);
  }

  if (!existsSync(join(EXT_DIR, "vendor", "pdf.mjs"))) {
    die("extension/vendor is empty - run `npm run vendor` first, the viewer needs pdf.js");
  }
  mkdirSync(OUT_DIR, { recursive: true });
  rmSync(WORK_DIR, { recursive: true, force: true });
  mkdirSync(WORK_DIR, { recursive: true });

  const { server, base } = await startFixtureServer();
  const chrome = findChrome();
  const profile = join(WORK_DIR, "profile");

  const child = spawn(
    chrome,
    [
      "--headless=new",
      `--user-data-dir=${profile}`,
      "--remote-debugging-port=0",
      `--window-size=${WIDTH},${HEIGHT}`,
      "--hide-scrollbars",
      "--no-first-run",
      "--no-default-browser-check",
      "--disable-features=Translate,MediaRouter",
      "--disable-background-timer-throttling",
      "about:blank",
    ],
    { stdio: ["ignore", "ignore", "pipe"] },
  );
  let chromeStderr = "";
  child.stderr.on("data", (d) => { chromeStderr += d.toString(); });

  let cdp;
  try {
    const portFile = join(profile, "DevToolsActivePort");
    const port = await waitFor("Chrome's debugging port", () => {
      if (!existsSync(portFile)) return null;
      const first = readFileSync(portFile, "utf8").split("\n")[0].trim();
      return first || null;
    }, 30_000).catch((e) => {
      throw new Error(`${e.message}${chromeStderr ? `\nchrome said: ${chromeStderr.trim()}` : ""}`);
    });

    const version = await (await fetch(`http://127.0.0.1:${port}/json/version`)).json();
    cdp = await CDP.connect(version.webSocketDebuggerUrl);

    const extensionId = await loadExtension(cdp);
    console.log(`  extension loaded as ${extensionId}`);

    const { targetId } = await cdp.send("Target.createTarget", { url: "about:blank" });
    const { sessionId } = await cdp.send("Target.attachToTarget", { targetId, flatten: true });
    await cdp.send("Page.enable", {}, sessionId);
    await cdp.send("Runtime.enable", {}, sessionId);
    await cdp.send("Emulation.setDeviceMetricsOverride", {
      width: WIDTH, height: HEIGHT, deviceScaleFactor: 1, mobile: false,
    }, sessionId);

    // The options page is an extension context, so it can seed storage for the viewer: the
    // interface-language override plus the dark reading theme the current listing frames use.
    const optionsUrl = `chrome-extension://${extensionId}/src/options.html`;

    for (const lang of languages) {
      await cdp.send("Page.navigate", { url: optionsUrl }, sessionId);
      await waitFor(`${optionsUrl} to be ready`, async () => {
        try {
          return await evaluate(cdp, sessionId, "typeof chrome !== 'undefined' && !!chrome.storage");
        } catch {
          return false;
        }
      });
      await evaluate(
        cdp,
        sessionId,
        `chrome.storage.local.set({uiLang: ${JSON.stringify(lang)}, viewerPrefs: {theme: "dark", family: "serif", size: 28}})`,
        true,
      );

      await openDocument(cdp, sessionId, extensionId, `${base}/${encodeURIComponent(PDF_NAME)}`);
      await assertChromeLanguage(cdp, sessionId, lang);
      await capture(cdp, sessionId, join(OUT_DIR, `shot1-${lang}.png`));

      await openDocument(cdp, sessionId, extensionId, `${base}/${encodeURIComponent(EPUB_NAME)}`);
      await assertChromeLanguage(cdp, sessionId, lang);
      await evaluate(cdp, sessionId, "document.getElementById('btn-toc').click()");
      await waitFor("the contents panel to open", async () => {
        const shown = await evaluate(
          cdp,
          sessionId,
          `(() => { const t = document.getElementById('toc');
             return !!(t && !t.classList.contains('hidden') && t.querySelectorAll('a').length > 0); })()`,
        );
        return shown;
      });
      await capture(cdp, sessionId, join(OUT_DIR, `shot2-${lang}.png`));

      console.log(`  ${lang}: shot1-${lang}.png shot2-${lang}.png`);
    }

    console.log(`${languages.length * 2} PNG(s) written to store/screenshots at ${WIDTH}x${HEIGHT}`);
  } finally {
    cdp?.close();
    child.kill();
    server.close();
  }
}

await main();

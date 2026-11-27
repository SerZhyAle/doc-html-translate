// End-to-end check of "Save media file for translation": the unpacked extension in Chromium saves
// cross-origin media with no downloads permission. Hand-run: node tools/uicheck/save-media-e2e.mjs
// Needs `npm run vendor` done in extension/, tools/sitecheck's playwright-core and a Playwright
// Chromium (DHT_CHROMIUM overrides the executable). Ends in "save-media-e2e: PASS|FAIL".
import { createServer } from "node:http";
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import { createHash, randomBytes } from "node:crypto";
import { fileURLToPath, pathToFileURL } from "node:url";

const repo = fileURLToPath(new URL("../../", import.meta.url));

// The newest Chromium Playwright installed; branded Chrome no longer loads an unpacked extension from the command line.
function findChromium() {
  if (process.env.DHT_CHROMIUM) return process.env.DHT_CHROMIUM;
  const cache = process.env.PLAYWRIGHT_BROWSERS_PATH || join(process.env.LOCALAPPDATA || join(homedir(), ".cache"), "ms-playwright");
  const found = (existsSync(cache) ? readdirSync(cache) : []).filter((d) => /^chromium-\d+$/.test(d)).sort().reverse()
    .flatMap((d) => ["chrome-win64/chrome.exe", "chrome-win/chrome.exe", "chrome-linux/chrome"].map((p) => join(cache, d, p))).filter(existsSync);
  if (!found.length) { console.log("save-media-e2e: COULD NOT VERIFY (no Playwright Chromium)"); process.exit(2); }
  return found[0];
}
const executablePath = findChromium();
const { chromium } = await import(pathToFileURL(join(repo, "tools/sitecheck/node_modules/playwright-core/index.mjs")).href);
const sha = (b) => createHash("sha256").update(b).digest("hex").slice(0, 16);
const png = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhWMw7wAAAABJRU5ErkJggg==", "base64");
const big = randomBytes(80 * 1024 * 1024);
const files = {
  "/photo.png": { body: png, headers: { "content-type": "image/png" } },
  "/named": { body: png, headers: { "content-type": "image/png", "content-disposition": 'attachment; filename="server name.png"' } },
  "/clip": { body: big, headers: { "content-type": "video/mp4" } },
  "/private.jpg": { body: png, headers: { "content-type": "image/jpeg" }, needsCookie: true },
};
// The media host sends no CORS headers at all: only the extension's host access can read it.
const media = createServer((req, res) => {
  const f = files[req.url.split("?")[0]];
  if (!f || (f.needsCookie && !/sid=1/.test(req.headers.cookie || ""))) { res.writeHead(f ? 403 : 404); return res.end("no"); }
  res.writeHead(200, { ...f.headers, "content-length": f.body.length });
  res.end(f.body);
});
await new Promise((r) => media.listen(0, "127.0.0.1", r));
const mediaBase = `http://127.0.0.1:${media.address().port}`;
const site = createServer((req, res) => { res.writeHead(200, { "content-type": "text/html" }); res.end("<html><body><p>page</p></body></html>"); });
await new Promise((r) => site.listen(0, "127.0.0.1", r));
const siteBase = `http://localhost:${site.address().port}/`;

const ext = join(repo, "extension");
const profile = mkdtempSync(join(tmpdir(), "dht-e2e-"));
const context = await chromium.launchPersistentContext(profile, {
  executablePath, headless: false, acceptDownloads: true,
  args: [`--headless=new`, `--disable-extensions-except=${ext}`, `--load-extension=${ext}`],
});
let failed = 0;
const check = (name, ok, detail = "") => { console.log(`${ok ? "PASS" : "FAIL"} ${name} ${detail}`); if (!ok) failed++; };
try {
  const worker = context.serviceWorkers()[0] || await context.waitForEvent("serviceworker");
  const id = new URL(worker.url()).host;
  const perms = await worker.evaluate(() => chrome.permissions.getAll());
  check("no downloads permission held", !perms.permissions.includes("downloads"), perms.permissions.join(","));
  const seen = [];
  const errors = [];
  context.on("page", (p) => {
    p.on("download", (d) => seen.push({ page: p, d }));
    p.on("pageerror", (e) => errors.push(String(e)));
  });
  await context.addCookies([{ name: "sid", value: "1", url: mediaBase, sameSite: "Lax" }]);
  const sitePage = await context.newPage();
  await sitePage.goto(siteBase);
  const driver = await context.newPage();
  await driver.goto(`chrome-extension://${id}/src/options.html`);
  const call = (info) => driver.evaluate(async ({ info, siteBase }) => {
    const m = await import("./media-save.js");
    const [tab] = await chrome.tabs.query({ url: `${new URL(siteBase).protocol}//localhost/*` });
    return m.saveMedia(info, tab);
  }, { info, siteBase });

  const wait = async (n, ms = 30000) => { const end = Date.now() + ms; while (seen.length < n && Date.now() < end) await new Promise((r) => setTimeout(r, 100)); };
  const cases = [["/photo.png?x=1", "photo.png", png], ["/named", "server name.png", png], ["/private.jpg", "private.jpg", png], ["/clip", "clip.mp4", big]];
  for (const [path, name, body] of cases) {
    const before = seen.length;
    const res = await call({ mediaType: "image", srcUrl: mediaBase + path, frameId: 0 });
    await wait(before + 1);
    const got = seen[before];
    if (!got) { check(`save ${path}`, false, `no download (${JSON.stringify(res)})`); continue; }
    const closed = got.page.waitForEvent("close", { timeout: 20000 }).then(() => true, () => false);
    const saved = readFileSync(await got.d.path());
    check(`save ${path}`, got.d.suggestedFilename() === name && sha(saved) === sha(body), `name="${got.d.suggestedFilename()}" bytes=${saved.length} failure=${await got.d.failure()}`);
    check(`tab closes itself after ${path}`, await closed);
  }
  // Several files from one saving page, one of them refused by the server.
  const before = seen.length;
  const key = "mediaJob:multi";
  await driver.evaluate(async ({ key, urls }) => chrome.storage.session.set({ [key]: { urls } }), { key, urls: [mediaBase + "/photo.png", mediaBase + "/missing.bin", mediaBase + "/named"] });
  const multi = await context.newPage();
  multi.on("download", (d) => { if (!seen.some((s) => s.d === d)) seen.push({ page: multi, d }); });
  await multi.goto(`chrome-extension://${id}/src/save.html#multi`);
  await wait(before + 2);
  await multi.waitForTimeout(1500);
  const lines = await multi.locator("#files li").allTextContents();
  check("several files saved, the refused one reported", seen.length === before + 2 && lines.length === 3 && /404/.test(lines[1]), JSON.stringify(lines));
  check("the page of several files stays open", !multi.isClosed());
  if (process.argv[2]) await multi.screenshot({ path: join(process.argv[2], "save-page.png") });
  await multi.reload();
  await multi.waitForTimeout(800);
  check("a reload saves nothing again", seen.length === before + 2 && /Nothing to save/.test(await multi.locator("#status").textContent()));
  check("no page errors", errors.length === 0, errors.join(" | "));
} finally {
  await context.close();
  media.close();
  site.close();
  rmSync(profile, { recursive: true, force: true });
}
console.log(failed ? `save-media-e2e: FAIL (${failed})` : "save-media-e2e: PASS");
process.exit(failed ? 1 : 0);

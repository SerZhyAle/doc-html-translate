// run.mjs - one edition's pass over the ticket 68 document corpus (DEV/doccorpus/cases.json).
//
// Both editions are measured by the same in-page probe, so a difference in the result is a
// difference in what the reader gets, not in how it was counted:
//
//   windows    converts each case with the desktop CLI, then opens the generated index.html in a
//              headless browser with no extension loaded - the page a user opens in Chrome.
//   extension  loads the unpacked extension and opens the same bytes in its document viewer,
//              served over a local http server as the viewer receives a real link.
//
// Nothing is judged here. The runner records what happened - exit code, log, probe, text, DOM and
// screenshots - into temp/doccorpus/<run>/<edition>/<case>/, and `go run ./tools/doccorpus
// report` grades it against the case expectations. A missing or changed source is recorded as
// such and never converted: the report turns it into COULD NOT VERIFY, not a pass.
//
// Usage (from the repository root, PowerShell):
//   node tools/doccorpus/run.mjs --edition windows [--case <id>].. [--split dev|holdout|all]
//        [--ocr default|on] [--cli <exe>] [--run <dir>] [--budget <seconds>]
//
// The DevTools client is the extension lab's own (extension/scripts/_ocrlab-cdp.mjs): Node's
// global WebSocket is the only dependency, as in every other script of this repository.

import { spawn, spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { createServer } from "node:http";
import { existsSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { release as osRelease } from "node:os";
import { basename, dirname, extname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { CDP, evaluate, findChrome, sleep, waitFor } from "../../extension/scripts/_ocrlab-cdp.mjs";
import { collect, OCR_COUNTER, PAGE_TOTAL } from "./collect.mjs";
import { caseKey, expectationSha, producerDigest, REUSE_SCHEMA, sizesOf, tryReuse } from "./reuse.mjs";

const HERE = dirname(fileURLToPath(import.meta.url));
const REPO = resolve(HERE, "..", "..");
const EXT_DIR = join(REPO, "extension");
const MANIFEST = join(REPO, "DEV", "doccorpus", "cases.json");
const SCHEMA_VERSION = 1;
const RESULT_SCHEMA = 1;

const VIEWPORT = { width: 1280, height: 800 };
const CONVERT_TIMEOUT_MS = 30 * 60_000;
const DOM_CAP = 30 * 1024 * 1024;

// Classes whose images the app OCRs without being asked: a standalone image and a comic
// archive (docs/PARITY.md "Comic forced-OCR decision"). They need the language pack even on the
// default path.
const FORCED_OCR = new Set(["comic-image", "comic-archive"]);

const USAGE = `run.mjs - one edition's pass over the ticket 68 corpus

  node tools/doccorpus/run.mjs --edition windows|extension [flags]

  --case <id>          run only this case (repeatable)
  --class <class>      run only this input class (repeatable)
  --split <s>          dev | holdout | all (default all)
  --ocr <mode>         default (the shipped default path) | on (OCR enabled for every image)
  --cli <exe>          desktop CLI (default temp/doccorpus/bin/doc-html-translate.exe)
  --run <dir>          run directory (default temp/doccorpus/<timestamp>)
  --budget <seconds>   probe budget per case before the result is marked incomplete (default 300)
  --fresh              collect every case even when an earlier settled, passing result with
                       identical inputs could be reused
  --reuse-from <dir>   search only this run directory (or folder of runs) for reusable cases
                       (default temp/doccorpus)
`;

function die(msg) {
  console.error(`doccorpus run: ${msg}`);
  process.exit(2);
}

function parseArgs(argv) {
  const out = { edition: "", case: [], class: [], split: "all", ocr: "default", cli: "", run: "", budget: "300", fresh: false, reuseFrom: "" };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--help" || a === "-h") return { help: true };
    if (!a.startsWith("--")) die(`unexpected argument ${a}`);
    const key = a.slice(2);
    if (key === "fresh") { out.fresh = true; continue; }
    const value = argv[++i];
    if (value === undefined) die(`--${key} needs a value`);
    if (key === "case") out.case.push(value);
    else if (key === "class") out.class.push(value);
    else if (key === "reuse-from") out.reuseFrom = value;
    else if (key in out) out[key] = value;
    else die(`unknown flag --${key}`);
  }
  if (!["windows", "extension"].includes(out.edition)) die("--edition must be windows or extension");
  if (!["default", "on"].includes(out.ocr)) die("--ocr must be default or on");
  return out;
}

// ---- corpus ----------------------------------------------------------------

function sha256(path) {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

function selectCases(args) {
  if (!existsSync(MANIFEST)) die(`no manifest at ${MANIFEST}`);
  const m = JSON.parse(readFileSync(MANIFEST, "utf8"));
  if (m.schemaVersion !== SCHEMA_VERSION) die(`manifest schemaVersion ${m.schemaVersion}, runner understands ${SCHEMA_VERSION}`);
  const wanted = new Set(args.case);
  for (const id of wanted) if (!m.cases.some((c) => c.id === id)) die(`unknown case ${id}`);
  const classes = new Set(args.class);
  const picked = m.cases.filter((c) => (wanted.size ? wanted.has(c.id) : args.split === "all" || c.split === args.split))
    .filter((c) => !classes.size || classes.has(c.class));
  if (!picked.length) die("no cases selected");
  return picked.map((c) => ({ ...c, path: join(REPO, m.root, ...c.file.split("/")) }));
}

// ---- environment -----------------------------------------------------------

function firstLine(cmd, argv) {
  const r = spawnSync(cmd, argv, { encoding: "utf8", timeout: 20_000, windowsHide: true });
  if (r.error || r.status !== 0) return "";
  return `${r.stdout || ""}${r.stderr || ""}`.split(/\r?\n/).map((s) => s.trim()).find(Boolean) || "";
}

function tessdataDir() {
  return join(process.env.LOCALAPPDATA || "", "doc-html-translate", "tessdata");
}

// The desktop's installed packs, fingerprinted by their bytes (the ocrlab runner's rule: a
// version string would be a mirrored constant, the hash is what was actually loaded).
function installedPacks() {
  const out = {};
  const dir = tessdataDir();
  if (!existsSync(dir)) return out;
  for (const f of readdirSync(dir)) {
    if (!f.endsWith(".traineddata")) continue;
    out[f.replace(/\.traineddata$/, "")] = `sha256:${sha256(join(dir, f)).slice(0, 16)}`;
  }
  return out;
}

// The catalog both editions offer (internal/ocr/tessdata.go Available == ocr-lang.js LANGS). A
// case whose pack is not in it cannot be OCR'd by either edition - that is a product gap the
// report must show, not a smaller test.
function offeredPacks() {
  const src = readFileSync(join(REPO, "internal", "ocr", "tessdata.go"), "utf8");
  const block = src.slice(src.indexOf("var Available"), src.indexOf("}\n\n", src.indexOf("var Available")));
  return [...block.matchAll(/\{"([a-z_]+)",/g)].map((x) => x[1]);
}

function environment(args, browserProduct) {
  const git = (argv) => firstLine("git", ["-C", REPO, ...argv]);
  const dirty = spawnSync("git", ["-C", REPO, "status", "--porcelain"], { encoding: "utf8" }).stdout.trim();
  const env = {
    commit: git(["rev-parse", "HEAD"]),
    treeDirty: dirty.length > 0,
    windows: `${firstLine("cmd", ["/c", "ver"])} (os.release ${osRelease()})`,
    browser: browserProduct,
    node: process.version,
    locale: Intl.DateTimeFormat().resolvedOptions().locale,
    ocrMode: args.ocr,
    offeredOcrPacks: offeredPacks(),
  };
  if (args.edition === "windows") {
    env.cli = args.cli;
    env.cliVersion = firstLine(args.cli, ["-version"]);
    // The exe's own bytes: the CLI under test may be older than the source tree it was built from.
    env.cliSha256 = sha256(args.cli);
    env.tesseract = firstLine("tesseract", ["--version"]);
    env.tessdataDir = tessdataDir();
    env.installedOcrPacks = installedPacks();
    env.pdftotext = existsSync(join(dirname(args.cli), "pdftotext.exe")) ? "next to the CLI" : "not next to the CLI";
  } else {
    const man = JSON.parse(readFileSync(join(EXT_DIR, "manifest.json"), "utf8"));
    const tjs = join(EXT_DIR, "node_modules", "tesseract.js", "package.json");
    env.extensionVersion = man.version;
    env.tesseractJs = existsSync(tjs) ? JSON.parse(readFileSync(tjs, "utf8")).version : "unknown";
    env.extensionTessdata = "tessdata_fast 4.0.0 via ocr-lang.js (eng vendored, the rest fetched on demand)";
  }
  return env;
}

// ---- the browser -------------------------------------------------------------

async function openBrowser(runDir, withExtension, seq) {
  const profile = join(runDir, `.profile-${withExtension ? "ext" : "win"}-${seq}`);
  const child = spawn(findChrome(), [
    "--headless=new",
    `--user-data-dir=${profile}`,
    "--remote-debugging-port=0",
    `--window-size=${VIEWPORT.width},${VIEWPORT.height}`,
    "--hide-scrollbars", "--no-first-run", "--no-default-browser-check",
    "--disable-features=Translate,MediaRouter",
    "--disable-background-timer-throttling",
    "--allow-file-access-from-files",
    "about:blank",
  ], { stdio: ["ignore", "ignore", "pipe"] });
  let stderr = "";
  child.stderr.on("data", (d) => { stderr += d.toString(); });

  const portFile = join(profile, "DevToolsActivePort");
  const cdp = await waitFor("the browser's debugging port", async () => {
    if (!existsSync(portFile)) return null;
    const port = readFileSync(portFile, "utf8").split("\n")[0].trim();
    if (!port) return null;
    try {
      const { webSocketDebuggerUrl } = await (await fetch(`http://127.0.0.1:${port}/json/version`)).json();
      return await CDP.connect(webSocketDebuggerUrl);
    } catch {
      return null;
    }
  }, 30_000);

  const { product } = await cdp.send("Browser.getVersion");
  let extensionId = "";
  if (withExtension) {
    const loaded = await cdp.send("Extensions.loadUnpacked", { path: EXT_DIR });
    extensionId = loaded.id;
  }
  const { targetId } = await cdp.send("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await cdp.send("Target.attachToTarget", { targetId, flatten: true });
  await cdp.send("Page.enable", {}, sessionId);
  await cdp.send("Runtime.enable", {}, sessionId);
  await cdp.send("Emulation.setDeviceMetricsOverride", { ...VIEWPORT, deviceScaleFactor: 1, mobile: false }, sessionId);
  return { child, cdp, product, session: sessionId, extensionId, said: () => stderr.trim().split("\n").slice(-6).join("\n") };
}

// setViewerOptions pins the extension's options from one of its own pages, so a run never
// inherits what a profile happened to hold.
async function setViewerOptions(b, options) {
  await b.cdp.send("Page.navigate", { url: `chrome-extension://${b.extensionId}/src/options.html` }, b.session);
  await waitFor("the options page", async () => {
    try { return await evaluate(b.cdp, b.session, "typeof chrome !== 'undefined' && !!chrome.storage"); } catch { return false; }
  }, 30_000);
  await evaluate(b.cdp, b.session, `chrome.storage.local.set(${JSON.stringify({ options, uiLang: "en" })})`, true);
}

// ---- the probe (identical for both editions) ---------------------------------

// Plates and their containers carry different class names per edition (the desktop writes
// .ocr-fig / .ocr-box, the extension .ocr-overlay / .ocr-plate); OCR-OVERLAY defines the roles,
// not the names, so the probe reads both.
const ROOT = `const ext = location.protocol === "chrome-extension:";
  const root = ext ? document.getElementById("content")
    : (document.querySelector("main.dht-single") || document.querySelector("main") || (document.body && document.body.children.length ? document.body : null));`;

// PROBE is taken on every scroll step, so it only counts. The text measure, the links and the
// computed styles cost a walk over the whole document and are read once, after the walk (DETAIL).
// The status patterns come from collect.mjs, which a test checks against the viewer's source.
const PROBE = `(() => {
  ${ROOT}
  // An empty body is the desktop's location.replace stub (an EPUB index) on its way to the real page.
  if (!root) return JSON.stringify({ state: "pending" });
  const plates = root.querySelectorAll(".ocr-plate, .ocr-box");
  const imgs = [...root.querySelectorAll("img")].filter((i) => i.getAttribute("src"));
  const toc = ext ? document.querySelectorAll("#toc a") : document.querySelectorAll(".dht-contents a");
  const status = ext ? (document.getElementById("status-text") || {}).textContent || "" : "";
  const ocrProgress = new RegExp(${JSON.stringify(OCR_COUNTER.source)}).exec(status);
  const totalMatch = ext ? new RegExp(${JSON.stringify(PAGE_TOTAL.source)}).exec((document.getElementById("page-total") || {}).textContent || "") : null;
  const notice = ext ? document.querySelector("#content .notice h1") : null;
  return JSON.stringify({
    state: "ready",
    title: document.title,
    lang: document.documentElement.lang || "",
    dir: document.documentElement.dir || "",
    scrollHeight: document.documentElement.scrollHeight,
    textLen: root.textContent.length,
    images: imgs.length,
    imagesLoaded: imgs.filter((i) => i.complete && i.naturalWidth > 0).length,
    imagesBroken: imgs.filter((i) => i.complete && i.naturalWidth === 0).length,
    imagesPending: imgs.filter((i) => !i.complete).length,
    canvases: root.querySelectorAll("canvas").length,
    tocEntries: toc.length,
    overlays: root.querySelectorAll(".ocr-overlay, .ocr-fig").length,
    plates: plates.length,
    pageUnits: ext ? root.querySelectorAll("section[data-page]").length : root.querySelectorAll(".dht-page").length,
    status,
    ocrDone: ocrProgress ? Number(ocrProgress[1]) : null,
    ocrTotal: ocrProgress ? Number(ocrProgress[2]) : null,
    ocrPending: root.querySelectorAll(".ocr-pending").length,
    // A PDF page keeps data-pdf-page (and a reserved box, when it has no text) until its
    // images are extracted; the desktop page has neither.
    extractionPending: root.querySelectorAll("section[data-pdf-page], .pdf-page-pending").length,
    pageTotal: totalMatch ? Number(totalMatch[1]) : null,
    notice: notice ? notice.textContent.trim() : "",
  });
})()`;

// DETAIL is read once, after the walk. textChars is the reader's text without OCR plates, page
// labels and scripts, whitespace collapsed - the measure the report compares with the source.
const DETAIL = `(() => {
  ${ROOT}
  const plateSel = ".ocr-plate, .ocr-box";
  const plates = [...root.querySelectorAll(plateSel)];
  const skip = (n) => n.closest && (n.closest(plateSel) || n.closest(".page-label, script, style, noscript"));
  let text = "";
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode: (t) => (skip(t.parentElement) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT),
  });
  while (walker.nextNode()) text += walker.currentNode.nodeValue + " ";
  text = text.replace(/\\s+/g, " ").trim();
  const links = [...root.querySelectorAll('a[href^="#"]')];
  const missing = links.filter((a) => {
    const id = decodeURIComponent(a.getAttribute("href").slice(1));
    return id && !document.getElementById(id) && !document.getElementsByName(id).length;
  });
  const heads = [...root.querySelectorAll("h1,h2,h3,h4,h5,h6")];
  const els = [...root.querySelectorAll("*")].slice(0, 6000);
  return JSON.stringify({
    textChars: text.length,
    textSample: text.slice(0, 600),
    headings: heads.length,
    headingSample: heads.slice(0, 12).map((h) => h.textContent.trim().slice(0, 80)),
    internalLinks: links.length,
    internalLinksMissing: missing.length,
    missingSample: missing.slice(0, 5).map((a) => a.getAttribute("href")),
    plateChars: plates.reduce((n, p) => n + p.textContent.trim().length, 0),
    verticalWriting: els.some((e) => getComputedStyle(e).writingMode.startsWith("vertical")),
    rubyElements: root.querySelectorAll("ruby").length,
  });
})()`;
const FULL_TEXT = `(() => {
  const ext = location.protocol === "chrome-extension:";
  const root = ext ? document.getElementById("content")
    : (document.querySelector("main.dht-single") || document.querySelector("main") || document.body);
  const plates = [...root.querySelectorAll(".ocr-plate, .ocr-box")].map((p) => p.textContent.replace(/\\s+/g, " ").trim());
  return JSON.stringify({ text: root.innerText, plates });
})()`;

async function probe(b) {
  return JSON.parse(await evaluate(b.cdp, b.session, PROBE));
}

// walk hands the page to collect(): scroll top to bottom, then wait until the reader has finished
// producing the document - not merely until the DOM holds still (see collect.mjs). The result
// carries the last ready probe and the collection record (outcome, stage, counts).
async function walk(b, budgetMs, stepFactor) {
  const { probe: last, ...collection } = await collect({
    probe: async () => {
      try {
        return await probe(b);
      } catch (err) {
        if (b.cdp.dead) err.fatal = true; // a dead browser is the case's error, not a retry
        throw err;
      }
    },
    scroll: (y) => evaluate(b.cdp, b.session, `window.scrollTo(0, ${y})`).catch((err) => { if (b.cdp.dead) throw err; }),
    sleep,
    now: Date.now,
    budgetMs,
    stepPx: Math.round(VIEWPORT.height * stepFactor),
  });
  return { probe: last, collection, truncated: collection.outcome !== "settled" };
}

// readLarge fetches a long string in slices. One reply carrying a whole CJK book's DOM - every
// character escaped - dropped the DevTools connection outright (zh-epub-journey-west, 2.3 MB of
// HTML), which took the browser's session and the case with it.
const SLICE = 256 * 1024;
async function readLarge(b, expr) {
  const n = await evaluate(b.cdp, b.session, `(window.__doccorpus = String(${expr})).length`);
  let out = "";
  for (let i = 0; i < n; i += SLICE) {
    out += await evaluate(b.cdp, b.session, `window.__doccorpus.slice(${i}, ${i + SLICE})`);
  }
  await evaluate(b.cdp, b.session, "delete window.__doccorpus");
  return out;
}

async function shoot(b, file, y) {
  await evaluate(b.cdp, b.session, `window.scrollTo(0, ${y})`);
  await sleep(400);
  const shot = await b.cdp.send("Page.captureScreenshot", { format: "png" }, b.session);
  writeFileSync(file, Buffer.from(shot.data, "base64"));
}

// capture writes the evidence of one rendered case: the probe, the text the reader sees, the
// DOM, and three screenshots - the top, the first OCR plate (or the middle) and the middle.
async function capture(b, dir, walked) {
  const full = JSON.parse(await readLarge(b, FULL_TEXT));
  writeFileSync(join(dir, "text.txt"), full.text);
  writeFileSync(join(dir, "plates.txt"), full.plates.join("\n"));
  let dom = await readLarge(b, "document.documentElement.outerHTML");
  if (dom.length > DOM_CAP) dom = `${dom.slice(0, DOM_CAP)}\n<!-- doccorpus: DOM truncated at ${DOM_CAP} chars -->`;
  writeFileSync(join(dir, "dom.html"), dom);
  const shots = {};
  shots.top = "top.png";
  await shoot(b, join(dir, shots.top), 0);
  const plateY = await evaluate(b.cdp, b.session,
    `(() => { const p = document.querySelector(".ocr-overlay, .ocr-fig"); return p ? Math.max(0, p.getBoundingClientRect().top + scrollY - 40) : -1; })()`);
  if (plateY >= 0) {
    shots.ocr = "ocr.png";
    await shoot(b, join(dir, shots.ocr), plateY);
  }
  shots.middle = "middle.png";
  await shoot(b, join(dir, shots.middle), Math.round(walked.probe.scrollHeight / 2));
  return shots;
}

// ---- one case ---------------------------------------------------------------

function findIndex(dir) {
  const direct = join(dir, "index.html");
  if (existsSync(direct)) return direct;
  if (!existsSync(dir)) return "";
  for (const e of readdirSync(dir)) {
    const p = join(dir, e, "index.html");
    if (existsSync(p) && statSync(join(dir, e)).isDirectory()) return p;
  }
  return "";
}

function convertWindows(args, c, dir, ocrLang) {
  const out = join(dir, "html");
  mkdirSync(out, { recursive: true });
  const argv = ["-notranslate", "-noopen", "-force", "-folder", out, "-src", c.language, "-ui-lang", "en"];
  if (ocrLang) argv.push("-ocr-lang", ocrLang);
  if (args.ocr === "on") argv.push("-ocr");
  argv.push(c.path);
  const t0 = Date.now();
  const r = spawnSync(args.cli, argv, { encoding: "utf8", timeout: CONVERT_TIMEOUT_MS, windowsHide: true, maxBuffer: 64 * 1024 * 1024 });
  const log = `> ${basename(args.cli)} ${argv.join(" ")}\n${r.stdout || ""}${r.stderr || ""}${r.error ? `\n[spawn error] ${r.error.message}` : ""}`;
  writeFileSync(join(dir, "convert.log"), log);
  return { exitCode: r.status, ms: Date.now() - t0, argv, index: findIndex(out), timedOut: r.error?.code === "ETIMEDOUT" };
}

async function runCase(ctx, c) {
  const dir = join(ctx.editionDir, c.id);
  mkdirSync(dir, { recursive: true });
  const res = { schema: RESULT_SCHEMA, caseId: c.id, edition: ctx.args.edition, ocrMode: ctx.args.ocr, startedAt: new Date().toISOString() };
  const save = () => writeFileSync(join(dir, "result.json"), `${JSON.stringify(res, null, 2)}\n`);

  if (!existsSync(c.path)) { res.source = "missing"; save(); return res; }
  const got = sha256(c.path);
  if (got !== c.sha256) { res.source = "hash-changed"; res.sourceSha256 = got; save(); return res; }
  res.source = "ok";
  res.sourceSha256 = got;

  // Still opened: the intentional difference is a "use the desktop app" notice, and that notice
  // is itself the behaviour the report checks.
  if (ctx.args.edition === "extension" && ["cbr", "cb7"].includes(extname(c.path).slice(1).toLowerCase())) {
    res.intentionalDifference = "the extension declines CBR/CB7 with a desktop-app notice (docs/PARITY.md)";
  }

  const needsOcr = ctx.args.ocr === "on" || FORCED_OCR.has(c.class);
  const offered = ctx.offered.includes(c.ocrLang);
  const installed = ctx.args.edition === "extension" || c.ocrLang in (ctx.env.installedOcrPacks || {});
  if (needsOcr && (!offered || !installed)) {
    res.environmentGap = !offered
      ? `OCR language ${c.ocrLang} is not offered by either edition (internal/ocr/tessdata.go Available / ocr-lang.js LANGS)`
      : `OCR language ${c.ocrLang} is not installed on this machine`;
  }
  const ocrLang = offered && installed ? c.ocrLang : "";

  // An earlier settled, passing result with an identical key stands in for the collection (see
  // reuse.mjs); the verdict is still asked of today's judge afterwards.
  const keyInfo = caseKey({ edition: ctx.args.edition, ocrMode: ctx.args.ocr, c, ocrLang, env: ctx.env, producer: ctx.producer, expectation: expectationSha(REPO, c.id) });
  const reuse = tryReuse({ repo: REPO, searchRoot: ctx.searchRoot, passName: ctx.passName, c, source: got, keyInfo, to: dir, fresh: ctx.args.fresh });
  if (reuse.result) return reuse.result;
  if (!ctx.args.fresh) console.log(`  fresh  ${c.id}: ${reuse.miss}`);

  let url;
  if (ctx.args.edition === "windows") {
    const conv = convertWindows(ctx.args, c, dir, ocrLang);
    res.convert = { exitCode: conv.exitCode, ms: conv.ms, argv: conv.argv, timedOut: conv.timedOut };
    if (conv.exitCode !== 0 || !conv.index) { res.convert.output = conv.index ? "present" : "missing"; save(); return res; }
    res.convert.index = conv.index.slice(dir.length + 1);
    url = pathToFileURL(conv.index).href;
  } else {
    await setViewerOptions(ctx.browser, { ocrImages: ctx.args.ocr === "on", ocrLang: ocrLang || "eng" });
    url = `chrome-extension://${ctx.browser.extensionId}/src/viewer.html?file=${encodeURIComponent(`${ctx.base}/case/${c.id}/${basename(c.path)}`)}`;
  }

  const t0 = Date.now();
  await ctx.browser.cdp.send("Page.navigate", { url }, ctx.browser.session);
  await waitFor(`${c.id} to open`, async () => {
    try { return (await probe(ctx.browser)).state === "ready"; } catch { return false; }
  }, 120_000);
  // The viewer's OCR queue is fed by an IntersectionObserver, so every image has to pass through
  // the viewport; the desktop page only lazy-loads, which a browser starts well ahead of it.
  const walked = await walk(ctx.browser, ctx.budgetMs, ctx.args.edition === "extension" ? 0.9 : 2.5);
  res.renderMs = Date.now() - t0;
  res.truncated = walked.truncated;
  res.collection = walked.collection;
  if (!walked.probe) { save(); return res; } // the page never became ready: nothing to measure
  res.probe = { ...walked.probe, ...JSON.parse(await evaluate(ctx.browser.cdp, ctx.browser.session, DETAIL)) };
  res.screenshots = await capture(ctx.browser, dir, walked);
  // Written last, over the finished folder: the key a later run compares and the sizes it checks.
  res.reuse = { schema: REUSE_SCHEMA, key: keyInfo.key, keyDigest: keyInfo.digest, files: sizesOf(dir) };
  save();
  return res;
}

// ---- main --------------------------------------------------------------------

function startServer(cases) {
  const routes = new Map(cases.map((c) => [`/case/${c.id}/${basename(c.path)}`, c.path]));
  const server = createServer((req, res) => {
    const path = routes.get(decodeURIComponent(req.url.split("?")[0]));
    if (!path || !existsSync(path)) { res.writeHead(404).end(); return; }
    const body = readFileSync(path);
    res.writeHead(200, { "Content-Type": "application/octet-stream", "Content-Length": body.length, "Access-Control-Allow-Origin": "*" });
    res.end(body);
  });
  return new Promise((ok) => server.listen(0, "127.0.0.1", () => ok({ server, base: `http://127.0.0.1:${server.address().port}` })));
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) { console.log(USAGE); return; }
  args.cli = resolve(REPO, args.cli || join("temp", "doccorpus", "bin", "doc-html-translate.exe"));
  if (args.edition === "windows" && !existsSync(args.cli)) die(`no CLI at ${args.cli} - go build -o it first`);
  if (args.edition === "extension" && !existsSync(join(EXT_DIR, "vendor", "tesseract", "tesseract.esm.min.js"))) {
    die("extension/vendor is empty - run `npm run vendor` in extension/ first");
  }
  const cases = selectCases(args);
  const stamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d+Z$/, "").replace("T", "-");
  const runDir = resolve(REPO, args.run || join("temp", "doccorpus", stamp));
  const editionDir = join(runDir, `${args.edition}-${args.ocr}`);
  mkdirSync(editionDir, { recursive: true });

  const { server, base } = await startServer(cases);
  let seq = 0;
  let browser = await openBrowser(runDir, args.edition === "extension", seq);
  const env = environment(args, browser.product);
  writeFileSync(join(editionDir, "environment.json"), `${JSON.stringify(env, null, 2)}\n`);
  const ctx = {
    args, env, offered: env.offeredOcrPacks, editionDir, base, browser, budgetMs: Number(args.budget) * 1000,
    // Always computed, even for --fresh: a fresh result is recorded under its key so later runs can reuse it.
    producer: producerDigest(REPO),
    searchRoot: resolve(REPO, args.reuseFrom || join("temp", "doccorpus")),
    passName: `${args.edition}-${args.ocr}`,
  };
  console.log(`doccorpus: ${cases.length} case(s), ${args.edition}, ocr=${args.ocr}, ${browser.product}\n  -> ${editionDir}`);

  let reused = 0;
  try {
    for (const c of cases) {
      let r;
      try {
        r = await runCase(ctx, c);
      } catch (err) {
        const said = ctx.browser.cdp.dead ? ctx.browser.said() : "";
        r = { schema: RESULT_SCHEMA, caseId: c.id, edition: args.edition, ocrMode: args.ocr, error: said ? `${err.message}; the browser said: ${said}` : err.message };
        mkdirSync(join(editionDir, c.id), { recursive: true });
        writeFileSync(join(editionDir, c.id, "result.json"), `${JSON.stringify(r, null, 2)}\n`);
      }
      const p = r.probe;
      if (r.reusedFrom) {
        reused += 1;
        console.log(`  reused ${c.id} from ${r.reusedFrom.caseDir} (collected ${r.reusedFrom.collectedAt}, verdict ${r.reusedFrom.verdict})`);
      }
      const line = r.error ? `ERROR ${r.error}`
        : r.source !== "ok" ? `source ${r.source}`
        : r.convert && r.convert.exitCode !== 0 ? `convert exit ${r.convert.exitCode}`
        : p ? `text ${p.textChars} img ${p.imagesLoaded}/${p.images} plates ${p.plates} toc ${p.tocEntries}${r.truncated ? ` INCOMPLETE (${r.collection.outcome}, ${r.collection.stage}, ${r.collection.rendered}/${r.collection.total ?? "?"})` : ""}${p.notice ? ` notice "${p.notice}"` : ""}`
        : "no probe";
      console.log(`  ${c.id.padEnd(40)} ${line}`);
      if (ctx.browser.cdp.dead) {
        ctx.browser.child.kill();
        seq += 1;
        ctx.browser = await openBrowser(runDir, args.edition === "extension", seq);
      }
    }
  } finally {
    ctx.browser.cdp.close();
    ctx.browser.child.kill();
    server.close();
  }
  console.log(`doccorpus: collected ${cases.length - reused}, reused ${reused} of ${cases.length} case(s)`);
}

main().catch((err) => die(err.stack || err.message));

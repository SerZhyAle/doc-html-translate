// ocrlab.mjs - the browser edition's evidence producer for the OCR visual-fidelity lab.
//
// The counterpart of tools/ocrlab/runner: same corpus, same pinned viewports, same evidence
// record, so one metrics package grades both editions against one set of annotations. It measures
// the shipped extension - the real viewer, the real tesseract.js worker, the real overlay and its
// re-fit - and re-implements none of it, because a benchmark that scored its own reimplementation
// would be measuring the wrong thing.
//
// Why a hand-rolled DevTools client: the viewer only exists under a chrome-extension:// origin
// (chrome.storage does not exist on a file:// page), so the run has to drive a browser with the
// extension loaded. Node ships a global WebSocket, which is the only piece the DevTools protocol
// needs, so this stays dependency-free like build.mjs and make-screenshots.mjs. Scenes are served
// over a local node:http server rather than file://, so fetch, lazy loading and the
// IntersectionObserver behave as they do in production.
//
// Usage: npm run ocrlab -- --help (from extension/).

import { createServer } from "node:http";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { basename, dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { die } from "./_lib.mjs";
import { CDP, evaluate, findChrome, sleep, waitFor } from "./_ocrlab-cdp.mjs";
import { EDITION_EXTENSION, makeRun, makeScene, validateRun } from "./_ocrlab-evidence.mjs";
import { assembleToNatural, bandsFor } from "./_ocrlab-image.mjs";

const HERE = dirname(fileURLToPath(import.meta.url));
const EXT_DIR = resolve(HERE, "..");
const REPO_ROOT = resolve(EXT_DIR, "..");

// Defaults mirror tools/ocrlab/main.go addCommonFlags, so both editions read one corpus.
const DEFAULT_MANIFEST = join(REPO_ROOT, "DEV", "ocrlab", "corpus.json");
const DEFAULT_ROOT = join(REPO_ROOT, "test_doc", "ocrlab");
const MANIFEST_SCHEMA_VERSION = 1;

// Pinned, and identical to tools/ocrlab/runner.Viewports. A floating window would produce numbers
// nobody could reproduce, and recording the same plate at deliberately different geometries is
// the whole point of the drift measurement.
const VIEWPORTS = [
  { name: "desktop", width: 1280, height: 800, deviceScaleFactor: 1 },
  { name: "tablet", width: 768, height: 1024, deviceScaleFactor: 1 },
  { name: "phone", width: 390, height: 844, deviceScaleFactor: 2 },
];
const PRIMARY_VIEWPORT = VIEWPORTS[0].name;

// Translation-stress cases, ported verbatim from tools/ocrlab/runner/stress.go - same six names,
// same literals, same order; a parity test compares the two lists. They are literals rather than
// translations because the runner must work offline and call no paid service, and geometry does
// not care whether a string means anything: it cares how long it is, which script it is in and
// which way it runs. Fixing the strings is also what makes a clipping result reproducible.
const STRESS_CASES = [
  { name: "none", text: "", factor: 1, dir: "ltr" },
  { name: "short", text: "Yes.", factor: 0.2, dir: "ltr" },
  { name: "long-latin", text: "Ausführliche Erläuterung der Sachlage ", factor: 2.5, dir: "ltr" },
  { name: "long-cyrillic", text: "Подробное разъяснение обстоятельств дела ", factor: 2.5, dir: "ltr" },
  { name: "rtl-arabic", text: "شرح مفصل للحالة ", factor: 1.8, dir: "rtl" },
  { name: "cjk", text: "详细说明情况 ", factor: 1.2, dir: "ltr" },
];

// The case that represents "the page as recognized", kept in step with runner.PrimaryStress.
const BASE_STRESS = "none";

const OCR_TIMEOUT_MS = 180_000; // a full-page scan through tesseract.js is minutes, not seconds
const SETTLE_MS = 120;

const MIME = {
  ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
  ".gif": "image/gif", ".bmp": "image/bmp", ".webp": "image/webp",
};

// ---- arguments -------------------------------------------------------------

// Local rather than _lib.parseFlags because --scene is repeatable, exactly as the Go runner's
// -scene is, and parseFlags keeps only the last value of a repeated flag.
function parseArgs(argv) {
  const out = { split: "dev", lang: "eng", scene: [], manifest: DEFAULT_MANIFEST, root: DEFAULT_ROOT, out: "" };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--help" || a === "-h") return { help: true };
    if (!a.startsWith("--")) die(`unexpected argument ${a} - see --help`);
    const key = a.slice(2);
    const value = argv[++i];
    if (value === undefined || value.startsWith("--")) die(`--${key} needs a value`);
    if (key === "scene") out.scene.push(value);
    else if (key in out) out[key] = value;
    else die(`unknown flag --${key} - see --help`);
  }
  return out;
}

const USAGE = `ocrlab - the extension edition's evidence producer

Usage (from extension/):
  npm run ocrlab -- [flags]

Flags:
  --split <dev|holdout|all>   which scenes to run (default dev)
  --scene <id>                run only this scene (repeatable)
  --lang <code>               tesseract language (default eng)
  --manifest <path>           default ../DEV/ocrlab/corpus.json
  --root <path>               corpus media root, default ../test_doc/ocrlab
  --out <dir>                 run directory (default ../temp/ocrlab/ext-<timestamp>)

Environment:
  CHROME=<path>               the browser to drive (Chrome or Edge)
`;

// ---- corpus ----------------------------------------------------------------

// selectScenes reads the manifest and refuses an incomplete corpus, exactly as the Go runner
// does: a report over the scenes that happened to be present is the one failure mode the
// strategic spec names outright.
function selectScenes({ manifest, root, split, scene }) {
  if (!existsSync(manifest)) die(`no corpus manifest at ${manifest}`);
  const m = JSON.parse(readFileSync(manifest, "utf8"));
  if (m.schemaVersion !== MANIFEST_SCHEMA_VERSION) {
    die(`${manifest}: schemaVersion ${m.schemaVersion}, this runner understands ${MANIFEST_SCHEMA_VERSION}`);
  }
  const wanted = new Set(scene);
  const picked = (m.scenes || []).filter((s) => (wanted.size ? wanted.has(s.id) : split === "all" || s.split === split));
  for (const id of wanted) {
    if (!picked.some((s) => s.id === id)) die(`unknown scene id ${id}`);
  }
  if (!picked.length) die(`no scenes selected (split ${split})`);

  const bad = [];
  for (const s of picked) {
    const path = join(root, ...s.file.split("/"));
    if (!existsSync(path)) { bad.push(`${s.id}: no media at ${path}`); continue; }
    if (!s.sha256) { bad.push(`${s.id}: no sha256 recorded, so the bytes prove nothing`); continue; }
    const got = createHash("sha256").update(readFileSync(path)).digest("hex");
    if (got !== s.sha256) bad.push(`${s.id}: hash differs from the manifest`);
    else s.mediaPath = path;
  }
  if (bad.length) die(`refusing to run on an incomplete corpus:\n  ${bad.sort().join("\n  ")}`);
  return picked;
}

// ---- the scene server ------------------------------------------------------

function startSceneServer(scenes) {
  const routes = new Map();
  for (const s of scenes) {
    const ext = extname(s.mediaPath).toLowerCase();
    routes.set(`/scene/${s.id}${ext}`, [MIME[ext] || "application/octet-stream", readFileSync(s.mediaPath)]);
    s.route = `/scene/${s.id}${ext}`;
  }
  const server = createServer((req, res) => {
    const hit = routes.get(decodeURIComponent(req.url.split("?")[0]));
    if (!hit) { res.writeHead(404).end(); return; }
    res.writeHead(200, { "Content-Type": hit[0], "Content-Length": hit[1].length });
    res.end(hit[1]);
  });
  return new Promise((ok) => {
    server.listen(0, "127.0.0.1", () => ok({ server, base: `http://127.0.0.1:${server.address().port}` }));
  });
}

// ---- what the page is asked ------------------------------------------------

// The overlay container is the viewer's own; this only observes it. `ocr-empty` is a real
// outcome, not a failure - the recognizer found nothing and the metrics report zero recall.
const PAGE_STATE = `(() => {
  if (document.querySelector("#content .notice")) {
    return JSON.stringify({ state: "failed", error: (document.querySelector("#content .notice h1")||{}).textContent || "the viewer declined the image" });
  }
  const c = document.querySelector(".ocr-overlay");
  if (!c) return JSON.stringify({ state: "pending" });
  const img = c.querySelector(".ocr-overlay-img");
  if (!img || !img.complete || !img.naturalWidth) return JSON.stringify({ state: "pending" });
  return JSON.stringify({ state: "ready", plates: c.querySelectorAll(".ocr-plate").length });
})()`;

// Map every plate's laid-out rect back into the source image's own pixels. Every number the lab
// stores is in that space, so three viewports produce directly comparable geometry.
const READ_PLATES = `(() => {
  const c = document.querySelector(".ocr-overlay");
  const img = c && c.querySelector(".ocr-overlay-img");
  const ir = img && img.getBoundingClientRect();
  if (!ir || !ir.width || !ir.height) return JSON.stringify({ ok: false, error: "the overlay image has no laid-out size" });
  const sx = img.naturalWidth / ir.width, sy = img.naturalHeight / ir.height;
  const plates = [];
  c.querySelectorAll(".ocr-plate").forEach((p) => {
    const r = p.getBoundingClientRect(), cs = getComputedStyle(p);
    plates.push({
      text: p.textContent,
      rect: {
        x0: Math.round((r.left - ir.left) * sx), y0: Math.round((r.top - ir.top) * sy),
        x1: Math.round((r.right - ir.left) * sx), y1: Math.round((r.bottom - ir.top) * sy),
      },
      fontPx: parseFloat(cs.fontSize) || 0,
      background: cs.backgroundColor,
      ink: cs.color,
      mode: p.dataset.ocrMode || "fill",
      modeConfidence: parseFloat(p.dataset.ocrModeConf || "0") || 0,
      scrollHeight: p.scrollHeight,
      clientHeight: p.clientHeight,
    });
  });
  return JSON.stringify({
    ok: true,
    image: {
      x: ir.left + scrollX, y: ir.top + scrollY, width: ir.width, height: ir.height,
      naturalWidth: img.naturalWidth, naturalHeight: img.naturalHeight,
    },
    plates,
  });
})()`;

// APPLY_STRESS swaps every plate's text, ported from the desktop probe's applyCase. The original
// is remembered on the element so each case is applied to the source rather than to the previous
// case's output, and the swap is a real text-node mutation - which is what makes the extension's
// own MutationObserver re-fit, the event this measurement exists to catch.
const APPLY_STRESS = (c) => `(() => {
  const c = ${JSON.stringify(c)};
  const repeatTo = (text, targetLen) => {
    let out = text;
    while (out.length < targetLen) out += text;
    return out.slice(0, Math.max(targetLen, text.length)).trim();
  };
  document.querySelectorAll(".ocr-overlay .ocr-plate").forEach((p) => {
    if (p.dataset.ocrlabOriginal === undefined) p.dataset.ocrlabOriginal = p.textContent;
    const original = p.dataset.ocrlabOriginal;
    p.textContent = c.text ? repeatTo(c.text, Math.max(1, Math.round(original.length * c.factor))) : original;
    p.style.direction = c.dir === "rtl" ? "rtl" : "";
  });
  return true;
})()`;

// Force a synchronous reflow before anything is measured, then let the shipped re-fit's
// setTimeout(0) and its observers run.
async function settle(cdp, session) {
  await evaluate(cdp, session, "void document.body.offsetHeight");
  await sleep(SETTLE_MS);
}

// ---- one scene -------------------------------------------------------------

async function runScene(ctx, s) {
  const scene = { sceneId: s.id, plates: [], screenshots: { stress: {} } };
  const viewerBase = `chrome-extension://${ctx.extensionId}/src/viewer.html?file=`;
  const url = viewerBase + encodeURIComponent(`${ctx.base}${s.route}`);

  const renderStart = Date.now();
  for (const v of VIEWPORTS) {
    await ctx.cdp.send("Emulation.setDeviceMetricsOverride", {
      width: v.width, height: v.height, deviceScaleFactor: v.deviceScaleFactor, mobile: false,
    }, ctx.session);

    const ocrStart = Date.now();
    await ctx.cdp.send("Page.navigate", { url }, ctx.session);
    const state = await waitFor(`${s.id} to be recognized at ${v.name}`, async () => {
      try {
        const raw = await evaluate(ctx.cdp, ctx.session, PAGE_STATE);
        const got = JSON.parse(raw);
        return got.state === "pending" ? null : got;
      } catch {
        return null; // navigation in flight - the old context is gone, try again
      }
    }, OCR_TIMEOUT_MS);
    if (state.state === "failed") throw new Error(state.error);
    if (v.name === PRIMARY_VIEWPORT) scene.ocrMs = Date.now() - ocrStart;

    for (const c of STRESS_CASES) {
      await evaluate(ctx.cdp, ctx.session, APPLY_STRESS(c));
      await settle(ctx.cdp, ctx.session);
      const read = JSON.parse(await evaluate(ctx.cdp, ctx.session, READ_PLATES));
      if (!read.ok) throw new Error(`${v.name}/${c.name}: ${read.error}`);
      if (v.name === PRIMARY_VIEWPORT) {
        if (c.name === BASE_STRESS) {
          scene.imageWidth = read.image.naturalWidth;
          scene.imageHeight = read.image.naturalHeight;
          captureSource(ctx, s, scene);
        }
        await captureStress(ctx, s, read.image, scene, c.name);
      }
      for (const p of read.plates) scene.plates.push({ ...p, viewport: v.name, stressCase: c.name });
    }
  }
  scene.renderMs = Date.now() - renderStart - (scene.ocrMs || 0);
  scene.peakRssBytes = await heapUsed(ctx);
  return scene;
}

// The source needs no browser: it is the corpus file itself, copied so the run folder is
// self-contained and can be moved, zipped or attached to a report.
function captureSource(ctx, s, scene) {
  mkdirSync(join(ctx.outDir, "shots", s.id), { recursive: true });
  const source = join(ctx.outDir, "shots", s.id, "source.png");
  copyFileSync(s.mediaPath, source);
  scene.screenshots.source = relTo(ctx.outDir, source);
}

// captureStress records one stress case's rendered state, clipped to the image and rescaled to
// its natural pixels so the concealment measurement can compare it with the source directly.
async function captureStress(ctx, s, imageRect, scene, caseName) {
  const file = join(ctx.outDir, "shots", s.id, `${caseName}.png`);
  await captureImage(ctx, imageRect, file);
  scene.screenshots.stress[caseName] = relTo(ctx.outDir, file);
  if (caseName === BASE_STRESS) scene.screenshots.rendered = relTo(ctx.outDir, file);
}

// The image as laid out is captured at the viewport's own resolution and in bands, then stitched
// and rescaled into the source's pixel space locally. Asking the browser to render the whole clip
// at the natural size instead produces a reply too large for the DevTools websocket to carry - see
// _ocrlab-image.mjs for the measurement, and tools/ocrlab/runner/browser.go CropToImage for the
// Go runner resampling locally for the same reason.
async function captureImage(ctx, r, file) {
  const parts = [];
  for (const band of bandsFor(r.width, r.height)) {
    const shot = await ctx.cdp.send("Page.captureScreenshot", {
      format: "png",
      captureBeyondViewport: true,
      clip: {
        x: r.x,
        y: r.y + (band.y0 * r.height) / Math.round(r.height),
        width: r.width,
        height: ((band.y1 - band.y0) * r.height) / Math.round(r.height),
        scale: 1,
      },
    }, ctx.session);
    parts.push({ png: Buffer.from(shot.data, "base64"), y0: band.y0, y1: band.y1 });
  }
  writeFileSync(file, await assembleToNatural(parts, r.width, r.height, r.naturalWidth, r.naturalHeight));
}

// heapUsed is the browser's own JS heap after the scene. Like the desktop runner's peakRSS it is
// an honest in-process proxy for the cost dimension rather than the operating system's RSS, and
// it moves when a scene makes the recognizer work, which is what the measurement is for.
async function heapUsed(ctx) {
  try {
    const { metrics } = await ctx.cdp.send("Performance.getMetrics", {}, ctx.session);
    return Math.round(metrics.find((m) => m.name === "JSHeapUsedSize")?.value || 0);
  } catch {
    return 0;
  }
}

const relTo = (base, path) => resolve(path).slice(resolve(base).length + 1).split("\\").join("/");

// ---- engine identity -------------------------------------------------------

// The engine is named from what is actually loaded, not from a mirrored constant. The traineddata
// is fingerprinted by its bytes exactly as the desktop runner does, so the two editions' engine
// records are directly comparable and a language-data swap shows up in both.
function engineFor(lang) {
  let tesseract = "tesseract.js (version unknown)";
  const pkg = join(EXT_DIR, "node_modules", "tesseract.js", "package.json");
  if (existsSync(pkg)) tesseract = `tesseract.js ${JSON.parse(readFileSync(pkg, "utf8")).version}`;

  let tessdataVersion = "";
  const data = join(EXT_DIR, "vendor", "tesseract", "lang", `${lang.split("+")[0]}.traineddata`);
  if (existsSync(data)) {
    tessdataVersion = `sha256:${createHash("sha256").update(readFileSync(data)).digest("hex").slice(0, 16)}`;
  }
  return { tesseract, tessdataVersion, lang };
}

// ---- the browser ------------------------------------------------------------

// openBrowser launches a headless browser, loads the unpacked extension, opens one page and pins
// the extension's options. It is a function rather than a stretch of main() because a scene can
// kill the renderer - an 11-megapixel scan through tesseract.js does - and a run that cannot
// start a second browser loses every scene after the one that crashed.
async function openBrowser(args, outDir, base, seq) {
  const profile = join(outDir, `.browser-profile${seq ? `-${seq}` : ""}`);
  const child = spawn(findChrome(), [
    "--headless=new",
    `--user-data-dir=${profile}`,
    "--remote-debugging-port=0",
    `--window-size=${VIEWPORTS[0].width},${VIEWPORTS[0].height}`,
    "--hide-scrollbars", "--no-first-run", "--no-default-browser-check",
    "--disable-features=Translate,MediaRouter",
    "--disable-background-timer-throttling",
    "about:blank",
  ], { stdio: ["ignore", "ignore", "pipe"] });
  let stderr = "";
  child.stderr.on("data", (d) => { stderr += d.toString(); });

  // The port file is re-read on every attempt, not once: a browser starting on a brand-new
  // profile can write a port, exit and be replaced by a second process on a different one, and
  // connecting to the first port then fails with ECONNREFUSED for a browser that is running fine.
  const portFile = join(profile, "DevToolsActivePort");
  const cdp = await waitFor("the browser's debugging port", async () => {
    if (!existsSync(portFile)) return null;
    const port = readFileSync(portFile, "utf8").split("\n")[0].trim();
    if (!port) return null;
    try {
      const { webSocketDebuggerUrl } = await (await fetch(`http://127.0.0.1:${port}/json/version`)).json();
      return await CDP.connect(webSocketDebuggerUrl);
    } catch {
      return null; // not accepting yet, or this port belonged to a process that has gone
    }
  }, 30_000).catch((e) => {
    throw new Error(`${e.message}${stderr ? `\nthe browser said: ${stderr.trim()}` : ""}`);
  });

  const { product } = await cdp.send("Browser.getVersion");
  const loaded = await cdp.send("Extensions.loadUnpacked", { path: EXT_DIR }).catch((e) => {
    die(`Extensions.loadUnpacked failed: ${e.message}\n` +
      "This browser is too old for the DevTools Extensions domain - use Chrome or Edge 12x or newer.");
  });
  const extensionId = loaded.id;

  const { targetId } = await cdp.send("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await cdp.send("Target.attachToTarget", { targetId, flatten: true });
  await cdp.send("Page.enable", {}, sessionId);
  await cdp.send("Runtime.enable", {}, sessionId);
  await cdp.send("Performance.enable", {}, sessionId).catch(() => { /* cost sample only */ });

  // Seed the extension's own options from an extension page, so the run pins the OCR language
  // instead of inheriting whatever a profile happened to hold.
  await cdp.send("Page.navigate", { url: `chrome-extension://${extensionId}/src/options.html` }, sessionId);
  await waitFor("the options page", async () => {
    try { return await evaluate(cdp, sessionId, "typeof chrome !== 'undefined' && !!chrome.storage"); }
    catch { return false; }
  }, 30_000);
  await evaluate(cdp, sessionId,
    `chrome.storage.local.set({options: {ocrImages: true, ocrLang: ${JSON.stringify(args.lang)}}, uiLang: "en"})`, true);

  // The browser's own words, kept for the scene that kills it. "the DevTools connection errored"
  // says a browser died; it does not say why, and a lab that records a death without its cause
  // makes the next person reproduce it by hand.
  const said = () => stderr.trim().split("\n").slice(-6).join("\n");
  return { child, cdp, product, said, ctx: { cdp, session: sessionId, extensionId, base, outDir } };
}

// ---- main ------------------------------------------------------------------

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) { console.log(USAGE); return; }

  if (!existsSync(join(EXT_DIR, "vendor", "tesseract", "tesseract.esm.min.js"))) {
    die("extension/vendor is empty - run `npm run vendor` first, the overlay needs tesseract.js");
  }
  const scenes = selectScenes(args);
  const stamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d+Z$/, "").replace("T", "-");
  const outDir = args.out ? resolve(args.out) : join(REPO_ROOT, "temp", "ocrlab", `ext-${stamp}`);
  mkdirSync(outDir, { recursive: true });

  const { server, base } = await startSceneServer(scenes);
  const run = makeRun({
    runId: basename(outDir),
    startedAt: new Date().toISOString().replace(/\.\d+Z$/, "Z"),
    edition: EDITION_EXTENSION,
    engine: engineFor(args.lang),
    viewports: VIEWPORTS,
  });
  let browser;
  try {
    browser = await openBrowser(args, outDir, base, 0);
    run.browser = { name: /edg/i.test(browser.product) ? "edge" : "chrome", version: browser.product };
    console.log(`ocrlab: ${scenes.length} scene(s), ${browser.product}, ${run.engine.tesseract}`);

    let relaunches = 0;
    for (const s of scenes) {
      try {
        run.scenes.push(makeScene(await runScene(browser.ctx, s)));
        const last = run.scenes[run.scenes.length - 1];
        console.log(`  ok   ${s.id.padEnd(40)} ${last.plates.length} plate-record(s), ocr ${last.ocrMs}ms`);
      } catch (err) {
        const said = browser.cdp.dead ? browser.said() : "";
        const why = said ? `${err.message}; the browser said: ${said}` : err.message;
        run.scenes.push(makeScene({ sceneId: s.id, error: why }));
        console.log(`  FAIL ${s.id.padEnd(40)} ${why}`);
      }
      // A scene can take the renderer down with it, and the next scene is then not a measurement
      // of anything. Start a fresh browser and carry on, so a crash costs the scene that caused it
      // rather than every scene after it. The crashed scene keeps its recorded error either way -
      // a scene that did not run must stay visible, never be dropped from the count.
      if (browser.cdp.dead) {
        browser.child.kill();
        relaunches += 1;
        console.log(`       .. the browser died on that scene, starting a fresh one (#${relaunches})`);
        browser = await openBrowser(args, outDir, base, relaunches);
      }
    }
  } finally {
    browser?.cdp.close();
    browser?.child.kill();
    server.close();
  }

  const evidence = join(outDir, "evidence.json");
  writeFileSync(evidence, `${JSON.stringify(run, null, 2)}\n`);
  const problems = validateRun(run);
  console.log(`\nevidence: ${evidence}`);
  if (problems.length) {
    console.error(`\nthe evidence does not satisfy the schema (${problems.length}):`);
    for (const p of problems) console.error(`  ${p}`);
    process.exit(1);
  }
}

await main();

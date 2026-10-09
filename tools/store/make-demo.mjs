// make-demo.mjs - records the silent product demo as a frame sequence: the master that
// make-demo.ps1 encodes into assets/demo.mp4 and assets/demo.gif.
//
// Everything on screen that is the product is the product's own output: the console lines are the
// real binary's stdout, and the pages are the HTML it converted, driven in headless Chrome through
// the page's own controls (contents button, text-size and theme controls, OCR toggle). The sample
// book and the comic page are authored for the demo (make-demo-assets.mjs); nothing is third-party.
// Chrome's own "Translate page" bar is native browser chrome and cannot be captured from a headless
// page, so it is not drawn or imitated - the closing card states that path instead.
//
// A frame is recorded only when something changes: each snapshot is held for a duration, and
// frames.txt (an ffconcat list) carries those durations exactly, so two runs of the same input
// give the same master and the encoder does the rest.
//
// Usage (make-demo.ps1 runs it): node tools/store/make-demo.mjs --cli <doc-html-translate.exe>
//   --work <dir> --frames <dir> [--tesseract <tesseract.exe>]

import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { CDP, evaluate, findChrome, sleep, waitFor } from "../../extension/scripts/_ocrlab-cdp.mjs";
import { die, parseFlags } from "../../extension/scripts/_lib.mjs";
import {
  BOOK_FILE, COMIC_FILE, COMIC_SIZE, captionScript, clearCaptionScript, closingHtml, comicHtml,
  consoleHtml, introHtml, sampleBookMarkdown,
} from "./make-demo-assets.mjs";

const VIEW = { width: 1280, height: 720 };
// The master's grid. An interface that changes in steps needs no more, and the GIF may not exceed
// 15 fps; the encoder pads the MP4 to 30.
const FPS = 15;
const CONSOLE_COMMAND = `doc-html-translate -noopen "${BOOK_FILE}"`;
// The converter writes next to the input, into a folder named after it.
const BOOK_DIR = BOOK_FILE.replace(/\.md$/i, "");
const COMIC_DIR = COMIC_FILE.replace(/\.png$/i, "");

const flags = parseFlags(process.argv.slice(2));
for (const need of ["cli", "work", "frames"]) if (typeof flags[need] !== "string") die(`missing --${need}`);
const CLI = resolve(flags.cli);
const WORK = resolve(flags.work);
const FRAMES = resolve(flags.frames);

// ---- the browser -------------------------------------------------------------------------

let child, cdp, session;

async function openBrowser() {
  const profile = join(WORK, "chrome-profile");
  child = spawn(findChrome(), [
    "--headless=new", `--user-data-dir=${profile}`, "--remote-debugging-port=0",
    `--window-size=${VIEW.width},${VIEW.height}`, "--hide-scrollbars", "--no-first-run",
    "--no-default-browser-check", "--disable-features=Translate,MediaRouter",
    "--disable-background-timer-throttling", "--force-device-scale-factor=1", "about:blank",
  ], { stdio: ["ignore", "ignore", "pipe"] });
  let stderr = "";
  child.stderr.on("data", (d) => { stderr += d.toString(); });

  const portFile = join(profile, "DevToolsActivePort");
  try {
    cdp = await waitFor("the browser's debugging port", async () => {
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
  } catch (e) {
    throw new Error(`${e.message}${stderr ? `\nchrome said: ${stderr.trim()}` : ""}`);
  }
  const { targetId } = await cdp.send("Target.createTarget", { url: "about:blank" });
  ({ sessionId: session } = await cdp.send("Target.attachToTarget", { targetId, flatten: true }));
  await cdp.send("Page.enable", {}, session);
  await cdp.send("Runtime.enable", {}, session);
  await setViewport(VIEW);
}

const setViewport = ({ width, height }) =>
  cdp.send("Emulation.setDeviceMetricsOverride", { width, height, deviceScaleFactor: 1, mobile: false }, session);

const run = (expression, awaitPromise = false) => evaluate(cdp, session, expression, awaitPromise);

// goto navigates and waits until the document is the one asked for and its marker element exists:
// the old document stays "complete" for a moment after Page.navigate returns, so readyState alone
// would capture the page being left.
async function goto(file, marker, selector) {
  await cdp.send("Page.navigate", { url: pathToFileURL(file).href }, session);
  await waitFor(`${marker} to load`, async () => {
    try {
      return await run(`document.readyState === "complete" && decodeURIComponent(location.pathname).includes(${JSON.stringify(marker)}) && !!document.querySelector(${JSON.stringify(selector)})`);
    } catch {
      return false; // navigation in flight - the old context is gone
    }
  }, 30_000);
}

async function centreOf(selector) {
  const at = await run(`(() => { const e = document.querySelector(${JSON.stringify(selector)});
    if (!e) return null; const b = e.getBoundingClientRect(); return { x: b.x + b.width / 2, y: b.y + b.height / 2 }; })()`);
  if (!at) throw new Error(`nothing there: ${selector}`);
  return at;
}

// hover moves the pointer onto an element, so its own hover style shows in the frame.
async function hover(selector) {
  const { x, y } = await centreOf(selector);
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseMoved", x, y }, session);
}

async function click(selector) {
  await hover(selector);
  const { x, y } = await centreOf(selector);
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x, y, button: "left", clickCount: 1 }, session);
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x, y, button: "left", clickCount: 1 }, session);
}

// ---- the frame sequence ------------------------------------------------------------------

const frames = [];
const scenes = [];
const dropped = [];

// snap records what is on screen for `seconds`. Two animation frames first, so a style change made
// by the last step has been painted.
async function snap(seconds) {
  await run("new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))", true);
  const shot = await cdp.send("Page.captureScreenshot", { format: "png" }, session);
  const png = Buffer.from(shot.data, "base64");
  // The encoder would rescale a frame of another size without a word, so the size is checked here.
  const [w, h] = [png.readUInt32BE(16), png.readUInt32BE(20)];
  if (w !== VIEW.width || h !== VIEW.height) throw new Error(`a frame is ${w}x${h}, expected ${VIEW.width}x${VIEW.height}`);
  const file = `k${String(frames.length + 1).padStart(4, "0")}.png`;
  writeFileSync(join(FRAMES, file), png);
  frames.push({ file, count: Math.max(1, Math.round(seconds * FPS)) });
}

const caption = (text, accent) => run(captionScript(text, accent));

async function scene(name, body) {
  const before = frames.reduce((n, f) => n + f.count, 0);
  await body();
  const count = frames.reduce((n, f) => n + f.count, 0) - before;
  scenes.push({ name, seconds: +(count / FPS).toFixed(2) });
  console.log(`  ${name.padEnd(12)} ${(count / FPS).toFixed(1)} s`);
}

function writeMaster() {
  const lines = ["ffconcat version 1.0"];
  for (const f of frames) lines.push(`file '${f.file}'`, `duration ${(f.count / FPS).toFixed(6)}`);
  // Older ffmpeg builds drop the last entry's duration unless the file is named once more; newer ones
  // honour it, so the repeat carries one frame's worth and costs 1/15 s either way.
  lines.push(`file '${frames.at(-1).file}'`, `duration ${(1 / FPS).toFixed(6)}`);
  writeFileSync(join(FRAMES, "frames.txt"), lines.join("\n") + "\n");
  const total = frames.reduce((n, f) => n + f.count, 0);
  writeFileSync(join(FRAMES, "scenes.json"), JSON.stringify({ fps: FPS, frames: total, seconds: +(total / FPS).toFixed(2), scenes, dropped }, null, 2));
  return total / FPS;
}

// ---- sources, conversion -----------------------------------------------------------------

function writeSources() {
  const dirs = { book: join(WORK, "book"), comic: join(WORK, "comic"), pages: join(WORK, "pages") };
  for (const d of Object.values(dirs)) mkdirSync(d, { recursive: true });
  writeFileSync(join(dirs.book, BOOK_FILE), sampleBookMarkdown(), "utf8");
  const pages = {
    intro: ["intro.html", introHtml()], closing: ["closing.html", closingHtml()],
    console: ["console.html", consoleHtml()], comic: ["comic.html", comicHtml()],
  };
  const files = {};
  for (const [key, [name, html]] of Object.entries(pages)) {
    files[key] = join(dirs.pages, name);
    writeFileSync(files[key], html, "utf8");
  }
  return { dirs, files };
}

// The converter runs from the folder that holds its input with a bare file name, as a person would
// run it, so the console shows no path.
function convert(cwd, args) {
  const env = { ...process.env };
  if (typeof flags.tesseract === "string") env.DOCHT_TESSERACT = flags.tesseract;
  const res = spawnSync(CLI, args, { cwd, env, encoding: "utf8" });
  if (res.status !== 0) throw new Error(`${CLI} ${args.join(" ")} exited ${res.status}\n${res.stdout}${res.stderr}`);
  return res.stdout;
}

// consoleLines turns the binary's stdout into what the console scene prints. The clock prefix is
// dropped so two runs draw the same pixels, and so is the banner line, which carries the build's
// version; every other line is verbatim.
function consoleLines(stdout) {
  return stdout.split(/\r?\n/).filter(Boolean).map((l) => l.replace(/^\[\d{2}:\d{2}:\d{2}\] ?/, "")).filter((l) => !/^doc-html-translate \S+$/.test(l));
}

async function renderComicPage(file, out) {
  await setViewport(COMIC_SIZE);
  await goto(file, "comic.html", "svg");
  const shot = await cdp.send("Page.captureScreenshot", { format: "png" }, session);
  writeFileSync(out, Buffer.from(shot.data, "base64"));
  await setViewport(VIEW);
}

// ---- the scenes --------------------------------------------------------------------------

async function sceneIntro(files) {
  await goto(files.intro, "intro.html", "main");
  await snap(2.6);
}

async function sceneConsole(files, lines) {
  await goto(files.console, "console.html", "#t");
  const term = (typed, printed, caret) =>
    run(`window.setTerm(${JSON.stringify(typed)}, ${JSON.stringify(printed)}, ${caret})`);
  await caption("Convert a book with one command");
  await term("", [], true);
  await snap(0.6);
  for (let n = 5; n < CONSOLE_COMMAND.length; n += 5) {
    await term(CONSOLE_COMMAND.slice(0, n), [], true);
    await snap(0.1);
  }
  await term(CONSOLE_COMMAND, [], true);
  await snap(0.6);
  for (let i = 1; i <= lines.length; i++) {
    await term(CONSOLE_COMMAND, lines.slice(0, i), false);
    await snap(0.4);
  }
  await snap(1.8);
}

async function sceneContents(bookIndex) {
  await goto(bookIndex, BOOK_DIR, ".dht-navbar");
  await caption("The result is an ordinary local web page");
  await snap(2.4);
  await click("#dht-contents-button");
  await waitFor("the contents panel", () => run(`!document.getElementById("dht-contents").hidden`), 5_000, 100);
  await caption("A real multi-level table of contents, generated for you", "multi-level");
  await snap(3.6);
  // The first link is the book title; the demo jumps to a chapter of part two instead, with a real
  // click on its link (the attribute only lets the click find it).
  await run(`(() => { const a = [...document.querySelectorAll("#dht-contents a")].find((x) => /^7\\. /.test(x.textContent));
    if (!a) throw new Error("no chapter 7 in the contents"); a.setAttribute("data-demo-target", ""); })()`);
  await click("[data-demo-target]");
  await sleep(250);
  await caption("Jump to any chapter in one click");
  await snap(2.6);
}

// A size change reflows the text above the reader, so the same scroll offset lands elsewhere in the
// book; the chapter the reader was on is brought back to the top, as a reader would scroll to it.
const keepChapterInView = () => run(`(() => { const h = [...document.querySelectorAll("main h3")].find((x) => /^7\\. /.test(x.textContent)); h.scrollIntoView(); })()`);

async function sceneReader() {
  await caption("Text size, font and spacing at a click");
  for (let i = 0; i < 3; i++) {
    await click("#dht-font-inc");
    await keepChapterInView();
    await snap(0.7);
  }
  await click("#dht-size-reset");
  await keepChapterInView();
  await snap(1.0);
  await caption("Light, sepia, dark and night reading themes", "sepia, dark and night");
  await hover("#dht-theme-sel");
  for (const [theme, hold] of [["sepia", 1.3], ["dark", 1.3], ["night", 1.3], ["light", 1.0]]) {
    await run(`(() => { const s = document.getElementById("dht-theme-sel"); s.value = ${JSON.stringify(theme)}; s.dispatchEvent(new Event("change", { bubbles: true })); })()`);
    await snap(hold);
  }
}

async function sceneComic(comicIndex) {
  await goto(comicIndex, COMIC_DIR, ".dht-navbar");
  await waitFor("the OCR plates", () => run(`document.querySelectorAll(".ocr-fig").length > 0`), 10_000, 100);
  await caption("Scans and comic pages get a real text layer too");
  await snap(2.8);
  await click("#dht-ocr-toggle");
  await caption("Switch it off and the original page is intact");
  await snap(1.8);
  await click("#dht-ocr-toggle");
  await snap(0.8);
  // Select the plates only (the picture precedes them in the DOM), as a reader would to copy the text.
  await run(`(() => { const p = document.querySelectorAll(".ocr-box"); const r = document.createRange();
    r.setStartBefore(p[0]); r.setEndAfter(p[p.length - 1]); const s = getSelection(); s.removeAllRanges(); s.addRange(r); })()`);
  await caption("Real text, so the browser can translate it", "the browser can translate it");
  await snap(2.6);
  await run(clearCaptionScript);
}

async function sceneClosing(files) {
  await goto(files.closing, "closing.html", "main");
  await snap(7.5);
}

// ---- main --------------------------------------------------------------------------------

async function main() {
  if (!existsSync(CLI)) die(`${CLI} not found - make-demo.ps1 builds it`);
  rmSync(FRAMES, { recursive: true, force: true });
  mkdirSync(FRAMES, { recursive: true });
  const { dirs, files } = writeSources();

  await openBrowser();
  try {
    console.log("Converting the sample book with the real binary..");
    const stdout = convert(dirs.book, ["-noopen", BOOK_FILE]);
    const lines = consoleLines(stdout);
    if (!lines.some((l) => l.startsWith("Done."))) throw new Error(`the converter did not finish:\n${stdout}`);
    const bookIndex = join(dirs.book, BOOK_DIR, "index.html");

    // The comic page is optional: it needs Tesseract and the bundled English data. A converter run
    // that leaves no plate is reported and the scene dropped, never faked.
    let comicIndex = null;
    try {
      await renderComicPage(files.comic, join(dirs.comic, COMIC_FILE));
      convert(dirs.comic, ["-ocr", "-noopen", COMIC_FILE]);
      comicIndex = join(dirs.comic, COMIC_DIR, "index.html");
      const html = readFileSync(comicIndex, "utf8");
      const plates = (html.match(/class="ocr-box"/g) ?? []).length;
      console.log(`  comic page: ${plates} OCR plate(s)`);
      if (plates < 3) {
        dropped.push({ name: "comic", reason: `OCR left ${plates} plate(s) on the comic page` });
        comicIndex = null;
      }
    } catch (e) {
      dropped.push({ name: "comic", reason: String(e.message).split("\n")[0] });
      comicIndex = null;
    }

    console.log("Recording scenes..");
    await scene("intro", () => sceneIntro(files));
    await scene("console", () => sceneConsole(files, lines));
    await scene("contents", () => sceneContents(bookIndex));
    await scene("reader", sceneReader);
    if (comicIndex) await scene("comic", () => sceneComic(comicIndex));
    await scene("closing", () => sceneClosing(files));
  } finally {
    cdp?.close();
    child?.kill();
  }
  const seconds = writeMaster();
  console.log(`Master: ${frames.length} snapshot(s), ${seconds.toFixed(1)} s at ${FPS} fps${dropped.length ? `, dropped: ${dropped.map((d) => d.name).join(", ")}` : ""}`);
}

await main();

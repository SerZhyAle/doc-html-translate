// build.mjs - dev tooling for the extension.
//
//   node build.mjs vendor   copy the pieces of pdfjs-dist we ship into vendor/
//   node build.mjs zip      produce a store-ready dist/doc-html-translate-extension.zip
//
// We vendor the *non-minified* pdfjs build on purpose: Chrome Web Store review is
// friendlier to readable third-party code than to minified blobs, and the size cost
// (~3 MB) is irrelevant for a locally-installed extension.

import { cp, mkdir, readFile, rm, stat, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { gunzipSync } from "node:zlib";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";

const root = dirname(fileURLToPath(import.meta.url));
const pdfjs = join(root, "node_modules", "pdfjs-dist");
const tesseract = join(root, "node_modules", "tesseract.js");
const tesseractCore = join(root, "node_modules", "tesseract.js-core");
const vendor = join(root, "vendor");

// tessdata_fast is tesseract.js's own default host; the English data is fetched once
// at build time so English OCR ships fully offline (see the OCR spec's privacy note).
// Version 4.0.0 must match ocr-lang.js CDN_LANG_PATH and the desktop app's pinned
// tessdata version (internal/ocr/tessdata.go cdnBase) - see ../docs/PARITY.md ("OCR").
const TESSDATA_BASE = "https://tessdata.projectnaptha.com/4.0.0_fast";

async function vendorPdfjs() {
  if (!existsSync(pdfjs)) {
    console.error("pdfjs-dist not installed. Run `npm install` first.");
    process.exit(1);
  }
  await rm(vendor, { recursive: true, force: true });
  await mkdir(vendor, { recursive: true });

  // Runtime files: the main entry + the worker. ES module builds.
  await cp(join(pdfjs, "build", "pdf.mjs"), join(vendor, "pdf.mjs"));
  await cp(join(pdfjs, "build", "pdf.worker.mjs"), join(vendor, "pdf.worker.mjs"));

  // Character maps (CJK etc.) and standard-font fallbacks improve text extraction
  // quality for PDFs that don't embed their fonts. Pointed at via getDocument().
  await cp(join(pdfjs, "cmaps"), join(vendor, "cmaps"), { recursive: true });
  await cp(join(pdfjs, "standard_fonts"), join(vendor, "standard_fonts"), { recursive: true });

  // Carry the upstream licence next to the vendored code.
  await cp(join(pdfjs, "LICENSE"), join(vendor, "LICENSE.pdfjs"));

  const version = JSON.parse(
    await import("node:fs/promises").then((fs) =>
      fs.readFile(join(pdfjs, "package.json"), "utf8"),
    ),
  ).version;
  console.log(`Vendored pdfjs-dist@${version} into vendor/`);
}

// Vendor the Tesseract.js OCR engine + bundled English data into vendor/tesseract/.
// Runs after vendorPdfjs() (which wipes vendor/), so it only writes its own subtree.
//
// minimum_chrome_version 105 guarantees WASM SIMD, and tesseract.js runs LSTM-only by
// default, so the single self-contained core `tesseract-core-simd-lstm.wasm.js` (wasm
// embedded) is the only core file the worker ever importScripts() - no need to ship the
// ~30 MB of non-SIMD / combined variants.
async function vendorTesseract() {
  if (!existsSync(tesseract) || !existsSync(tesseractCore)) {
    console.error("tesseract.js / tesseract.js-core not installed. Run `npm install` first.");
    process.exit(1);
  }
  const out = join(vendor, "tesseract");
  const langDir = join(out, "lang");
  await mkdir(langDir, { recursive: true });

  // Main-thread ESM entry + the worker script (loaded via workerPath at runtime).
  await cp(join(tesseract, "dist", "tesseract.esm.min.js"), join(out, "tesseract.esm.min.js"));
  await cp(join(tesseract, "dist", "worker.min.js"), join(out, "worker.min.js"));

  // The one core build the SIMD + LSTM-only path uses.
  await cp(
    join(tesseractCore, "tesseract-core-simd-lstm.wasm.js"),
    join(out, "tesseract-core-simd-lstm.wasm.js"),
  );

  // Neutralize Tesseract.js's baked-in CDN fallbacks. worker.min.js / tesseract.esm.min.js
  // ship jsdelivr defaults for corePath (the WASM core), the worker script and langPath; the
  // engine would importScripts()/import() them only when those paths are not supplied. We
  // always supply explicit local paths (ocr-lang.js workerAssets/workerOptions), so the
  // defaults are dead code - but Chrome Web Store MV3 review scans the *packaged* bytes and
  // rejects a `https://cdn.jsdelivr.net/...` load of executable code as "remotely hosted
  // code" (traineddata and other data are exempt; JS/WASM is not). Rewrite the CDN base to an
  // extension-local path so no remote-code URL ships. Behaviour is unchanged because the
  // defaults are never taken; the assertion fails the build if a Tesseract.js upgrade adds a
  // new remote default, so we catch it here rather than at store review.
  const cdnBase = "https://cdn.jsdelivr.net/npm/";
  const localBase = "vendor/tesseract/";
  for (const name of ["worker.min.js", "tesseract.esm.min.js"]) {
    const p = join(out, name);
    const sanitized = (await readFile(p, "utf8")).replaceAll(cdnBase, localBase);
    await writeFile(p, sanitized);
    if (/https?:\/\/[^\s"'`]*cdn\.jsdelivr\.net/.test(sanitized)) {
      console.error(`${name}: a jsdelivr CDN reference survived sanitization - Tesseract.js changed its remote defaults; update build.mjs.`);
      process.exit(1);
    }
  }

  // Bundled English data: fetch the gzipped tessdata_fast file once and store it
  // decompressed so the runtime loads it with gzip:false and no network.
  const url = `${TESSDATA_BASE}/eng.traineddata.gz`;
  const resp = await fetch(url);
  if (!resp.ok) {
    console.error(`Failed to fetch ${url}: ${resp.status} ${resp.statusText}`);
    process.exit(1);
  }
  const gz = new Uint8Array(await resp.arrayBuffer());
  const data = gunzipSync(gz);
  await writeFile(join(langDir, "eng.traineddata"), data);

  // Carry the upstream licences next to the vendored code.
  await cp(join(tesseract, "LICENSE.md"), join(out, "LICENSE.tesseract"));
  await cp(join(tesseractCore, "LICENSE"), join(out, "LICENSE.tesseract-core"));

  const version = JSON.parse(
    await import("node:fs/promises").then((fs) =>
      fs.readFile(join(tesseract, "package.json"), "utf8"),
    ),
  ).version;
  console.log(
    `Vendored tesseract.js@${version} + eng.traineddata (${(data.length / 1024 / 1024).toFixed(1)} MB) into vendor/tesseract/`,
  );
}

// Vendor the marked Markdown->HTML converter (single ESM file). Runs after the
// pdfjs/tesseract vendoring (which wiped vendor/), so it only adds its own files.
async function vendorMarked() {
  const marked = join(root, "node_modules", "marked");
  if (!existsSync(marked)) {
    console.error("marked not installed. Run `npm install` first.");
    process.exit(1);
  }
  await cp(join(marked, "lib", "marked.esm.js"), join(vendor, "marked.esm.js"));
  await cp(join(marked, "LICENSE.md"), join(vendor, "LICENSE.marked"));
  const version = JSON.parse(
    await import("node:fs/promises").then((fs) =>
      fs.readFile(join(marked, "package.json"), "utf8"),
    ),
  ).version;
  console.log(`Vendored marked@${version} into vendor/`);
}

// Vendor the foliate-js MOBI/KF8 reader for MOBI and AZW3. mobi.js is self-contained
// (no local imports) and receives its unzlib via the constructor, so only that one
// module plus fflate (KF8 font decompression) need vendoring. Pinned via package.json.
async function vendorFoliate() {
  const foliate = join(root, "node_modules", "foliate-js");
  const fflate = join(root, "node_modules", "fflate");
  if (!existsSync(foliate) || !existsSync(fflate)) {
    console.error("foliate-js / fflate not installed. Run `npm install` first.");
    process.exit(1);
  }
  const out = join(vendor, "foliate");
  await mkdir(out, { recursive: true });
  await cp(join(foliate, "mobi.js"), join(out, "mobi.js"));
  await cp(join(foliate, "LICENSE"), join(out, "LICENSE.foliate"));
  await cp(join(fflate, "esm", "browser.js"), join(vendor, "fflate.js"));
  await cp(join(fflate, "LICENSE"), join(vendor, "LICENSE.fflate"));
  const version = JSON.parse(
    await import("node:fs/promises").then((fs) =>
      fs.readFile(join(foliate, "package.json"), "utf8"),
    ),
  ).version;
  console.log(`Vendored foliate-js@${version} mobi.js + fflate into vendor/`);
}

// Exactly what the extension needs at runtime. An allow-list (not a deny-list) so
// dev/listing artifacts - store/, README.md, tests - can never silently leak into
// the store package.
const PACKAGE = ["manifest.json", "_locales", "src", "vendor", "icons"];

async function zip() {
  // The package is broken without the vendored pdfjs build; fail loudly rather
  // than shipping a viewer whose `import ../vendor/pdf.mjs` 404s.
  if (!existsSync(join(root, "vendor", "pdf.mjs"))) {
    console.error("vendor/pdf.mjs missing - run `npm run vendor` before zipping.");
    process.exit(1);
  }
  for (const e of PACKAGE) {
    if (!existsSync(join(root, e))) {
      console.error(`required package entry missing: ${e}`);
      process.exit(1);
    }
  }

  const dist = join(root, "dist");
  await mkdir(dist, { recursive: true });
  const out = join(dist, "doc-html-translate-extension.zip");
  await rm(out, { force: true });

  // Dependency-free zip, cross-platform so CI can run on cheap Linux runners:
  //   - Windows (dev machines): PowerShell Compress-Archive.
  //   - Linux/macOS (CI, ubuntu-latest): the `zip` CLI, run from `root` so the
  //     entries land at the archive root (matching Compress-Archive's layout).
  let r;
  if (process.platform === "win32") {
    const paths = PACKAGE.map((e) => `'${resolve(root, e).replace(/'/g, "''")}'`).join(",");
    const ps = `Compress-Archive -Path ${paths} -DestinationPath '${out.replace(/'/g, "''")}' -Force`;
    r = spawnSync("powershell", ["-NoProfile", "-Command", ps], { stdio: "inherit" });
  } else {
    // -r recurse dirs, -q quiet; relative entry names preserved via cwd: root.
    r = spawnSync("zip", ["-r", "-q", out, ...PACKAGE], { cwd: root, stdio: "inherit" });
  }
  if (r.status !== 0) {
    console.error("zip failed");
    process.exit(1);
  }
  const { size } = await stat(out);
  console.log(`Wrote ${out} (${(size / 1024 / 1024).toFixed(1)} MB)`);
}

// Stamp the extension version from the local date-time, like the desktop app's build.ps1
// but WITHOUT leading zeros: Chrome and Edge require every dot-separated part to be an
// integer 0-65535 with no leading zero on a non-zero part, so Edge rejects "26.0702.1342"
// (the "0702" part). Encode month/day and hour/minute as plain integers instead - identical
// to the desktop MSIX mapping (26.0702.1342 -> 26.702.1342). Still monotonic (mmdd 101..1231,
// hhmm 0..2359) and human-readable. Writes manifest.json + package.json in lockstep.
function dateVersion() {
  const d = new Date();
  const yy = d.getFullYear() % 100;
  const mmdd = (d.getMonth() + 1) * 100 + d.getDate();
  const hhmm = d.getHours() * 100 + d.getMinutes();
  return `${yy}.${mmdd}.${hhmm}`;
}

async function stamp() {
  const version = dateVersion();
  for (const file of ["manifest.json", "package.json"]) {
    const path = join(root, file);
    const json = JSON.parse(await readFile(path, "utf8"));
    json.version = version;
    await writeFile(path, JSON.stringify(json, null, 2) + "\n");
  }
  console.log(`Stamped version ${version} (manifest.json + package.json).`);
  return version;
}

const cmd = process.argv[2];
if (cmd === "stamp") { await stamp(); }
else if (cmd === "vendor") { await vendorPdfjs(); await vendorTesseract(); await vendorMarked(); await vendorFoliate(); }
else if (cmd === "zip") await zip();
else {
  console.error("usage: node build.mjs <stamp|vendor|zip>");
  process.exit(1);
}

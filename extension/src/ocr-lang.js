// ocr-lang.js - the single authority for OCR languages: the offline English path,
// on-demand download + IndexedDB caching of other languages, and installed-state.
//
// English (eng) ships in vendor/tesseract/lang/ (decompressed, loaded with gzip:false)
// and works with no network. Other languages are fetched on explicit user action from
// the tessdata_fast CDN and cached by Tesseract.js in IndexedDB; a language counts as
// "installed" only after a worker has successfully initialized with it.

import Tesseract from "../vendor/tesseract/tesseract.esm.min.js";

const { createWorker } = Tesseract;

// Supported OCR languages. `code` is the Tesseract traineddata name; jpn_vert is the
// vertical-text model used for manga. This catalog is shared with the desktop app's
// internal/ocr/tessdata.go `Available` - keep the two in sync (see ../../docs/PARITY.md).
export const LANGS = [
  { code: "eng", name: "English" },
  { code: "rus", name: "Russian" },
  { code: "ukr", name: "Ukrainian" },
  { code: "jpn", name: "Japanese" },
  { code: "jpn_vert", name: "Japanese (vertical)" },
  { code: "deu", name: "German" },
  { code: "fra", name: "French" },
  { code: "spa", name: "Spanish" },
  { code: "ita", name: "Italian" },
  { code: "por", name: "Portuguese" },
  { code: "pol", name: "Polish" },
  { code: "chi_sim", name: "Chinese (simplified)" },
  { code: "kor", name: "Korean" },
];

// langLabel renders a "+"-joined language string for a reader: each code keeps its traineddata
// name and gains the catalog's display name where there is one, so a line about "rus" also says
// Russian. The code stays first because it is what the user picks from the list. Twin of the
// desktop app's ocr.LangLabel (internal/ocr/tessdata.go) - see ../../docs/PARITY.md.
export function langLabel(lang) {
  const parts = String(lang || "")
    .split("+")
    .map((c) => c.trim())
    .filter(Boolean)
    .map((c) => {
      const known = LANGS.find((l) => l.code === c);
      return known ? `${c} (${known.name})` : c;
    });
  return parts.length ? parts.join(" + ") : String(lang || "");
}

// Languages that ship inside the extension package (fully offline, no network).
export const BUNDLED = ["eng"];

// tessdata_fast on tesseract.js's own default host - matches the bundled eng build.
// Version 4.0.0 must match build.mjs TESSDATA_BASE and the desktop app's pinned tessdata
// version (internal/ocr/tessdata.go cdnBase) so recognition matches - see ../../docs/PARITY.md.
export const CDN_LANG_PATH = "https://tessdata.projectnaptha.com/4.0.0_fast";

// storage key holding the list of successfully-downloaded language codes.
const INSTALLED_KEY = "ocrInstalledLangs";

export function isBundled(code) {
  return BUNDLED.includes(code);
}

// Directory that directly contains the bundled eng.traineddata.
export function localLangPath() {
  return chrome.runtime.getURL("vendor/tesseract/lang/");
}

// Where a language's traineddata is loaded from: local for bundled English, the CDN
// (with IndexedDB caching) for everything else. This is the seam the overlay engine
// consumes when it creates the recognition worker.
export function resolveLangPath(code) {
  return isBundled(code) ? localLangPath() : CDN_LANG_PATH;
}

// Vendored engine asset paths, shared by the recognition worker (Phase 03) and the
// download prewarm below. workerBlobURL:false keeps the worker a direct extension URL
// (MV3 CSP disallows wrapping it in a blob:).
export function workerAssets() {
  return {
    workerPath: chrome.runtime.getURL("vendor/tesseract/worker.min.js"),
    corePath: chrome.runtime.getURL("vendor/tesseract/tesseract-core-simd-lstm.wasm.js"),
    workerBlobURL: false,
  };
}

// Full createWorker options for a given language. Bundled English needs no cache and
// is not gzipped; downloaded languages are gzipped and cached in IndexedDB so later
// sessions load them offline.
export function workerOptions(code) {
  const bundled = isBundled(code);
  return {
    ...workerAssets(),
    langPath: resolveLangPath(code),
    gzip: !bundled,
    cacheMethod: bundled ? "none" : "write",
  };
}

// The set of currently usable languages: the always-present bundled set unioned with
// any that have been downloaded. English is reported installed with no storage entry.
export async function getInstalledLangs() {
  let downloaded = [];
  try {
    const got = await chrome.storage.local.get(INSTALLED_KEY);
    downloaded = got[INSTALLED_KEY] || [];
  } catch { /* storage may be unavailable in odd contexts */ }
  return Array.from(new Set([...BUNDLED, ...downloaded]));
}

export async function isInstalled(code) {
  return (await getInstalledLangs()).includes(code);
}

// Record a language as installed. Bundled languages are always installed and never
// written to storage.
export async function markInstalled(code) {
  if (isBundled(code)) return;
  try {
    const got = await chrome.storage.local.get(INSTALLED_KEY);
    const set = new Set(got[INSTALLED_KEY] || []);
    set.add(code);
    await chrome.storage.local.set({ [INSTALLED_KEY]: [...set] });
  } catch { /* ignore */ }
}

// Download a language on explicit user request: spin up a throwaway worker so
// Tesseract.js fetches the .traineddata from the CDN and caches it in IndexedDB, then
// mark it installed. Progress arrives via onProgress({status, progress}). Marking only
// happens after a successful init; a failed download throws and installs nothing.
export async function downloadLang(code, onProgress) {
  if (isBundled(code)) return;
  const options = workerOptions(code);
  if (onProgress) options.logger = (m) => onProgress(m);
  const worker = await createWorker(code, 1, options);
  await worker.terminate();
  await markInstalled(code);
}

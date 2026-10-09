// _ocrlab-lang.mjs - which OCR language a scene is read with, the extension lab's half of
// tools/ocrlab/evidence/lang.go.
//
// The lab used to read the whole campaign with one language while the truth declared Russian,
// French and English. A scene now carries its own declared language unless the operator forces one.
// The mapping tables and the order of precedence below are the Go lab's, entry for entry;
// TestParityOCRLabLangTable fails when either side or the app's own mapping (internal/ocr TessLang)
// moves alone, and the run cross-checks every scene against the declaration the Go side froze.

import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

export const DEFAULT_LANG = "eng";
export const LANG_PER_SCENE = "per-scene";

export const LANG_SOURCE_FLAG = "flag";
export const LANG_SOURCE_TRUTH = "truth";
export const LANG_SOURCE_CORPUS = "corpus";
export const LANG_SOURCE_DEFAULT = "default";

// Only OCR-catalogue packs (extension/src/ocr-lang.js LANGS): a declared language the catalogue
// cannot read has no entry and falls back to DEFAULT_LANG.
export const ISO_TO_TESS = {
  en: "eng", ru: "rus", uk: "ukr", de: "deu", fr: "fra",
  es: "spa", it: "ita", pt: "por", pl: "pol", ja: "jpn",
  zh: "chi_sim", ko: "kor",
};

// Subtags that select a different pack than their base language; consulted before the base.
export const REGION_TO_TESS = {
  "zh-hant": "chi_tra", "zh-tw": "chi_tra", "zh-hk": "chi_tra", "zh-mo": "chi_tra",
};

// tessCode maps one declared language to its catalogue pack, or null.
export function tessCode(declared) {
  const norm = String(declared ?? "").trim().replaceAll("_", "-").toLowerCase();
  if (Object.hasOwn(REGION_TO_TESS, norm)) return REGION_TO_TESS[norm];
  const base = norm.split("-")[0];
  return Object.hasOwn(ISO_TO_TESS, base) ? ISO_TO_TESS[base] : null;
}

function joinCodes(declared) {
  const codes = [];
  for (const d of declared || []) {
    const t = tessCode(d);
    if (t && !codes.includes(t)) codes.push(t);
  }
  return codes.join("+");
}

// loadAnnotation reads a scene's human-reviewed annotation, or null when it is absent or unreadable.
export function loadAnnotation(dir, sceneId) {
  const path = join(dir, `${sceneId}.json`);
  if (!existsSync(path)) return null;
  try {
    return JSON.parse(readFileSync(path, "utf8"));
  } catch {
    return null;
  }
}

// An OCR-seeded draft is the engine's own output, so its languages declare nothing.
const isTruth = (a) => Boolean(a) && a.origin === "human" && String(a.review?.annotatedBy ?? "").trim() !== "";

// sceneLang decides a scene's OCR language: an explicit operator choice wins, then the languages of
// the human-reviewed annotation's groups, then the manifest's languages, then DEFAULT_LANG. Several
// declared languages become one "+"-joined value in order of first appearance.
export function sceneLang({ explicit = "", annotation = null, scene = {} } = {}) {
  if (explicit) return { lang: explicit, source: LANG_SOURCE_FLAG };
  if (isTruth(annotation)) {
    const lang = joinCodes((annotation.groups || []).map((g) => g.language));
    if (lang) return { lang, source: LANG_SOURCE_TRUTH };
  }
  const lang = joinCodes(scene.languages);
  if (lang) return { lang, source: LANG_SOURCE_CORPUS };
  return { lang: DEFAULT_LANG, source: LANG_SOURCE_DEFAULT };
}

// missingLangData lists the packs of lang that are not vendored. The extension loads every other
// language from a CDN into the browser's cache on a user's explicit action; the lab has no opt-in
// for network access, so such a scene is recorded as unmeasured rather than read with the wrong one.
export function missingLangData(lang, langDir) {
  return String(lang).split("+").map((c) => c.trim())
    .filter((c) => c && !existsSync(join(langDir, `${c}.traineddata`)));
}

// runLangs names the run's language for the evidence header and the packs whose bytes identify its
// engine, as runner.runLangs does on the Go side.
export function runLangs(scenes, explicit, langDir) {
  if (explicit) return { label: explicit, packs: explicit };
  const codes = [];
  for (const s of scenes) {
    for (const code of s.lang.split("+")) {
      if (!missingLangData(code, langDir).length && !codes.includes(code)) codes.push(code);
    }
  }
  return { label: LANG_PER_SCENE, packs: codes.sort().join("+") };
}

// declarationProblems compares each scene's language with the one the Go side froze in
// declaration.json. A difference means the two tables or the precedence drifted; finding it before
// an hour of recognition is cheaper than finding it in the evidence issues afterwards.
export function declarationProblems(scenes, declaration) {
  const problems = [];
  for (const s of scenes) {
    const want = declaration?.scenes?.[s.id]?.lang;
    if (want !== undefined && want !== s.lang) problems.push(`${s.id}: this runner reads ${s.lang}, the declaration says ${want}`);
  }
  return problems;
}

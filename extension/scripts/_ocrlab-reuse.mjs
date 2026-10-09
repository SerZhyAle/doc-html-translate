// _ocrlab-reuse.mjs - the extension producer's side of scene reuse.
//
// Whether an earlier complete run can stand in for collecting a scene, and the copy of its captures,
// is decided once, in Go (tools/ocrlab/evidence reuse.go), so both editions apply the same rule and
// write the same `reusedFrom` field. This file only asks `ocrlab reuse` and hands back the record
// it prints. Collection is the expensive stage; scoring and the gate always rerun afterwards on
// whatever evidence the run ends up with.
//
// A lookup that fails for any reason is a miss, never an error: the scene is collected, which is
// always correct, and the reason is logged so a broken helper does not silently cost the saving.

import { spawnSync } from "node:child_process";

// reuseArgv is the argument list of `go run ./tools/ocrlab reuse` for one scene. The engine and
// browser are what only this producer knows: the key compares them with what the earlier run
// recorded, restricted to the language packs the scene reads.
export function reuseArgv({ outDir, sceneId, engine, browser, reuseFrom }) {
  return [
    "run", "./tools/ocrlab", "reuse",
    "-run", outDir,
    "-edition", "extension",
    "-scene", sceneId,
    "-tesseract", engine.tesseract,
    "-tessdata", engine.tessdataVersion,
    "-browser-name", browser.name,
    "-browser-version", browser.version,
    ...(reuseFrom ? ["-reuse-from", reuseFrom] : []),
  ];
}

const lastLine = (text) => String(text || "").trim().split(/\r?\n/).filter(Boolean).pop() || "";

// reuseScene returns { scene, bundle } when the scene was copied from an earlier run, or
// { miss: <reason> } when it must be collected. `fresh` skips the lookup altogether (--fresh).
// The language fields are overwritten with this run's own choice: the earlier record may have been
// read under another flag for the same language, and the declaration froze this run's.
export function reuseScene({ repoRoot, outDir, scene, engine, browser, reuseFrom = "", fresh = false, run = spawnSync }) {
  if (fresh) return { miss: "--fresh" };
  const r = run("go", reuseArgv({ outDir, sceneId: scene.id, engine, browser, reuseFrom }), {
    cwd: repoRoot, encoding: "utf8", maxBuffer: 64 * 1024 * 1024,
  });
  if (r.error || r.status !== 0) {
    return { miss: `lookup failed: ${r.error ? r.error.message : lastLine(r.stderr) || lastLine(r.stdout)}` };
  }
  const reply = lastLine(r.stdout);
  if (reply === "null") return { miss: lastLine(r.stderr).replace(/^no reuse for [^:]+: /, "") || "no reusable record" };
  let record;
  try {
    record = JSON.parse(reply);
  } catch {
    return { miss: "lookup reply is not JSON" };
  }
  if (!record || record.sceneId !== scene.id || !record.reusedFrom) {
    return { miss: "lookup reply is not a reused record of this scene" };
  }
  return { scene: { ...record, lang: scene.lang, langSource: scene.langSource }, bundle: record.reusedFrom.bundle };
}

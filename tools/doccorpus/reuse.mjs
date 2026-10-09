// reuse.mjs - the document campaign does not repeat a case whose inputs are identical to a
// previous complete run that passed.
//
// A long document case costs up to half an hour, and two unchanged passes over the same bytes with
// the same code answer the same question twice. A case is reused when its reuse key matches an
// earlier result exactly; any difference - the source, the edition, the OCR mode, one byte of
// product code, the browser, the CLI, the expectation - forces a fresh collection. What is reused
// is the collection only. The verdict is always asked of today's judge (`doccorpus verdict`) and
// reuse needs it to be PASS or PASS WITH ADVISORIES, so a changed judge takes effect without a
// browser run and an unsettled, timed-out or failing case is always collected again.
//
// Provenance is never silent: the copied result.json carries `reusedFrom`, run.mjs logs the reuse
// and the report reads it.

import { createHash } from "node:crypto";
import { spawnSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readdirSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { join, relative, resolve } from "node:path";

export const REUSE_SCHEMA = 1;

// The source bytes a collection depends on, relative to the repository. Test files and testdata
// are excluded (nothing in a collection executes them); everything else counts, whatever its
// extension, and the list is conservative on purpose.
export const PRODUCER_ENTRIES = ["internal", "cmd/doc-html-translate", "extension/src", "tools/doccorpus"];

const PASSING = new Set(["PASS", "PASS WITH ADVISORIES"]);

export const isTestFile = (name) => /_test\.go$|\.test\.(mjs|js)$/.test(name);

const sha256 = (data) => createHash("sha256").update(data).digest("hex");

function collectFiles(dir, out) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) {
      if (e.name !== "testdata" && e.name !== "node_modules") collectFiles(p, out);
    } else if (e.isFile() && !isTestFile(e.name)) {
      out.push(p);
    }
  }
}

// producerDigest hashes every non-test byte of the entries under repo. A missing entry throws: a
// smaller digest from a half-present tree would let two different trees compare equal.
export function producerDigest(repo, entries = PRODUCER_ENTRIES) {
  const files = [];
  for (const entry of entries) {
    const full = join(repo, ...entry.split("/"));
    if (statSync(full).isDirectory()) collectFiles(full, files);
    else files.push(full);
  }
  const rel = (p) => relative(repo, p).split("\\").join("/");
  files.sort((a, b) => (rel(a) < rel(b) ? -1 : rel(a) > rel(b) ? 1 : 0));
  const h = createHash("sha256");
  for (const f of files) h.update(`${rel(f)}\0${sha256(readFileSync(f))}\0`);
  return h.digest("hex");
}

const canonical = (v) => {
  if (Array.isArray(v)) return `[${v.map(canonical).join(",")}]`;
  if (v && typeof v === "object") return `{${Object.keys(v).sort().map((k) => `${JSON.stringify(k)}:${canonical(v[k])}`).join(",")}}`;
  return JSON.stringify(v ?? null);
};

export function expectationSha(repo, caseId) {
  const p = join(repo, "DEV", "doccorpus", "expect", `${caseId}.json`);
  return existsSync(p) ? sha256(readFileSync(p)) : "none";
}

// caseKey is everything one case's collection depends on. The windows and extension editions
// differ in `edition` and in the tool identity, so their keys cannot collide.
export function caseKey({ edition, ocrMode, c, ocrLang, env, producer, expectation }) {
  const tool = edition === "windows"
    ? { cliVersion: env.cliVersion || "", cliSha256: env.cliSha256 || "", tesseract: env.tesseract || "", pack: env.installedOcrPacks?.[ocrLang] || "", pdftotext: env.pdftotext || "" }
    : { extensionVersion: env.extensionVersion || "", tesseractJs: env.tesseractJs || "" };
  const key = {
    schema: REUSE_SCHEMA,
    edition,
    ocrMode,
    source: c.sha256,
    case: { file: c.file, class: c.class, language: c.language, ocrLang },
    expectation,
    producer,
    platform: `${process.platform}/${process.arch}`,
    browser: env.browser || "",
    tool,
  };
  return { key, digest: sha256(canonical(key)) };
}

// diffKey names the components in which two keys differ, as dotted paths.
export function diffKey(a, b, prefix = "") {
  const out = [];
  for (const k of new Set([...Object.keys(a || {}), ...Object.keys(b || {})])) {
    const x = a?.[k], y = b?.[k];
    if (x && y && typeof x === "object" && typeof y === "object") out.push(...diffKey(x, y, `${prefix}${k}.`));
    else if (canonical(x) !== canonical(y)) out.push(`${prefix}${k}`);
  }
  return out;
}

// sizesOf lists every file of a case folder with its size, except result.json (which carries the
// list). Written when a case finishes, so a later run can tell the folder is still what it left.
export function sizesOf(dir) {
  const out = {};
  const walk = (d) => {
    for (const e of readdirSync(d, { withFileTypes: true })) {
      const p = join(d, e.name);
      if (e.isDirectory()) walk(p);
      else if (e.isFile()) out[relative(dir, p).split("\\").join("/")] = statSync(p).size;
    }
  };
  walk(dir);
  delete out["result.json"];
  return out;
}

function sameSizes(want, have) {
  const wk = Object.keys(want), hk = Object.keys(have);
  if (wk.length !== hk.length) return `${hk.length} files, the case recorded ${wk.length}`;
  for (const k of wk) if (have[k] !== want[k]) return `a file changed since the case finished: ${k}`;
  return "";
}

function candidateDirs(searchRoot, passName, caseId) {
  if (!existsSync(searchRoot)) return [];
  const runs = existsSync(join(searchRoot, passName))
    ? [searchRoot]
    : readdirSync(searchRoot, { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => join(searchRoot, e.name));
  return runs.map((r) => join(r, passName, caseId)).filter((d) => existsSync(join(d, "result.json")));
}

// goVerdict asks today's judge for the verdict of one case folder.
export function goVerdict(repo, caseDir) {
  const r = spawnSync("go", ["run", "./tools/doccorpus", "verdict", caseDir], { cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024 });
  if (r.error || r.status !== 0) return { auto: "", error: r.error ? r.error.message : (r.stderr || r.stdout || "").trim().split(/\r?\n/).pop() };
  try {
    return JSON.parse(r.stdout.trim().split(/\r?\n/).pop());
  } catch {
    return { auto: "", error: "the judge's reply is not JSON" };
  }
}

// findReusable returns { dir, result, verdict } for the newest earlier case folder that may stand
// in for collecting this case, or { miss: <reason> } naming why the closest candidate was refused.
// searchRoot is one run directory or a folder of runs; passName is "<edition>-<ocr mode>".
export function findReusable({ searchRoot, passName, caseId, source, key, digest, exclude, verdictOf }) {
  const dirs = candidateDirs(searchRoot, passName, caseId)
    .filter((d) => resolve(d) !== resolve(exclude))
    .map((d) => ({ d, at: statSync(join(d, "result.json")).mtimeMs }))
    .sort((a, b) => b.at - a.at)
    .map((x) => x.d);
  let closest = "", rank = -1;
  const note = (r, dir, why) => { if (r > rank) { rank = r; closest = `${dir.split("\\").join("/")}: ${why}`; } };
  for (const dir of dirs) {
    let res;
    try {
      res = JSON.parse(readFileSync(join(dir, "result.json"), "utf8"));
    } catch {
      note(0, dir, "result.json is unreadable");
      continue;
    }
    if (!res.reuse || res.reuse.schema !== REUSE_SCHEMA || !res.reuse.key) { note(0, dir, "predates case reuse (no reuse key)"); continue; }
    if (res.caseId !== caseId || res.source !== "ok" || res.sourceSha256 !== source) { note(1, dir, "not the same source bytes"); continue; }
    if (res.reuse.keyDigest !== digest) { note(3, dir, `key differs: ${diffKey(res.reuse.key, key).join(", ") || "digest"}`); continue; }
    if (res.error) { note(2, dir, `the case errored: ${res.error}`); continue; }
    if (res.collection?.outcome !== "settled" || res.truncated) { note(2, dir, `collection was not settled (${res.collection?.outcome || "no collection"})`); continue; }
    const why = sameSizes(res.reuse.files || {}, sizesOf(dir));
    if (why) { note(4, dir, why); continue; }
    const verdict = verdictOf(dir);
    if (!PASSING.has(verdict.auto)) { note(5, dir, `the judge's verdict is ${verdict.auto || "unavailable"}${verdict.error ? ` (${verdict.error})` : ""}`); continue; }
    return { dir, result: res, verdict };
  }
  return { miss: closest || "no earlier result" };
}

// copyCase puts the earlier case folder into the new run and marks the copy. The copy is checked
// against the sizes the original recorded; on any failure nothing is left behind.
export function copyCase({ hit, to, repo, now = () => new Date() }) {
  if (existsSync(to) && readdirSync(to).length) throw new Error("the target case folder is not empty");
  try {
    cpSync(hit.dir, to, { recursive: true, force: false, errorOnExist: true });
    const why = sameSizes(hit.result.reuse.files, sizesOf(to));
    if (why) throw new Error(`the copy differs from the original: ${why}`);
    const res = JSON.parse(readFileSync(join(to, "result.json"), "utf8"));
    res.reusedFrom = {
      caseDir: relative(repo, hit.dir).split("\\").join("/"),
      keyDigest: res.reuse.keyDigest,
      producer: res.reuse.key.producer,
      collectedAt: res.reusedFrom?.collectedAt || res.startedAt,
      reusedAt: now().toISOString(),
      verdict: hit.verdict.auto,
    };
    writeFileSync(join(to, "result.json"), `${JSON.stringify(res, null, 2)}\n`);
    return res;
  } catch (err) {
    rmSync(to, { recursive: true, force: true });
    mkdirSync(to, { recursive: true });
    throw err;
  }
}

// tryReuse is the one call run.mjs makes. { result } is a copied, marked result; { miss } says why
// the case is collected. `fresh` skips the lookup altogether.
export function tryReuse({ repo, searchRoot, passName, c, source, keyInfo, to, fresh = false, verdictOf = (d) => goVerdict(repo, d), now }) {
  if (fresh) return { miss: "--fresh" };
  const hit = findReusable({ searchRoot, passName, caseId: c.id, source, key: keyInfo.key, digest: keyInfo.digest, exclude: to, verdictOf });
  if (hit.miss) return hit;
  try {
    return { result: copyCase({ hit, to, repo, now }) };
  } catch (err) {
    return { miss: `copy from ${hit.dir} failed: ${err.message}` };
  }
}

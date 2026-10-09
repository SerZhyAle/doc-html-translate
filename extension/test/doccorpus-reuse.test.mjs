// Tests for the document campaign's case reuse (tools/doccorpus/reuse.mjs).
//
// The rule they pin: a case is copied from an earlier run only when every input is identical, the
// earlier collection settled, its folder is still what it left and today's judge passes it. Any
// other case is collected again, and a reuse is never silent. The judge is a stub here; the real
// one is `doccorpus verdict`, tested in Go.

import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, utimesSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  REUSE_SCHEMA, caseKey, diffKey, findReusable, isTestFile, producerDigest, sizesOf, tryReuse,
} from "../../tools/doccorpus/reuse.mjs";

const tmp = () => mkdtempSync(join(tmpdir(), "doccorpus-reuse-"));
const put = (path, text) => { mkdirSync(join(path, ".."), { recursive: true }); writeFileSync(path, text); };

const CASE = { id: "ja-epub-x", file: "ja/x.epub", class: "epub", language: "ja", sha256: "a".repeat(64) };
const ENV_WINDOWS = {
  browser: "Chrome/151.0.1", cliVersion: "26.1009.1", cliSha256: "c".repeat(64), tesseract: "tesseract 5.4",
  installedOcrPacks: { jpn: "sha256:1234" }, pdftotext: "next to the CLI",
};
const ENV_EXTENSION = { browser: "Chrome/151.0.1", extensionVersion: "26.1009.1", tesseractJs: "7.0.0" };
const PASSING = () => ({ auto: "PASS" });

const keyOf = (over = {}) => caseKey({
  edition: "windows", ocrMode: "default", c: CASE, ocrLang: "jpn", env: ENV_WINDOWS, producer: "p".repeat(64), expectation: "e".repeat(64), ...over,
});

// An earlier run: <root>/<run>/<pass>/<case>/ holding a finished, settled result and its files.
function earlierRun(root, { run = "earlier", pass = "windows-default", keyInfo = keyOf(), edit = () => {}, files = { "text.txt": "hello", "html/index.html": "<html></html>" } } = {}) {
  const dir = join(root, run, pass, CASE.id);
  for (const [name, text] of Object.entries(files)) put(join(dir, name), text);
  const res = {
    schema: 1, caseId: CASE.id, edition: "windows", ocrMode: "default", startedAt: "2026-10-09T10:00:00.000Z",
    source: "ok", sourceSha256: CASE.sha256, truncated: false,
    collection: { outcome: "settled", stage: "settled" }, probe: { textChars: 5 },
    reuse: { schema: REUSE_SCHEMA, key: keyInfo.key, keyDigest: keyInfo.digest, files: sizesOf(dir) },
  };
  edit(res, dir);
  writeFileSync(join(dir, "result.json"), JSON.stringify(res));
  return dir;
}

function reuse(root, over = {}) {
  const keyInfo = over.keyInfo || keyOf();
  const to = join(root, "later", "windows-default", CASE.id);
  mkdirSync(to, { recursive: true });
  return tryReuse({
    repo: root, searchRoot: root, passName: "windows-default", c: CASE, source: CASE.sha256, keyInfo, to,
    verdictOf: PASSING, now: () => new Date("2026-10-09T11:00:00.000Z"), ...over,
  });
}

test("unchanged inputs reuse a settled passing case and the copy says where it came from", () => {
  const root = tmp();
  const from = earlierRun(root);
  const got = reuse(root);
  assert.ok(got.result, got.miss);
  assert.equal(got.result.reusedFrom.verdict, "PASS");
  assert.equal(got.result.reusedFrom.collectedAt, "2026-10-09T10:00:00.000Z");
  assert.equal(got.result.reusedFrom.reusedAt, "2026-10-09T11:00:00.000Z");
  assert.match(got.result.reusedFrom.caseDir, /earlier\/windows-default\/ja-epub-x$/);
  assert.equal(readFileSync(join(root, "later", "windows-default", CASE.id, "html", "index.html"), "utf8"), "<html></html>");
  const written = JSON.parse(readFileSync(join(root, "later", "windows-default", CASE.id, "result.json"), "utf8"));
  assert.deepEqual(written.reusedFrom, got.result.reusedFrom, "the marked result is what is on disk");
  assert.ok(!("reusedFrom" in JSON.parse(readFileSync(join(from, "result.json"), "utf8"))), "the original is untouched");
  rmSync(root, { recursive: true });
});

test("PASS WITH ADVISORIES reuses; FAIL, COULD NOT VERIFY and an unavailable judge never do", () => {
  const root = tmp();
  earlierRun(root);
  assert.ok(reuse(root, { verdictOf: () => ({ auto: "PASS WITH ADVISORIES" }) }).result);
  for (const verdict of [{ auto: "FAIL" }, { auto: "COULD NOT VERIFY" }, { auto: "", error: "go: not found" }]) {
    const got = reuse(root, { verdictOf: () => verdict });
    assert.ok(got.miss && !got.result, `${verdict.auto || "unavailable"} must be collected again`);
    assert.match(got.miss, /judge's verdict/);
  }
  rmSync(root, { recursive: true });
});

test("an unsettled, timed-out, truncated or errored case is collected again", () => {
  const edits = {
    timeout: (r) => { r.collection = { outcome: "timeout", stage: "rendering-active" }; r.truncated = true; },
    "producer error": (r) => { r.collection = { outcome: "producer-error", stage: "producer-error" }; },
    "no collection record": (r) => { delete r.collection; },
    truncated: (r) => { r.truncated = true; },
    errored: (r) => { r.error = "the browser died"; },
  };
  for (const [name, edit] of Object.entries(edits)) {
    const root = tmp();
    earlierRun(root, { edit });
    assert.ok(reuse(root).miss, `${name} must not be reused`);
    rmSync(root, { recursive: true });
  }
});

test("a case that predates reuse, or has other source bytes, is never reused", () => {
  const root = tmp();
  earlierRun(root, { edit: (r) => { delete r.reuse; } });
  assert.match(reuse(root).miss, /predates case reuse/);
  rmSync(root, { recursive: true });

  const other = tmp();
  earlierRun(other, { edit: (r) => { r.sourceSha256 = "b".repeat(64); } });
  assert.match(reuse(other).miss, /source bytes/);
  rmSync(other, { recursive: true });
});

test("each key component changed alone forces a fresh collection", () => {
  const root = tmp();
  earlierRun(root);
  const variants = {
    "edition": keyOf({ edition: "extension", env: ENV_EXTENSION }),
    "ocr mode": keyOf({ ocrMode: "on" }),
    "producer": keyOf({ producer: "q".repeat(64) }),
    "browser": keyOf({ env: { ...ENV_WINDOWS, browser: "Chrome/152.0.0" } }),
    "cli bytes": keyOf({ env: { ...ENV_WINDOWS, cliSha256: "d".repeat(64) } }),
    "cli version": keyOf({ env: { ...ENV_WINDOWS, cliVersion: "26.1010.1" } }),
    "tesseract": keyOf({ env: { ...ENV_WINDOWS, tesseract: "tesseract 5.5" } }),
    "language pack": keyOf({ env: { ...ENV_WINDOWS, installedOcrPacks: { jpn: "sha256:9999" } } }),
    "ocr language": keyOf({ ocrLang: "" }),
    "expectation": keyOf({ expectation: "x".repeat(64) }),
    "case record": keyOf({ c: { ...CASE, language: "zh" } }),
    "source bytes": keyOf({ c: { ...CASE, sha256: "f".repeat(64) } }),
  };
  const base = keyOf().digest;
  for (const [name, keyInfo] of Object.entries(variants)) {
    assert.notEqual(keyInfo.digest, base, `${name} must change the key`);
    const got = reuse(root, { keyInfo });
    assert.ok(got.miss && !got.result, `${name} must not reuse`);
    assert.match(got.miss, /key differs|source bytes/, name);
  }
  rmSync(root, { recursive: true });
});

test("the two editions' keys cannot collide, and the miss names what differs", () => {
  const w = keyOf(), e = keyOf({ edition: "extension", env: ENV_EXTENSION });
  assert.notEqual(w.digest, e.digest);
  assert.deepEqual(diffKey(w.key, e.key).sort(), [
    "edition", "tool.cliSha256", "tool.cliVersion", "tool.extensionVersion", "tool.pack", "tool.pdftotext", "tool.tesseract", "tool.tesseractJs",
  ]);
  assert.equal(keyOf().digest, keyOf().digest, "the digest is stable");
});

test("an altered, shortened or extended case folder is never reused", () => {
  const damage = {
    "file rewritten": (dir) => writeFileSync(join(dir, "text.txt"), "hello, edited"),
    "file removed": (dir) => rmSync(join(dir, "text.txt")),
    "file added": (dir) => writeFileSync(join(dir, "stray.png"), "x"),
    "nested file removed": (dir) => rmSync(join(dir, "html", "index.html")),
  };
  for (const [name, hurt] of Object.entries(damage)) {
    const root = tmp();
    hurt(earlierRun(root));
    const got = reuse(root);
    assert.ok(got.miss && !got.result, `${name} must not be reused`);
    assert.match(got.miss, /file|files/, name);
    rmSync(root, { recursive: true });
  }
});

test("--fresh never looks back and never calls the judge", () => {
  const root = tmp();
  earlierRun(root);
  let asked = 0;
  assert.deepEqual(reuse(root, { fresh: true, verdictOf: () => { asked += 1; return PASSING(); } }), { miss: "--fresh" });
  assert.equal(asked, 0);
  rmSync(root, { recursive: true });
});

test("the newest candidate wins, the case being written is excluded, a chain keeps the first collection time", () => {
  const root = tmp();
  const older = earlierRun(root, { run: "older" });
  const newer = earlierRun(root, { run: "newer", edit: (r) => { r.startedAt = "2026-10-10T10:00:00.000Z"; r.reusedFrom = { collectedAt: "2026-10-01T08:00:00.000Z" }; } });
  utimesSync(join(older, "result.json"), new Date("2026-10-01"), new Date("2026-10-01"));
  utimesSync(join(newer, "result.json"), new Date("2026-10-02"), new Date("2026-10-02"));
  const got = reuse(root);
  assert.ok(got.result, got.miss);
  assert.match(got.result.reusedFrom.caseDir, /newer\//);
  assert.equal(got.result.reusedFrom.collectedAt, "2026-10-01T08:00:00.000Z");
  rmSync(root, { recursive: true });
});

test("a non-empty target is refused and left alone", () => {
  const root = tmp();
  earlierRun(root);
  const to = join(root, "later", "windows-default", CASE.id);
  put(join(to, "keep.txt"), "mine");
  const got = reuse(root);
  assert.match(got.miss, /not empty/);
  assert.equal(readFileSync(join(to, "keep.txt"), "utf8"), "mine");
  rmSync(root, { recursive: true });
});

test("a search root with nothing in it is a plain miss", () => {
  const root = tmp();
  const got = findReusable({ searchRoot: join(root, "nothing"), passName: "windows-default", caseId: CASE.id, source: CASE.sha256, key: keyOf().key, digest: keyOf().digest, exclude: root, verdictOf: PASSING });
  assert.deepEqual(got, { miss: "no earlier result" });
  rmSync(root, { recursive: true });
});

test("the producer digest ignores test files and testdata and sees every product byte", () => {
  const repo = tmp();
  put(join(repo, "internal", "ocr", "ocr.go"), "package ocr");
  put(join(repo, "internal", "ocr", "ocr_test.go"), "package ocr");
  put(join(repo, "internal", "ocr", "testdata", "fixture.bin"), "one");
  put(join(repo, "internal", "ocr", "embedded.txt"), "asset");
  put(join(repo, "extension", "src", "ocr.js"), "export {}");
  put(join(repo, "extension", "src", "ocr.test.mjs"), "test");
  put(join(repo, "tools", "doccorpus", "run.mjs"), "run");
  put(join(repo, "tools", "doccorpus", "judge_test.go"), "package main");
  const entries = ["internal", "extension/src", "tools/doccorpus"];
  const digest = () => producerDigest(repo, entries);
  const base = digest();

  put(join(repo, "internal", "ocr", "ocr_test.go"), "package ocr // edited");
  put(join(repo, "internal", "ocr", "testdata", "fixture.bin"), "two");
  put(join(repo, "extension", "src", "ocr.test.mjs"), "edited");
  put(join(repo, "tools", "doccorpus", "judge_test.go"), "package main // edited");
  assert.equal(digest(), base, "editing tests must not discard a collection");

  for (const rel of [["internal", "ocr", "ocr.go"], ["internal", "ocr", "embedded.txt"], ["extension", "src", "ocr.js"], ["tools", "doccorpus", "run.mjs"]]) {
    const p = join(repo, ...rel);
    const before = readFileSync(p, "utf8");
    writeFileSync(p, `${before} `);
    assert.notEqual(digest(), base, `${rel.join("/")} is a product byte`);
    writeFileSync(p, before);
  }
  put(join(repo, "internal", "ocr", "new.go"), "package ocr");
  assert.notEqual(digest(), base, "a new product file changes the digest");
  assert.throws(() => producerDigest(repo, ["internal", "absent"]), "a missing entry is an error, not a smaller digest");
  assert.ok(isTestFile("x_test.go") && isTestFile("a.test.mjs") && !isTestFile("run.mjs"));
  rmSync(repo, { recursive: true });
});

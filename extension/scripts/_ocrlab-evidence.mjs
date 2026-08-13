// _ocrlab-evidence.mjs - the browser edition's half of the cross-edition evidence contract.
//
// The desktop lab (tools/ocrlab/evidence) and this file describe a run in the same terms so one
// metrics package can grade both editions against one set of annotations. The Go encoder emits
// its struct fields in declaration order, so the factories here build their objects in that same
// order: two evidence.json files for the same scene then diff line by line instead of looking
// like different formats. TestParityOCRLabEvidenceSchema reads the names out of both sides and
// fails when they drift.
//
// Everything geometric is in natural image pixels, exactly as on the desktop side.

export const SCHEMA_VERSION = 1;

export const EDITION_DESKTOP = "desktop";
export const EDITION_EXTENSION = "extension";

// Concealment modes, mirroring the Go Mode constants. The extension records MODE_FILL today
// because that is what it draws; the field exists from the first run so a later mode change is a
// diff in the evidence rather than an unexplained movement in the scores.
export const MODE_FILL = "fill";
export const MODE_RECONSTRUCT = "reconstruct";
export const MODE_MASK = "mask";

const MODES = [MODE_FILL, MODE_RECONSTRUCT, MODE_MASK];
const EDITIONS = [EDITION_DESKTOP, EDITION_EXTENSION];

const num = (v) => (Number.isFinite(Number(v)) ? Number(v) : 0);
const int = (v) => Math.round(num(v));
const str = (v) => (v === undefined || v === null ? "" : String(v));

export function makeRect({ x0 = 0, y0 = 0, x1 = 0, y1 = 0 } = {}) {
  return { x0: int(x0), y0: int(y0), x1: int(x1), y1: int(y1) };
}

export function makeViewport({ name = "", width = 0, height = 0, deviceScaleFactor = 1 } = {}) {
  return { name: str(name), width: int(width), height: int(height), deviceScaleFactor: num(deviceScaleFactor) };
}

// makePlate is one rendered text plate as the browser actually laid it out. scrollHeight and
// clientHeight are read back from the DOM rather than computed: whether a translation clipped is
// a fact about the rendered box, and the runtime re-fit means an estimate would be wrong.
export function makePlate(p = {}) {
  return {
    text: str(p.text),
    rect: makeRect(p.rect),
    viewport: str(p.viewport),
    stressCase: str(p.stressCase),
    fontPx: num(p.fontPx),
    background: str(p.background),
    ink: str(p.ink),
    mode: str(p.mode || MODE_FILL),
    modeConfidence: num(p.modeConfidence),
    scrollHeight: int(p.scrollHeight),
    clientHeight: int(p.clientHeight),
  };
}

// CLIP_SLACK_PX is how much overflow is layout rounding rather than hidden text.
//
// The shipped re-fit uses one pixel, and the lab needs more: scrollHeight and clientHeight are
// each rounded to an integer from a fractional layout, so a box that fits exactly can still
// report a two or three pixel difference. Measured over the 45-scene corpus on 2026-08-12, every
// plate the old one-pixel rule called clipped overshot by exactly 2 or 3 px and none by more -
// the re-fit's height:auto escape had fired on all of them and nothing was hidden. Four pixels
// clears that residue and cannot hide a real clip, because hiding text costs at least one line
// and a line is at least a font size tall.
export const CLIP_SLACK_PX = 4;

// clipped reports whether the browser said this plate's content overflows its box.
export const clipped = (plate) => plate.scrollHeight > plate.clientHeight + CLIP_SLACK_PX;

export function makeScreenshots({ source = "", rendered = "", stress = {} } = {}) {
  const out = { source: str(source), rendered: str(rendered) };
  if (stress && Object.keys(stress).length) out.stress = { ...stress };
  return out;
}

// makeScene is one corpus scene's outcome. error is written when the runner could not process the
// scene: a run that silently dropped it would report better numbers over fewer scenes, which is
// the one failure mode the strategic spec names outright.
export function makeScene(s = {}) {
  const out = {
    sceneId: str(s.sceneId),
    imageWidth: int(s.imageWidth),
    imageHeight: int(s.imageHeight),
    plates: (s.plates || []).map(makePlate),
    screenshots: makeScreenshots(s.screenshots),
    ocrMs: int(s.ocrMs),
    renderMs: int(s.renderMs),
    peakRssBytes: int(s.peakRssBytes),
  };
  if (s.error) out.error = str(s.error);
  return out;
}

export function makeEngine({ tesseract = "", tessdataVersion = "", lang = "" } = {}) {
  return { tesseract: str(tesseract), tessdataVersion: str(tessdataVersion), lang: str(lang) };
}

export function makeBrowser({ name = "", version = "" } = {}) {
  return { name: str(name), version: str(version) };
}

export function makeRun(r = {}) {
  return {
    schemaVersion: SCHEMA_VERSION,
    runId: str(r.runId),
    startedAt: str(r.startedAt),
    edition: str(r.edition || EDITION_EXTENSION),
    engine: makeEngine(r.engine),
    browser: makeBrowser(r.browser),
    viewports: (r.viewports || []).map(makeViewport),
    scenes: (r.scenes || []).map(makeScene),
  };
}

// validateRun returns a list of field-level problems, empty when the run is well formed. It is
// the check both sides run over an evidence file: the Node test feeds it a run the Go runner
// wrote, so a schema change that only landed on one side is caught before a scorer sees it.
export function validateRun(run) {
  const problems = [];
  const bad = (where, what) => problems.push(`${where}: ${what}`);

  if (!run || typeof run !== "object") return ["run: not an object"];
  if (run.schemaVersion !== SCHEMA_VERSION) {
    bad("schemaVersion", `is ${JSON.stringify(run.schemaVersion)}, this build understands ${SCHEMA_VERSION}`);
  }
  if (!run.runId) bad("runId", "empty");
  if (!run.startedAt) bad("startedAt", "empty");
  if (!EDITIONS.includes(run.edition)) bad("edition", `is ${JSON.stringify(run.edition)}, expected one of ${EDITIONS.join(", ")}`);
  if (!run.engine || typeof run.engine !== "object") bad("engine", "missing");
  else if (!run.engine.lang) bad("engine.lang", "empty - a score cannot be attributed to a language that is not recorded");
  if (!run.browser || typeof run.browser !== "object") bad("browser", "missing");

  const viewports = Array.isArray(run.viewports) ? run.viewports : [];
  if (!viewports.length) bad("viewports", "empty");
  const known = new Set();
  viewports.forEach((v, i) => {
    if (!v || !v.name) bad(`viewports[${i}].name`, "empty");
    else known.add(v.name);
    if (!(v && v.width > 0 && v.height > 0)) bad(`viewports[${i}]`, "width and height must be positive");
    if (!(v && v.deviceScaleFactor > 0)) bad(`viewports[${i}].deviceScaleFactor`, "must be positive");
  });

  const scenes = Array.isArray(run.scenes) ? run.scenes : [];
  if (!scenes.length) bad("scenes", "empty");
  const seen = new Set();
  scenes.forEach((s, i) => {
    const where = `scenes[${i}]`;
    if (!s || typeof s !== "object") { bad(where, "not an object"); return; }
    if (!s.sceneId) bad(`${where}.sceneId`, "empty");
    else if (seen.has(s.sceneId)) bad(`${where}.sceneId`, `duplicate ${s.sceneId}`);
    else seen.add(s.sceneId);
    if (s.error) return; // a failed scene carries its reason and nothing else worth checking
    if (!(s.imageWidth > 0 && s.imageHeight > 0)) bad(`${where}`, "imageWidth and imageHeight must be positive");
    if (!Array.isArray(s.plates)) { bad(`${where}.plates`, "not an array"); return; }
    s.plates.forEach((p, j) => validatePlate(p, `${where}.plates[${j}]`, known, bad));
  });
  return problems;
}

function validatePlate(p, where, viewportNames, bad) {
  if (!p || typeof p !== "object") { bad(where, "not an object"); return; }
  for (const key of Object.keys(makePlate())) {
    if (!(key in p)) bad(`${where}.${key}`, "missing");
  }
  if (p.viewport && viewportNames.size && !viewportNames.has(p.viewport)) {
    bad(`${where}.viewport`, `${p.viewport} is not one of the run's declared viewports`);
  }
  if (!p.stressCase) bad(`${where}.stressCase`, "empty");
  if (p.mode && !MODES.includes(p.mode)) bad(`${where}.mode`, `${p.mode} is not one of ${MODES.join(", ")}`);
  const r = p.rect;
  if (!r || typeof r !== "object") bad(`${where}.rect`, "missing");
  else if (r.x1 < r.x0 || r.y1 < r.y0) bad(`${where}.rect`, "x1/y1 must not be smaller than x0/y0");
}

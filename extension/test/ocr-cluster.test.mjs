// Tests for the line-to-plate grouping (ocr-cluster.js). Mirrors internal/ocr/cluster_test.go -
// the two implementations are hand-ported, so they are given the same input and must reach the
// same plates (see docs/PARITY.md).

import { test } from "node:test";
import assert from "node:assert/strict";
import {
  clusterLines, droppedLines, keepLine, medianLinePitch, releaseOversized, resultStrength, sameTypeSize,
  strictlyBetter, trimOutlierWords,
  OCR_MAX_PLATE_COVERAGE,
} from "../src/ocr-cluster.js";

// The line boxes below are not invented: they are what tesseract returned for the lab's two
// grouping scenes on 2026-08-11, in the upscaled space the desktop app clusters in. The extension
// divides by the upscale factor before clustering instead, so it sees these halved - both spaces
// are asserted below, because the decision must not depend on which one it is made in.
const balloonOnPanel = [
  { bbox: { x0: 644, y0: 184, x1: 930, y1: 218 }, conf: 96.0, text: "WELL, THAT IS" },
  { bbox: { x0: 645, y0: 256, x1: 890, y1: 285 }, conf: 96.7, text: "ONE WAY TO" },
  { bbox: { x0: 646, y0: 328, x1: 838, y1: 357 }, conf: 93.6, text: "SOLVE IT!" },
  { bbox: { x0: 156, y0: 126, x1: 1034, y1: 826 }, conf: 0.0, text: "" }, // the panel: no text
];

// What this edition's own engine returns for the same scene, which is not what the desktop engine
// returns: tesseract.js reads the balloon's left outline as "|" and folds it into the line, so the
// line box grows to 37 px beside a 13 px neighbour. Every fixture above came from the desktop
// engine, which is why the split below was invisible to this suite until the lab measured the
// rendered page on 2026-08-15 (DEV/research/ocrlab/2026-08-15__extension-parity-run.md): one
// balloon became two plates and the taller plate reached onto the protected outline.
//
// wordH is what the fix reads - the words' own heights, whose median is the size a reader sees.
const outlineArtefact = [
  { bbox: { x0: 116, y0: 123, x1: 390, y1: 151 }, conf: 96.4, text: "ARE YOU SURE", wordH: [28, 28, 28] },
  { bbox: { x0: 116, y0: 175, x1: 362, y1: 203 }, conf: 95.7, text: "ABOUT THIS?", wordH: [28, 28] },
  { bbox: { x0: 78, y0: 259, x1: 298, y1: 333 }, conf: 90.1, text: "| NOT EVEN", wordH: [74, 26, 26] },
  { bbox: { x0: 117, y0: 311, x1: 308, y1: 337 }, conf: 95.9, text: "SLIGHTLY.", wordH: [26] },
];

const adjacentBalloons = [
  { bbox: { x0: 116, y0: 123, x1: 390, y1: 151 }, conf: 96.4, text: "ARE YOU SURE" },
  { bbox: { x0: 116, y0: 175, x1: 362, y1: 203 }, conf: 95.7, text: "ABOUT THIS?" },
  { bbox: { x0: 118, y0: 259, x1: 298, y1: 287 }, conf: 96.4, text: "NOT EVEN" },
  { bbox: { x0: 117, y0: 311, x1: 308, y1: 339 }, conf: 95.9, text: "SLIGHTLY." },
];

const halved = (lines) => lines.map((l) => ({
  ...l,
  bbox: {
    x0: Math.round(l.bbox.x0 / 2), y0: Math.round(l.bbox.y0 / 2),
    x1: Math.round(l.bbox.x1 / 2), y1: Math.round(l.bbox.y1 / 2),
  },
}));

// The splitting half of the pair: all-caps lettering boxes far shorter than the line it came from
// (29-34 px ink for 72 px leading here), so grouping it by ink height tore one balloon into three
// plates - three sentence fragments for a translator that has no sentence to work with.
test("one balloon is one plate", () => {
  for (const [lines, w, h] of [[balloonOnPanel, 1120, 840], [halved(balloonOnPanel), 560, 420]]) {
    const blocks = clusterLines(lines, 80, w, h);
    assert.equal(blocks.length, 1);
    assert.equal(blocks[0].text, "WELL, THAT IS ONE WAY TO SOLVE IT!");
  }
});

// A tall artefact inside a line must not end its balloon. The type-size test reads the median of
// the words' heights, so "|" at 74 px sits beside two 26 px words and the line still measures 26.
test("an outline artefact does not split a balloon", () => {
  const blocks = clusterLines(outlineArtefact, 80, 1240, 600);
  assert.equal(blocks.length, 2);
  assert.equal(blocks[1].text, "| NOT EVEN SLIGHTLY.");
  // And the plate stays off the outline: its box is the union of the two lines, no wider than the
  // artefact line itself was.
  assert.equal(blocks[1].bbox.x0, 78);
});

// The same lines with no word heights - an engine that reports none - must keep the old behaviour
// rather than silently becoming looser: the line box is then all there is to measure.
test("without word heights the type-size test falls back to the line box", () => {
  const stripped = outlineArtefact.map(({ wordH, ...rest }) => rest);
  const blocks = clusterLines(stripped, 80, 1240, 600);
  assert.equal(blocks.length, 3);
});

// Grouping the balloon back together is not enough: the artefact still pulls the line box - and so
// the plate - out over the protected outline. Measured on the same scene, the damage went 148 px
// before the type-size fix to 160 px after it, because the plate then spanned both lines.
test("a tall non-text token does not stretch the line box", () => {
  const words = [
    { text: "|", bbox: { x0: 78, y0: 259, x1: 88, y1: 333 } },
    { text: "NOT", bbox: { x0: 116, y0: 265, x1: 210, y1: 291 } },
    { text: "EVEN", bbox: { x0: 220, y0: 265, x1: 298, y1: 291 } },
  ];
  const box = { x0: 78, y0: 259, x1: 298, y1: 333 };
  assert.deepEqual(trimOutlierWords(box, words, 1), { x0: 116, y0: 265, x1: 298, y1: 291 });

  // Ordinary punctuation is not an artefact: it is short, so it stays and the box is untouched.
  const withComma = [
    { text: "NOT", bbox: { x0: 116, y0: 265, x1: 210, y1: 291 } },
    { text: ",", bbox: { x0: 212, y0: 281, x1: 220, y1: 293 } },
  ];
  const commaBox = { x0: 116, y0: 265, x1: 220, y1: 293 };
  assert.deepEqual(trimOutlierWords(commaBox, withComma, 1), commaBox);

  // A line that is nothing but the artefact keeps its box - there is no lettering to shrink to.
  const onlyRule = [{ text: "|", bbox: { x0: 78, y0: 259, x1: 88, y1: 333 } }];
  const ruleBox = { x0: 78, y0: 259, x1: 88, y1: 333 };
  assert.deepEqual(trimOutlierWords(ruleBox, onlyRule, 1), ruleBox);
});

// The merging half: the fix above must not be paid for with a plate that spans two balloons. The
// pitch inside a balloon is 52 px here and the step across to the next one is 84 px.
test("adjacent balloons stay two plates", () => {
  for (const [lines, w, h] of [[adjacentBalloons, 1240, 600], [halved(adjacentBalloons), 620, 300]]) {
    const blocks = clusterLines(lines, 80, w, h);
    assert.equal(blocks.length, 2);
    assert.equal(blocks[0].text, "ARE YOU SURE ABOUT THIS?");
    assert.equal(blocks[1].text, "NOT EVEN SLIGHTLY.");
  }
});

// poster-display-type-on-flat-colour as the desktop app reads it: the grey rendition at PSM 11 with
// rus data, rows in the order tesseract returned them (PSM 11 is "sparse text, in no particular
// order", and one line did come back out of reading order). ЗАЧЕМ and ОБ ЗЛОМ fall under the rescue
// floor. The step from the headline to the next kept line is 335 px and the step between two body
// lines a reader takes as one sentence is 381 px, so no pitch bound cuts in the right place - the
// type sizes are what separate them, 281 px of ink against a 155 px median.
const displayHeadlineOverBody = [
  { bbox: { x0: 76, y0: 79, x1: 780, y1: 366 }, conf: 69.2, text: "ЗАЧЕМ" },
  { bbox: { x0: 74, y0: 415, x1: 1328, y1: 696 }, conf: 80.7, text: "ТРАХАТЬСЯ:" },
  { bbox: { x0: 79, y0: 750, x1: 521, y1: 905 }, conf: 96.1, text: "МЫ ЖЕ" },
  { bbox: { x0: 76, y0: 1131, x1: 456, y1: 1302 }, conf: 92.6, text: "ЛЮДИ," },
  { bbox: { x0: 79, y0: 1319, x1: 538, y1: 1472 }, conf: 95.9, text: "МОЖЕМ" },
  { bbox: { x0: 79, y0: 1694, x1: 498, y1: 1836 }, conf: 73.9, text: "ОБ ЗЛОМ" },
  { bbox: { x0: 80, y0: 1509, x1: 477, y1: 1658 }, conf: 87.2, text: "ПРОСТО" },
  { bbox: { x0: 80, y0: 1872, x1: 644, y1: 2009 }, conf: 95.0, text: "ПОГОВОРИТЬ" },
];

// samson-and-delilah-03-scroll's 19 hand-drawn line boxes - the corpus's widest within-group spread
// (ink 23-34 px, worst line 1.42x the group's own median). One caption, one sentence.
const scrollCaption = [
  [103, 40, 469, 74, "LONG AGO IN ISRAEL"], [130, 74, 441, 102, "THE SMALL TOWN OF"],
  [130, 109, 469, 134, "ASHKELON WAS RULED"], [130, 142, 469, 165, "BY A PHILISTINE KING,"],
  [132, 172, 469, 196, "WHO WAS CRUEL AND"], [135, 203, 469, 227, "HEARTLESS. ALL THE"],
  [132, 235, 469, 258, "DANITES HATED HIM."], [130, 265, 469, 289, "HIS TAXES WERE HEAVY,"],
  [130, 297, 469, 320, "HIS PUNISHMENTS SEVERE!"], [130, 328, 469, 353, "SAMSON, JUDGE OF THE"],
  [130, 362, 463, 385, "DANITES, WAS CHOSEN"], [130, 392, 469, 415, "TO PROTECT HIS"],
  [130, 423, 469, 446, "DOWNTRODDEN PEOPLE."], [130, 453, 469, 477, "IN ISRAEL WAS DELILAH,"],
  [130, 484, 447, 510, "IN ISRAEL WAS THEIR"], [130, 516, 381, 540, "GREAT LOVE, IN ISRAEL"],
  [134, 547, 381, 570, "WAS THE DEATH OF----"], [130, 574, 457, 607, "SAMSON AND"],
  [164, 613, 402, 645, "DELILAH!"],
].map(([x0, y0, x1, y1, text]) => ({ bbox: { x0, y0, x1, y1 }, conf: 95, text }));

// A headline and the body under it are two texts even when they sit within the page's own line pitch
// of each other. Pitch cannot see it here - the headline's step is smaller than one inside the body -
// so the type size has to. Mirrors internal/ocr TestClusterLinesSeparatesDisplayTypeFromBody.
// A headline and the body under it are two texts even when they sit within the page's own line pitch
// of each other. Pitch cannot see it here - the headline's step is smaller than one inside the body -
// so the type size has to. ЗАЧЕМ stays under the rescue floor: the 2026-08-15 re-measurement found
// no floor and no length rule that recovers it without plating debris on a Cyrillic poster read with
// English data (DEV/research/ocr_rescue_floor_2026-08-15.md).
test("display type and the body under it are two plates", () => {
  const blocks = clusterLines(displayHeadlineOverBody, 80, 1920, 2560);
  assert.equal(blocks.length, 2);
  assert.equal(blocks[0].text, "ТРАХАТЬСЯ:");
  assert.equal(blocks[1].text, "МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ");
  assert.deepEqual(blocks[1].bbox, { x0: 76, y0: 750, x1: 644, y1: 2009 });
  // The extension clusters in the un-upscaled space, so the same decision has to survive halving.
  const small = clusterLines(halved(displayHeadlineOverBody), 80, 960, 1280);
  assert.equal(small.length, 2);
  assert.equal(small[1].text, "МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ");
});

test("a section break is not a line pitch", () => {
  const headingAndDistantBody = [
    { bbox: { x0: 20, y0: 10, x1: 120, y1: 30 }, conf: 90, text: "Heading" },
    { bbox: { x0: 10, y0: 200, x1: 190, y1: 220 }, conf: 88, text: "This body" },
  ];
  assert.equal(medianLinePitch(headingAndDistantBody, 20), 0);
  assert.equal(clusterLines(headingAndDistantBody).length, 2);

  const body = [
    { bbox: { x0: 10, y0: 10, x1: 190, y1: 30 }, conf: 95, text: "Alpha beta" },
    { bbox: { x0: 10, y0: 34, x1: 190, y1: 54 }, conf: 95, text: "gamma delta" },
    { bbox: { x0: 10, y0: 58, x1: 160, y1: 78 }, conf: 95, text: "epsilon" },
  ];
  assert.equal(medianLinePitch(body, 20), 24);
  assert.equal(clusterLines(body).length, 1);
});

test("lines that share no column measure no pitch off each other", () => {
  const columns = [
    { bbox: { x0: 10, y0: 10, x1: 100, y1: 30 }, conf: 95, text: "left one" },
    { bbox: { x0: 300, y0: 20, x1: 390, y1: 40 }, conf: 95, text: "right one" },
  ];
  assert.equal(medianLinePitch(columns, 20), 0);
});

test("a caption whose lines vary in height is not torn by the size rule", () => {
  const blocks = clusterLines(scrollCaption, 80, 520, 720);
  assert.ok(blocks.length >= 1);
  assert.equal(blocks[0].bbox.y0, 40);
  assert.equal(blocks[0].bbox.y1, 607);
  assert.ok(blocks[0].text.startsWith("LONG AGO IN ISRAEL"));
  assert.ok(blocks[0].text.endsWith("SAMSON AND"));
});

// The six rows of test_doc/accounts.jpg as the desktop app reads them (640x563). One type size, one
// column, an even pitch - nothing in the typography separates a row from the next - so before the
// coverage rule the six list rows landed in one plate over 0.6829 of the picture whose own lines
// filled 0.6608 of its height. The boxes are the recognizer's, unaltered; the row texts are not -
// the screenshot is a private account list and its rows name real people, so each keeps its shape
// and loses its content. Mirrors internal/ocr TestClusterLinesReleasesAPlateThatCoversItsPage.
const accountsWindow = [
  [15, 17, 290, 38, "Your family group members"],
  [15, 51, 339, 66, "View and manage your family group. Learn more ©"],
  [13, 99, 527, 151, "Se) Given Family Family manager"],
  [22, 182, 467, 231, "i) Second Family Parent"],
  [15, 262, 479, 310, "6 Third Family Member"],
  [13, 341, 479, 393, "wi Fourth Family Member"],
  [13, 423, 555, 474, "te Fifth Family Supervised member"],
  [15, 505, 479, 553, "© Sixth Family Member"],
].map(([x0, y0, x1, y1, text]) => ({ bbox: { x0, y0, x1, y1 }, conf: 90, text }));

test("a plate that covers its page is released into its lines", () => {
  const blocks = clusterLines(accountsWindow, 50, 640, 563);
  assert.ok(blocks.length >= 6, `blocks = ${blocks.length}, want the six rows released`);
  for (const b of blocks) {
    const cover = (b.bbox.x1 - b.bbox.x0) * (b.bbox.y1 - b.bbox.y0) / (640 * 563);
    assert.ok(cover <= OCR_MAX_PLATE_COVERAGE, `plate "${b.text}" covers ${cover.toFixed(4)}`);
  }
  const joined = blocks.map((b) => b.text).join(" ");
  for (const want of ["Given Family", "Second Family", "Sixth Family", "Supervised member"]) {
    assert.ok(joined.includes(want), `released plates lost "${want}"`);
  }
});

// The price side: each half of the rule alone would release a scene the corpus says is one plate.
// Mirrors internal/ocr TestReleaseOversizedNeedsBothConditions.
test("releasing a plate needs both conditions", () => {
  const cur = (x1, y1, lines) => ({
    x0: 0, y0: 0, x1, y1, lines, texts: lines.map((_, i) => String.fromCharCode(97 + i)),
  });
  const tight = [{ x0: 0, y0: 0, x1: 100, y1: 79 }, { x0: 0, y0: 80, x1: 100, y1: 158 }];
  assert.equal(releaseOversized(cur(100, 200, tight), 120, 170), null, "a tightly packed caption was released");
  const loose = [
    { x0: 0, y0: 0, x1: 100, y1: 20 }, { x0: 0, y0: 40, x1: 100, y1: 60 }, { x0: 0, y0: 80, x1: 100, y1: 100 },
  ];
  assert.equal(releaseOversized(cur(100, 100, loose), 500, 500), null, "an ordinary paragraph was released");
  assert.equal(releaseOversized(cur(100, 100, loose), 120, 120).length, 3);
});

// The ratio has to admit the widest spread one text shows on its own and reject the narrowest step
// between two texts. Both numbers come from the corpus's hand-drawn line boxes.
test("the type-size ratio brackets the measured bands", () => {
  assert.equal(sameTypeSize(34, 24), true, "the widest within-caption spread (1.42x) is one text");
  assert.equal(sameTypeSize(281, 155), false, "display type over body text (1.81x) is two texts");
  assert.equal(sameTypeSize(0, 155), true, "an unmeasured height does not end a plate");
  assert.equal(sameTypeSize(155, 0), true, "an unmeasured cluster height does not end a plate");
});



// The rescue ladder's comparator. Mirrors internal/ocr/strength_test.go case for case: which rung a
// reader ends up seeing is decided here, and the two editions must decide the same way.
test("resultStrength counts the words a reader would see", () => {
  assert.equal(resultStrength([]), 0);
  assert.equal(resultStrength([{ text: "| МОЖЕМ" }]), 2, "one stray word on a large poster");
  assert.equal(resultStrength([{ text: "THE END" }]), 2, "a sparse caption is not weak for being short");
  assert.equal(
    resultStrength([{ text: "ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ," }, { text: "МОЖЕМ ПРОСТО ПОГОВОРИТЬ" }]),
    7,
    "several plates count together",
  );
});

test("strictlyBetter keeps the incumbent on a tie", () => {
  const poor = [{ text: "| МОЖЕМ" }];
  const rich = [{ text: "ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ" }];
  assert.equal(strictlyBetter(rich, poor), true, "a richer rung replaces a poorer one");
  assert.equal(strictlyBetter(poor, rich), false, "a poorer rung does not replace a richer one");
  assert.equal(strictlyBetter([{ text: "ЕЩЁ ДВА" }], poor), false, "an equal count keeps the incumbent");
  assert.equal(strictlyBetter(poor, []), true, "anything replaces nothing");
  assert.equal(strictlyBetter([], []), false, "nothing replaces nothing");
});

// The floor's discard record. Mirrors internal/ocr/cluster_test.go TestDroppedLinesRecordTheFloor:
// the two editions must agree on what "the reader lost this" means, because any re-derivation of
// the floor is measured from these records on one edition and applied to both.
test("droppedLines records the text the floor rejected, and only that", () => {
  // The measured staging of poster-display-type-on-flat-colour, rus data, the sparse rung.
  const lines = [
    { bbox: { x0: 38, y0: 40, x1: 390, y1: 183 }, conf: 69.2, text: "ЗАЧЕМ" },
    { bbox: { x0: 37, y0: 207, x1: 664, y1: 348 }, conf: 80.7, text: "ТРАХАТЬСЯ:" },
    { bbox: { x0: 39, y0: 375, x1: 261, y1: 453 }, conf: 73.9, text: "ОБ ЗЛОМ" },
    { bbox: { x0: 0, y0: 0, x1: 10, y1: 10 }, conf: 12.0, text: "" }, // no text: not a loss
  ];
  const dropped = droppedLines(lines, 80);
  assert.deepEqual(dropped.map((d) => d.text), ["ЗАЧЕМ", "ОБ ЗЛОМ"]);
  assert.deepEqual(dropped.map((d) => d.conf), [69.2, 73.9]);
  assert.deepEqual(dropped[0].bbox, { x0: 38, y0: 40, x1: 390, y1: 183 });
  // The record names the gate it failed, so drops from the ordinary pass and from the rescue
  // ladder stay separable in one file.
  assert.deepEqual(dropped.map((d) => d.floor), [80, 80]);
  assert.deepEqual(droppedLines(lines, 50).map((d) => d.floor), []);

  // The predicate is the same one clusterLines applies, so the two can never describe different
  // decisions: every line is either clustered or recorded as dropped, never both and never neither.
  const kept = lines.filter((l) => keepLine(l, 80)).length;
  const empty = lines.filter((l) => !l.text).length;
  assert.equal(kept, 1);
  assert.equal(dropped.length + kept + empty, lines.length, "every line is kept, dropped or textless");

  // A floor of 0 loses nothing, and the empty line is still not a loss.
  assert.equal(droppedLines(lines, 0).length, 0);
});

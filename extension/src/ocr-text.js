// ocr-text.js - decide whether a recognized OCR block is worth turning into a translatable
// plate. Rejects OCR noise that has nothing to translate: fewer than 5 letters (also kills
// pure numbers / symbols), a run of letters with no vowels (consonant soup), text that is
// wholly an address (URL / email / domain / path), and low-quality "mishmash" where few
// whitespace tokens look like real words. Short CJK phrases are kept. Mirrors the desktop
// app's internal/ocr/text.go isTranslatable - keep the two in sync (see docs/PARITY.md).

// CJK by Unicode script, as text.go isCJK reads it with Go's script tables - so halfwidth
// katakana, compatibility jamo and the astral Han planes count, and the Common-script marks that
// sit in the kana blocks (the prolonged sound mark, the middle dot) do not.
const CJK = /[\p{Script=Han}\p{Script=Hiragana}\p{Script=Katakana}\p{Script=Hangul}]/u;
const VOWEL = /[aeiouyàáâãäåæèéêëìíîïòóôõöøùúûüýÿаеёиоуыэюяєії]/i;
const ADDRESS = /^(?:https?:\/\/|www\.)\S+$|^\S+@\S+\.\S+$|^[\w-]+(?:\.[\w-]+)+(?:[/?#]\S*)?$|^[a-z]:\\|^\/[\w./-]+$/i;
const LETTER = /\p{L}/u;

// OCR-OVERLAY rule 13: policy - five letters, a vowel, half the lettered tokens word-like, a CJK
// bypass (OCR-PIPELINE 2.6).
export function isTranslatable(raw) {
  const t = (raw || "").replace(/\s+/g, " ").trim();
  if (!t) return false;

  let cjk = 0, letters = 0;
  for (const c of t) {
    if (CJK.test(c)) cjk++;
    else if (LETTER.test(c)) letters++;
  }
  if (cjk >= 2) return true;      // short CJK phrases are translatable
  if (letters < 5) return false;  // too few letters (also numbers / symbols)
  if (!VOWEL.test(t)) return false;   // consonant soup, no vowels
  if (ADDRESS.test(t)) return false;  // wholly an address

  // Mishmash: among tokens that carry letters, how many look like real words?
  let lettered = 0, wordlike = 0;
  for (const w of t.split(" ")) {
    let l = 0;
    for (const c of w) if (LETTER.test(c) && !CJK.test(c)) l++;
    if (l === 0) continue;
    lettered++;
    if (l >= 2 && VOWEL.test(w)) wordlike++;
  }
  if (lettered >= 3 && wordlike / lettered < 0.5) return false;
  return true;
}

// OCR_HANGUL_JOIN_GAP_RATIO is how close two Hangul words must stand, as a fraction of the line's
// median word height, to be one word the recognizer cut in two rather than two words the writer
// spaced. Japanese and Chinese are written without spaces, so tokens that meet on Han or kana are
// joined outright; Korean spaces its words, never its syllables, and the recognizer cuts both
// ("코 모 나 시 장 으로서," for "코모나 시장으로서,"), so a Hangul pair is judged on the gap between
// its boxes. Measured on the 12 Korean Pepper&Carrot pages of the ticket 68 corpus against the
// author's lettering: a plateau of 46-48/235 exact lettering lines from 0.30 to 0.36, against 19 with
// every space kept and 13 with none; 0.33 is the plateau's geometric middle
// (DEV/research/RESEARCH_cjk-word-join_2026-09-29.md). Shared invariant - see docs/PARITY.md and
// text.go ocrHangulJoinGapRatio.
// OCR-OVERLAY rule 13: derived - RESEARCH_cjk-word-join_2026-09-29 (OCR-PIPELINE amendment 1.8 A).
export const OCR_HANGUL_JOIN_GAP_RATIO = 0.33;

const HANGUL = /\p{Script=Hangul}/u;

// The punctuation CJK text is set with, which Unicode files under the Common script and CJK above
// therefore does not see: the CJK Symbols and Punctuation block (、。「」), the katakana middle dot and
// prolonged sound mark (・ー), and the halfwidth and fullwidth forms (，！？). Joining on these lifts
// the corpus' exactly matched Japanese lines from 46 to 52 of 270 and Chinese from 21 to 25 of 155;
// ASCII punctuation keeps its space (joining it too drops Japanese back to 47). Mirrors text.go
// isCJKPunct.
const CJK_PUNCT = /[\u3000-\u303F\u30FB\u30FC\uFF00-\uFFEF]/u;

// cjkMeet reports whether two consecutive tokens meet on CJK text - the last character of prev and
// the first of next each a CJK letter or CJK punctuation - and whether either of those two is Hangul.
// Mirrors text.go cjkMeet.
function cjkMeet(prev, next) {
  const a = Array.from(String(prev || "")).pop() || "";
  const b = Array.from(String(next || ""))[0] || "";
  const side = (c) => CJK.test(c) || CJK_PUNCT.test(c);
  if (!side(a) || !side(b)) return { cjk: false, hangul: false };
  return { cjk: true, hangul: HANGUL.test(a) || HANGUL.test(b) };
}

// joinLineWords builds one recognizer line's text from its words (OCR-PIPELINE amendment 1.8 A). Two
// consecutive words are joined with a space, except where they meet on CJK text (cjkMeet): on Han,
// kana and CJK punctuation with no space at all, and where Hangul is involved with no space only when
// the two boxes stand closer than OCR_HANGUL_JOIN_GAP_RATIO times the median height of these words. A
// space between a CJK character and a Latin word, a digit or ASCII punctuation is kept; a word with no
// box, or words with no measurable height, leave the Hangul space in place. Boxes are read through `scale` as
// splitWideGaps reads them. Mirrors text.go joinLineWords - keep the two in sync (docs/PARITY.md).
export function joinLineWords(words, scale = 1) {
  const all = words || [];
  const at = (v) => Math.round(v / scale);
  const hs = all.filter((w) => w && w.bbox).map((w) => at(w.bbox.y1) - at(w.bbox.y0)).filter((h) => h > 0);
  const med = hs.length ? hs.slice().sort((p, q) => p - q)[hs.length >> 1] : 0;
  let out = "";
  for (let i = 0; i < all.length; i++) {
    const w = all[i];
    const text = typeof w.text === "string" ? w.text : "";
    if (i > 0 && !joinsUnspaced(all[i - 1], w, med, at)) out += " ";
    out += text;
  }
  return out;
}

function joinsUnspaced(prev, next, med, at) {
  const { cjk, hangul } = cjkMeet(prev.text, next.text);
  if (!cjk) return false;
  if (!hangul) return true;
  if (!prev.bbox || !next.bbox || med <= 0) return false;
  const gap = Math.max(at(next.bbox.x0) - at(prev.bbox.x1), at(prev.bbox.x0) - at(next.bbox.x1));
  return gap < med * OCR_HANGUL_JOIN_GAP_RATIO;
}

// joinPlateLines builds a plate's text from its lines, in reading order (OCR-PIPELINE amendment 1.8
// B). Lines are joined with a space, except where they meet on Han, kana or CJK punctuation: a
// Japanese or Chinese phrase wraps with no space at the line break. A Hangul line break keeps its
// space - Korean lettering breaks its lines between words. Mirrors text.go joinPlateLines - keep the
// two in sync (docs/PARITY.md).
export function joinPlateLines(lines) {
  let out = "";
  (lines || []).forEach((l, i) => {
    if (i > 0) {
      const { cjk, hangul } = cjkMeet(lines[i - 1], l);
      if (!cjk || hangul) out += " ";
    }
    out += l;
  });
  return out;
}

// repairPipeMisreads rewrites a line's bare "|" tokens into "I", sparing the text-token indexes in
// protectedAt, and returns the text unchanged when nothing was rewritten.
//
// A serif capital I is a bare vertical stroke, and the recognizer reads it as a pipe: on the
// 2026-09-28 field repro (a photographed school-text page) every one of the page's 15 standalone
// "I" tokens came back as "|" on an otherwise well recognized page, and the translation kept each
// bar and lost the subject with it - "I get up at seven o'clock" arrived as the imperative
// "Вставай в семь". This is the token mechanic; the guards that spare a bar - the outline the
// outlier trim handles, a table's column grid - live at the caller, in the cluster flush
// (OCR-PIPELINE amendment 1.5).
// Mirrors the desktop app's internal/ocr/text.go repairPipeMisreads - keep the two in sync
// (docs/PARITY.md).
export function repairPipeMisreads(line, protectedAt = null) {
  const toks = String(line || "").split(" ");
  let changed = false;
  for (let i = 0; i < toks.length; i++) {
    if (toks[i] !== "|" || (protectedAt && protectedAt.has(i))) continue;
    toks[i] = "I";
    changed = true;
  }
  return changed ? toks.join(" ") : String(line || "");
}

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

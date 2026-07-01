// ocr-text.js - decide whether a recognized OCR block is worth turning into a translatable
// plate. Rejects OCR noise that has nothing to translate: fewer than 5 letters (also kills
// pure numbers / symbols), a run of letters with no vowels (consonant soup), text that is
// wholly an address (URL / email / domain / path), and low-quality "mishmash" where few
// whitespace tokens look like real words. Short CJK phrases are kept. Mirrors the desktop
// app's internal/ocr/text.go isTranslatable - keep the two in sync (see docs/PARITY.md).

const CJK = /[぀-ヿ㐀-鿿가-힯豈-﫿]/; // Kana, CJK, Hangul
const VOWEL = /[aeiouyàáâãäåæèéêëìíîïòóôõöøùúûüýÿаеёиоуыэюяєії]/i;
const ADDRESS = /^(?:https?:\/\/|www\.)\S+$|^\S+@\S+\.\S+$|^[\w-]+(?:\.[\w-]+)+(?:[/?#]\S*)?$|^[a-z]:\\|^\/[\w./-]+$/i;
const LETTER = /\p{L}/u;

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

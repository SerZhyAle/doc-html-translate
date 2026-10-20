// ocr-lang-ui.js - the OCR-language picker both the popup and the options page render.
//
// Each surface used to keep its own copy of this renderer and they drifted by a hint line;
// the rendering now lives here once. Installed languages are selectable, the rest download
// on demand, and a failed download is reported inline, next to the button that started it,
// naming the language and offering a retry - it used to vanish into the console while the
// button silently snapped back to "Download".
//
// The Download button reserves the width of its longest state, "Downloading 100%" (BUTTONS 2.0 section 3
// rule 3), so the row does not move as the progress text changes.

import { LANGS, getInstalledLangs, downloadLang } from "./ocr-lang.js";
import { getOptions, setOptions } from "./options-store.js";
import { glyph } from "./glyphs.js";
import { t } from "./i18n.js";

const msg = (key, fallback, ...args) => t(key, fallback, ...args);

// commandButton builds a command with its vocabulary glyph before a caption span; the caption is what
// later states rewrite, so the glyph survives them.
function commandButton(glyphId, caption, reserve) {
  const btn = document.createElement("button");
  btn.type = "button";
  const cap = document.createElement("span");
  cap.className = "cap";
  cap.textContent = caption;
  if (reserve) cap.dataset.reserve = reserve;
  btn.append(glyph(glyphId), cap);
  return btn;
}

// showDownloadError puts the failure between the failed row and the next one. role="alert"
// because it appears on its own - a screen reader must announce it, not discover it on tab.
function showDownloadError(el, row, lang, retry) {
  const err = document.createElement("div");
  err.className = "ocr-error";
  err.setAttribute("role", "alert");
  const text = document.createElement("span");
  text.textContent = msg("ocrDownloadFailed", "Couldn't download {1}", lang.name);
  const btn = commandButton("action.retry", msg("ocrRetry", "Retry"));
  btn.addEventListener("click", retry);
  err.append(text, btn);
  row.after(err);
}

// runDownload fetches one language, showing progress on the row's own button. A failure puts
// the button back the way it was and reports the failure inline, where the retry reruns this.
async function runDownload(el, row, lang, rerender) {
  const btn = row.querySelector("button");
  const cap = btn.querySelector(".cap");
  const prev = el.querySelector(".ocr-error");
  if (prev) prev.remove();
  btn.disabled = true;
  try {
    await downloadLang(lang.code, (m) => {
      if (m && typeof m.progress === "number") {
        cap.textContent = `${msg("ocrDownloading", "Downloading")} ${Math.round(m.progress * 100)}%`;
      }
    });
    await rerender();
  } catch (e) {
    console.error("language download failed", e);
    cap.textContent = msg("ocrDownload", "Download");
    btn.disabled = false;
    showDownloadError(el, row, lang, () => runDownload(el, row, lang, rerender));
  }
}

// renderOcrLangs fills el with the language rows. `hint` is an optional line above them (the
// popup keeps its list compact under one heading; the options page labels the block in its
// own markup). While OCR is off, a call-to-action points at the switch instead of showing a
// greyed-out, dead-looking list (which reads as "unavailable").
export async function renderOcrLangs(el, { hint = "" } = {}) {
  const o = await getOptions();
  el.replaceChildren();
  el.classList.remove("disabled");

  if (!o.ocrImages) {
    const off = document.createElement("div");
    off.className = "hint";
    off.textContent = msg("ocrOffHint", "Turn on to recognize text in images - then pick or download a language (English is built-in).");
    el.append(off);
    return;
  }

  if (hint) {
    const hintEl = document.createElement("div");
    hintEl.className = "hint";
    hintEl.textContent = hint;
    el.append(hintEl);
  }

  const installed = await getInstalledLangs();
  const widest = `${msg("ocrDownloading", "Downloading")} 100%`;
  for (const lang of LANGS) {
    const row = document.createElement("div");
    row.className = "ocr-lang";
    if (installed.includes(lang.code)) {
      // An exclusive choice, not a command: a native radio inside the label that is its target.
      const label = document.createElement("label");
      label.className = "choice";
      const radio = document.createElement("input");
      radio.type = "radio";
      radio.name = "ocrLang";
      radio.id = `ocrlang-${lang.code}`;
      radio.checked = o.ocrLang === lang.code;
      radio.addEventListener("change", async () => {
        const oo = await getOptions();
        oo.ocrLang = lang.code;
        await setOptions(oo);
      });
      const name = document.createElement("span");
      name.textContent = `${lang.name} (${msg("ocrInstalled", "installed")})`;
      label.append(radio, name);
      row.append(label);
    } else {
      const name = document.createElement("span");
      name.className = "name";
      name.textContent = lang.name;
      const btn = commandButton("action.download", msg("ocrDownload", "Download"), widest);
      const rerender = () => renderOcrLangs(el, { hint });
      btn.addEventListener("click", () => runDownload(el, row, lang, rerender));
      row.append(name, btn);
    }
    el.append(row);
  }
}

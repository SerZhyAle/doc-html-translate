// ocr-lang-ui.js - the OCR-language picker both the popup and the options page render.
//
// Each surface used to keep its own copy of this renderer and they drifted by a hint line;
// the rendering now lives here once. Installed languages are selectable, the rest download
// on demand, and a failed download is reported inline, next to the button that started it,
// naming the language and offering a retry - it used to vanish into the console while the
// button silently snapped back to "Download".

import { LANGS, getInstalledLangs, downloadLang } from "./ocr-lang.js";
import { DEFAULT_OPTIONS } from "./defaults.js";
import { t } from "./i18n.js";

const msg = (key, fallback, ...args) => t(key, fallback, ...args);

async function getOptions() {
  const got = await chrome.storage.local.get("options");
  return { ...DEFAULT_OPTIONS, ...(got.options || {}) };
}
async function setOptions(opts) {
  await chrome.storage.local.set({ options: opts });
}

// showDownloadError puts the failure between the failed row and the next one. role="alert"
// because it appears on its own - a screen reader must announce it, not discover it on tab.
function showDownloadError(el, row, lang, retry) {
  const err = document.createElement("div");
  err.className = "ocr-error";
  err.setAttribute("role", "alert");
  const text = document.createElement("span");
  text.textContent = msg("ocrDownloadFailed", "Couldn't download {1}", lang.name);
  const btn = document.createElement("button");
  btn.type = "button";
  btn.textContent = msg("ocrRetry", "Retry");
  btn.addEventListener("click", retry);
  err.append(text, btn);
  row.after(err);
}

// runDownload fetches one language, showing progress on the row's own button. A failure puts
// the button back the way it was and reports the failure inline, where the retry reruns this.
async function runDownload(el, row, lang, rerender) {
  const btn = row.querySelector("button");
  const prev = el.querySelector(".ocr-error");
  if (prev) prev.remove();
  btn.disabled = true;
  try {
    await downloadLang(lang.code, (m) => {
      if (m && typeof m.progress === "number") {
        btn.textContent = `${msg("ocrDownloading", "Downloading")} ${Math.round(m.progress * 100)}%`;
      }
    });
    await rerender();
  } catch (e) {
    console.error("language download failed", e);
    btn.textContent = msg("ocrDownload", "Download");
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
  for (const lang of LANGS) {
    const row = document.createElement("div");
    row.className = "ocr-lang";
    if (installed.includes(lang.code)) {
      const id = `ocrlang-${lang.code}`;
      const radio = document.createElement("input");
      radio.type = "radio";
      radio.name = "ocrLang";
      radio.id = id;
      radio.checked = o.ocrLang === lang.code;
      radio.addEventListener("change", async () => {
        const oo = await getOptions();
        oo.ocrLang = lang.code;
        await setOptions(oo);
      });
      const label = document.createElement("label");
      label.htmlFor = id;
      label.textContent = `${lang.name} (${msg("ocrInstalled", "installed")})`;
      row.append(radio, label);
    } else {
      const name = document.createElement("span");
      name.textContent = lang.name;
      const btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = msg("ocrDownload", "Download");
      const rerender = () => renderOcrLangs(el, { hint });
      btn.addEventListener("click", () => runDownload(el, row, lang, rerender));
      row.append(name, btn);
    }
    el.append(row);
  }
}

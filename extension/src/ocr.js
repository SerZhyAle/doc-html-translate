// ocr.js - controller for the standalone image-OCR page opened from the right-click
// "OCR & translate this image" menu. Reads ?src=, runs the shared overlay unit with the
// user's preferred OCR language, shows progress, and sets <html lang> so the browser
// offers "Translate page".

import { overlayImage, ocrLangToHtmlLang, makeBadge } from "./ocr-overlay.js";
import { langLabel } from "./ocr-lang.js";

const statusEl = document.getElementById("status");
const barEl = document.getElementById("progress-bar");
const textEl = document.getElementById("status-text");
const mount = document.getElementById("ocr-mount");

// Trailing arguments fill {1}, {2}, .. the same way i18n.js's t() does, so a message file
// written for one surface reads correctly on the other.
const msg = (key, fallback, ...args) => {
  let text = fallback;
  try { text = chrome.i18n.getMessage(key) || fallback; } catch { text = fallback; }
  if (args.length === 0) return text;
  return text.replace(/\{(\d)\}/g, (m, i) => (args[i - 1] === undefined ? m : String(args[i - 1])));
};

function parseSrc() {
  const m = /[?&]src=([^&]*)/.exec(location.search);
  if (!m) return "";
  try { return decodeURIComponent(m[1]); } catch { return m[1]; }
}

function isSafe(url) {
  return /^(https?:|file:|blob:|data:)/i.test(url);
}

function setStatus(t) { textEl.textContent = t; }
function setProgress(f) { barEl.style.width = `${Math.round(f * 100)}%`; }
function hideStatus() { statusEl.classList.add("done"); }

async function getOcrLang() {
  try {
    const got = await chrome.storage.local.get("options");
    return (got.options && got.options.ocrLang) || "eng";
  } catch {
    return "eng";
  }
}

async function main() {
  const src = parseSrc();
  if (!src || !isSafe(src)) {
    mount.replaceChildren();
    setStatus(msg("ocrLoadError", "Cannot load this image."));
    return;
  }
  const lang = await getOcrLang();
  setStatus(msg("ocrProgress", "Recognizing text.."));
  try {
    const container = await overlayImage(src, {
      lang,
      onProgress: (m) => {
        if (m && typeof m.progress === "number") setProgress(m.progress);
        if (m && m.status) setStatus(`${msg("ocrProgress", "Recognizing text..")} (${m.status})`);
      },
    });
    mount.replaceChildren(container);
    document.documentElement.lang = ocrLangToHtmlLang(lang);
    setProgress(1);
    if (container.classList.contains("ocr-empty")) {
      // Short on the picture, the reason on the status line: what looks like a verdict on the
      // image is usually a verdict on the recognition language, which defaults to English.
      container.append(makeBadge(msg("ocrNoText", "No text found")));
      setStatus(msg("ocrNoTextLang", "No text found using {1} - if this page is in another language, pick it in the extension popup.", langLabel(lang)));
    } else {
      setStatus(msg("ocrDone", "Done - use the browser's \"Translate page\""));
    }
    setTimeout(hideStatus, 1400);
  } catch (e) {
    console.error(e);
    setStatus(msg("ocrLoadError", "Could not process this image."));
  }
}

main();

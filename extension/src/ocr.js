// ocr.js - controller for the standalone image-OCR page opened from the right-click
// "OCR & translate this image" menu. Reads ?src=, runs the shared overlay unit with the
// user's preferred OCR language, shows progress, and sets <html lang> so the browser
// offers "Translate page".

import { overlayImage, ocrLangToHtmlLang, makeBadge } from "./ocr-overlay.js";

const statusEl = document.getElementById("status");
const barEl = document.getElementById("progress-bar");
const textEl = document.getElementById("status-text");
const mount = document.getElementById("ocr-mount");

const msg = (key, fallback) => {
  try { return chrome.i18n.getMessage(key) || fallback; } catch { return fallback; }
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
      container.append(makeBadge(msg("ocrNoText", "No text found")));
      setStatus(msg("ocrNoText", "No text found"));
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

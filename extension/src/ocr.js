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

function isFileUrl(url) { return /^file:/i.test(url); }

// A file URL with a host is a UNC path (\\server\share). File-scheme match patterns only
// cover empty-host URLs, so no toggle can ever make it readable - viewer.js fails the same
// case fast rather than running a doomed fetch.
function isUncUrl(url) { return /^file:\/\/[^/]/i.test(url); }

function setStatus(t) { textEl.textContent = t; }
function setProgress(f) { barEl.style.width = `${Math.round(f * 100)}%`; }
function hideStatus() { statusEl.classList.add("done"); }

// The right-click menu is offered on every image, including the ones on a file:// page, but an
// extension may not read local files until the user turns on "Allow access to file URLs" - off
// by default on a store install. Without this check the fetch simply throws and the page said
// only "Could not process this image.", which names neither the cause nor the cure. The viewer
// already answers the same question for documents (vLoadFailFile).
function fileAccessAllowed() {
  return new Promise((resolve) => {
    try {
      chrome.extension.isAllowedFileSchemeAccess((allowed) => resolve(allowed !== false));
    } catch {
      resolve(true); // API missing: let the fetch decide rather than block a working path
    }
  });
}

// showNotice replaces the "Loading image.." placeholder - an error used to leave it standing, so
// the page read as still working while the status line said it had failed.
function showNotice(lines, action) {
  const box = document.createElement("div");
  box.className = "ocr-hint";
  for (const line of lines) {
    const p = document.createElement("p");
    p.textContent = line;
    box.append(p);
  }
  if (action) {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "ocr-action";
    btn.textContent = action.label;
    btn.addEventListener("click", action.onClick);
    box.append(btn);
  }
  mount.replaceChildren(box);
}

// Chromium refuses a tabs.create() to chrome://extensions from an extension page in some
// builds, so the button falls back to showing the address for the user to paste.
function openExtensionsPage(btn) {
  const url = `chrome://extensions/?id=${chrome.runtime.id}`;
  try {
    chrome.tabs.create({ url }, () => {
      if (chrome.runtime.lastError) btn.textContent = url;
    });
  } catch {
    btn.textContent = url;
  }
}

function showFileBlocked() {
  showNotice(
    [msg("ocrFileBlocked", 'This image is a local file, and the extension may not read local files yet. Open the extensions page, turn on "Allow access to file URLs" for this extension, then run the OCR again.')],
    {
      label: msg("ocrOpenExtensions", "Open the extensions page"),
      onClick: (ev) => openExtensionsPage(ev.currentTarget),
    },
  );
  setStatus(msg("ocrFileBlockedStatus", "Local files are not allowed for this extension"));
}

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
  if (isUncUrl(src)) {
    showNotice([msg("ocrUncBlocked", "This image sits on a network path (\\\\server\\share), which extensions cannot read. Map the share to a drive letter and open the image from there.")]);
    setStatus(msg("ocrLoadError", "Could not process this image."));
    return;
  }
  if (isFileUrl(src) && !(await fileAccessAllowed())) {
    showFileBlocked();
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
    // The reason first, because a local file reaching this point may be missing rather than
    // blocked - the pre-check already ruled out the plain "no file access" case. The toggle is
    // still the likeliest cure on file://, so it follows as a hint instead of a verdict.
    const lines = [msg("ocrLoadError", "Could not process this image."), msg("vReason", "Reason: {1}", e.message)];
    if (isFileUrl(src)) {
      lines.push(msg("ocrFileBlocked", 'This image is a local file, and the extension may not read local files yet. Open the extensions page, turn on "Allow access to file URLs" for this extension, then run the OCR again.'));
      showNotice(lines, { label: msg("ocrOpenExtensions", "Open the extensions page"), onClick: (ev) => openExtensionsPage(ev.currentTarget) });
    } else {
      showNotice(lines);
    }
    setStatus(msg("ocrLoadError", "Could not process this image."));
  }
}

main();

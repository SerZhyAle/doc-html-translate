// media-save.js - "Save media file for translation", the right-click action that puts the file a page
// shows into a folder the reader named in the settings, for translating it somewhere else (the
// desktop app, an OCR tool). Ticket D2HTML-0171.
//
// What is saved, in order:
//   0. every large file of a carousel, when the clicked file is one of its slides or, with no file
//      clicked, when a carousel is what the reader is looking at (findCarouselMedia);
//   1. the picture, video or audio the reader right-clicked, when the browser names its address;
//   2. the file the clicked frame shows by itself (an image, a video or a PDF opened directly);
//   3. the largest file the clicked frame shows - by its byte size where the browser's resource
//      timing knows it, by an estimate from its pixel size or duration where it does not;
//   4. when the frame cannot be read at all, its own address if that names a file rather than a page.
// Any extension is saved as it is; nothing is converted.
//
// Where: the browser lets an extension write only inside its download folder, so the setting is a
// folder there ("" = the download folder itself). The browser picks the file name - from the
// server's headers, the address and the type - and this module only puts the folder in front of it.
//
// The "downloads" permission is optional: declaring it outright would make the browser switch the
// extension off on update for every existing user until they accept a new warning. It is asked for
// the first time the action is used (or from the settings page), and never before.

import { DEFAULT_OPTIONS } from "./defaults.js";

export const DOWNLOADS_PERMISSION ={ permissions: ["downloads"] };
const NOTICE_MS = 4000;
const PENDING_MS = 120000;
const CAROUSEL_STEP_MS = 400;
const MAX_FOLDER_LENGTH = 200;
const WEB_PAGE_EXTS = new Set(["html", "htm", "xhtml", "shtml", "php", "asp", "aspx", "jsp", "cgi"]);
const WINDOWS_RESERVED = /^(con|prn|aux|nul|com\d|lpt\d)$/i;

// normalizeMediaFolder turns what the reader typed into a folder path the downloads API accepts:
// relative, "/"-separated, no drive letter, no "." or ".." segment, no character Windows refuses in a
// name, no trailing dot or space. An empty result means the download folder itself.
export function normalizeMediaFolder(raw) {
  const text = String(raw ?? "").trim().replace(/^[a-zA-Z]:/, "");
  const parts = [];
  for (const piece of text.split(/[\\/]+/)) {
    let seg = piece.replace(/[\u0000-\u001f<>:"|?*]/g, "_").trim().replace(/^\.+/, "").replace(/[. ]+$/, "");
    if (!seg) continue;
    if (WINDOWS_RESERVED.test(seg)) seg = `_${seg}`;
    parts.push(seg);
  }
  let out = parts.join("/");
  if (out.length > MAX_FOLDER_LENGTH) out = out.slice(0, MAX_FOLDER_LENGTH).replace(/[/. ]+$/, "");
  return out;
}

// savableUrl says whether the downloads API can fetch an address on its own. A blob: address belongs
// to the page that made it (a streamed video is one) and is not reachable from outside the page.
export function savableUrl(url) {
  return typeof url === "string" && /^(https?:|file:|data:)/i.test(url);
}

// directFileUrl returns the address of a frame the extension could not read when that address names
// a file - its last path segment has an extension that is not a web page's - and "" otherwise.
export function directFileUrl(url) {
  let parsed;
  try { parsed = new URL(url); } catch { return ""; }
  if (!/^(https?|file):$/.test(parsed.protocol)) return "";
  const last = decodeURIComponent(parsed.pathname.split("/").pop() || "");
  const dot = last.lastIndexOf(".");
  if (dot <= 0 || dot === last.length - 1) return "";
  return WEB_PAGE_EXTS.has(last.slice(dot + 1).toLowerCase()) ? "" : parsed.href;
}

// clickedMediaUrl is the right-clicked picture, video or audio, when the browser names a savable one.
export function clickedMediaUrl(info) {
  if (!info || !["image", "video", "audio"].includes(info.mediaType)) return "";
  return savableUrl(info.srcUrl) ? info.srcUrl : "";
}

// findFrameMedia runs inside the clicked frame (chrome.scripting.executeScript serializes it, so it
// must not reach anything outside its own body). It returns { url, from } or null.
export function findFrameMedia() {
  const ownType = String(document.contentType || "").toLowerCase();
  if (ownType && ownType !== "text/html" && ownType !== "application/xhtml+xml") {
    return { url: location.href, from: "document" };
  }
  const usable = (u) => typeof u === "string" && /^(https?:|file:|data:)/i.test(u);
  const bytes = new Map();
  try {
    for (const e of performance.getEntriesByType("resource")) {
      const n = Math.max(e.encodedBodySize || 0, e.decodedBodySize || 0, e.transferSize || 0);
      if (n > (bytes.get(e.name) || 0)) bytes.set(e.name, n);
    }
  } catch { /* no timing - the estimates below decide */ }
  let bestUrl = "";
  let bestSize = 0;
  // A size the browser measured wins over an estimate of the same file; the estimates are rough
  // byte counts, so a measured file and an estimated one still compare on one scale.
  const consider = (url, estimate) => {
    if (!usable(url)) return;
    const size = bytes.get(url) || (url.startsWith("data:") ? url.length : estimate);
    if (size > bestSize) { bestUrl = url; bestSize = size; }
  };
  const area = (el) => {
    const r = el.getBoundingClientRect();
    return Math.max(0, r.width) * Math.max(0, r.height);
  };
  for (const img of document.images) {
    // About half a byte per pixel: a typical photo JPEG.
    consider(img.currentSrc || img.src, ((img.naturalWidth * img.naturalHeight) || area(img)) * 0.5 + 1);
  }
  for (const media of document.querySelectorAll("video, audio")) {
    const seconds = Number.isFinite(media.duration) && media.duration > 0 ? media.duration : 60;
    // Bytes per second of a modest web video or an ordinary audio stream.
    const rate = media.tagName === "VIDEO" ? 250000 : 16000;
    const urls = [media.currentSrc, media.src, ...Array.from(media.querySelectorAll("source"), (s) => s.src)];
    for (const u of urls) consider(u, seconds * rate);
  }
  for (const el of document.querySelectorAll("embed[src], object[data]")) {
    consider(el.src || el.data, area(el) * 2 + 1);
  }
  return bestUrl ? { url: bestUrl, from: "frame" } : null;
}

// findCarouselMedia runs inside the clicked frame, self-contained like findFrameMedia. A carousel is
// a row of large slides seen through a window about one slide wide. It is the one holding
// `clickedUrl` or, with none, the one filling most of the viewport - unless a large file outside any
// carousel fills more. Sites that keep only the neighbouring slides in the page are stepped through
// with the carousel's own arrows and put back where they were. Returns { urls } or null.
export async function findCarouselMedia(clickedUrl, stepMs) {
  const MIN_SIDE = 200;
  const MAX_STEPS = 40;
  const MAX_DEPTH = 25;
  const BUDGET_MS = 25000;
  const usable = (u) => typeof u === "string" && /^(https?:|file:|data:)/i.test(u);
  const box = (el) => el.getBoundingClientRect();
  const urlOf = (el) => {
    const all = [el.currentSrc, el.src];
    if (el.tagName === "VIDEO") all.push(...Array.from(el.querySelectorAll("source"), (s) => s.src));
    return all.find(usable) || "";
  };
  const largeIn = (root) => Array.from(root.querySelectorAll("img, video")).filter((el) => {
    const r = box(el);
    return r.width >= MIN_SIDE && r.height >= MIN_SIDE;
  });
  const visibleArea = (el) => {
    const r = box(el);
    const w = Math.min(r.right, innerWidth) - Math.max(r.left, 0);
    const h = Math.min(r.bottom, innerHeight) - Math.max(r.top, 0);
    return w > 0 && h > 0 ? w * h : 0;
  };
  // The nearest ancestor that clips sideways and is too narrow to show two slides.
  const windowOf = (track, slideWidth) => {
    for (let el = track; el; el = el.parentElement) {
      if (/(hidden|auto|scroll|clip)/.test(getComputedStyle(el).overflowX)) {
        return box(el).width <= slideWidth * 1.6 ? el : null;
      }
    }
    return null;
  };
  // A slide is measured by the file it shows: its own element may be a one-pixel positioning box.
  // Sites wrap each file in many layers, so the search for the row goes far up.
  const trackOf = (media) => {
    let node = media;
    for (let depth = 0; depth < MAX_DEPTH && node.parentElement; depth++, node = node.parentElement) {
      const parent = node.parentElement;
      const inside = largeIn(parent);
      const rects = Array.from(parent.children, (c) => inside.find((m) => c === m || c.contains(m)))
        .filter(Boolean).map(box).sort((a, b) => a.left - b.left);
      if (rects.length < 2) continue;
      const first = rects[0];
      const inRow = rects.every((r, i) => Math.abs(r.top - first.top) <= first.height * 0.1
        && (i === 0 || r.left - rects[i - 1].left >= first.width * 0.8));
      // Layers lying on one another (a poster under its video) are not the row; it may be higher up.
      const view = inRow ? windowOf(parent, first.width) : null;
      if (view) return { track: parent, view };
    }
    return null;
  };

  const tracks = [];
  const loose = [];
  for (const media of largeIn(document)) {
    if (tracks.some((t) => t.track.contains(media))) continue;
    const found = trackOf(media);
    if (found) tracks.push(found);
    else loose.push(media);
  }
  let chosen = null;
  if (clickedUrl) {
    chosen = tracks.find((t) => largeIn(t.track).some((m) => urlOf(m) === clickedUrl)) || null;
  } else {
    for (const t of tracks) {
      if (visibleArea(t.view) > (chosen ? visibleArea(chosen.view) : 0)) chosen = t;
    }
    if (chosen && loose.some((m) => visibleArea(m) > visibleArea(chosen.view))) chosen = null;
  }
  if (!chosen) return null;

  const { track, view } = chosen;
  const urls = new Set();
  const collect = () => {
    for (const m of largeIn(track)) {
      const u = urlOf(m);
      if (u) urls.add(u);
    }
  };
  // An arrow is a small button lying over the left or right edge of the window, never a link.
  const arrow = (side) => {
    if (!track.isConnected) return null;
    const w = box(view);
    for (let scope = view, i = 0; scope && i < 4; i++, scope = scope.parentElement) {
      for (const b of scope.querySelectorAll('button, [role="button"]')) {
        if (b.disabled || b.getAttribute("aria-disabled") === "true" || b.closest("a[href]")) continue;
        const r = box(b);
        if (!r.width || !r.height || r.width > 90 || r.height > 90) continue;
        const x = r.left + r.width / 2;
        const y = r.top + r.height / 2;
        if (y < w.top + w.height * 0.25 || y > w.top + w.height * 0.75) continue;
        const atEdge = side === "next"
          ? x >= w.right - w.width * 0.25 && x <= w.right + 8
          : x <= w.left + w.width * 0.25 && x >= w.left - 8;
        if (atEdge) return b;
      }
    }
    return null;
  };
  const deadline = Date.now() + BUDGET_MS;
  const press = async (button) => {
    button.click();
    await new Promise((resolve) => setTimeout(resolve, stepMs));
  };
  // Walks one way until the arrow is gone, or until more steps brought nothing new than there are
  // slides - an endless carousel going round.
  const walk = async (side) => {
    let steps = 0;
    let dry = 0;
    while (steps < MAX_STEPS && dry <= urls.size && Date.now() < deadline) {
      const button = arrow(side);
      if (!button) break;
      await press(button);
      steps++;
      const before = urls.size;
      collect();
      dry = urls.size > before ? 0 : dry + 1;
    }
    return steps;
  };
  collect();
  let offset = await walk("next");
  offset -= await walk("prev");
  for (; offset !== 0; offset -= Math.sign(offset)) {
    const button = arrow(offset > 0 ? "prev" : "next");
    if (!button) break;
    await press(button);
  }
  return urls.size ? { urls: [...urls] } : null;
}

// showPageNotice runs inside the page: a short status line in the corner, gone after a few seconds.
export function showPageNotice(text, ms) {
  const id = "dht-media-save-notice";
  const old = document.getElementById(id);
  if (old) old.remove();
  const box = document.createElement("div");
  box.id = id;
  box.setAttribute("role", "status");
  box.textContent = text;
  Object.assign(box.style, {
    position: "fixed", zIndex: "2147483647", right: "16px", bottom: "16px",
    maxWidth: "min(460px, calc(100vw - 32px))", padding: "10px 14px", borderRadius: "8px",
    background: "#202124", color: "#ffffff", font: "14px/1.4 system-ui, sans-serif",
    boxShadow: "0 2px 10px rgba(0,0,0,.35)", pointerEvents: "none", overflowWrap: "anywhere",
  });
  (document.body || document.documentElement).append(box);
  setTimeout(() => box.remove(), ms);
}

// message reads the browser-language catalog (a worker has no interface-language override loaded)
// and fills {1}, {2} the way i18n.js's t() does.
function message(key, fallback, ...subs) {
  let text = "";
  try { text = chrome.i18n.getMessage(key) || ""; } catch { /* no catalog - English */ }
  return subs.reduce((out, sub, i) => out.replace(`{${i + 1}}`, () => String(sub)), text || fallback);
}

function notify(tabId, text) {
  if (tabId == null) return Promise.resolve();
  return chrome.scripting.executeScript({
    target: { tabId },
    func: showPageNotice,
    args: [text, NOTICE_MS],
  }).catch(() => { /* a page the extension may not touch - the download bar still shows the file */ });
}

// Downloads this module started, by address, with the folder each goes to. The browser asks for the
// file name a moment after the download starts; the entry is dropped then, or after a while if the
// download never got that far.
const pendingFolders = new Map();
let namingHooked = false;

function nameDownload(item, suggest) {
  if (!item || item.byExtensionId !== chrome.runtime.id || !pendingFolders.has(item.url)) return;
  const folder = pendingFolders.get(item.url);
  pendingFolders.delete(item.url);
  if (!folder) return;
  const name = String(item.filename || "").split(/[\\/]/).pop() || "download";
  suggest({ filename: `${folder}/${name}`, conflictAction: "uniquify" });
}

// hookDownloadNaming registers the file-name listener once the downloads API is there - at start when
// the permission is already held, or right after it is granted.
export function hookDownloadNaming() {
  if (namingHooked || !globalThis.chrome || !chrome.downloads || !chrome.downloads.onDeterminingFilename) return;
  chrome.downloads.onDeterminingFilename.addListener(nameDownload);
  namingHooked = true;
}

// pickTargets finds what to save for one right-click: a whole carousel, else one file, else nothing.
async function pickTargets(info, tab) {
  const clicked = clickedMediaUrl(info);
  const target = { tabId: tab.id, frameIds: [Number.isInteger(info.frameId) ? info.frameId : 0] };
  const inFrame = async (script) => {
    const [result] = await chrome.scripting.executeScript({ target, ...script });
    return result && result.result;
  };
  try {
    const carousel = await inFrame({ func: findCarouselMedia, args: [clicked, CAROUSEL_STEP_MS] });
    const urls = ((carousel && carousel.urls) || []).filter(savableUrl);
    if (urls.length) return urls;
  } catch { /* an unreadable frame, or the page moved on - the single file below */ }
  if (clicked) return [clicked];
  try {
    const found = await inFrame({ func: findFrameMedia });
    return found && savableUrl(found.url) ? [found.url] : [];
  } catch {
    const own = directFileUrl(info.frameUrl || info.pageUrl || (tab && tab.url) || "");
    return own ? [own] : [];
  }
}

// mediaFolderOption is the stored folder, or the default one for a profile that never set it.
async function mediaFolderOption() {
  let raw = DEFAULT_OPTIONS.mediaFolder;
  try {
    const got = await chrome.storage.local.get("options");
    if (got.options && typeof got.options.mediaFolder === "string") raw = got.options.mediaFolder;
  } catch { /* storage unreadable - the default folder */ }
  return normalizeMediaFolder(raw);
}

// openMediaSettings opens the settings on the page that holds the folder and the permission button.
async function openMediaSettings() {
  try {
    const got = await chrome.storage.local.get("uiState");
    const state = got.uiState && typeof got.uiState === "object" ? got.uiState : {};
    await chrome.storage.local.set({ uiState: { ...state, page: "general" } });
  } catch { /* the page opens where it was last left */ }
  try { await chrome.runtime.openOptionsPage(); } catch { /* nothing else to show */ }
}

// requestDownloads asks for the optional permission. It must be the first thing a click handler
// does: the browser accepts the request only while the click still counts as the reader's gesture.
// Resolves to true (held or granted), false (refused) or null (the browser would not ask from here).
export function requestDownloads() {
  let asked;
  try { asked = chrome.permissions.request(DOWNLOADS_PERMISSION); } catch (e) { asked = Promise.reject(e); }
  return Promise.resolve(asked).then((granted) => granted === true, async () => {
    // The request was refused for want of a gesture; a permission held already still counts.
    try { return (await chrome.permissions.contains(DOWNLOADS_PERMISSION)) ? true : null; } catch { return null; }
  });
}

// saveMedia is the menu action. `granted` is requestDownloads()'s answer, started before anything
// else in the click handler.
export async function saveMedia(info, tab, granted) {
  const tabId = tab && tab.id != null ? tab.id : null;
  if (tabId == null) return { ok: false, reason: "no-tab" };
  const allowed = await granted;
  if (allowed !== true) {
    // Refused, or the browser would not ask from the menu: the settings page asks with a button.
    if (allowed === null) await openMediaSettings();
    else await notify(tabId, message("mediaSavePermissionRefused", "Saving files is not allowed. Allow it in the extension's settings."));
    return { ok: false, reason: "permission" };
  }
  hookDownloadNaming();
  const urls = await pickTargets(info, tab);
  if (!urls.length) {
    await notify(tabId, message("mediaSaveNoneFound", "No media file found here to save."));
    return { ok: false, reason: "none" };
  }
  if (!chrome.downloads || !chrome.downloads.download) {
    await notify(tabId, message("mediaSaveFailed", "Could not save the media file: {1}", "downloads API unavailable - reopen the browser"));
    return { ok: false, reason: "api" };
  }
  const folder = await mediaFolderOption();
  const saved = [];
  let failure = "";
  for (const url of urls) {
    try { await startDownload(url, folder); saved.push(url); }
    catch (e) { failure = String((e && e.message) || e); }
  }
  if (failure) {
    await notify(tabId, message("mediaSaveFailed", "Could not save the media file: {1}", failure));
    return { ok: false, reason: "download" };
  }
  const where = folder ? `${message("mediaSaveDownloads", "Downloads")}/${folder}` : message("mediaSaveDownloads", "Downloads");
  await notify(tabId, saved.length > 1
    ? message("mediaSaveStartedMany", "Saving the media files ({1}) to {2}", saved.length, where)
    : message("mediaSaveStarted", "Saving the media file to {1}", where));
  return { ok: true, urls: saved, folder };
}

// startDownload hands one address to the browser, remembering the folder for the file-name question.
// The name itself is left as the browser chose it.
async function startDownload(url, folder) {
  pendingFolders.set(url, folder);
  const forget = setTimeout(() => pendingFolders.delete(url), PENDING_MS);
  // A worker timer is a number; a test runner's is an object that must not hold the process open.
  if (forget && typeof forget.unref === "function") forget.unref();
  try {
    await chrome.downloads.download({ url, conflictAction: "uniquify", saveAs: false });
  } catch (e) {
    pendingFolders.delete(url);
    throw e;
  }
}

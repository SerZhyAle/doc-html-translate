// media-save.js - "Save media file for translation", the right-click action that puts the file a page
// shows into the browser's download folder, for translating it somewhere else (the desktop app, an
// OCR tool). Ticket D2HTML-0171.
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
// How: no extra permission. The extension's own page (save.html) reads the file with the host access
// the extension already holds and hands it to the browser behind a link with the `download` attribute
// (save-file.js), under the file's own name. A service worker has no document for such a link, so this
// module only finds the addresses, leaves them in session storage and opens that page.

const NOTICE_MS = 4000;
const CAROUSEL_STEP_MS = 400;
const MAX_NAME_LENGTH = 150;
const FALLBACK_NAME = "media";
// Where the addresses wait for the page that saves them.
export const JOB_PREFIX = "mediaJob:";
const WEB_PAGE_EXTS = new Set(["html", "htm", "xhtml", "shtml", "php", "asp", "aspx", "jsp", "cgi"]);
const WINDOWS_RESERVED = /^(con|prn|aux|nul|com\d|lpt\d)$/i;
// The extension a file gets when neither the server nor the address names one.
const TYPE_EXTS = {
  "image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp", "image/avif": "avif",
  "image/jxl": "jxl", "image/bmp": "bmp", "image/tiff": "tif", "image/svg+xml": "svg",
  "video/mp4": "mp4", "video/webm": "webm", "video/ogg": "ogv", "video/quicktime": "mov",
  "audio/mpeg": "mp3", "audio/mp4": "m4a", "audio/ogg": "ogg", "audio/wav": "wav", "audio/webm": "weba",
  "application/pdf": "pdf",
};

// savableUrl says whether the extension can read an address on its own. A blob: address belongs
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
  }).catch(() => { /* a page the extension may not touch - the saving page still says what happened */ });
}

// dispositionName is the file name a server offers in Content-Disposition, "" when it offers none.
function dispositionName(value) {
  const encoded = /filename\*\s*=\s*[\w-]*'[\w-]*'([^;]+)/i.exec(value);
  if (encoded) {
    try { return decodeURIComponent(encoded[1].trim()); } catch { /* a broken escape - the plain name */ }
  }
  const quoted = /filename\s*=\s*"((?:\\.|[^"\\])*)"/i.exec(value);
  if (quoted) return quoted[1].replace(/\\(.)/g, "$1");
  const bare = /filename\s*=\s*([^;"]+)/i.exec(value);
  return bare ? bare[1].trim() : "";
}

// addressName is the last path segment of a web or file address; a data: address names nothing.
function addressName(url) {
  let parsed;
  try { parsed = new URL(url); } catch { return ""; }
  if (!/^(https?|file):$/.test(parsed.protocol)) return "";
  const last = parsed.pathname.split("/").pop() || "";
  try { return decodeURIComponent(last); } catch { return last; }
}

// cleanName makes one file name Windows accepts: no separator or reserved character, no leading dot,
// no trailing dot or space, bounded length with the extension kept.
function cleanName(raw) {
  let name = String(raw || "").replace(/[\u0000-\u001f<>:"/\\|?*]+/g, "_").trim().replace(/^\.+/, "").replace(/[. ]+$/, "");
  if (!name) return "";
  if (WINDOWS_RESERVED.test(name.split(".")[0])) name = `_${name}`;
  if (name.length > MAX_NAME_LENGTH) {
    const dot = name.lastIndexOf(".");
    const ext = dot > 0 && name.length - dot <= 12 ? name.slice(dot) : "";
    name = name.slice(0, MAX_NAME_LENGTH - ext.length).replace(/[. ]+$/, "") + ext;
  }
  return name;
}

// fileNameFor is the name a saved file gets: the server's, else the address's, else "media", with an
// extension from the content type when the name carries none. The name is otherwise left as it is.
export function fileNameFor({ url, disposition, type }) {
  const name = cleanName(dispositionName(String(disposition || ""))) || cleanName(addressName(url)) || FALLBACK_NAME;
  if (/\.[A-Za-z0-9]{1,8}$/.test(name)) return name;
  const ext = TYPE_EXTS[String(type || "").split(";")[0].trim().toLowerCase()];
  return ext ? `${name}.${ext}` : name;
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

// saveMedia is the menu action: find the files, then open the page that saves them. One file is saved
// in a background tab that closes itself; several open the tab in front, where the browser may ask
// whether to allow more than one download.
export async function saveMedia(info, tab) {
  const tabId = tab && tab.id != null ? tab.id : null;
  if (tabId == null) return { ok: false, reason: "no-tab" };
  const urls = await pickTargets(info, tab);
  if (!urls.length) {
    await notify(tabId, message("mediaSaveNoneFound", "No media file found here to save."));
    return { ok: false, reason: "none" };
  }
  try {
    const id = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
    await chrome.storage.session.set({ [JOB_PREFIX + id]: { urls } });
    const page = { url: `${chrome.runtime.getURL("src/save.html")}#${id}`, active: urls.length > 1, openerTabId: tabId };
    if (Number.isInteger(tab.index)) page.index = tab.index + 1;
    await chrome.tabs.create(page);
  } catch (e) {
    await notify(tabId, message("mediaSaveFailed", "Could not save the media file: {1}", String((e && e.message) || e)));
    return { ok: false, reason: "page" };
  }
  const where = message("mediaSaveDownloads", "Downloads");
  await notify(tabId, urls.length > 1
    ? message("mediaSaveStartedMany", "Saving the media files ({1}) to {2}", urls.length, where)
    : message("mediaSaveStarted", "Saving the media file to {1}", where));
  return { ok: true, urls };
}

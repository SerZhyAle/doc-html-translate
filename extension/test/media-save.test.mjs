// "Save media file for translation" (media-save.js, ticket D2HTML-0171): which file a right-click
// saves, where it goes, and what the reader sees when it cannot be saved. The browser is a fake
// that records what the module asked of it.

import { test, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { parseHTML } from "linkedom";

const calls = { scripts: [], downloads: [], suggests: [], optionsOpened: 0, stored: [] };
let stored = {};                 // chrome.storage.local
let frameResult = null;          // what findFrameMedia "returns" from the page
let carouselResult = null;       // what findCarouselMedia "returns"
let frameThrows = false;         // the page refuses injection
let permissionAnswer = true;     // chrome.permissions.request
let permissionThrows = false;
let permissionHeld = false;      // chrome.permissions.contains
const filenameListeners = [];

globalThis.chrome = {
  runtime: {
    id: "ext-id",
    openOptionsPage: async () => { calls.optionsOpened++; },
  },
  i18n: { getMessage: () => "" },
  storage: {
    local: {
      get: async (key) => (key in stored ? { [key]: stored[key] } : {}),
      set: async (obj) => { Object.assign(stored, obj); calls.stored.push(obj); },
    },
  },
  permissions: {
    request: (p) => {
      if (permissionThrows) throw new Error("This function must be called during a user gesture");
      assert.deepEqual(p, { permissions: ["downloads"] });
      return Promise.resolve(permissionAnswer);
    },
    contains: async () => permissionHeld,
  },
  scripting: {
    executeScript: async (arg) => {
      calls.scripts.push(arg);
      if (arg.func && arg.func.name === "findCarouselMedia") {
        if (frameThrows) throw new Error("Cannot access contents of the page");
        return [{ result: carouselResult }];
      }
      if (arg.func && arg.func.name === "findFrameMedia") {
        if (frameThrows) throw new Error("Cannot access contents of the page");
        return [{ result: frameResult }];
      }
      return [{ result: undefined }];
    },
  },
  downloads: {
    download: async (arg) => { calls.downloads.push(arg); return 7; },
    onDeterminingFilename: { addListener: (fn) => filenameListeners.push(fn) },
  },
};

const media = await import("../src/media-save.js");
const { normalizeMediaFolder, directFileUrl, clickedMediaUrl, savableUrl, findFrameMedia, findCarouselMedia, saveMedia, requestDownloads } = media;

beforeEach(() => {
  calls.scripts.length = 0;
  calls.downloads.length = 0;
  calls.suggests.length = 0;
  calls.stored.length = 0;
  calls.optionsOpened = 0;
  stored = {};
  frameResult = null;
  carouselResult = null;
  frameThrows = false;
  permissionAnswer = true;
  permissionThrows = false;
  permissionHeld = false;
});

// nameAs fires the browser's file-name question for a download and returns what the module suggested.
function nameAs(item) {
  const got = [];
  for (const fn of filenameListeners) fn({ byExtensionId: "ext-id", ...item }, (s) => got.push(s));
  return got;
}

function notices() {
  return calls.scripts.filter((s) => s.func && s.func.name === "showPageNotice").map((s) => s.args[0]);
}

test("the folder is made relative and safe for the downloads API", () => {
  assert.equal(normalizeMediaFolder("Media for translation"), "Media for translation");
  assert.equal(normalizeMediaFolder("  a\\b//c/ "), "a/b/c");
  assert.equal(normalizeMediaFolder("C:\\Users\\me\\Translate"), "Users/me/Translate");
  assert.equal(normalizeMediaFolder("../../etc/./x"), "etc/x");
  assert.equal(normalizeMediaFolder("bad<>:\"|?*name. "), "bad_______name");
  assert.equal(normalizeMediaFolder("con/aux1/nul"), "_con/aux1/_nul");
  assert.equal(normalizeMediaFolder(""), "");
  assert.equal(normalizeMediaFolder(undefined), "");
  assert.ok(normalizeMediaFolder("x".repeat(500)).length <= 200);
});

test("only addresses the downloads API can fetch are savable; a page's blob is not", () => {
  assert.ok(savableUrl("https://cdn.test/a.jpg"));
  assert.ok(savableUrl("data:image/png;base64,AAAA"));
  assert.ok(savableUrl("file:///C:/x.mkv"));
  assert.ok(!savableUrl("blob:https://site.test/1234"));
  assert.ok(!savableUrl(""));
  assert.equal(clickedMediaUrl({ mediaType: "image", srcUrl: "https://cdn.test/a.jpg" }), "https://cdn.test/a.jpg");
  assert.equal(clickedMediaUrl({ mediaType: "video", srcUrl: "blob:https://site.test/1" }), "");
  assert.equal(clickedMediaUrl({ srcUrl: "https://cdn.test/a.jpg" }), "");
});

test("an unreadable frame's own address is saved only when it names a file, of any extension", () => {
  assert.equal(directFileUrl("https://x.test/v/movie.mkv?x=1"), "https://x.test/v/movie.mkv?x=1");
  assert.equal(directFileUrl("https://x.test/data/book.weird-ext"), "https://x.test/data/book.weird-ext");
  assert.equal(directFileUrl("https://x.test/page.php"), "");
  assert.equal(directFileUrl("https://x.test/index.html"), "");
  assert.equal(directFileUrl("https://x.test/feed/"), "");
  assert.equal(directFileUrl("https://x.test/.hidden"), "");
  assert.equal(directFileUrl("chrome://settings/a.png"), "");
});

// fakePage installs the globals findFrameMedia reads inside a frame.
function fakePage({ type = "text/html", href = "https://site.test/p", images = [], media = [], embeds = [], sizes = {} }) {
  const rect = (w, h) => () => ({ width: w, height: h });
  globalThis.location = { href };
  globalThis.performance = {
    getEntriesByType: () => Object.entries(sizes).map(([name, n]) => ({ name, encodedBodySize: n, decodedBodySize: 0, transferSize: 0 })),
  };
  globalThis.document = {
    contentType: type,
    images: images.map((i) => ({ currentSrc: i.src, src: i.src, naturalWidth: i.w || 0, naturalHeight: i.h || 0, getBoundingClientRect: rect(i.w || 0, i.h || 0) })),
    querySelectorAll: (sel) => {
      if (sel === "video, audio") {
        return media.map((m) => ({ tagName: m.tag, currentSrc: m.src, src: m.src, duration: m.duration, querySelectorAll: () => [] }));
      }
      if (sel.startsWith("embed")) return embeds.map((e) => ({ src: e.src, getBoundingClientRect: rect(e.w, e.h) }));
      return [];
    },
  };
}

test("a frame that shows a file by itself saves that file", () => {
  fakePage({ type: "image/webp", href: "https://cdn.test/photo.webp" });
  assert.deepEqual(findFrameMedia(), { url: "https://cdn.test/photo.webp", from: "document" });
  fakePage({ type: "video/mp4", href: "https://cdn.test/clip" });
  assert.equal(findFrameMedia().url, "https://cdn.test/clip");
});

test("a page saves its largest file: a measured size first, an estimate where none is known", () => {
  fakePage({
    images: [{ src: "https://cdn.test/thumb.jpg", w: 150, h: 150 }, { src: "https://cdn.test/big.jpg", w: 1080, h: 1350 }],
  });
  assert.equal(findFrameMedia().url, "https://cdn.test/big.jpg");

  // The browser measured the small-looking one as the larger file.
  fakePage({
    images: [{ src: "https://cdn.test/a.png", w: 400, h: 400 }, { src: "https://cdn.test/b.jpg", w: 1000, h: 1000 }],
    sizes: { "https://cdn.test/a.png": 2_000_000 },
  });
  assert.equal(findFrameMedia().url, "https://cdn.test/a.png");

  // A video outweighs a photo; a streamed (blob:) video cannot be saved, so the photo is taken.
  fakePage({
    images: [{ src: "https://cdn.test/big.jpg", w: 1080, h: 1350 }],
    media: [{ tag: "VIDEO", src: "https://cdn.test/v.mp4", duration: 30 }],
  });
  assert.equal(findFrameMedia().url, "https://cdn.test/v.mp4");
  fakePage({
    images: [{ src: "https://cdn.test/big.jpg", w: 1080, h: 1350 }],
    media: [{ tag: "VIDEO", src: "blob:https://site.test/9", duration: 30 }],
  });
  assert.equal(findFrameMedia().url, "https://cdn.test/big.jpg");

  fakePage({ embeds: [{ src: "https://site.test/doc.pdf", w: 800, h: 1000 }] });
  assert.equal(findFrameMedia().url, "https://site.test/doc.pdf");

  fakePage({});
  assert.equal(findFrameMedia(), null);
});

test("a right-clicked picture is saved into the default folder under the browser's own file name", async () => {
  const info = { menuItemId: "save-media", mediaType: "image", srcUrl: "https://cdn.test/p.jpg?stp=1", frameId: 0 };
  const res = await saveMedia(info, { id: 5 }, requestDownloads());
  assert.equal(res.ok, true);
  assert.deepEqual(calls.downloads, [{ url: "https://cdn.test/p.jpg?stp=1", conflictAction: "uniquify", saveAs: false }]);
  assert.ok(!calls.scripts.some((s) => s.func && s.func.name === "findFrameMedia"), "the clicked picture needs no page scan");
  assert.deepEqual(nameAs({ url: "https://cdn.test/p.jpg?stp=1", filename: "p.jpg" }),
    [{ filename: "Media for translation/p.jpg", conflictAction: "uniquify" }]);
  // Asked once: a second question about the same address is someone else's download.
  assert.deepEqual(nameAs({ url: "https://cdn.test/p.jpg?stp=1", filename: "p.jpg" }), []);
  assert.match(notices().at(-1), /Downloads\/Media for translation/);
});

test("other downloads keep the name the browser chose", async () => {
  await saveMedia({ mediaType: "image", srcUrl: "https://cdn.test/q.png" }, { id: 5 }, Promise.resolve(true));
  assert.deepEqual(nameAs({ url: "https://cdn.test/q.png", filename: "q.png", byExtensionId: undefined }), []);
  assert.deepEqual(nameAs({ url: "https://elsewhere.test/r.zip", filename: "r.zip" }), []);
});

test("the stored folder is used, and an empty one saves straight into the download folder", async () => {
  stored.options = { mediaFolder: "Translate/In" };
  await saveMedia({ mediaType: "video", srcUrl: "https://cdn.test/v.mp4" }, { id: 5 }, Promise.resolve(true));
  assert.deepEqual(nameAs({ url: "https://cdn.test/v.mp4", filename: "v.mp4" }),
    [{ filename: "Translate/In/v.mp4", conflictAction: "uniquify" }]);

  stored.options = { mediaFolder: "" };
  await saveMedia({ mediaType: "audio", srcUrl: "https://cdn.test/a.ogg" }, { id: 5 }, Promise.resolve(true));
  assert.deepEqual(nameAs({ url: "https://cdn.test/a.ogg", filename: "a.ogg" }), []);
  assert.equal(notices().at(-1), "Saving the media file to Downloads");
});

test("a click beside the media scans the clicked frame and saves what it found", async () => {
  frameResult = { url: "https://cdn.test/found.jpg", from: "frame" };
  const res = await saveMedia({ frameId: 3, pageUrl: "https://site.test/p/1" }, { id: 9 }, Promise.resolve(true));
  assert.equal(res.ok, true);
  const scan = calls.scripts.find((s) => s.func && s.func.name === "findFrameMedia");
  assert.deepEqual(scan.target, { tabId: 9, frameIds: [3] });
  assert.equal(calls.downloads.at(-1).url, "https://cdn.test/found.jpg");
});

test("a frame the extension cannot read falls back to its address only when that names a file", async () => {
  frameThrows = true;
  await saveMedia({ frameId: 0, pageUrl: "https://x.test/files/scan.tiff" }, { id: 9 }, Promise.resolve(true));
  assert.equal(calls.downloads.at(-1).url, "https://x.test/files/scan.tiff");

  calls.downloads.length = 0;
  const res = await saveMedia({ frameId: 0, pageUrl: "https://x.test/page.php" }, { id: 9 }, Promise.resolve(true));
  assert.deepEqual(res, { ok: false, reason: "none" });
  assert.equal(calls.downloads.length, 0);
  assert.equal(notices().at(-1), "No media file found here to save.");
});

// fakeCarousel is a page whose slider keeps only the current slide and its neighbours, 600px wide,
// with an arrow over each edge while there is somewhere to go. `at` is the slide shown.
function fakeCarousel({ count, at = 0, grid = false, wrappers = 12 }) {
  const { document } = parseHTML("<html><body><div id='view'><ul id='track'></ul></div><img id='aside' src='https://cdn.test/aside.jpg'></body></html>");
  const state = { at, document };
  const rect = (left, top, width, height) => () => ({ left, top, width, height, right: left + width, bottom: top + height });
  const view = document.getElementById("view");
  const track = document.getElementById("track");
  view.getBoundingClientRect = rect(0, 0, grid ? 600 * count : 600, 600);
  document.getElementById("aside").getBoundingClientRect = rect(0, 5000, 600, 600);
  const render = () => {
    track.innerHTML = "";
    view.querySelectorAll("button").forEach((b) => b.remove());
    for (let i = 0; i < count; i++) {
      if (!grid && Math.abs(i - state.at) > 1) continue;
      const li = document.createElement("li");
      const img = document.createElement("img");
      img.setAttribute("src", `https://cdn.test/slide-${i}.jpg`);
      const left = (grid ? i : i - state.at) * 600;
      const box = rect(left, 0, 600, 600);
      // As real sliders do it: the slide element is a one-pixel positioning box, the picture sits
      // many wrappers down, with a second layer lying over it.
      li.getBoundingClientRect = rect(left, 0, 1, 1);
      img.getBoundingClientRect = box;
      let inner = li;
      for (let d = 0; d < wrappers; d++) {
        const div = document.createElement("div");
        div.getBoundingClientRect = box;
        inner.append(div);
        inner = div;
      }
      const cover = document.createElement("img");
      cover.setAttribute("src", `https://cdn.test/slide-${i}.jpg`);
      cover.getBoundingClientRect = box;
      inner.append(img, cover);
      track.append(li);
    }
    for (const [step, left, exists] of [[-1, 10, state.at > 0], [1, 550, state.at < count - 1]]) {
      if (grid || !exists) continue;
      const button = document.createElement("button");
      button.getBoundingClientRect = rect(left, 285, 30, 30);
      button.click = () => { state.at += step; render(); };
      view.append(button);
    }
  };
  render();
  globalThis.document = document;
  globalThis.innerWidth = 1200;
  globalThis.innerHeight = 800;
  globalThis.getComputedStyle = (el) => ({ overflowX: el.id === "view" ? "hidden" : "visible" });
  return state;
}

const slides = (n) => Array.from({ length: n }, (_, i) => `https://cdn.test/slide-${i}.jpg`);

test("a carousel gives every slide, stepping through it and putting it back", async () => {
  for (const at of [0, 3, 6]) {
    const page = fakeCarousel({ count: 7, at });
    const found = await findCarouselMedia("", 0);
    assert.deepEqual([...found.urls].sort(), slides(7), `starting at slide ${at}`);
    assert.equal(page.at, at, "the carousel is put back where it was");
  }
});

test("only the clicked file's own carousel is expanded, and a grid of pictures is not one", async () => {
  fakeCarousel({ count: 4 });
  assert.equal((await findCarouselMedia("https://cdn.test/slide-1.jpg", 0)).urls.length, 4);
  assert.equal(await findCarouselMedia("https://cdn.test/aside.jpg", 0), null);
  fakeCarousel({ count: 3, grid: true });
  assert.equal(await findCarouselMedia("", 0), null);
});

test("every file of a carousel is saved under the browser's own names", async () => {
  carouselResult = { urls: ["https://cdn.test/1_n.jpg", "https://cdn.test/2_n.jpg", "blob:https://site.test/3"] };
  const res = await saveMedia({ frameId: 0, mediaType: "image", srcUrl: "https://cdn.test/1_n.jpg" }, { id: 5 }, Promise.resolve(true));
  assert.deepEqual(res.urls, ["https://cdn.test/1_n.jpg", "https://cdn.test/2_n.jpg"]);
  assert.deepEqual(calls.downloads.map((d) => d.url), res.urls);
  assert.deepEqual(nameAs({ url: "https://cdn.test/2_n.jpg", filename: "2_n.jpg" }),
    [{ filename: "Media for translation/2_n.jpg", conflictAction: "uniquify" }]);
  assert.equal(notices().at(-1), "Saving the media files (2) to Downloads/Media for translation");
});

test("nothing is saved without the permission: a refusal says so, a browser that would not ask opens the settings", async () => {
  permissionAnswer = false;
  const refused = await saveMedia({ mediaType: "image", srcUrl: "https://cdn.test/p.jpg" }, { id: 5 }, requestDownloads());
  assert.deepEqual(refused, { ok: false, reason: "permission" });
  assert.equal(calls.downloads.length, 0);
  assert.match(notices().at(-1), /not allowed/);

  permissionThrows = true;
  const unasked = await saveMedia({ mediaType: "image", srcUrl: "https://cdn.test/p.jpg" }, { id: 5 }, requestDownloads());
  assert.deepEqual(unasked, { ok: false, reason: "permission" });
  assert.equal(calls.optionsOpened, 1);
  assert.equal(stored.uiState.page, "general", "the settings open on the page with the folder and the button");
  assert.equal(calls.downloads.length, 0);
});

test("a permission held already counts even when the browser would not ask from the menu", async () => {
  permissionThrows = true;
  permissionHeld = true;
  assert.equal(await requestDownloads(), true);
  permissionHeld = false;
  assert.equal(await requestDownloads(), null);
});

test("a download the browser refuses is reported on the page", async () => {
  const download = chrome.downloads.download;
  chrome.downloads.download = async () => { throw new Error("Invalid URL"); };
  try {
    const res = await saveMedia({ mediaType: "image", srcUrl: "https://cdn.test/p.jpg" }, { id: 5 }, Promise.resolve(true));
    assert.deepEqual(res, { ok: false, reason: "download" });
    assert.equal(notices().at(-1), "Could not save the media file: Invalid URL");
  } finally {
    chrome.downloads.download = download;
  }
});

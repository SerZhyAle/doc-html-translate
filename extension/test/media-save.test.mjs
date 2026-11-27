// "Save media file for translation" (media-save.js, ticket D2HTML-0171): which files a right-click
// saves, the page that is handed them, the names they get, and what the reader sees when nothing can
// be saved. The browser is a fake that records what the module asked of it.

import { test, beforeEach } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import { parseHTML } from "linkedom";

const calls = { scripts: [], tabs: [], jobs: [] };
let frameResult = null;          // what findFrameMedia "returns" from the page
let carouselResult = null;       // what findCarouselMedia "returns"
let frameThrows = false;         // the page refuses injection
let tabThrows = false;           // the browser refuses to open the saving page

globalThis.chrome = {
  runtime: { id: "ext-id", getURL: (p) => `chrome-extension://ext-id/${p}` },
  i18n: { getMessage: () => "" },
  storage: { session: { set: async (obj) => { calls.jobs.push(obj); } } },
  tabs: { create: async (arg) => { if (tabThrows) throw new Error("No window"); calls.tabs.push(arg); return { id: 11 }; } },
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
};

const media = await import("../src/media-save.js");
const { directFileUrl, clickedMediaUrl, savableUrl, findFrameMedia, findCarouselMedia, saveMedia, fileNameFor, JOB_PREFIX } = media;

beforeEach(() => {
  calls.scripts.length = 0;
  calls.tabs.length = 0;
  calls.jobs.length = 0;
  frameResult = null;
  carouselResult = null;
  frameThrows = false;
  tabThrows = false;
});

function notices() {
  return calls.scripts.filter((s) => s.func && s.func.name === "showPageNotice").map((s) => s.args[0]);
}

// lastJob is the addresses handed to the saving page, with the key its tab was opened on.
function lastJob() {
  const [key, job] = Object.entries(calls.jobs.at(-1))[0];
  assert.ok(key.startsWith(JOB_PREFIX));
  assert.equal(calls.tabs.at(-1).url, `chrome-extension://ext-id/src/save.html#${key.slice(JOB_PREFIX.length)}`);
  return job.urls;
}

test("the extension asks for no downloads permission, declared or optional", () => {
  const manifest = JSON.parse(fs.readFileSync(new URL("../manifest.json", import.meta.url), "utf8"));
  const asked = [...manifest.permissions, ...(manifest.optional_permissions || [])];
  assert.ok(!asked.includes("downloads"));
  for (const file of ["media-save.js", "save.js", "background.js", "options.js"]) {
    const source = fs.readFileSync(new URL(`../src/${file}`, import.meta.url), "utf8");
    assert.ok(!/chrome\.(downloads|permissions)\b/.test(source), `${file} calls no downloads or permissions API`);
  }
});

test("only addresses the extension can read are savable; a page's blob is not", () => {
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

test("a right-clicked picture goes to a saving page in a background tab", async () => {
  const info = { menuItemId: "save-media", mediaType: "image", srcUrl: "https://cdn.test/p.jpg?stp=1", frameId: 0 };
  const res = await saveMedia(info, { id: 5, index: 2 });
  assert.equal(res.ok, true);
  assert.deepEqual(lastJob(), ["https://cdn.test/p.jpg?stp=1"]);
  assert.deepEqual({ ...calls.tabs[0], url: undefined }, { url: undefined, active: false, openerTabId: 5, index: 3 });
  assert.ok(!calls.scripts.some((s) => s.func && s.func.name === "findFrameMedia"), "the clicked picture needs no page scan");
  assert.equal(notices().at(-1), "Saving the media file to Downloads");
});

test("a click beside the media scans the clicked frame and saves what it found", async () => {
  frameResult = { url: "https://cdn.test/found.jpg", from: "frame" };
  const res = await saveMedia({ frameId: 3, pageUrl: "https://site.test/p/1" }, { id: 9 });
  assert.equal(res.ok, true);
  const scan = calls.scripts.find((s) => s.func && s.func.name === "findFrameMedia");
  assert.deepEqual(scan.target, { tabId: 9, frameIds: [3] });
  assert.deepEqual(lastJob(), ["https://cdn.test/found.jpg"]);
});

test("a frame the extension cannot read falls back to its address only when that names a file", async () => {
  frameThrows = true;
  await saveMedia({ frameId: 0, pageUrl: "https://x.test/files/scan.tiff" }, { id: 9 });
  assert.deepEqual(lastJob(), ["https://x.test/files/scan.tiff"]);

  calls.tabs.length = 0;
  const res = await saveMedia({ frameId: 0, pageUrl: "https://x.test/page.php" }, { id: 9 });
  assert.deepEqual(res, { ok: false, reason: "none" });
  assert.equal(calls.tabs.length, 0);
  assert.equal(notices().at(-1), "No media file found here to save.");
});

test("a file takes the server's name, else the address's, with the type's extension when it has none", () => {
  const url = "https://cdn.test/v/1_n.jpg?stp=1";
  assert.equal(fileNameFor({ url, disposition: null, type: "image/jpeg" }), "1_n.jpg");
  assert.equal(fileNameFor({ url, disposition: 'attachment; filename="My photo.png"' }), "My photo.png");
  assert.equal(fileNameFor({ url, disposition: "attachment; filename*=UTF-8''%D1%84%D0%BE%D1%82%D0%BE.webp" }), "фото.webp");
  assert.equal(fileNameFor({ url, disposition: "inline; filename=plain.gif" }), "plain.gif");
  assert.equal(fileNameFor({ url: "https://cdn.test/media/abc", type: "video/mp4; codecs=avc1" }), "abc.mp4");
  assert.equal(fileNameFor({ url: "data:image/png;base64,AAAA", type: "image/png" }), "media.png");
  assert.equal(fileNameFor({ url: "https://cdn.test/", type: "application/x-unknown" }), "media");
  assert.equal(fileNameFor({ url: "https://x.test/data/book.weird-ext" }), "book.weird-ext");
});

test("a name from outside cannot leave the download folder or break on Windows", () => {
  const url = "https://cdn.test/x.jpg";
  assert.equal(fileNameFor({ url, disposition: 'attachment; filename="..\\\\..\\\\evil.exe"' }), "_.._evil.exe");
  assert.equal(fileNameFor({ url, disposition: 'attachment; filename="a<b>:c|?.png"' }), "a_b_c_.png");
  assert.equal(fileNameFor({ url, disposition: 'attachment; filename="con.png"' }), "_con.png");
  assert.equal(fileNameFor({ url: "https://cdn.test/%2E%2E%2Fsecret.txt" }), "_secret.txt");
  const long = fileNameFor({ url: `https://cdn.test/${"x".repeat(400)}.jpeg` });
  assert.ok(long.length <= 150 && long.endsWith(".jpeg"));
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

test("every file of a carousel goes to a saving page in front", async () => {
  carouselResult = { urls: ["https://cdn.test/1_n.jpg", "https://cdn.test/2_n.jpg", "blob:https://site.test/3"] };
  const res = await saveMedia({ frameId: 0, mediaType: "image", srcUrl: "https://cdn.test/1_n.jpg" }, { id: 5 });
  assert.deepEqual(res.urls, ["https://cdn.test/1_n.jpg", "https://cdn.test/2_n.jpg"]);
  assert.deepEqual(lastJob(), res.urls);
  assert.equal(calls.tabs[0].active, true);
  assert.equal(notices().at(-1), "Saving the media files (2) to Downloads");
});

test("a saving page the browser will not open is reported on the page", async () => {
  tabThrows = true;
  const res = await saveMedia({ mediaType: "image", srcUrl: "https://cdn.test/p.jpg" }, { id: 5 });
  assert.deepEqual(res, { ok: false, reason: "page" });
  assert.equal(notices().at(-1), "Could not save the media file: No window");
});

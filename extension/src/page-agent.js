// page-agent.js - the only part of the whole-page OCR run that touches the reader's document.
// Injected on demand by the broker (page-ocr.js); it is never a declared content script, so an
// ordinary page the reader never asked about is untouched.
//
// It does no recognition and holds no engine: it enumerates the pictures, draws the plates it is
// handed, keeps them attached as the page reflows, and takes the whole layer away again. The plate
// rules themselves come from ocr-plates.js, loaded here as a module so there is one implementation
// of them and not a second one written for this surface (docs/PARITY.md "OCR").
//
// The layer is drawn beside the page rather than inside it: every plate lives in one container
// appended to <html>, positioned in document coordinates over the picture it belongs to. The page's
// own boxes are never wrapped, moved or re-styled, so its layout, its links and its scripts see the
// document they had.

(() => {
  if (window.__dhtPageOcr) { window.__dhtPageOcr.reinjected = true; return; }

  const NS = "page-ocr";
  const ROOT_ID = "dht-ocr-root";
  const BAR_ID = "dht-ocr-bar";
  const FRAME_ID = "dht-ocr-host-frame";
  const SELECTABLE_CLASS = "dht-ocr-selectable";
  const LAYER_OFF_CLASS = "ocr-layer-off";

  // A picture smaller than this on either side carries no readable text at web resolution - it is
  // an icon, a sprite, a spacer or an avatar. Recognizing them is the bulk of the cost on an
  // ordinary page and none of the value.
  // OCR-OVERLAY rule 13: policy - smallest picture worth a recognition pass on a live page
  // (OCR-PIPELINE amendment 1.2 L).
  const MIN_PICTURE_PX = 96;

  // A run outlives the browser's patience with an idle background worker, and the worker is what
  // owns the queue. An incoming message resets that timer, so the agent says hello while a run is
  // going and stops as soon as it is not.
  const PING_MS = 20000;
  // The broker reports after every picture, and gives one picture at most 150 s. Past this much
  // silence there is no run left to keep alive - the worker restarted, or the run ended without a
  // word - and pinging on would only keep an idle worker awake.
  const PING_STALE_MS = 180000;

  // Positioning is event-driven: scroll, resize, the pictures' own size changes and page mutations
  // each ask for one frame of placement. This slow poll is the safety net for what raises no event
  // at all - a CSS animation or transition carrying a picture - and runs only while a layer exists.
  const FALLBACK_POLL_MS = 1000;

  // Trailing arguments fill {1}, {2}, .. the same way i18n.js's t() does, so a message file written
  // for one surface reads correctly on the other.
  const msg = (key, fallback, ...args) => {
    let text = fallback;
    try { text = chrome.i18n.getMessage(key) || fallback; } catch { text = fallback; }
    if (!args.length) return text;
    return text.replace(/\{(\d)\}/g, (m, i) => (args[i - 1] === undefined ? m : String(args[i - 1])));
  };

  const platesReady = import(chrome.runtime.getURL("src/ocr-plates.js"));

  const state = {
    reinjected: false,
    anchors: new Map(), // picture id -> { img, layer, stopFit, visible }
    nextId: 1,
    ids: new WeakMap(), // img -> picture id, so a rescan never offers the same picture twice
    root: null,
    plates: null, // ocr-plates.js, once loaded - a layer exists only after it is
    bar: null,
    els: {},
    raf: 0,
    poll: 0,
    ro: null,
    listening: false,
    lastStatusAt: 0,
    io: null,
    mo: null,
    onLoad: null,
    onMessage: null,
    pendingNew: 0,
    offered: new WeakSet(), // pictures already counted into pendingNew
    selectable: false,
    ping: 0,
    running: false,
  };
  window.__dhtPageOcr = state;

  // ---- The layer root ------------------------------------------------------
  // Appended to <html>, not to <body>: an absolutely positioned element whose ancestors are all
  // unpositioned is laid out against the initial containing block, which is document coordinates.
  // Hanging it off <body> instead would make every plate inherit whatever offset, margin or
  // positioning the page gave its body, and the layer would sit off the art on a great many sites.
  function ensureRoot() {
    if (state.root && state.root.isConnected) return state.root;
    const root = document.createElement("div");
    root.id = ROOT_ID;
    document.documentElement.append(root);
    state.root = root;
    return root;
  }

  // ---- Picture discovery ---------------------------------------------------
  // The run reads what the page itself shows and could read, never more. The broker holds the
  // extension's access to every site; a picture the page merely names - a file: path, an intranet
  // address, an image that never loaded - would otherwise be fetched with that access and its words
  // written into this DOM, where the page's scripts read them (ticket 42, B49). file: and the
  // browser's internal schemes are left out even where a page shows them: a web page cannot read
  // them, so recognizing them would read on the page's behalf.
  function pictureSrc(img) {
    const src = img.currentSrc || img.src || "";
    return /^(https?:|data:|blob:)/i.test(src) ? src : "";
  }

  // A zero natural size is an image that failed or has not arrived yet: there are no pixels the
  // page could have read, so there is nothing to recognize either.
  function isLoaded(img) {
    return !!img.complete && img.naturalWidth > 0 && img.naturalHeight > 0;
  }

  // readPixels copies the picture as the page decoded it. The canvas is subject to the page's own
  // origin rules, so a cross-origin picture without CORS taints it and reading fails with
  // "tainted" - the broker then decides whether a re-fetch is acceptable. PNG keeps the lettering
  // lossless; drawImage already applies the EXIF orientation the <img> is displayed with.
  function readPixels(id) {
    const a = state.anchors.get(id);
    const img = a && a.img;
    if (!img || !img.isConnected || !pictureSrc(img) || !isLoaded(img)) return { ok: false, error: "unloaded" };
    let url;
    try {
      const canvas = document.createElement("canvas");
      canvas.width = img.naturalWidth;
      canvas.height = img.naturalHeight;
      const ctx = canvas.getContext("2d");
      if (!ctx) return { ok: false, error: "unreadable" };
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
      url = canvas.toDataURL("image/png");
    } catch (e) {
      return { ok: false, error: e && e.name === "SecurityError" ? "tainted" : "unreadable" };
    }
    // A canvas past the browser's size limit serializes to "data:," instead of throwing.
    if (typeof url !== "string" || !url.startsWith("data:image/png")) return { ok: false, error: "unreadable" };
    return { ok: true, src: url };
  }

  function collect() {
    prune();
    const out = [];
    for (const img of document.images) {
      if (!img.isConnected) continue;
      if (state.ids.has(img)) continue;
      const src = pictureSrc(img);
      if (!src || !isLoaded(img)) continue;
      const rect = img.getBoundingClientRect();
      if (rect.width < MIN_PICTURE_PX || rect.height < MIN_PICTURE_PX) continue;
      if (img.naturalWidth < MIN_PICTURE_PX) continue;
      const id = `p${state.nextId++}`;
      state.ids.set(img, id);
      state.anchors.set(id, { img, layer: null, stopFit: null, visible: true });
      out.push({ id, src, top: rect.top, height: rect.height, bottom: rect.bottom });
    }
    // The reader's eye first: pictures at or below the top of the viewport in document order, then
    // the ones already scrolled past. A long archive page is useful long before it is finished.
    out.sort((a, b) => score(a) - score(b));
    observeAll();
    return out.map(({ id, src }) => ({ id, src }));
  }

  function score(r) {
    return r.bottom < 0 ? 1e6 - r.bottom : Math.max(0, r.top);
  }

  function observeAll() {
    if (!state.io && typeof IntersectionObserver !== "undefined") {
      // rootMargin gives the loop a head start, so a plate is already in place by the time the
      // picture it belongs to has finished scrolling in.
      state.io = new IntersectionObserver((entries) => {
        for (const e of entries) {
          const id = state.ids.get(e.target);
          const a = id && state.anchors.get(id);
          if (a) a.visible = e.isIntersecting;
        }
      }, { rootMargin: "300px" });
    }
    if (!state.io) return;
    for (const a of state.anchors.values()) {
      try { state.io.observe(a.img); } catch { /* ignore */ }
    }
  }

  // ---- Placement -----------------------------------------------------------
  // Placement reads each drawn picture's live rect, touching only the pictures in or near the
  // viewport. It used to run every animation frame for as long as any plate existed - a page left
  // open with plates on it did layout work sixty times a second while nothing moved. Now it runs
  // one frame per change signal (see watchLayout), which still covers what a page does to a
  // picture: responsive re-sourcing, a sticky or transformed container, a lazily grown box, a
  // virtualized list that moves it.
  function place(a) {
    const r = a.img.getBoundingClientRect();
    if (a.badgeLayer) put(a.badgeLayer, r);
    if (a.layer) putPlates(a.layer, a.img, r);
  }

  // rotatedChain asks the picture and every ancestor whether it turns, skews or mirrors it: a
  // rotated container carries the picture with it as surely as a rotated <img>.
  function rotatedChain(el, transformRotates) {
    for (let n = el; n && n.nodeType === 1; n = n.parentElement) {
      const cs = getComputedStyle(n);
      if (transformRotates(cs.transform, cs.rotate, cs.scale)) return true;
    }
    return false;
  }

  // putPlates lays the plate layer over the picture as drawn (ocr-plates.js pictureBox), cut to
  // what the element shows. A picture it cannot place truthfully - rotated, skewed, mirrored, or
  // positioned in a way the arithmetic does not know - gets no plates until it can be placed again:
  // OCR-OVERLAY rule 4, clear rather than draw in the wrong place.
  function putPlates(layer, img, r) {
    const { pictureBox, transformRotates } = state.plates;
    const cs = getComputedStyle(img);
    const pb = pictureBox({
      rect: r, style: cs, naturalWidth: img.naturalWidth, naturalHeight: img.naturalHeight,
      offsetWidth: img.offsetWidth, offsetHeight: img.offsetHeight,
      rotated: rotatedChain(img, transformRotates),
    });
    if (!pb) {
      layer.style.display = "none";
      layer.dataset.unplaced = "1";
      layer.dataset.l = "";
      return;
    }
    delete layer.dataset.unplaced;
    const { box, clip } = pb;
    const inset = [clip.top - box.top, box.left + box.width - clip.left - clip.width,
      box.top + box.height - clip.top - clip.height, clip.left - box.left];
    const cut = inset.some((v) => v > 0.5) ? `inset(${inset.map((v) => `${Math.max(0, v)}px`).join(" ")})` : "";
    put(layer, { left: box.left, top: box.top, width: box.width, height: box.height });
    if (layer.style.clipPath !== cut) layer.style.clipPath = cut;
  }

  function put(layer, r) {
    if (r.width < 1 || r.height < 1) { layer.style.display = "none"; return; }
    const left = r.left + window.scrollX;
    const top = r.top + window.scrollY;
    if (layer.style.display === "none" || layer.dataset.l !== String(left) || layer.dataset.t !== String(top)
      || layer.dataset.w !== String(r.width) || layer.dataset.h !== String(r.height)) {
      layer.dataset.l = String(left); layer.dataset.t = String(top);
      layer.dataset.w = String(r.width); layer.dataset.h = String(r.height);
      layer.style.display = "";
      layer.style.left = `${left}px`;
      layer.style.top = `${top}px`;
      layer.style.width = `${r.width}px`;
      layer.style.height = `${r.height}px`;
    }
  }

  function hasLayers() {
    for (const a of state.anchors.values()) if (a.layer || a.badgeLayer) return true;
    return false;
  }

  // prune forgets pictures the page removed. The anchor held the <img> itself, so a single-page
  // app swapping its content kept every old picture - and its plates - alive for the tab's life.
  function prune() {
    for (const [id, a] of state.anchors) {
      if (a.img.isConnected) continue;
      clearLayer(a);
      if (a.badgeLayer) { a.badgeLayer.remove(); a.badgeLayer = null; }
      if (state.io) { try { state.io.unobserve(a.img); } catch { /* ignore */ } }
      if (state.ro) { try { state.ro.unobserve(a.img); } catch { /* ignore */ } }
      state.anchors.delete(id);
    }
  }

  function tick() {
    state.raf = 0;
    prune();
    for (const a of state.anchors.values()) {
      if ((a.layer || a.badgeLayer) && a.visible !== false) place(a);
    }
    if (!hasLayers()) unwatchLayout();
  }

  function schedule() {
    if (state.raf || !state.anchors.size) return;
    state.raf = requestAnimationFrame(tick);
  }

  function watchLayout() {
    if (state.listening) return;
    state.listening = true;
    // Capture: a scroll inside a scrolling container does not bubble to window.
    window.addEventListener("scroll", schedule, { capture: true, passive: true });
    window.addEventListener("resize", schedule, { passive: true });
    if (!state.ro && typeof ResizeObserver !== "undefined") {
      state.ro = new ResizeObserver(schedule);
      try { state.ro.observe(document.documentElement); } catch { /* ignore */ }
    }
    for (const a of state.anchors.values()) {
      if (state.ro && (a.layer || a.badgeLayer)) { try { state.ro.observe(a.img); } catch { /* ignore */ } }
    }
    state.poll = setInterval(schedule, FALLBACK_POLL_MS);
  }

  function unwatchLayout() {
    if (!state.listening) return;
    state.listening = false;
    window.removeEventListener("scroll", schedule, { capture: true });
    window.removeEventListener("resize", schedule);
    if (state.ro) { state.ro.disconnect(); state.ro = null; }
    if (state.poll) { clearInterval(state.poll); state.poll = 0; }
  }

  function track(a) {
    watchLayout();
    if (state.ro) { try { state.ro.observe(a.img); } catch { /* ignore */ } }
    schedule();
  }

  // ---- Drawing -------------------------------------------------------------
  async function drawPlates(id, specs, htmlLang) {
    const a = state.anchors.get(id);
    if (!a || !a.img.isConnected) return;
    const plates = await platesReady;
    const { renderPlates, scheduleFit } = plates;
    state.plates = plates;
    if (!state.anchors.has(id)) return; // removed while the module was loading
    clearLayer(a);
    if (!specs || !specs.length) return;
    const layer = document.createElement("div");
    layer.className = "ocr-overlay dht-ocr-layer";
    renderPlates(layer, specs);
    if (htmlLang) {
      // Tag the plates with the language they were recognized in. The page's own lang attribute is
      // left alone - it is the page's - and this is what tells the browser's translator that these
      // particular words are not in the page's language.
      for (const p of layer.children) p.setAttribute("lang", htmlLang);
    }
    ensureRoot().append(layer);
    a.layer = layer;
    place(a);
    a.stopFit = scheduleFit(layer);
    track(a);
  }

  function clearLayer(a) {
    if (a.stopFit) { try { a.stopFit(); } catch { /* ignore */ } a.stopFit = null; }
    if (a.layer) { a.layer.remove(); a.layer = null; }
  }

  async function markBusy(id, on) {
    const a = state.anchors.get(id);
    if (!a) return;
    if (!on) {
      if (a.badgeLayer) { a.badgeLayer.remove(); a.badgeLayer = null; }
      return;
    }
    const { makeBadge } = await platesReady;
    if (!state.anchors.has(id) || a.badgeLayer) return;
    const holder = document.createElement("div");
    holder.className = "dht-ocr-layer dht-ocr-badge-layer";
    holder.append(makeBadge(msg("pageOcrBadge", "Reading..")));
    ensureRoot().append(holder);
    a.badgeLayer = holder;
    place(a);
    track(a);
  }

  // ---- The reader's controls ----------------------------------------------
  // The vocabulary drawings the bar uses (BUTTONS 2.0 section 2 rule 1: the glyph precedes the caption).
  // This script is injected into a page, so it cannot import glyphs.js - that file is not web accessible and
  // adding it would expose it to every site - and carries the four drawings itself. extension/test/
  // page-agent.test.mjs holds each to the table in glyphs.js, which tests/iconography_test.go holds to the
  // vendored catalog files.
  const BAR_GLYPHS = {
    "action.cancel": "M12,2C6.47,2 2,6.47 2,12s4.47,10 10,10 10,-4.47 10,-10S17.53,2 12,2zM17,15.59L15.59,17 12,13.41 8.41,17 7,15.59 10.59,12 7,8.41 8.41,7 12,10.59 15.59,7 17,8.41 13.41,12 17,15.59z",
    "action.extract-text": "M4.5,9h3c0.6,0 1,0.5 1,1v4c0,0.5 -0.4,1 -1,1h-3c-0.6,0 -1,-0.5 -1,-1v-4c0,-0.5 0.4,-1 1,-1zM5,10.5v3h2v-3zM13.5,9h-3c-0.6,0 -1,0.5 -1,1v4c0,0.5 0.4,1 1,1h3c0.6,0 1,-0.5 1,-1v-0.5h-3.5v-3h3.5V10c0,-0.5 -0.4,-1 -1,-1zM15.5,9v6h1.5v-2h1.1l1.4,2h1.7l-1.5,-2.1c0.5,-0.2 0.8,-0.7 0.8,-1.2V10c0,-0.6 -0.5,-1 -1,-1zM17,10.5h2v1h-2z",
    "view.text-layer": "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z",
    "action.remove": "M7,11v2h10v-2H7zM12,2C6.48,2 2,6.48 2,12s4.48,10 10,10 10,-4.48 10,-10S17.52,2 12,2zM12,20c-4.41,0 -8,-3.59 -8,-8s3.59,-8 8,-8 8,3.59 8,8 -3.59,8 -8,8z",
  };
  const SVG_NS = "http://www.w3.org/2000/svg";

  function barGlyph(id) {
    const svg = document.createElementNS(SVG_NS, "svg");
    svg.setAttribute("class", "dht-ocr-glyph");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    svg.setAttribute("focusable", "false");
    const path = document.createElementNS(SVG_NS, "path");
    path.setAttribute("fill", "currentColor");
    // The catalog's recognized-text drawing is an even-odd shape; filled non-zero its letters close up.
    if (id === "action.extract-text") path.setAttribute("fill-rule", "evenodd");
    path.setAttribute("d", BAR_GLYPHS[id]);
    svg.append(path);
    return svg;
  }

  // The bar lives in the page's DOM, so the page's scripts can find these buttons and click them.
  // Only the reader's own click counts: a scan started by the page would run on the page's say-so.
  // The caption sits in its own span so a state change rewrites the caption and keeps the glyph; `reserve`
  // names the longest alternative caption, whose width the button keeps (BUTTONS 3 rule 3).
  function button(label, onClick, glyphId, reserve) {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "dht-ocr-btn";
    const cap = document.createElement("span");
    cap.className = "dht-ocr-cap";
    cap.textContent = label;
    if (reserve) cap.setAttribute("data-reserve", reserve);
    if (glyphId) b.append(barGlyph(glyphId));
    b.append(cap);
    b.addEventListener("click", (e) => { if (e.isTrusted) onClick(e); });
    return b;
  }

  // A two-state caption keeps the other state's text as its reserve, so the button is always as wide as the
  // wider of the two and its neighbours do not move when it flips.
  const setCaption = (btn, text, reserve) => {
    const cap = btn.querySelector(".dht-ocr-cap");
    cap.textContent = text;
    if (reserve) cap.setAttribute("data-reserve", reserve);
  };

  function toBroker(t, extra) {
    try {
      const p = chrome.runtime.sendMessage({ dht: NS, t, ...extra });
      if (p && typeof p.catch === "function") p.catch(() => {});
    } catch { /* the broker is gone; the local controls still work */ }
  }

  // The bar is the extension's own chrome inside someone else's page. Its language and direction are the
  // interface's - the browser's language, as its messages are - not the page's, so it mirrors with its own
  // text; its size, text size and controls profile are the reader's saved choices from the options page,
  // resolved once when the bar is made and never from an event (UI-SIZING 1.1 section 2 rule 1). Until the
  // choices arrive the bar stays hidden, so it never resizes under the reader. page-overlay.css turns the
  // attributes into the same tokens the extension's pages use.
  const RTL_UI_LANGS = ["ar", "ur"];
  function applyBarProfile(bar) {
    let ui = "en";
    try { ui = (chrome.i18n.getUILanguage() || "en").split("-")[0]; } catch { /* English */ }
    bar.setAttribute("lang", ui);
    bar.setAttribute("dir", RTL_UI_LANGS.includes(ui) ? "rtl" : "ltr");
    let touch = false;
    try { touch = matchMedia("(pointer: coarse)").matches || matchMedia("(any-pointer: none)").matches; } catch { /* a pointer surface */ }
    bar.setAttribute("data-dht-input", touch ? "touch" : "pointer");
    bar.setAttribute("data-dht-pending", "");
    let stored;
    try { stored = chrome.storage.local.get("options"); } catch { /* storage unavailable: the defaults */ }
    Promise.resolve(stored).then((got) => {
      const o = (got && got.options) || {};
      // An unknown value matches no rule in page-overlay.css and leaves the ordinary size.
      if (typeof o.uiSize === "string") bar.setAttribute("data-dht-size", o.uiSize);
      if (typeof o.textSize === "string") bar.setAttribute("data-dht-text", o.textSize);
      if (o.inputProfile === "touch" || o.inputProfile === "pointer") bar.setAttribute("data-dht-input", o.inputProfile);
    }).catch(() => { /* the defaults */ }).then(() => bar.removeAttribute("data-dht-pending"));
  }

  function ensureBar() {
    if (state.bar && state.bar.isConnected) return state.bar;
    const bar = document.createElement("div");
    bar.id = BAR_ID;
    bar.setAttribute("role", "status");
    applyBarProfile(bar);

    const status = document.createElement("span");
    status.className = "dht-ocr-status";
    bar.append(status);

    const stop = button(msg("pageOcrStop", "Cancel"), () => toBroker("stop"), "action.cancel");
    const rescan = button(msg("pageOcrRescan", "Scan new pictures"), () => {
      state.pendingNew = 0;
      state.offered = new WeakSet();
      toBroker("rescan");
      render();
    }, "action.extract-text");
    rescan.hidden = true;
    const hideText = msg("pageOcrHide", "Hide text");
    const showText = msg("pageOcrShow", "Show text");
    const hide = button(hideText, () => {
      const off = document.documentElement.classList.toggle(LAYER_OFF_CLASS);
      // The action that changes (Hide becomes Show) says what it will do next, so it carries no pressed state.
      if (off) setCaption(hide, showText, hideText); else setCaption(hide, hideText, showText);
    }, "view.text-layer", showText);
    // A toggle with a stable name says whether it is on through aria-pressed and a pressed look, not through
    // a second caption; the tooltip tells what the next press does. There is no vocabulary meaning for it, so
    // it is text only (BUTTONS 2.0 section 2 rule 1).
    const select = button(msg("pageOcrSelect", "Select text"), () => {
      state.selectable = !state.selectable;
      document.documentElement.classList.toggle(SELECTABLE_CLASS, state.selectable);
      select.title = state.selectable ? msg("pageOcrSelectOff", "Let clicks through") : "";
      select.setAttribute("aria-pressed", String(state.selectable));
    });
    select.setAttribute("aria-pressed", "false");
    const remove = button(msg("pageOcrRemove", "Remove"), () => toBroker("remove"), "action.remove");

    bar.append(stop, rescan, hide, select, remove);
    document.documentElement.append(bar);
    state.bar = bar;
    state.els = { status, stop, rescan, hide, select, remove };
    watchForNewPictures();
    return bar;
  }

  function render() {
    const e = state.els;
    if (!e.status) return;
    e.rescan.hidden = state.pendingNew === 0 || state.running;
    e.stop.hidden = !state.running;
  }

  function setPing(on) {
    if (on && !state.ping) {
      state.ping = setInterval(() => {
        if (Date.now() - state.lastStatusAt > PING_STALE_MS) { setPing(false); return; }
        toBroker("ping");
      }, PING_MS);
    }
    if (!on && state.ping) { clearInterval(state.ping); state.ping = 0; }
  }

  function setStatus(s) {
    ensureBar();
    state.lastStatusAt = Date.now();
    state.running = !!s.running;
    setPing(state.running);
    let text;
    if (s.error) text = s.error;
    else if (s.running) text = msg("pageOcrRunning", "Reading pictures.. {1} of {2}", s.done, s.total);
    else if (s.stopped) text = msg("pageOcrStopped", "Stopped - {1} of {2} pictures read", s.done, s.total);
    else text = msg("pageOcrDone", "{1} of {2} pictures read - use the browser's \"Translate page\"", s.done, s.total);
    if (s.failed) text += ` ${msg("pageOcrFailed", "({1} could not be read)", s.failed)}`;
    state.els.status.textContent = text;
    render();
  }

  // Pictures that arrive after the run started - lazy loading, infinite scroll - are not recognized
  // behind the reader's back: the bar offers a rescan instead, so a run is always something the
  // reader asked for and can see the size of. A picture already in the page counts once it loads,
  // since only a loaded picture is collected.
  function offer(img) {
    if (state.ids.has(img) || state.offered.has(img)) return 0;
    state.offered.add(img);
    return 1;
  }

  function watchForNewPictures() {
    if (state.mo || typeof MutationObserver === "undefined") return;
    state.mo = new MutationObserver((muts) => {
      let found = 0;
      for (const mu of muts) {
        for (const n of mu.addedNodes) {
          if (n.nodeType !== 1) continue;
          if (n.tagName === "IMG") found += offer(n);
          else if (n.querySelectorAll) {
            for (const img of n.querySelectorAll("img")) found += offer(img);
          }
        }
      }
      if (found) { state.pendingNew += found; render(); }
      // A mutation can move a picture without resizing it; one placement frame settles that.
      if (state.listening) schedule();
    });
    state.mo.observe(document.documentElement, { childList: true, subtree: true });
    // load does not bubble; capture on the document sees every picture's.
    state.onLoad = (e) => {
      const img = e.target;
      if (!img || img.tagName !== "IMG" || !isLoaded(img) || !offer(img)) return;
      state.pendingNew++;
      render();
    };
    document.addEventListener("load", state.onLoad, true);
  }

  // ---- The recognizer host, when it has to live in this page ---------------
  // Only used where the browser offers the extension no document of its own. The frame is on the
  // extension's origin, so the engine inside it runs under the extension's policy, not this page's.
  // It is parked off-screen rather than hidden with display:none, which would keep the browser from
  // ever running it.
  function insertHostFrame(url) {
    let f = document.getElementById(FRAME_ID);
    if (f) f.remove();
    f = document.createElement("iframe");
    f.id = FRAME_ID;
    f.src = url;
    document.documentElement.append(f);
    return { ok: true };
  }

  function removeHostFrame() {
    const f = document.getElementById(FRAME_ID);
    if (f) f.remove();
  }

  // ---- Teardown ------------------------------------------------------------
  // "Remove" has to leave the page indistinguishable from before, so everything this agent ever
  // added or observed goes: the layers, the bar, the frame, the classes it put on <html>, the
  // observers and the frame loop. The injected stylesheet is removed by the broker, which is what
  // added it.
  function teardown() {
    for (const a of state.anchors.values()) {
      clearLayer(a);
      if (a.badgeLayer) a.badgeLayer.remove();
    }
    state.anchors.clear();
    state.ids = new WeakMap();
    if (state.io) { state.io.disconnect(); state.io = null; }
    if (state.mo) { state.mo.disconnect(); state.mo = null; }
    if (state.onLoad) { document.removeEventListener("load", state.onLoad, true); state.onLoad = null; }
    if (state.raf) { cancelAnimationFrame(state.raf); state.raf = 0; }
    unwatchLayout();
    setPing(false);
    removeHostFrame();
    if (state.root) { state.root.remove(); state.root = null; }
    if (state.bar) { state.bar.remove(); state.bar = null; }
    state.els = {};
    document.documentElement.classList.remove(LAYER_OFF_CLASS, SELECTABLE_CLASS);
    state.selectable = false;
    state.running = false;
    state.pendingNew = 0;
    // A later Start injects a fresh agent; a listener left here would answer its messages first.
    if (state.onMessage) { chrome.runtime.onMessage.removeListener(state.onMessage); state.onMessage = null; }
    try { delete window.__dhtPageOcr; } catch { window.__dhtPageOcr = undefined; }
  }

  state.onMessage = (m, sender, sendResponse) => {
    if (!m || m.dht !== NS) return;
    switch (m.t) {
      case "collect":
        ensureBar();
        sendResponse({ ok: true, images: collect() });
        return;
      case "pixels":
        sendResponse(readPixels(m.id));
        return;
      case "host-frame":
        sendResponse(insertHostFrame(m.url));
        return;
      case "drop-host-frame":
        removeHostFrame();
        sendResponse({ ok: true });
        return;
      case "busy":
        markBusy(m.id, m.on);
        sendResponse({ ok: true });
        return;
      case "plates":
        markBusy(m.id, false);
        drawPlates(m.id, m.specs, m.htmlLang);
        sendResponse({ ok: true });
        return;
      case "status":
        setStatus(m);
        sendResponse({ ok: true });
        return;
      case "teardown":
        teardown();
        sendResponse({ ok: true });
        return;
      default:
        return;
    }
  };
  chrome.runtime.onMessage.addListener(state.onMessage);
})();

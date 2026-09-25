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
    bar: null,
    els: {},
    raf: 0,
    poll: 0,
    ro: null,
    listening: false,
    lastStatusAt: 0,
    io: null,
    mo: null,
    pendingNew: 0,
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
  function usableSrc(img) {
    const src = img.currentSrc || img.src || "";
    return /^(https?:|file:|data:|blob:)/i.test(src) ? src : "";
  }

  function collect() {
    prune();
    const out = [];
    for (const img of document.images) {
      if (!img.isConnected) continue;
      if (state.ids.has(img)) continue;
      const src = usableSrc(img);
      if (!src) continue;
      const rect = img.getBoundingClientRect();
      if (rect.width < MIN_PICTURE_PX || rect.height < MIN_PICTURE_PX) continue;
      if (img.naturalWidth && img.naturalWidth < MIN_PICTURE_PX) continue;
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
    if (a.layer) put(a.layer, r);
  }

  function put(layer, r) {
    if (r.width < 1 || r.height < 1) { layer.style.display = "none"; return; }
    const left = r.left + window.scrollX;
    const top = r.top + window.scrollY;
    if (layer.dataset.l !== String(left) || layer.dataset.t !== String(top)
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
    const { renderPlates, scheduleFit } = await platesReady;
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
  function button(label, onClick) {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "dht-ocr-btn";
    b.textContent = label;
    b.addEventListener("click", onClick);
    return b;
  }

  function toBroker(t, extra) {
    try {
      const p = chrome.runtime.sendMessage({ dht: NS, t, ...extra });
      if (p && typeof p.catch === "function") p.catch(() => {});
    } catch { /* the broker is gone; the local controls still work */ }
  }

  function ensureBar() {
    if (state.bar && state.bar.isConnected) return state.bar;
    const bar = document.createElement("div");
    bar.id = BAR_ID;
    bar.setAttribute("role", "status");

    const status = document.createElement("span");
    status.className = "dht-ocr-status";
    bar.append(status);

    const stop = button(msg("pageOcrStop", "Stop"), () => toBroker("stop"));
    const rescan = button(msg("pageOcrRescan", "Scan new pictures"), () => { state.pendingNew = 0; toBroker("rescan"); render(); });
    rescan.hidden = true;
    const hide = button(msg("pageOcrHide", "Hide text"), () => {
      const off = document.documentElement.classList.toggle(LAYER_OFF_CLASS);
      hide.textContent = off ? msg("pageOcrShow", "Show text") : msg("pageOcrHide", "Hide text");
    });
    const select = button(msg("pageOcrSelect", "Select text"), () => {
      state.selectable = !state.selectable;
      document.documentElement.classList.toggle(SELECTABLE_CLASS, state.selectable);
      select.textContent = state.selectable ? msg("pageOcrSelectOff", "Let clicks through") : msg("pageOcrSelect", "Select text");
      select.setAttribute("aria-pressed", String(state.selectable));
    });
    select.setAttribute("aria-pressed", "false");
    const remove = button(msg("pageOcrRemove", "Remove"), () => toBroker("remove"));

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
  // reader asked for and can see the size of.
  function watchForNewPictures() {
    if (state.mo || typeof MutationObserver === "undefined") return;
    state.mo = new MutationObserver((muts) => {
      let found = 0;
      for (const mu of muts) {
        for (const n of mu.addedNodes) {
          if (n.nodeType !== 1) continue;
          if (n.tagName === "IMG" && !state.ids.has(n)) found++;
          else if (n.querySelectorAll) {
            for (const img of n.querySelectorAll("img")) if (!state.ids.has(img)) found++;
          }
        }
      }
      if (found) { state.pendingNew += found; render(); }
      // A mutation can move a picture without resizing it; one placement frame settles that.
      if (state.listening) schedule();
    });
    state.mo.observe(document.documentElement, { childList: true, subtree: true });
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
    try { delete window.__dhtPageOcr; } catch { window.__dhtPageOcr = undefined; }
  }

  chrome.runtime.onMessage.addListener((m, sender, sendResponse) => {
    if (!m || m.dht !== NS) return;
    switch (m.t) {
      case "collect":
        ensureBar();
        sendResponse({ ok: true, images: collect() });
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
  });
})();

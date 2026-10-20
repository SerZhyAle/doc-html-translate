// In-page readers for the reader-chrome check (ticket 119, slice R1). Every function here is serialized by
// Playwright and run inside the page, so each is self-contained: it reads what the engine laid out, never what the
// stylesheet says. A profile P names the chrome of one edition:
//   P.root      the bar(s) that hold the controls            P.controls  the interactive controls of a bar
//   P.items     non-interactive members of a row (text)      P.panels    panel roots whose controls are not bar rows
// The Go reader (internal/htmlgen) and the extension viewer (extension/src/viewer.*) are described by one profile
// each in reader.mjs; the measurement is the same for both so the two editions are held to one list.

// collect: geometry of every visible control and item, plus the page-level numbers the checks compare.
export function collect(P) {
  const q = (s, r = document) => [...r.querySelectorAll(s)];
  const visible = (e) => {
    const r = e.getBoundingClientRect();
    if (!(r.width > 0 && r.height > 0)) return false;
    for (let n = e; n; n = n.parentElement) {
      const cs = getComputedStyle(n);
      if (cs.display === 'none' || cs.visibility === 'hidden') return false;
    }
    return true;
  };
  const inPanel = (e) => P.panels && e.closest(P.panels);
  const label = (e) => (e.id ? '#' + e.id : (typeof e.className === 'string' && e.className.trim() ? '.' + e.className.trim().split(/\s+/).join('.') : e.tagName.toLowerCase()));
  const out = { coarse: matchMedia('(pointer: coarse)').matches, iw: document.documentElement.clientWidth, ih: window.innerHeight, roots: [], controls: [], items: [], dir: getComputedStyle(document.documentElement).direction };
  for (const root of q(P.root)) {
    if (!visible(root)) continue;
    const r = root.getBoundingClientRect(), cs = getComputedStyle(root);
    out.roots.push({ x: r.x, y: r.y, w: r.width, h: r.height, dir: cs.direction, pl: parseFloat(cs.paddingLeft), pr: parseFloat(cs.paddingRight), bl: parseFloat(cs.borderLeftWidth), br: parseFloat(cs.borderRightWidth), sticky: cs.position === 'sticky', scrollW: root.scrollWidth, clientW: root.clientWidth, pad: cs.padding, rowGap: parseFloat(cs.rowGap), colGap: parseFloat(cs.columnGap) });
  }
  const seen = new Set();
  for (const root of q(P.root)) {
    for (const e of q(P.controls, root)) {
      if (seen.has(e) || !visible(e) || inPanel(e)) continue;
      seen.add(e);
      const r = e.getBoundingClientRect(), cs = getComputedStyle(e), g = e.querySelector('svg');
      const gb = g ? g.getBoundingClientRect() : null;
      const tag = e.tagName.toLowerCase();
      const caption = [...e.childNodes].filter((n) => n.nodeType === 3 || (n.nodeType === 1 && n.tagName.toLowerCase() !== 'svg')).map((n) => n.textContent).join('').trim();
      out.controls.push({
        key: label(e), tag, kind: tag === 'a' ? 'link' : tag, icon: tag !== 'select' && tag !== 'input' && !caption, caption,
        x: r.x, y: r.y, w: r.width, h: r.height,
        radius: parseFloat(cs.borderTopLeftRadius), radiusPct: cs.borderTopLeftRadius.includes('%'),
        bw: parseFloat(cs.borderTopWidth), fs: parseFloat(cs.fontSize), lh: cs.lineHeight,
        glyph: gb ? { w: gb.width, h: gb.height } : null,
        clipX: tag === 'select' || tag === 'input' ? false : e.scrollWidth > e.clientWidth + 1,
        clipY: tag === 'select' || tag === 'input' ? false : e.scrollHeight > e.clientHeight + 1,
        pressed: e.getAttribute('aria-pressed'), expanded: e.getAttribute('aria-expanded'), disabledAria: e.getAttribute('aria-disabled'),
        title: e.getAttribute('title'), ariaLabel: e.getAttribute('aria-label'), href: e.getAttribute('href'), target: e.getAttribute('target'),
        shadow: cs.boxShadow, bg: cs.backgroundColor, color: cs.color, borderColor: cs.borderTopColor,
      });
    }
    for (const e of q(P.items || ':not(*)', root)) {
      if (!visible(e) || inPanel(e) || e.closest(P.controls)) continue;
      const r = e.getBoundingClientRect();
      out.items.push({ key: label(e), x: r.x, y: r.y, w: r.width, h: r.height, glyphOnly: !!e.closest('svg') });
    }
  }
  // natural (max-content) width of each button and link, to tell a content width from a stretched one
  const els = [...seen];
  out.controls.forEach((c, i) => {
    const e = els[i];
    if (c.kind === 'select' || c.kind === 'input') { c.natural = c.w; return; }
    // !important so a stylesheet's own important flex or width cannot hide a stretch from this measurement
    const w0 = e.style.cssText;
    e.style.setProperty('flex', 'none', 'important'); e.style.setProperty('width', 'max-content', 'important'); e.style.setProperty('margin', '0', 'important');
    c.natural = e.getBoundingClientRect().width;
    e.style.cssText = w0;
  });
  out.docScrollW = document.documentElement.scrollWidth;
  return out;
}

// overflowWithoutChrome: the page's scroll width with the chrome taken out, to show the chrome adds none.
export function overflowWithoutChrome(P) {
  const hide = [...document.querySelectorAll(P.root + ', ' + (P.panels || ':not(*)'))];
  const before = hide.map((e) => e.style.display);
  const withChrome = document.documentElement.scrollWidth;
  hide.forEach((e) => { e.style.display = 'none'; });
  const without = document.documentElement.scrollWidth;
  hide.forEach((e, i) => { e.style.display = before[i]; });
  return { withChrome, without };
}

// contrastOf: text and glyph contrast of one control on the face it is actually drawn on, in the state it is in now.
export function contrastOf(arg) {
  const [sel, index] = arg;
  const U = window.__uic;
  const e = [...document.querySelectorAll(sel)].filter((x) => x.getBoundingClientRect().width > 0)[index];
  if (!e) return null;
  const face = U.bgOf(e);
  const ink = U.over(U.fgOf(e), face);
  const path = e.querySelector('svg path');
  let glyph = null;
  if (path) { const f = U.rgba(getComputedStyle(path).fill); if (f) glyph = U.contrast(U.over(f, face), face); }
  const caption = [...e.childNodes].some((n) => (n.nodeType === 3 && n.textContent.trim()) || (n.nodeType === 1 && n.tagName.toLowerCase() !== 'svg' && n.textContent.trim())) || e.tagName === 'SELECT' || e.tagName === 'INPUT';
  const bd = U.rgba(getComputedStyle(e).borderTopColor), side = U.bgOf(e.parentElement || e);
  const border = bd && parseFloat(getComputedStyle(e).borderTopWidth) > 0 ? U.contrast(U.over(bd, side), side) : null;
  return { key: e.id ? '#' + e.id : e.className || e.tagName, border, disabled: e.getAttribute('aria-disabled') === 'true', text: caption ? U.contrast(ink, face) : null, glyph, face: face.map((v) => Math.round(v)).join(','), ink: ink.map((v) => Math.round(v)).join(',') };
}

// focusInfo: what the keyboard focus looks like on the active element, and the face behind the ring.
export function focusInfo() {
  const U = window.__uic, e = document.activeElement;
  if (!e || e === document.body) return null;
  const cs = getComputedStyle(e);
  const ring = U.rgba(cs.outlineColor);
  const face = U.bgOf(e.closest('.dht-navbar, .dht-toolbar, #toolbar') || e);
  return {
    key: e.id ? '#' + e.id : (typeof e.className === 'string' && e.className.trim() ? '.' + e.className.trim().split(/\s+/).join('.') : e.tagName.toLowerCase()), tag: e.tagName.toLowerCase(),
    disabled: e.getAttribute('aria-disabled') === 'true',
    chrome: !!e.closest('.dht-navbar, .dht-toolbar, .dht-search-panel, .dht-contents, #toolbar, #search-panel, #toc'),
    style: cs.outlineStyle, width: parseFloat(cs.outlineWidth), offset: parseFloat(cs.outlineOffset), contrast: ring ? U.contrast(U.over(ring, face), face) : null,
  };
}

// panelBox: a panel against the bar it hangs under.
export function panelBox(arg) {
  const [panelSel, barSel] = arg;
  const p = document.querySelector(panelSel), b = document.querySelector(barSel);
  if (!p) return null;
  const pr = p.getBoundingClientRect(), br = b ? b.getBoundingClientRect() : null, cs = getComputedStyle(p);
  return { visible: !p.hidden && pr.width > 0, top: pr.top, left: pr.left, right: pr.right, bottom: pr.bottom, barBottom: br ? br.bottom : null, barTop: br ? br.top : null, iw: window.visualViewport ? window.visualViewport.width : document.documentElement.clientWidth, ih: window.innerHeight, position: cs.position };
}

// markerOf: the external-navigation marker painted after a link.
export function markerOf(sel) {
  const e = document.querySelector(sel);
  if (!e) return null;
  const a = getComputedStyle(e, '::after'), r = e.getBoundingClientRect();
  return { content: a.content, w: parseFloat(a.width), h: parseFloat(a.height), mask: (a.maskImage || a.webkitMaskImage || 'none').slice(0, 60), bg: a.backgroundColor, transform: a.transform, linkH: r.height };
}

// centers: the viewport centre of every visible control of a selector, in document order (the mouse goes there to
// read the hover face). Same filter as contrastOf, so index i names the same control in both.
export function centers(sel) {
  return [...document.querySelectorAll(sel)].filter((x) => x.getBoundingClientRect().width > 0).map((e) => {
    const r = e.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2, disabled: e.getAttribute('aria-disabled') === 'true' };
  });
}

// pressedProbe: the face and shadow of a toggle before and after it is switched on, then restored.
export function pressedProbe(arg) {
  const [sel, attr] = arg;
  const e = document.querySelector(sel);
  if (!e) return null;
  const was = e.getAttribute(attr);
  const read = () => { const cs = getComputedStyle(e); return { shadow: cs.boxShadow, bg: cs.backgroundColor, color: cs.color, border: cs.borderTopColor }; };
  e.setAttribute(attr, 'false');
  const before = read();
  e.setAttribute(attr, 'true');
  const after = read();
  if (was === null) e.removeAttribute(attr); else e.setAttribute(attr, was);
  return { before, after };
}

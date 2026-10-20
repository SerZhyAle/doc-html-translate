package htmlgen

// chromeUICSS is the command layer of the generated reader chrome: the size tokens (UI-SIZING 1.1)
// and the button, group and state rules (BUTTONS 2.0) of the chapter bar, the single-page bar, the
// index toolbar and the two panels. It sits inside readerCSS's <style>, which every page carries, so
// the index toolbar (no navbar CSS) gets it too. The extension's viewer holds the same numbers in
// viewer.css; docs/UI_PROFILE.md "Reader chrome" is the record both follow.
//
// Colours are the reading theme's own roles (--dht-*, internal/appearance) or system colours inside
// forced-colors; no palette value is declared here (tests/appearance_parity_test.go).
//
// A var, not a const, because the external-link marker is the vendored drawing as a CSS mask
// (glyphMaskURL), built from the same table the tests hold to assets/glyphs.
var chromeUICSS = `
  /* UI-SIZING 1.1 / BUTTONS 2.0 section 3: ordinary targets are 32 CSS px for a fine pointer and 48
     under @media (pointer: coarse) - chosen by the primary pointer, never per event. --dht-ui is the
     interface scale the chrome would take from a later size selector; clamp() keeps an unknown value
     from producing a zero-size target (the supported range is 0.875 .. 2, the smallest target
     28 / 42). Every length of the chrome below is one of these tokens. */
  :root { --dht-ui: 1; }
  .dht-navbar, .dht-toolbar, .dht-contents {
    --dht-s: clamp(0.875, var(--dht-ui), 2);
    --dht-tgt: calc(32px * var(--dht-s));
    --dht-pad: calc(12px * var(--dht-s));
    --dht-glyph: calc(16px * var(--dht-s));
    --dht-gap: calc(8px * var(--dht-s));
    --dht-radius: calc(4px * var(--dht-s));
    --dht-fs: calc(14px * var(--dht-s));
    --dht-fs-note: calc(12px * var(--dht-s));
  }
  /* A pinned line height: the bar must not grow and shrink with the reader's line-spacing choice or the
     book's own CSS. (The contents panel sets its own, in singleContentsCSS.) */
  .dht-navbar, .dht-toolbar { line-height: 1.35; }
  @media (pointer: coarse) {
    .dht-navbar, .dht-toolbar, .dht-contents {
      --dht-tgt: calc(48px * var(--dht-s));
      --dht-pad: calc(16px * var(--dht-s));
      --dht-glyph: calc(20px * var(--dht-s));
    }
  }
  .dht-glyph { width: var(--dht-glyph); height: var(--dht-glyph); flex: none; }
  /* ICON-RENDER section 13 C: document paging mirrors; transport glyphs stay fixed. */
  .dht-navbar[dir="rtl"] .nav-turn .dht-glyph { transform: scaleX(-1); }

  /* One command button: a rectangle with the baseline corner, a transparent (secondary) face with the
     theme's border, the glyph before the caption, content width plus padding - never stretched. A
     whole button wraps to the next row before its caption does; only a button that cannot fit alone
     wraps its caption (overflow-wrap) and grows. Buttons, button-styled links (navigation keeps link
     semantics, BUTTONS 2.5) and the panels' Close share it. */
  .dht-btn, .dht-navbar a.dht-nav-link, .dht-navbar a.disabled, .dht-toolbar a.dht-continue,
  #dht-search-close, .dht-contents-head button {
    display: inline-flex; align-items: center; justify-content: center; gap: var(--dht-gap);
    box-sizing: border-box; min-width: var(--dht-tgt); min-height: var(--dht-tgt); max-width: 100%;
    padding: calc(4px * var(--dht-s)) var(--dht-pad);
    background: transparent; color: var(--dht-bar-fg);
    border: 1px solid var(--dht-border); border-radius: var(--dht-radius);
    font: inherit; font-size: var(--dht-fs); line-height: 1.35; text-align: center;
    text-decoration: none; overflow-wrap: anywhere; cursor: pointer;
  }
  /* A glyph-only button is the square target itself: no padding, the glyph centred. */
  .dht-ico { padding: 0; }
  .dht-btn[hidden], .dht-group[hidden] { display: none; }
  /* APP-STYLE section 5: the hover tint stays outside the palette on purpose - a neutral grey that
     darkens a light face and lightens a dark one, so one value reads on all four reading themes while
     the ink keeps its themed colour. */
  .dht-btn:hover, .dht-navbar a.dht-nav-link:hover, .dht-contents-head button:hover, #dht-search-close:hover {
    border-color: var(--dht-accent); background: rgba(127, 127, 127, 0.14);
  }
  /* Selected (a toggle that is on, a disclosure that is open): a distinct face, the accent border and
     an inset bar along the block end in the ink colour - a state a reader who cannot tell the hues
     apart still sees (BUTTONS 3 rule 5). The bar is an inset shadow, so no state moves a neighbour. */
  .dht-btn[aria-pressed="true"], .dht-btn[aria-expanded="true"] {
    background: var(--dht-border); border-color: var(--dht-accent);
    box-shadow: inset 0 calc(-3px * var(--dht-s)) 0 currentColor;
  }
  /* A disabled end of the book keeps its place, border and glyph, muted together. */
  .dht-navbar a.disabled { color: var(--dht-muted); pointer-events: none; cursor: default; }
  /* Primary rank: the one accent button of the index toolbar. The ink is the theme's own paper colour,
     not white: white passes 4.5:1 on the light and sepia accents but lands at 3.1:1 on the dark and
     night accents, while --dht-bg clears it on all four (5.0 / 5.0 / 5.5 / 6.5) - ticket 57. */
  .dht-toolbar a.dht-continue { display: none; background: var(--dht-accent); color: var(--dht-bg); border-color: var(--dht-accent); }
  .dht-toolbar a.dht-continue:hover { filter: brightness(1.08); }

  /* Native editors keep their platform arrow and popup but take the target height; they are not
     buttons (BUTTONS section 6). */
  .dht-navbar select, .dht-toolbar select, .dht-search-panel select {
    box-sizing: border-box; min-width: var(--dht-tgt); min-height: var(--dht-tgt);
    padding: 0 var(--dht-gap); background: transparent; color: var(--dht-bar-fg);
    border: 1px solid var(--dht-border); border-radius: var(--dht-radius);
    font: inherit; font-size: var(--dht-fs); cursor: pointer;
  }
  .dht-navbar select:hover, .dht-toolbar select:hover, .dht-search-panel select:hover { border-color: var(--dht-accent); }
  /* The select's own face is transparent, so its popup list falls back to the system's face under the theme's ink -
     light text on white in the dark themes. The options carry the bar's face and ink instead. */
  .dht-navbar select option, .dht-toolbar select option, .dht-search-panel select option { background: var(--dht-bar-bg); color: var(--dht-bar-fg); }
  .dht-sel { display: inline-flex; align-items: stretch; gap: var(--dht-gap); }
  .dht-sel > .dht-glyph { align-self: center; }

  /* Command rows flow from the reading start (BUTTONS 4 rule 2): a group is a wrapping flex row, so a
     small group moves as one unit when it fits a line and wraps its own buttons when it is wider than
     the line. Every button of one visual line takes that line's height (align-items: stretch), widths
     stay content-based, spare space stays empty, the gap between buttons and wrapped rows is one token. */
  .dht-group, .nav-turn { display: flex; flex-wrap: wrap; align-items: stretch; gap: var(--dht-gap); min-width: 0; max-width: 100%; }
  /* The bar's own children are flex items; the actions wrapper only exists for the paging script. */
  .dht-navbar .nav-actions { display: contents; }
  .dht-navbar .nav-file, .dht-navbar .nav-title, .dht-navbar .nav-info { align-self: center; }
  .dht-navbar .nav-info { font-size: var(--dht-fs-note); color: var(--dht-muted); }
  .dht-toolbar { display: flex; flex-wrap: wrap; align-items: stretch; gap: var(--dht-gap); margin: 0 0 1.2em; font-family: "Segoe UI", system-ui, Arial, sans-serif; font-size: var(--dht-fs); }

  /* The version link opens the project site: an external destination, so it carries the vendored
     nav.open-external marker (BUTTONS 2 rule 3; rtl: mirror). It is a text link, not a button, with
     the target height of the row it sits in. */
  .dht-navbar a.nav-version {
    display: inline-flex; align-items: center; gap: var(--dht-gap); box-sizing: border-box;
    min-height: var(--dht-tgt); padding: 0 var(--dht-gap); border-radius: var(--dht-radius);
    font-size: var(--dht-fs-note); color: var(--dht-muted); text-decoration: none;
  }
  .dht-navbar a.nav-version:hover { color: var(--dht-bar-fg); background: rgba(127, 127, 127, 0.14); }
  .dht-navbar a.nav-version::after {
    content: ""; flex: none; width: var(--dht-glyph); height: var(--dht-glyph); background-color: currentColor;
    -webkit-mask: ` + glyphMaskURL("nav.open-external") + ` center / contain no-repeat; mask: ` + glyphMaskURL("nav.open-external") + ` center / contain no-repeat;
  }
  .dht-navbar[dir="rtl"] a.nav-version::after { transform: scaleX(-1); }

  /* Forced colours and more contrast: the reader's own palette is replaced by the system's, so what a
     colour carried must survive in system colours - the selected face, the progress fill and the
     markers drawn as masks (a mask is painted with the background colour). */
  @media (forced-colors: active) {
    .dht-btn[aria-pressed="true"], .dht-btn[aria-expanded="true"] { background: Highlight; color: HighlightText; border-color: Highlight; }
    .dht-navbar a.disabled { color: GrayText; }
    .dht-progress { background: Highlight; }
    .dht-navbar a.nav-version::after { background-color: LinkText; }
  }
  @media (prefers-contrast: more) {
    .dht-btn, .dht-navbar a.dht-nav-link, .dht-navbar a.disabled, .dht-toolbar a.dht-continue,
    .dht-navbar select, .dht-toolbar select, .dht-search-panel select, #dht-search-close, .dht-contents-head button { border-color: currentColor; }
  }
`

// ui-boot.js - the first lines an extension page runs, as a classic script in <head> so they run before
// the first paint (a module is deferred until the document is parsed).
//
// 1. It resolves the automatic input profile ONCE, from the primary pointer: a coarse pointer, or no
//    pointer at all, is a touch surface (UI-SIZING 2 rule 1). The answer is kept on the root as
//    data-input-auto and never recomputed, so a mouse movement or a touch can never reflow a page.
// 2. It reveals the page itself after a moment if the module that applies the saved settings never ran
//    (a failed import, a storage error): a hidden page forever is worse than a flash of defaults.
(() => {
  const root = document.documentElement;
  let touch = false;
  try {
    touch = matchMedia("(pointer: coarse)").matches || matchMedia("(any-pointer: none)").matches;
  } catch { /* no media queries: treat it as a pointer surface */ }
  root.dataset.inputAuto = touch ? "touch" : "pointer";
  root.dataset.input = root.dataset.inputAuto;
  setTimeout(() => root.removeAttribute("data-boot"), 2500);
})();

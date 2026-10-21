// missing-image.js - what the viewer shows in place of a document picture it cannot show.
//
// ICON-EXTERNAL section 13 F: the pictures inside a document the user opened are local user content,
// and one that is missing or broken shows the vocabulary's `content.image` with the source's alt text -
// never the browser's broken-image icon and never a blank box. The alt text is the document's own
// words, so no string of this product is involved. The desktop reader pages do the same from their
// head (internal/htmlgen/missingimage.go); docs/PARITY.md "Missing document pictures".

import { glyph } from "./glyphs.js";

export const MISSING_IMG_CLASS = "missing-img";

// failed <img> -> the placeholder that took its place. The OCR queue puts a picture back where it
// found it when recognition fails; without this it would bring the broken picture back over the
// placeholder.
const placeholders = new WeakMap();

// missingImagePlaceholder builds the stand-in: the glyph, then the alt text when the source gave
// one. An empty alt marks a decorative picture, so its stand-in is the glyph alone.
export function missingImagePlaceholder(alt) {
  const box = document.createElement("span");
  box.className = MISSING_IMG_CLASS;
  box.append(glyph("content.image"));
  const text = String(alt == null ? "" : alt).trim();
  if (text) {
    const label = document.createElement("span");
    label.textContent = text;
    box.append(label);
  }
  return box;
}

// showMissingImage swaps a picture that cannot be shown for its placeholder, in place. The picture's
// src is never touched, so nothing is requested again. Returns the placeholder, or null when the
// picture is not in a tree.
export function showMissingImage(img) {
  if (!img.parentNode) return null;
  const box = missingImagePlaceholder(img.getAttribute("alt"));
  placeholders.set(img, box);
  img.replaceWith(box);
  return box;
}

// missingImageFor returns the placeholder that replaced img, or null.
export function missingImageFor(img) {
  return placeholders.get(img) || null;
}

// installMissingImageFallback listens for a picture's load error. An error event does not bubble, so
// only a capture-phase listener on the document sees every picture, including those inserted later.
export function installMissingImageFallback(doc = document) {
  doc.addEventListener("error", (event) => {
    const img = event.target;
    if (img && img.localName === "img") showMissingImage(img);
  }, true);
}

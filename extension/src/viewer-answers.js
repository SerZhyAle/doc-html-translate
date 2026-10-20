// viewer-answers.js - the answer buttons of the viewer's notices, offers and export card (BUTTONS 2.0, ticket 119).
//
// One builder and one arrangement for every surface that asks the reader something: the export card, the
// remote-images notice, the resume offer, the error and empty-state notices and the password prompt. The look
// (the neutral face, the primary rank, the targets) is viewer.css; this file only decides the structure that
// look needs: a glyph before a caption, a row that wraps whole buttons, and at most one primary answer.

import { glyph } from "./glyphs.js";

// answerButton builds one answer: the vocabulary glyph first when there is one, then the caption in its own
// span, so a caption read back (a test, a screen reader) is the caption alone. `rank` is "primary" or empty:
// the neutral face is the default, and only the answer the surface recommends is ever accented.
export function answerButton(caption, { glyphId = null, rank = "" } = {}) {
  const button = document.createElement("button");
  button.type = "button";
  if (rank) button.className = rank;
  if (glyphId) button.append(glyph(glyphId));
  const text = document.createElement("span");
  text.textContent = caption;
  button.append(text);
  return button;
}

// answerRow lays answers out as a row that flows from the reading start and wraps whole buttons, every button
// of one visual line taking the line's height (viewer.css). The order given is the reading order: the
// dismissing answer first, any helping action next, the final affirmative last (BUTTONS 4 rule 4).
export function answerRow(buttons, className = "notice-actions") {
  const row = document.createElement("div");
  row.className = className;
  row.append(...buttons);
  return row;
}

// arrangeNotice gathers the buttons a notice was built with into one row and accents the first of them. The
// notices are assembled by many call sites that each append a button or two after their text; the first is the
// recovery the notice offers, and a second one, when there is one, is the ordinary alternative - never two
// accent faces on one surface (BUTTONS 2 rule 2). A button inside a form row (the password prompt) keeps its
// row and is still the one primary.
export function arrangeNotice(box) {
  const loose = [...box.children].filter((node) => node.matches("button"));
  if (loose.length) box.append(answerRow(loose));
  const first = box.querySelector("button");
  if (first) first.classList.add("primary");
}

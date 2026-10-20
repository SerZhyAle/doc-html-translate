// The extension has one set of interface numbers (UI-SIZING 1.1, BUTTONS 2.0 section 3): the options page, the
// popup and the image-OCR page read them from ui-sizing.css; the viewer's toolbar carries its own scoped copy in
// viewer.css (written by slice R1 and left alone by slice X1); the generated reader and the launcher have theirs.
// This test pins the first two to each other, token by token, so the two cannot drift apart unnoticed.

import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const src = path.join(import.meta.dirname, "..", "src");
const tokens = fs.readFileSync(path.join(src, "ui-sizing.css"), "utf8");
const viewer = fs.readFileSync(path.join(src, "viewer.css"), "utf8");

// The first pixel literal of each `--name: calc(Npx ..)` declaration inside a block.
function pixelTokens(block) {
  return Object.fromEntries([...block.matchAll(/--([a-z-]+):\s*calc\((\d+)px/g)].map((m) => [m[1], Number(m[2])]));
}

// The viewer's token scope: the toolbar, the panels and - since slice X1 - the notices, the offers, the export card and the status bar.
const VIEWER_TOKEN_SCOPE = "#toolbar, #toc, #search-panel, #status, .notice, .remote-notice, .resume-notice, .export-dialog {";

// The text of the first rule whose selector line starts with `selector`.
function rule(css, selector) {
  const i = css.indexOf(selector);
  assert.ok(i >= 0, `rule ${selector} not found`);
  return css.slice(css.indexOf("{", i) + 1, css.indexOf("}", i));
}

test("the token block of ui-sizing.css has the viewer's ordinary pointer numbers", () => {
  const mine = pixelTokens(rule(tokens, "html {"));
  const theirs = pixelTokens(rule(viewer, VIEWER_TOKEN_SCOPE));
  for (const name of ["tgt", "pad", "glyph", "gap", "radius", "fs", "fs-note"]) {
    assert.ok(name in theirs, `viewer.css lost --${name}`);
    assert.equal(mine[name], theirs[name], `--${name}: ui-sizing.css ${mine[name]}, viewer.css ${theirs[name]}`);
  }
});

test("the touch profile of ui-sizing.css has the viewer's coarse-pointer numbers", () => {
  const mine = pixelTokens(rule(tokens, 'html[data-input="touch"]'));
  const coarse = viewer.slice(viewer.indexOf("@media (pointer: coarse)"));
  const theirs = pixelTokens(rule(coarse, VIEWER_TOKEN_SCOPE));
  for (const name of ["tgt", "pad", "glyph"]) assert.equal(mine[name], theirs[name], `touch --${name}`);
});

test("the scale is clamped to the declared range, and the declared floors follow from it", () => {
  assert.match(tokens, /--s:\s*clamp\(0\.875, var\(--ui\), 2\)/, "interface scale clamp");
  assert.match(viewer, /--s:\s*clamp\(0\.875, var\(--ui\), 2\)/, "the viewer clamps the same way");
  assert.match(tokens, /--t:\s*clamp\(1, var\(--tx\), 2\)/, "text enlargement clamp");
  // Smallest declared targets (docs/UI_PROFILE.md): 28 pointer and 42 touch, at the compact scale.
  const base = pixelTokens(rule(tokens, "html {")).tgt;
  const touch = pixelTokens(rule(tokens, 'html[data-input="touch"]')).tgt;
  assert.equal(base * 0.875, 28);
  assert.equal(touch * 0.875, 42);
  for (const [attr, ui] of [["compact", 0.875], ["large", 1.25], ["xlarge", 1.5], ["huge", 2]]) {
    assert.match(tokens, new RegExp(`html\\[data-size="${attr}"\\]\\s*\\{\\s*--ui:\\s*${ui.toString().replace(".", "\\.")};`), `size ${attr}`);
  }
});

test("every length of the shared components is a token or a scaled literal, never a bare pixel", () => {
  // A bare px in ui-components.css would not follow the interface size: only 0, 1px hairlines and
  // calc(.. * var(--s)) forms are allowed outside the shadow and dialog-edge rules listed here.
  const css = fs.readFileSync(path.join(src, "ui-components.css"), "utf8")
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/url\("[^"]*"\)/g, "url()");
  const bare = [];
  for (const line of css.split("\n")) {
    if (/box-shadow|max-width: min\(calc|min-width: min\(calc|inset-block-start|100cqw|100vw/.test(line)) continue;
    for (const m of line.matchAll(/(?<![\w.(*-])(\d+)px/g)) {
      const around = line.slice(Math.max(0, m.index - 6), m.index + m[0].length + 14);
      if (/calc\(|\* var\(--s\)|var\(--s\) \*/.test(line) || m[1] === "1" || m[1] === "2") continue;
      bare.push(`${m[0]} in: ${around.trim()}`);
    }
  }
  assert.deepEqual(bare, []);
});

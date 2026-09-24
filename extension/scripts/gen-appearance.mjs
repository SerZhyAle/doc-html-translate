// gen-appearance.mjs - write the extension's copy of the shared appearance (the OCR overlay unit
// and the reader theme palette) from ../internal/appearance/appearance.json, the one place those
// values are edited. The desktop app builds its copy from the same file at run time
// (internal/appearance/appearance.go), and tests/appearance_parity_test.go compares the two.
//
//   node scripts/gen-appearance.mjs          rewrite the generated regions in place
//   node scripts/gen-appearance.mjs --check  exit 1 if a region is stale, write nothing
//
// Generated content lives between marker comments inside the stylesheets that already ship, not in
// a file of its own: the manifest, ocr.html, the page-OCR injection and the viewer's export path
// all name these two files, and a third would have to be added to each of them for nothing. A
// missing marker pair is an error in both modes - a region that cannot be found is not fresh.
//
// Names are per-edition and live here; the source carries declarations only. Each rule is
// preceded by a role or theme comment, which is what the parity gate keys on instead of selectors.

import { readFile, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const extRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const SOURCE = join(extRoot, "..", "internal", "appearance", "appearance.json");

export const BEGIN = "/* >>> generated from internal/appearance - do not edit by hand */";
export const END = "/* <<< generated */";

// Same text as the Go side's HiddenPlateDecl: the toggle's meaning, not a styling choice.
const HIDDEN_PLATE = { property: "display", value: "none" };

const OVERLAY_NAMES = {
  container: ".ocr-overlay",
  image: ".ocr-overlay-img",
  plate: ".ocr-plate",
  hiddenPlate: "html.ocr-layer-off .ocr-plate",
};
const PALETTE_NAMES = { prefix: "", attr: "data-theme" };

const WIDTH = 100;

// wrapComment renders a note as a block comment wrapped at WIDTH, continuation lines aligned
// under the first word the way the hand-written comments in these files are.
function wrapComment(text, indent) {
  const lines = [];
  let line = `${indent}/* `;
  const pad = `${indent}   `;
  for (const word of text.split(/\s+/)) {
    if (line.length + word.length + 1 > WIDTH && line.trim() !== "/*") {
      lines.push(line.trimEnd());
      line = pad;
    }
    line += `${word} `;
  }
  lines.push(`${line.trimEnd()} */`);
  return lines.join("\n");
}

function rule(label, selector, decls) {
  const body = decls.map((d) =>
    (d.note ? `${wrapComment(d.note, "  ")}\n` : "") + `  ${d.property}: ${d.value};`);
  return `/* ${label} */\n${selector} {\n${body.join("\n")}\n}`;
}

export const tokenProperty = (token) => token.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`);

export function overlayCSS(src, names = OVERLAY_NAMES) {
  return [
    rule("role: container", names.container, src.roles.container),
    rule("role: image", names.image, src.roles.image),
    rule("role: plate", names.plate, src.roles.plate),
    rule("role: hidden-plate", names.hiddenPlate, [HIDDEN_PLATE]),
  ].join("\n\n");
}

export function paletteCSS(src, names = PALETTE_NAMES) {
  return Object.entries(src.themes).map(([theme, colors], i) => rule(
    `theme: ${theme}`,
    i === 0 ? ":root" : `html[${names.attr}="${theme}"]`,
    Object.entries(colors).map(([token, value]) => ({ property: `--${names.prefix}${tokenProperty(token)}`, value })),
  )).join("\n\n");
}

// replaceRegion swaps the content between the one marker pair in css for body. Throws when the
// pair is missing, duplicated or out of order.
export function replaceRegion(css, body, file) {
  const b = css.indexOf(BEGIN);
  const e = css.indexOf(END);
  if (b < 0 || e < 0 || e < b || css.indexOf(BEGIN, b + 1) >= 0 || css.indexOf(END, e + 1) >= 0) {
    throw new Error(`${file}: expected exactly one "${BEGIN}" ... "${END}" pair`);
  }
  return `${css.slice(0, b + BEGIN.length)}\n${body}\n${css.slice(e)}`;
}

export const TARGETS = [
  { file: "src/ocr-overlay.css", build: overlayCSS },
  { file: "src/viewer.css", build: paletteCSS },
];

async function main() {
  const check = process.argv.includes("--check");
  const src = JSON.parse(await readFile(SOURCE, "utf8"));
  let stale = 0;
  for (const t of TARGETS) {
    const path = join(extRoot, t.file);
    const before = (await readFile(path, "utf8")).replace(/\r\n/g, "\n");
    const after = replaceRegion(before, t.build(src), t.file);
    if (after === before) continue;
    if (check) {
      console.error(`${t.file}: generated region is stale - run \`npm run appearance\``);
      stale++;
    } else {
      await writeFile(path, after);
      console.log(`${t.file}: regenerated from internal/appearance`);
    }
  }
  if (stale) process.exit(1);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  try {
    await main();
  } catch (err) {
    console.error(err.message);
    process.exit(1);
  }
}

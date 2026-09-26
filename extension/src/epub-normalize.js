// epub-normalize.js - the parts of the desktop app's internal/epub/normalize.go the viewer needs
// before a chapter reaches the DOM: XHTML markup rewritten so the HTML parser reads it as an XML
// parser would, and the rule for which SVG is a cover wrapper. They live apart from epub.js so the
// markup pass is testable without a DOM, and both run the shared fixture
// tests/testdata/content_fidelity_cases.json with internal/epub (docs/PARITY.md, "EPUB and HTML
// content fidelity").

// HTML elements that never have an end tag, so their XML self-closing form already means the same
// thing to the HTML parser. Mirrors htmlVoidElements in normalize.go.
const VOID = new Set([
  "area", "base", "br", "col", "embed", "hr", "img", "input", "keygen", "link", "meta",
  "param", "source", "track", "wbr",
]);

// Elements whose content the tokenizer reads as raw text up to the matching end tag, as
// golang.org/x/net/html's tokenizer does - a "<b/>" inside a <script> is not a tag.
const RAW_TEXT = new Set(["iframe", "noembed", "noframes", "noscript", "plaintext", "script", "style", "textarea", "title", "xmp"]);

// A start tag: name, then attributes where a quoted value may hold ">" or "/".
const START_TAG = /<([A-Za-z][^\t\n\f\r />]*)((?:[^>"']|"[^"]*"|'[^']*')*?)(\/?)>/y;
const END_TAG = /<\/([A-Za-z][^\t\n\f\r />]*)[^>]*>/y;

function skipTo(src, from, marker) {
  const at = src.indexOf(marker, from);
  return at < 0 ? src.length : at + marker.length;
}

// xhtmlToHtmlSyntax is the port of xhtmlToHTMLSyntax: an HTML parser ignores the "/" of
// <a id="x"/> or <script src=".."/>, leaves the element open and swallows the rest of the chapter,
// so such non-void elements become a start and an end tag. Inside <svg> and <math> the HTML parser
// honours self-closing tags, so foreign content passes through, as does every other byte. The XML
// declaration is dropped: HTML would keep it as a bogus comment.
export function xhtmlToHtmlSyntax(src) {
  let out = "";
  let i = 0;
  let foreign = 0;
  while (i < src.length) {
    const lt = src.indexOf("<", i);
    if (lt < 0) {
      out += src.slice(i);
      break;
    }
    out += src.slice(i, lt);
    i = lt;
    if (src.startsWith("<!--", i)) {
      const end = skipTo(src, i + 4, "-->");
      out += src.slice(i, end);
      i = end;
      continue;
    }
    if (src.startsWith("<?", i) || src.startsWith("<!", i)) {
      const end = skipTo(src, i, ">");
      const raw = src.slice(i, end);
      if (!/^<\?xml[\t\n\f\r ]/.test(raw)) out += raw;
      i = end;
      continue;
    }
    if (src[i + 1] === "/") {
      END_TAG.lastIndex = i;
      const m = END_TAG.exec(src);
      if (!m) {
        out += "<";
        i++;
        continue;
      }
      const name = m[1].toLowerCase();
      if ((name === "svg" || name === "math") && foreign > 0) foreign--;
      out += m[0];
      i += m[0].length;
      continue;
    }
    START_TAG.lastIndex = i;
    const m = START_TAG.exec(src);
    if (!m) {
      out += "<";
      i++;
      continue;
    }
    const name = m[1].toLowerCase();
    i += m[0].length;
    if (m[3] === "/") {
      if (foreign > 0 || VOID.has(name)) out += m[0];
      else out += `${m[0].slice(0, -2).replace(/[\t\n\f\r ]+$/, "")}></${name}>`;
      continue;
    }
    out += m[0];
    if (name === "svg" || name === "math") foreign++;
    if (RAW_TEXT.has(name)) {
      const close = new RegExp(`</${name}[\\t\\n\\f\\r />]`, "ig");
      close.lastIndex = i;
      const e = close.exec(src);
      const stop = e ? e.index : src.length;
      out += src.slice(i, stop);
      i = stop;
    }
  }
  return out;
}

const SVG_IGNORED = new Set(["title", "desc", "defs", "metadata"]);

// coverSvgImage returns the one <image> of svg when that is all it draws - whitespace, comments
// and the non-rendered title, desc, defs and metadata aside - or null. Only such an SVG is a
// cover wrapper; one with text, a second picture or any shape is an illustration and stays whole.
// Mirrors singleImageHref in normalize.go.
export function coverSvgImage(svg) {
  let image = null;
  for (const c of Array.from(svg.childNodes)) {
    if (c.nodeType === 3) {
      if (c.textContent.trim() !== "") return null;
    } else if (c.nodeType === 1) {
      const tag = c.localName.toLowerCase();
      if (SVG_IGNORED.has(tag)) continue;
      if (tag !== "image" || image) return null;
      image = c;
    }
  }
  return image;
}

// outermostSvg is the <svg> the desktop app would judge: its walk stops at the first <svg> it
// meets, so an SVG nested in another is never a cover wrapper of its own.
export function outermostSvg(el) {
  let svg = el.closest("svg");
  for (let up = svg && svg.parentElement ? svg.parentElement.closest("svg") : null; up;
    up = up.parentElement ? up.parentElement.closest("svg") : null) {
    svg = up;
  }
  return svg;
}

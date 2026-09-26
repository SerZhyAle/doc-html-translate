// url-policy.js - what a document's own markup may point at, shared by the generic sanitizer
// (sanitize.js: HTML, Markdown, MOBI) and the EPUB chapter renderer (epub.js).
//
// Two questions, answered once for both paths:
//   - links (a/area href): only schemes that navigate somewhere a reader expects. A script
//     scheme is inert inside the viewer only because of the extension's content policy; the
//     "save as HTML" export has no such policy, so the link itself has to go.
//   - resources (src, srcset, poster, background, SVG href): local data only. A remote URL is
//     a request the document makes on its own - a tracking pixel, an intranet GET - so it is
//     parked under a data- attribute until the reader allows remote content, and anything
//     that is neither local nor remote is dropped.
// Names are removed and ids namespaced here too, because both paths merge every section into
// one viewer document where a `name` can shadow a document method (DOM clobbering).

const LINK_SCHEMES = new Set(["http", "https", "mailto", "tel"]);
const LOCAL_RESOURCE_SCHEMES = new Set(["blob", "data"]);
const REMOTE_SCHEMES = new Set(["http", "https"]);
const RESOURCE_ATTRS = new Set(["src", "srcset", "poster", "background", "lowsrc", "dynsrc"]);
// Never meaningful in a reader, and each one either sends a request or submits somewhere.
const DROP_ATTRS = new Set(["ping", "action", "formaction", "srcdoc", "xml:base"]);
const LINK_TAGS = new Set(["a", "area"]);
// Attributes whose value is one id, or a space-separated list of ids, in the same document. The
// ids they name are namespaced, so the references are too - or a label loses its control, a
// table cell its headers, and a screen reader every aria relationship.
const IDREF_ATTRS = new Set([
  "for", "headers", "list", "form", "popovertarget", "commandfor",
  "aria-activedescendant", "aria-controls", "aria-describedby", "aria-details",
  "aria-errormessage", "aria-flowto", "aria-labelledby", "aria-owns",
]);
// url(#id) in a presentation attribute: SVG fill, stroke, clip-path, mask, filter, marker-*.
// Inline styles are dropped, so attributes are the only place one survives.
const URL_FRAGMENT_REF = /url\(\s*(['"]?)#([^'")\s]+)\1\s*\)/gi;

// SVG animation can rewrite an href to a script URL after sanitizing, so it is dropped with
// the other unsafe tags; a reader loses nothing by it. SVG names are listed in their own case:
// a type selector matches a foreign (SVG) element case-sensitively.
export const DROP_TAGS =
  "script,style,link,base,meta,title,noscript,iframe,frame,frameset,object,embed,applet,form," +
  "animate,set,animateMotion,animateTransform,animatemotion,animatetransform";

export const REMOTE_ATTR_PREFIX = "data-dht-remote-";
export const REMOTE_MARK = "data-dht-remote";

// urlKind classifies an attribute value: "fragment" (#x), "relative" (no scheme),
// "protocol-relative" (//host), or the lower-case scheme. Whitespace and control characters
// are stripped first because the URL parser ignores them - "java\tscript:" is a script URL.
export function urlKind(value) {
  const s = String(value == null ? "" : value).replace(/[\u0000- \u007f]+/g, "");
  if (s.startsWith("#")) return "fragment";
  if (s.startsWith("//") || s.startsWith("\\\\")) return "protocol-relative";
  const m = /^([a-z][a-z0-9+.-]*):/i.exec(s);
  return m ? m[1].toLowerCase() : "relative";
}

export function isSafeLinkHref(value) {
  const k = urlKind(value);
  return k === "fragment" || k === "relative" || LINK_SCHEMES.has(k);
}

// resourceVerdict: "keep", "remote" (park until allowed) or "drop".
export function resourceVerdict(value) {
  const k = urlKind(value);
  if (k === "relative" || k === "fragment" || LOCAL_RESOURCE_SCHEMES.has(k)) return "keep";
  if (k === "protocol-relative" || REMOTE_SCHEMES.has(k)) return "remote";
  return "drop";
}

// srcset is a list; the worst candidate decides the whole attribute.
function srcsetVerdict(value) {
  let worst = "keep";
  for (const cand of String(value).split(",")) {
    const url = cand.trim().split(/\s+/)[0];
    if (!url) continue;
    const v = resourceVerdict(url);
    if (v === "drop") return "drop";
    if (v === "remote") worst = "remote";
  }
  return worst;
}

function parkName(attrName) {
  // xlink:href parks as href: SVG 2 honours a plain href, so the restore needs no namespace.
  return REMOTE_ATTR_PREFIX + (attrName === "xlink:href" ? "href" : attrName);
}

// scrubTree strips everything a document must not carry into the viewer, in place:
//   - on* handlers and inline styles;
//   - link hrefs with a disallowed scheme (the element stays, as inert text);
//   - resource URLs per resourceVerdict, remote ones parked and the element marked;
//   - name attributes (an anchor's name becomes its id first, so "#name" links still land);
//     an image map's name is the one kept, namespaced like an id, because usemap finds a map
//     by name and a map's name is not exposed on window or document, so it cannot clobber;
//   - ids namespaced as d<index>-<id>, together with every same-section reference to one:
//     "#x" hrefs (links, SVG <use>/<image>), url(#x), usemap and the IDREF_ATTRS. EPUB
//     retargets its own <a> links first and passes rewriteFragments false, which leaves
//     only those alone.
// Returns the number of elements with parked remote content.
export function scrubTree(root, index, { rewriteFragments = true } = {}) {
  let remote = 0;
  const ns = (id) => `d${index}-${id}`;
  for (const e of Array.from(root.querySelectorAll("*"))) {
    const tag = e.localName.toLowerCase();
    const isLink = LINK_TAGS.has(tag);
    let parked = false;
    const nm = e.getAttribute("name");
    if (nm && tag === "a" && !e.id) e.id = nm;
    for (const attr of Array.from(e.attributes)) {
      const n = attr.name.toLowerCase();
      const v = attr.value;
      if (n === "name" && tag === "map" && v.trim()) {
        e.setAttribute(attr.name, ns(v.trim()));
        continue;
      }
      if (n.startsWith("on") || n === "style" || n === "name" || DROP_ATTRS.has(n) || n.startsWith(REMOTE_ATTR_PREFIX) || n === REMOTE_MARK) {
        e.removeAttribute(attr.name);
        continue;
      }
      if (IDREF_ATTRS.has(n)) {
        const ids = v.split(/\s+/).filter(Boolean);
        if (ids.length) e.setAttribute(attr.name, ids.map(ns).join(" "));
        continue;
      }
      if (n === "usemap") {
        if (urlKind(v) === "fragment" && v.trim().length > 1) e.setAttribute(attr.name, `#${ns(v.trim().slice(1))}`);
        else e.removeAttribute(attr.name); // a map in another document never resolves here
        continue;
      }
      if (v.includes("url(")) {
        const rewritten = v.replace(URL_FRAGMENT_REF, (m, q, id) => `url(${q}#${ns(id)}${q})`);
        if (rewritten !== v) e.setAttribute(attr.name, rewritten);
      }
      if (n === "href" || n === "xlink:href") {
        if (isLink) {
          if (!isSafeLinkHref(v)) e.removeAttribute(attr.name);
          else if ((rewriteFragments || tag !== "a") && urlKind(v) === "fragment" && v.trim().length > 1) {
            e.setAttribute(attr.name, `#${ns(v.trim().slice(1))}`);
          }
          continue;
        }
        // Any other element's href is a resource: an SVG <image>, <use>, <feImage>..
        // A same-document reference follows the ids it points at.
        if (urlKind(v) === "fragment") {
          if (v.trim().length > 1) e.setAttribute(attr.name, `#${ns(v.trim().slice(1))}`);
          continue;
        }
      } else if (!RESOURCE_ATTRS.has(n)) {
        continue;
      }
      const verdict = n === "srcset" ? srcsetVerdict(v) : resourceVerdict(v);
      if (verdict === "keep") continue;
      e.removeAttribute(attr.name);
      if (verdict === "remote") {
        e.setAttribute(parkName(n), v);
        parked = true;
      }
    }
    if (parked) {
      e.setAttribute(REMOTE_MARK, "");
      remote++;
    }
    if (e.id) e.id = ns(e.id);
  }
  return remote;
}

// restoreRemote puts parked remote URLs back on every marked element under root, once the
// reader has allowed remote content. Returns the elements restored.
export function restoreRemote(root) {
  const restored = [];
  for (const e of Array.from(root.querySelectorAll(`[${REMOTE_MARK}]`))) {
    for (const attr of Array.from(e.attributes)) {
      if (!attr.name.startsWith(REMOTE_ATTR_PREFIX)) continue;
      e.setAttribute(attr.name.slice(REMOTE_ATTR_PREFIX.length), attr.value);
      e.removeAttribute(attr.name);
    }
    e.removeAttribute(REMOTE_MARK);
    restored.push(e);
  }
  return restored;
}

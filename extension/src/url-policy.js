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
//   - ids namespaced as d<index>-<id>, and with rewriteFragments, same-section "#x" links
//     retargeted to match. EPUB rewrites its own links first and passes false.
// Returns the number of elements with parked remote content.
export function scrubTree(root, index, { rewriteFragments = true } = {}) {
  let remote = 0;
  for (const e of Array.from(root.querySelectorAll("*"))) {
    const tag = e.localName.toLowerCase();
    const isLink = LINK_TAGS.has(tag);
    let parked = false;
    const nm = e.getAttribute("name");
    if (nm && tag === "a" && !e.id) e.id = nm;
    for (const attr of Array.from(e.attributes)) {
      const n = attr.name.toLowerCase();
      const v = attr.value;
      if (n.startsWith("on") || n === "style" || n === "name" || DROP_ATTRS.has(n) || n.startsWith(REMOTE_ATTR_PREFIX) || n === REMOTE_MARK) {
        e.removeAttribute(attr.name);
        continue;
      }
      if (n === "href" || n === "xlink:href") {
        if (isLink) {
          if (!isSafeLinkHref(v)) e.removeAttribute(attr.name);
          else if (rewriteFragments && urlKind(v) === "fragment" && v.trim().length > 1) {
            e.setAttribute(attr.name, `#d${index}-${v.trim().slice(1)}`);
          }
          continue;
        }
        // Any other element's href is a resource: an SVG <image>, <use>, <feImage>..
        // A same-document reference follows the ids it points at.
        if (urlKind(v) === "fragment") {
          if (rewriteFragments && v.trim().length > 1) e.setAttribute(attr.name, `#d${index}-${v.trim().slice(1)}`);
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
    if (e.id) e.id = `d${index}-${e.id}`;
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

// site-host.js - the viewer's ?file= parameter, and the site the popup's per-site switch acts on.
//
// The switch controls the active site's list. background.js matches both the document's own host
// and the host of the page it was opened from, so a PDF on a CDN linked from that site follows
// the same setting. On the reflow viewer's own tab the
// site is the one the shown document came from: the extension's id is not a site anyone can mean.

// The interception rule substitutes the original URL after `file=` *without*
// URL-encoding, so it may itself contain `?`/`&` (its own query string). Take the
// whole raw tail after `file=` rather than URLSearchParams, which would split on
// the embedded `&`. If the tail isn't already an absolute URL, treat it as
// percent-encoded (the manual viewer.html?file=<encoded> entry point).
export function parseFileParam(search) {
  const m = /[?&]file=(.*)$/s.exec(search);
  if (!m) return "";
  const raw = m[1];
  if (/^(https?|file):/i.test(raw)) return raw;
  try { return decodeURIComponent(raw); } catch { return raw; }
}

// siteHost returns the host the per-site switch acts on for a tab showing tabUrl, or "" when the
// tab is on no website (a local file, a browser page, the viewer with a local document).
// viewerUrl is the viewer page's own address, chrome.runtime.getURL("src/viewer.html").
export function siteHost(tabUrl, viewerUrl) {
  let url = String(tabUrl || "");
  if (viewerUrl && url.startsWith(viewerUrl)) {
    const q = url.indexOf("?");
    url = q >= 0 ? parseFileParam(url.slice(q)) : "";
  }
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:" ? u.hostname : "";
  } catch {
    return "";
  }
}

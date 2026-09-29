// intercept.js - the URL shapes document interception agrees on, in one place.
//
// background.js builds its DNR rules from these patterns, and the popup and the
// command handler reuse them to tell whether the tab on screen is a document the
// extension would take over. One module, so the three surfaces can never disagree
// about what a "supported document" is.

// The document extension must end the URL's *path*: `[^?#]*` keeps the match out of the query
// and fragment, so `https://site/viewer?file=a.pdf` - a web app's own page - is not taken over.
// The query string itself is still captured and carried to the viewer.
const INTERCEPT_EXT = "(?:pdf|epub|rtf|fb2|mobi|azw3|cbz|cbt)";
export const HTTPS_INTERCEPT_REGEX = `^(https?://[^?#]*\\.${INTERCEPT_EXT}(?:[?#].*)?)$`;
// Three slashes: only empty-host file URLs. UNC paths (file://server/share) can't be granted to
// extensions by any match pattern, so the viewer could never fetch them - leave those to
// Chrome's built-in viewer.
export const FILE_INTERCEPT_REGEX = `^(file:///[^?#]*\\.${INTERCEPT_EXT}(?:[?#].*)?)$`;

const HTTPS_RE = new RegExp(HTTPS_INTERCEPT_REGEX);
const FILE_RE = new RegExp(FILE_INTERCEPT_REGEX);

// isInterceptableUrl says whether a main-frame navigation to url would be redirected to the
// reflow viewer by the dynamic rules. DNR evaluates these as RE2; the subset used here reads
// the same in JS (background.test.mjs pins that), so the JS check is the same check.
export function isInterceptableUrl(url) {
  const s = String(url || "");
  return HTTPS_RE.test(s) || FILE_RE.test(s);
}

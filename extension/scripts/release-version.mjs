// The extension's version, as the stores read it: YY.MDD.Hmm, three integers with no leading zero, because Chrome and
// Edge refuse "26.0702.1342" (the desktop stamp 26.0702.1342 maps to 26.702.1342). Still monotonic: mmdd 101..1231,
// hhmm 0..2359.
//
// A store release pins it (canon RELEASE_AND_DISTRIBUTION section 4, PACKAGE-VERSIONING E1-E4, ticket E0111 F15): the
// instant is recorded in docs/release-notes.json before the tag, the ext-cws-v / ext-edge-v tag names it, and the CI build
// stamps exactly that - so the site can show the version before the stores do, and no run invents a later one.

const VERSION = /^\d{2}\.[1-9]\d{2,3}\.(0|[1-9]\d{0,3})$/;
const STORE_TAG = /^ext-(?:cws|edge)-v(.+)$/;
const RECORDED = /("extension":\s*\{\s*"version":\s*")([^"]*)(")/;

export function versionFromInstant(d) {
  const yy = d.getFullYear() % 100;
  const mmdd = (d.getMonth() + 1) * 100 + d.getDate();
  const hhmm = d.getHours() * 100 + d.getMinutes();
  return `${yy}.${mmdd}.${hhmm}`;
}

export function isStoreVersion(v) {
  return typeof v === "string" && VERSION.test(v);
}

// The version a store tag releases. The tag must name the recorded version: a tag that does not was cut from a tree whose
// site shows something else, and it stops here rather than publish.
export function versionFromTag(tag, recorded) {
  const m = STORE_TAG.exec(tag);
  if (!m) throw new Error(`${tag} is not an ext-cws-v<version> or ext-edge-v<version> tag`);
  if (!isStoreVersion(m[1])) throw new Error(`${tag}: '${m[1]}' is not the stores' integer shape YY.MDD.Hmm`);
  if (m[1] !== recorded) {
    throw new Error(`${tag} names ${m[1]}, but docs/release-notes.json records extension.version ${recorded}: run node extension/build.mjs record-version and scripts/release-notes.ps1 -Render, commit, then tag that version`);
  }
  return m[1];
}

function parts(v) {
  return v.split(".").map(Number);
}

function isLater(a, b) {
  const [x, y] = [parts(a), parts(b)];
  for (let i = 0; i < x.length; i++) {
    if (x[i] !== y[i]) return x[i] > y[i];
  }
  return false;
}

// docs/release-notes.json with the extension's version replaced, every other byte kept. A later release orders above an
// earlier one on each channel (PACKAGE-VERSIONING E3), so a version that does not is refused.
export function withRecordedVersion(notesText, version) {
  const m = RECORDED.exec(notesText);
  if (!m) throw new Error('docs/release-notes.json has no "extension": {"version": ..} member');
  if (isStoreVersion(m[2]) && !isLater(version, m[2])) {
    throw new Error(`${version} does not order above the recorded extension.version ${m[2]}`);
  }
  return notesText.replace(RECORDED, `$1${version}$3`);
}

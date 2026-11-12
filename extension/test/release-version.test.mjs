// The extension's store version is pinned to its tag (ticket E0111 F15, PACKAGE-VERSIONING E1-E4).
import test from "node:test";
import assert from "node:assert/strict";
import { isStoreVersion, versionFromInstant, versionFromTag, withRecordedVersion } from "../scripts/release-version.mjs";

test("the stamp is three integers with no leading zero", () => {
  assert.equal(versionFromInstant(new Date(2026, 6, 2, 13, 42)), "26.702.1342");
  assert.equal(versionFromInstant(new Date(2026, 9, 21, 0, 41)), "26.1021.41");
  assert.equal(versionFromInstant(new Date(2026, 0, 1, 0, 0)), "26.101.0");
  assert.equal(isStoreVersion("26.702.1342"), true);
  assert.equal(isStoreVersion("26.1021.41"), true);
  assert.equal(isStoreVersion("26.0702.1342"), false);
  assert.equal(isStoreVersion("26.1021.041"), false);
  assert.equal(isStoreVersion("26.1021"), false);
  assert.equal(isStoreVersion(undefined), false);
});

test("a store tag releases the version it names, when that is the recorded one", () => {
  assert.equal(versionFromTag("ext-cws-v26.1108.1530", "26.1108.1530"), "26.1108.1530");
  assert.equal(versionFromTag("ext-edge-v26.1108.1530", "26.1108.1530"), "26.1108.1530");
});

test("a tag that names another version, or none, stops the build", () => {
  assert.throws(() => versionFromTag("ext-cws-v26.1108.1530", "26.1021.41"), /records extension\.version 26\.1021\.41/);
  assert.throws(() => versionFromTag("ext-cws-v26.1021", "26.1021.41"), /integer shape/);
  assert.throws(() => versionFromTag("v26.1021.0300", "26.1021.41"), /is not an ext-cws-v/);
  assert.throws(() => versionFromTag("ext-cws-v26.0702.1342", "26.0702.1342"), /integer shape/);
});

test("recording replaces the version and keeps every other byte", () => {
  const before = '{\r\n  "link": {"en": "x"},\r\n  "extension": {"version": "26.1021.41"},\r\n  "releases": []\r\n}\r\n';
  const after = withRecordedVersion(before, "26.1108.1530");
  assert.equal(after, before.replace("26.1021.41", "26.1108.1530"));
});

test("recording refuses a version that does not order above the recorded one", () => {
  const notes = '{"extension": {"version": "26.1108.1530"}}';
  assert.throws(() => withRecordedVersion(notes, "26.1108.1530"), /does not order above/);
  assert.throws(() => withRecordedVersion(notes, "26.1021.41"), /does not order above/);
  assert.match(withRecordedVersion(notes, "26.1108.1531"), /26\.1108\.1531/);
  assert.match(withRecordedVersion(notes, "27.101.0"), /27\.101\.0/);
});

test("recording refuses a source with no extension member", () => {
  assert.throws(() => withRecordedVersion('{"releases": []}', "26.1108.1530"), /no "extension"/);
});

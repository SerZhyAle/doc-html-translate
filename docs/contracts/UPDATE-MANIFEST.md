# Pointer: UPDATE-MANIFEST

- **Id:** `UPDATE-MANIFEST`
- **Version:** 0.13 draft
- **Home:** the shared contracts catalog, `app-update-feed/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** planned, not bound (amendment 2026-10-29 item 1); no manifest client is implemented
- **Wire carrier:** `schemaVersion` (JSON int)

**Read at 0.13 on 2026-10-29 (ticket 122).** The catalog accepted the `app-update-feed` proposal of 2026-10-20: this
product left the header's `consumers:` line for the planned, not bound products of section 1, and an in-app notice
that reads a release listing, keeps its own cache and comparison, links to the release page and installs nothing is
not a client of the feed. It becomes bound when it ships a manifest client, on its proposal. The product holds no
manifest parser or client and no fixture suite. Its update check (ticket 64, `cmd/doc-html-ui/update.go`) is such a
notice: the GUI reads the GitHub Releases API, caches the answer, degrades silently, compares the registered stamp
with the leading `v` removed (a different shape gives no offer) and offers only a link to the release page. The
Store edition delegates to the Store, and winget-managed installs are not detected.

The registry exception for the declared-consumer gap is closed (2026-10-29, ticket 122). No catalog vectors exist.

**Conformance.** `TestUpdateCheckOfflineIsQuiet`, `TestUpdateCheckVersionAndRequest` and
`TestUpdateVersionRejectsUntrustedTag` in `cmd/doc-html-ui` hold the link check; nothing holds the manifest.

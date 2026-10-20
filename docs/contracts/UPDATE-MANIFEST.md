# Pointer: UPDATE-MANIFEST

- **Id:** `UPDATE-MANIFEST`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `app-update-feed/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** declared consumer, not bound for the GitHub Releases link check; no manifest client is implemented
- **Wire carrier:** `schemaVersion` (JSON int)

**Read at 0.12 on 2026-10-20 (ticket 120).** The product holds no manifest parser or client and no fixture suite, so
the feed sections and the hash rule are not applicable. It does have an update check (ticket 64,
`cmd/doc-html-ui/update.go`): the GUI reads the GitHub Releases API, caches the answer, degrades silently, compares
the registered stamp with the leading `v` removed (a different shape gives no offer) and offers only a link to the
release page. Section 1 (0.10) and 0.12 put such a browser Releases link outside this manifest, owing no parser,
cache or hash profile. The Store edition delegates to the Store, and winget-managed installs are not detected.
This is a declaration, not conformance evidence.

The registry exception for the declared-consumer gap is corrected, not closed (until 2026-12-31); the owner is asked
to move this product from declared to planned in the contract header (the `app-update-feed` proposal file of
2026-10-20). No catalog vectors exist.

**Conformance.** `TestUpdateCheckOfflineIsQuiet`, `TestUpdateCheckVersionAndRequest` and
`TestUpdateVersionRejectsUntrustedTag` in `cmd/doc-html-ui` hold the link check; nothing holds the manifest.

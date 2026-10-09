# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1007.0054

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1007.0054 | release.yml run 37547852267 success; 8 assets incl the universal installer built from the tag; exe stamp verified 26.1007.0054 (installer: resource version matches, the Go link-stamp check does not apply to the Inno stub); CI hashes match local; auto-generated body replaced with the approved What's new + Downloads | 2026-10-07 01:44 |
| winget | live | PR#447950 | 15 manifests via wingetcreate submit winget (folder, because InstallationNotes is new vs 26.0930.1107); SHA256 matches the release .sha256 asset; winget validate + local install-test verified the hash end-to-end and upgraded 26.0930.1107 in place; fork was synced by the owner; license/cla pass (already on file); PR body filled from the live upstream template; previous PR#444113 merged; live read 2026-10-09: PR#447950 merged 2026-10-07T00:58:43Z and manifests/s/SerZhyAle/DocHtmlTranslate/26.1007.0054 is the newest folder upstream | 2026-10-09 17:41 |
| Store | pending | SZA.Doc-HTML-Translate_26.1007.54.0_x64.msix | 26.1007.0054: unsigned package built from the tag v26.1007.0054 in msix/out (clean tree, HEAD = tag commit); awaiting manual upload in Partner Center (product 9PMHSWQPR6V1) with msix/out/store-import (listing CSV ReleaseNotes refreshed for 13 locales); live read 2026-10-09: the public product endpoint answers 200 with LastUpdateDateUtc 2026-10-07T00:11:24Z and RatingCount 0, but exposes no served package version (Payload.Version is empty), so the upload of this package is not confirmed and the status is left as recorded | 2026-10-09 17:41 |
| Chrome | live | ext-cws-v26.1007 | publish-cws run 37548550167 success; zip stamped 26.1006.2348 (build-time stamp, UTC), uploaded to the Chrome Web Store, review runs out-of-band in the dashboard; live read 2026-10-09: the store page serves 26.1006.2348 (the extension stamp of this release), 199 users | 2026-10-09 17:41 |
| Edge | live | ext-edge-v26.1007 | publish-edge run 37548550179 success; zip stamped 26.1006.2348; Upload Succeeded, draft submission Publish Succeeded; Microsoft certification runs out-of-band; live read 2026-10-09: the product-details endpoint serves 26.1006.2348 (the extension stamp of this release), activeInstallCount 17 | 2026-10-09 17:41 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1007.0054

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1007.0054 | release.yml run 37547852267 success; 8 assets incl the universal installer built from the tag; exe stamp verified 26.1007.0054 (installer: resource version matches, the Go link-stamp check does not apply to the Inno stub); CI hashes match local; auto-generated body replaced with the approved What's new + Downloads | 2026-10-07 01:44 |
| winget | submitted | PR#447950 | 15 manifests via wingetcreate submit winget (folder, because InstallationNotes is new vs 26.0930.1107); SHA256 matches the release .sha256 asset; winget validate + local install-test verified the hash end-to-end and upgraded 26.0930.1107 in place; fork was synced by the owner; license/cla pass (already on file); PR body filled from the live upstream template; previous PR#444113 merged | 2026-10-07 01:55 |
| Store | pending | SZA.Doc-HTML-Translate_26.1007.54.0_x64.msix | 26.1007.0054: unsigned package built from the tag v26.1007.0054 in msix/out (clean tree, HEAD = tag commit); awaiting manual upload in Partner Center (product 9PMHSWQPR6V1) with msix/out/store-import (listing CSV ReleaseNotes refreshed for 13 locales) | 2026-10-07 01:44 |
| Chrome | submitted | ext-cws-v26.1007 | publish-cws run 37548550167 success; zip stamped 26.1006.2348 (build-time stamp, UTC), uploaded to the Chrome Web Store, review runs out-of-band in the dashboard | 2026-10-07 01:50 |
| Edge | submitted | ext-edge-v26.1007 | publish-edge run 37548550179 success; zip stamped 26.1006.2348; Upload Succeeded, draft submission Publish Succeeded; Microsoft certification runs out-of-band | 2026-10-07 01:50 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

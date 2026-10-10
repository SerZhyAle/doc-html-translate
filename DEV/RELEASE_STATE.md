# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1010.0142

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1010.0142 | release.yml run 38006806135 success; 7 CI assets + setup.exe built from the tag and uploaded; exe stamp verified 26.1010.0142 (resource, linked stamp, reports); CI hashes match local; auto-generated body replaced with the approved What's new + Downloads (the body was blanked for a moment by a scripting slip and restored with the same hashes) | 2026-10-10 02:05 |
| winget | submitted | PR#449861 | 15 manifests restamped to 26.1010.0142, winget validate OK, local winget install --manifest verified the zip SHA256 end to end; fork synced first; submitted with wingetcreate submit winget (descriptions and tags changed); PR body filled from the live template; CLA on file | 2026-10-10 02:05 |
| Store | pending | SZA.Doc-HTML-Translate_26.1010.142.0_x64.msix | unsigned package built from tag v26.1010.0142 (clean tree, HEAD = tag commit) in msix/out; store-import CSV in msix/out/store-import carries the refreshed ReleaseNotes, search terms and Feature9; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1); the 26.1007.54.0 package was also never confirmed uploaded | 2026-10-10 02:05 |
| Chrome | submitted | ext-cws-v26.1010 | publish-cws run 38007344497 success; crxVersion 26.1010.3, submittedItemRevisionStatus PENDING_REVIEW; review runs out-of-band in the dashboard | 2026-10-10 02:05 |
| Edge | blocked | ext-edge-v26.1010 | publish-edge run 38007346992: upload Succeeded (draft 26.1010.3), publish FAILED: ModuleStateUnPublishable Invalid module LISTING - a dashboard-side listing field is invalid, not the package (appDesc lengths all <= 132); owner must open the Edge Partner Center listing, fix the named language, and press Publish on the existing draft; Edge still serves 26.1006.2348 | 2026-10-10 02:05 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

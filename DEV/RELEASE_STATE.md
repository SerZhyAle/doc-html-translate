# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0930.1107

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0930.1107 | release.yml run 36699011635 success; 8 assets incl the universal installer built from the tag; exe stamp verified 26.0930.1107; auto-generated body replaced with the approved What's new | 2026-09-30 11:56 |
| winget | submitted | PR#444113 | 15 manifests; SHA256 from the release .sha256 asset (matches Get-FileHash); winget validate + local install-test verified the hash end-to-end; PR body filled from the live upstream template; previous PR#433869 merged | 2026-09-30 12:01 |
| Store | pending | SZA.Doc-HTML-Translate_26.930.1107.0_x64.msix | unsigned package built from the tag in msix/out; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1); listing CSV ReleaseNotes not refreshed yet | 2026-09-30 12:01 |
| Chrome | submitted | ext-cws-v26.0930 | publish-cws run 36699584346 success; upload SUCCEEDED, item state PENDING_REVIEW - the previous revision stays published until it clears | 2026-09-30 12:01 |
| Edge | submitted | ext-edge-v26.0930 | publish-edge run 36699584033 first failed HTTP 401 (expired EDGE_API_KEY); new key stored as the repo secret, rerun success: upload Succeeded, draft submission published; Microsoft certification runs out-of-band | 2026-09-30 12:04 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

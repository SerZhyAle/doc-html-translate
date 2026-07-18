# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0718.0252

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0718.0252 | 7 assets incl installer, curated notes | 2026-07-18 02:55 |
| winget | submitted | PR#404039 | install-test hash verified; desc adds comics/images | 2026-07-18 02:58 |
| Store | submitted | SZA.Doc-HTML-Translate_26.718.259.0_x64.msix | uploaded to Partner Center, in certification (product 9PMHSWQPR6V1) | 2026-07-18 03:13 |
| Chrome | live | ext-cws-v26.0718 | publish-cws.yml success (26s) | 2026-07-18 03:02 |
| Edge | blocked | ext-edge-v26.0718 | InProgressSubmission: new draft uploaded, publish blocked by prior cert still in progress; rerun/publish-draft when it clears | 2026-07-18 03:02 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

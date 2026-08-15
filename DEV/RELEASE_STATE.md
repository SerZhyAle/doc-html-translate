# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0815.2020

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0815.2020 | 7 assets incl the universal installer rebuilt with -Stamp to match the tag; auto-generated body replaced with curated What's new | 2026-08-15 20:36 |
| winget | submitted | PR#418115 | 15 manifests; SHA256 from the release's own .sha256 asset; install-test verified the hash end-to-end. Fork sync needs the workflow scope, so the branch was built through the API from the fork's own master - diff is only the new version folder. PR body filled from the live upstream template | 2026-08-15 20:40 |
| Store | pending | SZA.Doc-HTML-Translate_26.815.2043.0_x64.msix | unsigned package built in msix/out; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1). Supersedes the 26.0729 and 26.0813 packages, neither of which was ever uploaded | 2026-08-15 20:44 |
| Chrome | submitted | ext-cws-v26.0815 | publish-cws run 31901906596 success; uploaded to the Chrome Web Store, review pending - the previous revision stays published until it clears | 2026-08-15 20:44 |
| Edge | submitted | ext-edge-v26.0815 | publish-edge run 31901908230 success; uploaded and published by CI, Microsoft certification runs out-of-band | 2026-08-15 20:44 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

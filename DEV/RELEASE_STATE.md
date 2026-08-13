# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0813.1917

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0813.1917 | 7 assets incl universal installer; curated notes replace the auto-generated body | 2026-08-13 19:20 |
| winget | submitted | PR#417020 | install-test hash verified end-to-end; 15 manifests; fork could not be synced (token lacks workflow scope) so the branch was built through the API from the fork's own master - the PR diff is still only the new version folder | 2026-08-13 20:51 |
| Store | pending | SZA.Doc-HTML-Translate_26.813.2051.0_x64.msix | unsigned package built in msix/out; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1). The 26.0729 package was never uploaded either, so this supersedes it. 11 locales added on 26.0729 still need adding in the dashboard, re-exporting and the CSV re-run | 2026-08-13 20:53 |
| Chrome | submitted | ext-cws-v26.0813 | crx 26.813.1851 uploaded, PENDING_REVIEW (item nmcckamdocainafmmompkbmelkpbnmic; the previous revision stays published until review clears) | 2026-08-13 20:53 |
| Edge | submitted | ext-edge-v26.0813 | crx 26.813.1851 uploaded and published by CI; Microsoft certification runs out-of-band | 2026-08-13 20:53 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

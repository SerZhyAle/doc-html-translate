# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0729.0134

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0729.0134 | 7 assets incl universal installer; curated notes replace the auto-generated body | 2026-07-29 02:05 |
| winget | submitted | PR#409212 | install-test hash verified end-to-end; adds 12 locale manifests; PR body replaced | 2026-07-29 02:06 |
| Store | pending | SZA.Doc-HTML-Translate_26.729.207.0_x64.msix | unsigned package built; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1). Import folder: temp/store-import (CSV + 39 per-locale screenshots). 11 new locales must be added in the dashboard, re-exported, then the CSV re-run | 2026-07-29 02:07 |
| Chrome | submitted | ext-cws-v26.0729 | crx 26.729.7 uploaded, PENDING_REVIEW (26.718.1849 still the published revision until review clears) | 2026-07-29 02:08 |
| Edge | submitted | ext-edge-v26.0729 | upload + publish both Succeeded (the 26.0718 block cleared); Microsoft certification runs out-of-band | 2026-07-29 02:10 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

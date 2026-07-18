# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0718.0252

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0718.0252 | 7 assets incl installer, curated notes | 2026-07-18 02:55 |
| winget | submitted | PR#404039 | install-test hash verified; desc adds comics/images | 2026-07-18 02:58 |
| Store | pending |  |  |  |
| Chrome | pending |  |  |  |
| Edge | pending |  |  |  |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

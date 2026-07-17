# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : (unset)

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | pending |  |  |  |
| winget | pending |  |  |  |
| Store | pending |  |  |  |
| Chrome | pending |  |  |  |
| Edge | pending |  |  |  |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

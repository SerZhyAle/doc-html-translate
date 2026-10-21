# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1021.0300

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | pending | v26.1021.0300 | tag not pushed yet; needs a green check.ps1 on the tagged tree | 2026-10-21 02:26 |
| winget | pending | - | previous PR#449861 is MERGED (26.1010.0142 is in the index); the new manifests are restamped only after the GitHub release exists | 2026-10-21 02:26 |
| Store | pending | - | MSIX not built yet; the unsigned packages 26.1010.142.0 and 26.1007.54.0 were never confirmed uploaded, the new one supersedes them | 2026-10-21 02:26 |
| Chrome | pending | - | previous ext-cws-v26.1010 (crxVersion 26.1010.3) was PENDING_REVIEW when last recorded; not rechecked | 2026-10-21 02:26 |
| Edge | pending | - | previous publish blocked on ModuleStateUnPublishable Invalid module LISTING (dashboard-side listing field); unless the owner fixed the listing the new publish fails the same way | 2026-10-21 02:26 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

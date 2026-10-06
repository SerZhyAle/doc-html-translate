# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1007.0054

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1007.0054 | release.yml run 37547852267 success; 8 assets incl the universal installer built from the tag; exe stamp verified 26.1007.0054 (installer: resource version matches, the Go link-stamp check does not apply to the Inno stub); CI hashes match local; auto-generated body replaced with the approved What's new + Downloads | 2026-10-07 01:44 |
| winget | pending | - | 26.1007.0054: not started (needs the release zip hash; confirmation pending) | 2026-10-07 01:44 |
| Store | pending | SZA.Doc-HTML-Translate_26.1007.54.0_x64.msix | 26.1007.0054: unsigned package built from the tag v26.1007.0054 in msix/out (clean tree, HEAD = tag commit); awaiting manual upload in Partner Center (product 9PMHSWQPR6V1) with msix/out/store-import (listing CSV ReleaseNotes refreshed for 13 locales) | 2026-10-07 01:44 |
| Chrome | pending | - | 26.1007.0054: not started (tag ext-cws-v26.1007 pending confirmation); ext-cws-v26.0930 was PENDING_REVIEW at the last record, owner says the review state is ok | 2026-10-07 01:44 |
| Edge | pending | - | 26.1007.0054: not started (tag ext-edge-v26.1007 pending confirmation) | 2026-10-07 01:44 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

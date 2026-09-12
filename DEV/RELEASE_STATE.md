# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0912.2026

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0912.2026 | release.yml run 34711217340 success; 7 assets incl the universal installer built with -Stamp to match the tag; auto-generated body replaced with curated What's new | 2026-09-12 21:57 |
| winget | submitted | PR#433869 | 15 manifests; SHA256 from the release's own .sha256 asset; install-test verified the hash end-to-end. wingetcreate could not sync the fork again, so the branch was built through the API from the fork's own master - diff is only the new version folder. PR body filled from the live upstream template | 2026-09-12 21:58 |
| Store | pending | SZA.Doc-HTML-Translate_26.912.2027.0_x64.msix | unsigned package built in msix/out; listing CSV + import folder in msix/out/store-import (ReleaseNotes refreshed x13); awaiting manual upload in Partner Center (product 9PMHSWQPR6V1). Supersedes the never-uploaded 26.815.2043 package | 2026-09-12 21:55 |
| Chrome | submitted | ext-cws-v26.0912 | publish-cws run 34715569240 success; uploaded to the Chrome Web Store, review pending - the previous revision stays published until it clears | 2026-09-12 21:57 |
| Edge | submitted | ext-edge-v26.0912 | publish-edge run 34715569256 success; uploaded by CI, Microsoft certification runs out-of-band | 2026-09-12 21:57 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

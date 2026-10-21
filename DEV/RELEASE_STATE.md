# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1021.0300

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1021.0300 | release.yml run 39522258307 success; 6 CI assets + setup.exe built from the tag and uploaded; exe stamp verified 26.1021.0300 (resource, linked stamp, reports); CI hashes match local; body replaced with the approved What's new + Downloads | 2026-10-21 02:37 |
| winget | submitted | PR#457550 | 15 manifests restamped to 26.1021.0300, winget validate OK, local winget install --manifest verified the zip SHA256 and upgraded the real 26.1010.0142 install; fork synced first; submitted with wingetcreate submit winget (13 descriptions changed: JPEG XL in the image format list); PR body filled from the live template; license/cla SUCCESS | 2026-10-21 02:39 |
| Store | pending | - | MSIX not built yet; the unsigned packages 26.1010.142.0 and 26.1007.54.0 were never confirmed uploaded, the new one supersedes them | 2026-10-21 02:26 |
| Chrome | pending | - | previous ext-cws-v26.1010 (crxVersion 26.1010.3) was PENDING_REVIEW when last recorded; not rechecked | 2026-10-21 02:26 |
| Edge | pending | - | previous publish blocked on ModuleStateUnPublishable Invalid module LISTING (dashboard-side listing field); unless the owner fixed the listing the new publish fails the same way | 2026-10-21 02:26 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

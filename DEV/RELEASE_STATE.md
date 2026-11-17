# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1117.1450

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1117.1450 | release.yml run 44104616080 success; 7 CI assets + setup.exe built from the tag and uploaded; exe stamp verified 26.1117.1450 (resource, linked stamp, reports); zip SHA256 equals the GitHub asset digest; body replaced with the approved What's new + Downloads; main pushed (Pages) | 2026-11-17 15:43 |
| winget | submitted | PR#482354 | manifests validated, local winget install --manifest OK (hash verified); PR open, license/cla SUCCESS, MERGEABLE, 15 files; submitted through the API from the fork branch (fork sync needs the workflow scope); not merged yet | 2026-11-17 15:43 |
| Store | pending | - | Store preflight 2026-11-17 14:58: ready; submission published; price Free; package 26.1117.1450.0 above every live version. MSIX built from tag v26.1117.1450, msix/out/SZA.Doc-HTML-Translate_26.1117.1450.0_x64.msix, unsigned, read-back Identity OK; owner uploads by hand in Partner Center | 2026-11-17 15:43 |
| Chrome | pending | ext-cws-v26.1117.1543 | extension version recorded in docs/release-notes.json; tag not pushed yet | 2026-11-17 15:43 |
| Edge | pending | ext-edge-v26.1117.1543 | extension version recorded in docs/release-notes.json; tag not pushed yet | 2026-11-17 15:43 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

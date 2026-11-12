# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1112.2325

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1112.2325 | release.yml run 43327212930 success; 7 CI assets built from the tag + setup.exe built from the tag and uploaded; exe stamp verified 26.1112.2325 (resource, linked stamp, reports); zip SHA256 equals the GitHub asset digest; body replaced with the approved What's new + Downloads | 2026-11-13 00:23 |
| winget | submitted | PR#478928 | PR open, license/cla completed, MERGEABLE, 15 files / 640 additions; manifests validated and local winget install --manifest OK (hash verified); submitted through the API from the fork branch (wingetcreate could not sync the fork: gh token has no workflow scope); not merged yet | 2026-11-13 00:23 |
| Store | pending | - | Store preflight 2026-11-13 00:14: ready; submission published; price Free; package above every live version. MSIX built from tag v26.1112.2325: msix/out/SZA.Doc-HTML-Translate_26.1112.2325.0_x64.msix, unsigned, read-back Identity OK; owner uploads by hand in Partner Center | 2026-11-13 00:23 |
| Chrome | submitted | ext-cws-v26.1021 | publish-cws run 39522779744 success; crxVersion 26.1021.41; PENDING_REVIEW at upload, store review not rechecked | 2026-10-21 10:03 |
| Edge | submitted | ext-edge-v26.1021 | publish-edge run 39522782394 success; Upload and Publish Succeeded, version 26.1021.41; the earlier Invalid module LISTING block did not recur; store certification not rechecked | 2026-10-21 10:03 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

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
| Chrome | submitted | ext-cws-v26.1117.1543 | publish-cws run 44106344131 success; upload SUCCEEDED, submitted revision crxVersion 26.1117.1543 PENDING_REVIEW (published revision still 26.1113.22); store review not rechecked | 2026-11-17 15:45 |
| Edge | blocked | ext-edge-v26.1117.1543 | publish-edge run 44106343944 FAILED at publish: Upload Succeeded (draft 26.1117.1543 is in the Edge dashboard) but 'Can't publish extension as your extension submission is in progress' (InProgressSubmission: the 26.1113.22 submission is still in Microsoft certification); once it clears, run: gh run rerun 44106343944 --failed | 2026-11-17 15:45 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

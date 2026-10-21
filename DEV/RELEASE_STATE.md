# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.1021.0300

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.1021.0300 | release.yml run 39522258307 success; 6 CI assets + setup.exe built from the tag and uploaded; exe stamp verified 26.1021.0300 (resource, linked stamp, reports); CI hashes match local; body replaced with the approved What's new + Downloads | 2026-10-21 02:37 |
| winget | submitted | PR#457550 | PR MERGED 2026-10-21 06:37 UTC (Validation-Completed, Moderator-Approved); not live yet: after winget source update at 10:02 local the index still lists 26.1010.0142 as newest | 2026-10-21 10:03 |
| Store | pending | - | owner uploads by hand; MSIX NOT built yet: build-msix.ps1 -Tag needs HEAD at the tag commit 08a63b3 and the git switch is blocked by the guard hook for the agent, so it waits for the owner to run it; msix/out/store-import is already rebuilt (listingData.csv with JPEG XL + images) | 2026-10-21 10:03 |
| Chrome | submitted | ext-cws-v26.1021 | publish-cws run 39522779744 success; crxVersion 26.1021.41; PENDING_REVIEW at upload, store review not rechecked | 2026-10-21 10:03 |
| Edge | submitted | ext-edge-v26.1021 | publish-edge run 39522782394 success; Upload and Publish Succeeded, version 26.1021.41; the earlier Invalid module LISTING block did not recur; store certification not rechecked | 2026-10-21 10:03 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

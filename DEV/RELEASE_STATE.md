# Release publish-state

Durable state of the current release across all 5 channels. Updated via
`scripts/release-state.ps1` (alias `a rs`). This file records what already happened;
publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / `scripts/release.ps1`.

- **Version** : 26.0813.1917

| Channel | Status | Ref | Note | Updated |
|---|---|---|---|---|
| GitHub | live | v26.0813.1917 | 7 assets incl universal installer; curated notes replace the auto-generated body | 2026-08-13 19:20 |
| winget | live | PR#417020 | merged 2026-08-13 20:02 UTC; verified with `gh pr view` on 2026-08-15 - this row read `submitted` until then | 2026-08-15 |
| Store | pending | SZA.Doc-HTML-Translate_26.813.2051.0_x64.msix | unsigned package built in msix/out; awaiting manual upload in Partner Center (product 9PMHSWQPR6V1). The 26.0729 package was never uploaded either, so this supersedes it. 11 locales added on 26.0729 still need adding in the dashboard, re-exporting and the CSV re-run | 2026-08-13 20:53 |
| Chrome | submitted | ext-cws-v26.0813 | crx 26.813.1851 uploaded, PENDING_REVIEW (item nmcckamdocainafmmompkbmelkpbnmic; the previous revision stays published until review clears) | 2026-08-13 20:53 |
| Edge | submitted | ext-edge-v26.0813 | crx 26.813.1851 uploaded and published by CI; Microsoft certification runs out-of-band | 2026-08-13 20:53 |

Status vocabulary: `pending` -> `submitted` -> `live` (or `blocked` / `n/a`).

## Release-gate decision, 26.0815.2016

The OCR visual-fidelity lab's **concealment gate is red at ship time** and this is a decision, not
an oversight. `go run ./tools/ocrlab gate temp/ocrlab/20260815-190756` reports
`worst residual ink 0.9992` against a `0.28` bound.

**The gate went red because it was fixed.** Until 2026-08-15 the scorer defaulted concealment to
`0` - the value perfect concealment gets - on any scene where the runner stored no post-overlay
render, which is exactly the scene where recognition found nothing. Three of the four failing dev
scenes are that case, so the `0.28` bound was derived against a scorer blind to them. The `comic`
category, whose number was measured on a real scene, still passes at `0.2705`.

So the red row does not describe a regression in the product; it describes a bound that was never
honest. Re-deriving `DEV/ocrlab/thresholds.json` belongs to a dated baseline run against
`temp/ocrlab/20260815-190756` - the first run whose concealment numbers cover every scene - and it
is the first work item after this release, not a part of it.

Owner asked for the release across all channels after this was raised. Recorded here so a later
session reads the decision instead of re-discovering the red row and stopping.

The gates that do describe the product are green: `./scripts/test.ps1` exit 0, `./scripts/lint.ps1`,
`./scripts/typo.ps1`, `npm test` 140/140, and the lab's hard rows for protected-area damage (0),
clipped plates (0) and drift (0).

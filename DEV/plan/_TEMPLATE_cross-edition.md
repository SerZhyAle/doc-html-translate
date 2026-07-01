# <Feature name>

**Status:** Draft
**Priority:** <n>
**Date:** <YYYY-MM-DD>

> Template for a **cross-edition** feature ticket. One feature = one ticket that covers every edition.
> Copy this file to `DEV/plan/<YYYY-MM-DD>_<slug>.md`, delete this quote block, and fill in.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it if you touch a shared invariant.

## What / why

<One paragraph: the user-facing behaviour and the reason. No file paths, no symbols - that goes in the
tactical breakdown per the [spec lifecycle](../../docs/SPEC_LIFECYCLE.md).>

## Edition parity checklist

For each edition: **Done**, or **Declined** with a one-line rationale (record lasting declines in
`docs/PARITY.md` under "Intentional divergences"). "Not applicable" needs a reason too.

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[ ]` | |
| GUI (`doc-html-ui`) | `[ ]` | GUI must expose every new CLI flag |
| MSIX Store app | `[ ]` | usually inherits the GUI; note any packaging impact (read-only dir, manifest assoc) |
| Browser extension | `[ ]` | JS is a separate codebase - port, do not assume it follows |
| Website / docs | `[ ]` | landing page + extension page if user-facing |

## Shared invariants touched

<List any constant/palette/host/default from `docs/PARITY.md` this feature changes. If none, say "none".
If it introduces a *new* shared value, add it to the invariant tables in `docs/PARITY.md` in this ticket.>

## Cross-references

<Link the code on each side that must stay in sync (Go path <-> JS path), so a future change to one
surfaces the other. Mirror the port map in `docs/PARITY.md`.>

## Done criteria

- [ ] Every edition row above is `Done` or `Declined (reason)`.
- [ ] `docs/PARITY.md` updated if a shared invariant changed or a new one was introduced.
- [ ] Tests / gate green on the Go side (`scripts/test.ps1`) and `npm test` on the extension side.
- [ ] Changelog entry in `DEV/CHANGELOG.md`.

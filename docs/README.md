# docs/

The public-facing engineering docs: things a reader outside a single work session needs - the parity
contract, the method behind the spec skills, the OCR mechanism, and the integration contract other
products build against. Session-scoped material (tickets, research notes, the changelog, release state)
lives under [`DEV/`](../DEV/) instead, and the agent-facing rules live in
[`AGENTS.md`](../AGENTS.md) / [`CLAUDE.md`](../CLAUDE.md).

| File | What it is | Read it when |
| --- | --- | --- |
| [PARITY.md](PARITY.md) | The cross-edition contract: the Go -> JS port map, the values that must stay identical across the desktop app and the browser extension, and the intentional divergences. | Before adding or changing any user-facing feature. |
| [ocr-pipeline.md](ocr-pipeline.md) | How the OCR overlay finds image text and lays it back over the picture as real HTML - the mechanism, not the CLI contract. | Porting, reviewing, or rebuilding the overlay elsewhere. |
| [integration-image-translate.md](integration-image-translate.md) | The integration contract for embedding a "Translate image" button in another product. | Wiring another app into this one. |
| [SPEC_LIFECYCLE.md](SPEC_LIFECYCLE.md) | The tooling-agnostic methodology behind the `/spec*` skills. | Writing or auditing a ticket. |
| [how-i-posted-this-project-to-winget.md](how-i-posted-this-project-to-winget.md) | A walkthrough of the winget submission, with every blocker and fix. | Repeating or debugging a winget release. |

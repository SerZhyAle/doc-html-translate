# docs/

The public-facing engineering docs: things a reader outside a single work session needs - the parity
contract and the method behind the spec skills. The **cross-project contracts** other products build
against - the image-translate CLI contract, the OCR overlay mechanism and the overlay's own function
contract - live in the shared contracts catalog, not here; [`contracts/`](contracts/) holds one pointer
per contract and [`AGENTS.md`](../AGENTS.md) is the one file naming where the catalog is.
The collaborative specifications and tactical plans live in
[`DEV/plan/`](../DEV/plan/), while research notes, the changelog and release state
live under the rest of [`DEV/`](../DEV/). The agent-facing rules live in
[`AGENTS.md`](../AGENTS.md) / [`CLAUDE.md`](../CLAUDE.md).

| File | What it is | Read it when |
| --- | --- | --- |
| [PARITY.md](PARITY.md) | The cross-edition contract: the Go -> JS port map, the values that must stay identical across the desktop app and the browser extension, and the intentional divergences. | Before adding or changing any user-facing feature. |
| [GLYPH-MAP.md](GLYPH-MAP.md) | Every glyph each surface draws, mapped to the shared icon vocabulary's id, with its status (conforms, dated exception, artwork). | Before adding or changing any icon, glyph or control label. |
| [contracts/](contracts/README.md) | Pointers to the cross-project contracts this product produces or consumes: id, version, role, and what this repo owes each one. | Before changing anything at a boundary another product builds against. |
| [SPEC_LIFECYCLE.md](SPEC_LIFECYCLE.md) | The tooling-agnostic methodology behind the `/spec*` skills. | Writing or auditing a ticket. |
| [how-i-posted-this-project-to-winget.md](how-i-posted-this-project-to-winget.md) | A walkthrough of the winget submission, with every blocker and fix. | Repeating or debugging a winget release. |

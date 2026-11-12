# Pointer: WINDOWS-STORE

- **Id:** `WINDOWS-STORE`
- **Version:** 0.4 (draft; owner StreamsPlayer)
- **Home:** the shared contracts catalog, `windows-store/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer, `partial` - a declared consumer of the contract. This product hands Partner Center a packed MSIX
  ([`../../msix/`](../../msix/)) and a listing import file ([`../../tools/store/`](../../tools/store/)), and reads its Store
  submission through the release flow's read-only preflight (section D, rules 19 to 26). Adopted and read against the
  tree on 2026-11-12 (ticket E0133).

What it covers: what the artifacts must be so that Partner Center accepts them and the user meets one honest page - the
reserved identity read back from the packed archive, an unsigned upload, the version remap, the notices inside the
archive, the listing import file (header-named columns, the export's dialect, the caps, the search terms) and, since 0.4,
the machine channel to the Store submission. The order of channels and the click path are the canon's, not this
contract's.

**Where this product stands (2026-11-12).** The registry row says `partial`; the held and the open parts are:

- Held: the identity lives in one file ([`../../msix/identity.json`](../../msix/identity.json), rule 19) and the build
  reads it; the live package `26.1021.300.0` read back by hand carries the reserved Identity, no `AppxSignature.p7x`
  entry and `THIRD-PARTY-NOTICES.txt` (rules 1 to 4); the manifest declares the file association instead of relying on
  what the container hides (rule 5); the committed export holds all 13 language columns, every cell inside its cap and
  one `Title` and one `CopyrightTrademarkInformation` for every language (rules 7, 9, 14); the listing builder exits 1
  and writes nothing for a shipped language with no column, a cell over a cap and a break of the export's dialect, writes
  that dialect, lists every cell a `-Refresh` replaces with its length, and both committed CSVs (the export and the
  fixture with the copy cells emptied) are pinned `-text` and round-trip byte for byte, the cases running as one fast
  test (rules 7 to 9, 12, 13, rungs 1 and 2, ticket E0135, 2026-11-12); the product has its own age
  rating (rule 18); the release flow's third header verdict is the shared preflight (rules 23 (a), (b), (d), 25), which
  never fails a release after the tag (rule 21).
- Open (dated exceptions in the catalog registry): the build asserts nothing on the packed archive (ticket E0134, rules
  1, 2, 4 and rung 3); the optional screenshot attach writes a bare file name
  for the dashboard's folder import (rule 11, a supported variant); the release notes of the 13 columns are translated
  (rule 16, a supported variant).
- Not yet run: the preflight against the live Store, and the first write through the channel; both stay with the
  release and the owner (rule 22, section 4 rung 6); and a Partner Center import of a file made by the refusing
  builder (rung 4).

**Supported variants (design record, 2026-11-12).** Two deliberate differences from the sealed text, each a permanent
variant with a review condition and not a migration debt:

- Rule 11, item A of 0.3: the optional `-Screenshots` run of the listing builder writes the bare image name
  (`<view>-<locale>.png`) for the dashboard's folder import, because on 2026-08-13 the CSV alone was refused for every
  language and the folder import resolved the bare names. Only empty cells are filled and the deliverable is the folder,
  never the bare CSV. Review: when the owner of the contract answers the proposal filed with the catalog that day.
- Rule 16: the release notes are translated in all 13 columns, because the listing is localized as a whole; only ru and
  uk are proofread. It is an assumption the owner may overrule (the alternative is the English text in the ten
  machine-translated columns). Review: the owner's decision on that copy.

**What this repo owes it**

- Keep the Store identity in `msix/identity.json` and nowhere else; never change `IdentityName` or `Publisher`.
- Read the packed archive back before a package is offered for upload (E0134 closes this).
- Build the listing import file from the product's own fresh export with the refusals and caps of rules 7 to 9 (done,
  E0135); `tools/store/listingData.csv` is the authoritative listing file, `msix/out/store-import/` a gitignored copy
  of it.
- Write no credential, Seller ID, Tenant ID or client ID into any tracked file (rules 19 and 20); the release flow takes
  only the credential's expiry date, as a parameter.
- When Partner Center refuses something the rules did not predict, amend the contract first (comply or amend), then the
  builder.

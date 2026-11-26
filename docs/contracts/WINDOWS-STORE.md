# Pointer: WINDOWS-STORE

- **Id:** `WINDOWS-STORE`
- **Version:** 0.6 (draft; owner StreamsPlayer)
- **Home:** the shared contracts catalog, `windows-store/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer, `partial` - a declared consumer of the contract. This product hands Partner Center a packed MSIX
  ([`../../msix/`](../../msix/)) and a listing import file ([`../../tools/store/`](../../tools/store/)), and since E0156 (2026-11-17) also
  prepares and, behind the pre-authorized commit switch, submits its Store submission through the machine channel of section D (rules
  19 to 26) - `tools/store/submit-submission.ps1`, the release flow's Store step. First adopted 2026-11-12 (ticket E0133); read against the
  tree at 0.4 (E0132, E0133), 0.5 (E0148, E0151) and 0.6 (E0156, 2026-11-17).

What it covers: what the artifacts must be so that Partner Center accepts them and the user meets one honest page - the
reserved identity read back from the packed archive, an unsigned upload, the version remap, the notices inside the
archive, the listing import file (header-named columns, the export's dialect, the caps, the search terms) and,
since 0.4, the machine channel to the Store submission: what the channel accepts, refuses, records and never touches
(section D, rules 19 to 26, each rule marked **Requirement**, **Measured** or **UNVERIFIED** - the UNVERIFIED marks stand until conformance
rung 6, which this repo has not met). The order of channels and the click path are the canon's, not this contract's.

**Where this product stands (2026-11-17).** The registry row says `partial`; the held and the open parts are:

- Held: the identity lives in one file ([`../../msix/identity.json`](../../msix/identity.json), rule 19) and both the MSIX build and the
  channel step read it; [`../../msix/package-readback.ps1`](../../msix/package-readback.ps1) (wired in `msix/build-msix.ps1:244`, ticket E0134) reads the packed archive back and refuses a mismatched Identity, Version or PublisherDisplayName, any `AppxSignature.p7x` entry (rule 23 (c)) or a missing `THIRD-PARTY-NOTICES.txt`, and `-SelfSign` builds write `_selftest.msix` (rules 1 to 4, rung 3); the manifest declares the file association instead of relying on
  what the container hides (rule 5); the committed export holds all 13 language columns, every cell inside its cap and
  one `Title` and one `CopyrightTrademarkInformation` for every language (rules 7, 9, 14); the listing builder exits 1
  and writes nothing for a shipped language with no column, a cell over a cap and any break of the export's dialect, writes
  that dialect, lists every cell a `-Refresh` replaces with its length, offers `-VerifyExport` to check that a Partner Center re-export holds uploaded asset URLs in every `DesktopScreenshotN` cell (0.5 item A condition 3, ticket E0151), and both committed CSVs (the export and
  the fixture with the copy cells emptied) are pinned `-text` and round-trip byte for byte, the cases running as one fast
  test (rules 7 to 9, 12, 13, rungs 1 and 2, tickets E0135 and E0151, `cases=42 failed=0`); the product has its own age
  rating (rule 18); the release flow's third header verdict is the shared preflight (rules 23 (a), (b), (d), 25), which
  never fails a release after the tag (rule 21); and the write half of the channel - the Store step of the release
  ([`../../tools/store/submit-submission.ps1`](../../tools/store/submit-submission.ps1), ticket E0156) - re-runs that preflight inline,
  refuses a pending submission this product's own record does not hold (rule 23 (a), (f), never deleting one), refreshes only the
  thirteen `ReleaseNotes` fields of the draft through the JSON path (rule 24), proves the draft read-back before
  any commit, keeps the distinct `-Commit` switch, and writes the rule-26 record to `DEV/release-verdicts/`; its refusal half
  is stub-proven (`tools/store/test-store-submission.ps1`, rung 5 stub half).
- Rule 22 is held with a recorded difference: the per-release typed confirmation of the commit is replaced by the owner's
  standing pre-authorization of 2026-11-17 (ticket E0156), recorded as the dated exception row in the catalog registry and asked
  as the catalog proposal `PROPOSAL-2026-11-17-doc-html-translate-store-commit-preauthorization.md` (proposed; the proposal binds nobody, the
  exception covers the difference until the owner folds or rejects it).
- Open (dated exceptions in the catalog registry): the release notes of the 13 columns are translated
  (rule 16, a supported variant).
- Not yet run: one full channel run against the real Store - rung 5's real half and rung 6 (one real draft, then the first
  committed submission of the next release) - so every UNVERIFIED mark of section D stands; and a Partner Center import of a file made by the refusing
  builder (rung 4).

**What this repo owes it**

- Keep the Store identity in `msix/identity.json` and nowhere else; never change `IdentityName` or `Publisher`.
- Read the packed archive back before a package is offered for upload (E0134 closes this).
- Build the listing import file from the product's own fresh export with the refusals and caps of rules 7 to 9 (done,
  E0135); `tools/store/listingData.csv` is the authoritative listing file, `msix/out/store-import/` a gitignored copy
  of it.
- The machine channel re-reads the shared preflight in the same run it writes in, never writes a credential, Seller ID,
  Tenant ID or client ID anywhere (rules 19 and 20; the release flow takes only the credential's expiry date, as a parameter),
  and keeps the rule-26 evidence without account identifiers or upload addresses.
- When Partner Center refuses something the rules did not predict, amend the contract first (comply or amend), then the
  builder.

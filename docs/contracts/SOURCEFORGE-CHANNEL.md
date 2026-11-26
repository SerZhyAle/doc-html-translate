# Pointer: SOURCEFORGE-CHANNEL

- **Id:** `SOURCEFORGE-CHANNEL`
- **Version:** 0.1 draft
- **Home:** the shared contracts catalog, `sourceforge/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - prepared 2026-11-21 (ticket D2HTML-0165)
- **Wire carrier:** none (live SourceForge project forms and release downloads, read under contract section 4)

The SourceForge listing channel for this MIT-licensed Windows desktop and CLI product (`SerZhyAle/doc-html-translate`).

What the repository holds today (under promotion campaign ticket `D2HTML-0111`):
- **Listing draft** (`outreach/sourceforge.md`): eligibility (MIT, Windows desktop and CLI shapes, rule 1), slug `doc-html-translate` (18 ASCII bytes), Short Summary (64 UTF-8 bytes, rule 12), Full Description (330 characters, rule 12), homepage, GitHub issues support URL, `store.ms.logo` logo path, categories, preferred default Windows x64 ZIP download, Files and Source Code Git (`mount git`, no Tickets or Wiki, rules 6 and 11), and executor-tagged sequence.
- **Adapted SHA-256 script** (`outreach/sourceforge.md`): adapts the promote reference script to `[ValidatePattern('^v\d{2}\.\d{4}\.\d{4}$')]` and `SerZhyAle/doc-html-translate` (rules 16 and 17).
- **Published `v26.1117.1450` sidecars:** `.sha256` sidecars for `doc-html-translate-setup-26.1117.1450.exe` and `THIRD-PARTY-NOTICES.txt` were uploaded to GitHub release `v26.1117.1450` and read back on 2026-11-21 (all five assets of that release now have published `.sha256` files).
- **Go ledger** (`GO_LEDGER.md`): records the executed `v26.1117.1450` sidecar upload and holds separate named `2026-11-21` goes for project creation, Files import, Git import and metadata save.

**Conformance (`partial` 0.1).** Rules 1, 11 and 12 are prepared in the tree draft, and `v26.1117.1450` meets the exact-tag `.sha256` prerequisite on GitHub. All five live section 4 checks wait for the owner's attended browser sitting (`serzhyale` sign-in, CAPTCHA/code/phone PIN, Terms of Use box and Create press) so creation, Files and Git imports, release download SHA-256 checks and Admin > Metadata save and read-back run in one sitting (rules 2 to 18):
1. Project URL answering 200 with author username read back.
2. SourceForge Files folders compared against `gh release list`.
3. Every newest-release file downloaded through its SourceForge button and verified locally (`MATCH` or `MISMATCH`, or `integrity check not run` if a transfer stalls).
4. Saved metadata and default download choice read back from Admin > Metadata.
5. Project tools and empty tools left in place read back.

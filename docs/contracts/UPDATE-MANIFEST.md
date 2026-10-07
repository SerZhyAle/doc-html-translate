# Pointer: UPDATE-MANIFEST

- **Id:** `UPDATE-MANIFEST`
- **Version:** 0.11 draft
- **Home:** the shared contracts catalog, `app-update-feed/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** declared consumer; no UPDATE-MANIFEST client is implemented
- **Wire carrier:** `schemaVersion` (JSON int)

**0.11 review (2026-10-07, ticket 108).** Section 2 rule 6 compares decimal parts
numerically, including ten-digit compact stamps, and offers nothing for another shape.
The GUI and website resolve GitHub Releases instead; winget scripts read package metadata.
None parses the manifest of this contract. This is a declaration, not conformance evidence.
No comparison helper or fixture test is owed until a manifest client is implemented.
No catalog vectors exist. The adoption row stays `pending` with this missing implementation.

# Pointer: UPDATE-MANIFEST

- **Id:** `UPDATE-MANIFEST`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `app-update-feed/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - release discovery and winget package synchronization
- **Wire carrier:** `schemaVersion` (JSON int)

Release manifests and non-Store package delivery:
- Non-Store distribution through GitHub Releases and Windows Package Manager (`winget/`).
- Package integrity verified via SHA-256 digests in manifests.
- Packaged MSIX edition updates are managed natively via Microsoft Store.

**Conformance.** Package manifest verification and `scripts/verify-exe-version.ps1`.

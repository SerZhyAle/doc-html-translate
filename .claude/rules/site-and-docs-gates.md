---
paths:
  - "*.html"
  - "*/index.html"
  - "docs/**"
  - "sitemap.xml"
  - "configs/doc-registry-exclusions.jsonl"
  - "configs/site-held-addresses.jsonl"
  - "scripts/doc-registry.ps1"
  - "scripts/security-posture.ps1"
  - "scripts/site-addresses.ps1"
  - "scripts/release-notes.ps1"
  - "scripts/lib/docregistry.ps1"
  - "extension/manifest.json"
  - "extension/store/*.md"
  - "msix/AppxManifest.xml"
  - "msix/README.md"
---

# Site pages, documents and the whole-tree checks

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Whole-tree checks

- `check.ps1` also holds two whole-tree checks. `scripts/doc-registry.ps1`: every maintained document is a
  record of `docs/DOCUMENT_REGISTRY.jsonl` (record shape 1, declared in the stamp), every `.md` / `.html`
  file is claimed by a record or by `configs/doc-registry-exclusions.jsonl`, every announced page carries
  the SEO block, and `sitemap.xml` is generated - after adding or renaming a site page, run it with
  `-Generate`, never edit the sitemap. `scripts/security-posture.ps1`: the permission and network-surface
  inventories in `docs/security-posture.json` against `extension/manifest.json`, `msix/AppxManifest.xml`,
  the code and the dependency set; the privacy pages, `extension/store/PRIVACY.md`, the permission
  justifications in `extension/store/LISTING.md` and the runFullTrust text in `msix/README.md` are
  rendered from those rows - edit the rows, then run it with `-Render`. A new permission, network call
  or telemetry-shaped dependency fails it until a row covers it. `scripts/site-addresses.ps1`: every
  address of the site that a program, extension message, README, listing source, manifest or test holds is
  in `configs/site-held-addresses.jsonl` and still answers (page, query, section anchor); a new holder or a
  new address fails it until the list has it (`-Suggest` prints the record), and a moved page leaves a
  forwarder - the policy is [docs/SITE_ADDRESSES.md](../../docs/SITE_ADDRESSES.md). `scripts/release-notes.ps1`: the site's
  release-notes page is the render of `docs/release-notes.json` (one record per `v*` app tag, the same bullets in
  en/ru/uk, newest first); the check fails on a tag with no record, a record out of order or a hand-edited block,
  and a release adds its record first (`-Render` rewrites the page) - never edit the page per ticket. The same
  render writes the release facts of the other pages: the landing's newest-record block, the version beside each
  download and store button, the asset names. The extension's `ext-cws-v` / `ext-edge-v` tags name the version
  `node extension/build.mjs record-version` recorded there, and the store build stops on any other.

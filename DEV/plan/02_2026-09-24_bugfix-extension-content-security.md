# Strategic spec: 02_2026-09-24_bugfix-extension-content-security - Extension: untrusted documents stay inert

**Ticket:** 02_2026-09-24_bugfix-extension-content-security
**Status:** Draft
**Priority:** 60
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/02_2026-09-24_bugfix-extension-content-security/` (created by /spec-tech)
**Findings:** B14 B15 B16 B17 B18 B22 B24 B26 B29 (see the [findings register](../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The extension renders untrusted documents.
- **Script links in exports:** its sanitizer keeps script-scheme links. They are inert inside the viewer only because of the extension's content policy, and the "save as HTML" export drops that policy, so a saved file can run script when a link is clicked.
- **Exposed pages:** internal pages are exposed to every website, and the OCR broker accepts host messages from any sender. Any site can embed an OCR host, receive other tabs' jobs, or make the extension fetch local and intranet URLs.
- **Tracking:** documents can make the extension request remote resources (tracking pixels), and OCR re-fetches them with extension privileges.
- **Clobbering:** `name` attributes can shadow document methods and break the viewer.
- **Broken in-page links:** they break for HTML, Markdown and MOBI.
- **Other defects:**
  - The interception rule matches `.pdf` inside query strings.
  - Login-protected PDFs fail because the fetch sends no credentials.
  - The TAR reader drifts from the desktop edition on long names.
  - The build ships an unpinned OCR data file.

## 2. Goals
1. No document content can execute script, in the viewer or in any exported file.
2. Only extension pages can talk to the OCR broker and host, and internal pages are not reachable by websites except where a feature strictly needs it.
3. Remote resources referenced by a document are not fetched without the user's opt-in.
4. Document markup cannot interfere with the viewer's own code.
5. In-document links work for every format.
6. Automatic interception triggers only for URLs whose path is a supported document.
7. Documents behind a login open when the user can see them in the browser.
8. TAR comics read the same entries as the desktop edition.
9. The bundled OCR data is verified at build time.

**Non-goals:**
- Resource leaks (ticket `bugfix-extension-lifecycle-leaks`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** MV3 on Chrome and Edge, store review policies (a narrower web-accessible surface is favoured).
- **Performance:** n/a.
- **Localization:** the opt-in UI for remote content in all extension locales.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-extension-lifecycle-leaks`, `hotfix-epub-href-containment` (the extension's href decoding).
- **Platform constraints:** cross-edition. The desktop HTML output has the same `javascript:` question for TOC links (see `bugfix-reader-layer-and-single-page`). The TAR drift fix restores parity, so update docs/PARITY.md.
- **Copy/tone policy:** the remote-content notice.
- **Validation level:** sanitizer tests for script schemes, `xlink:href`, `name` and `#` links; a manifest check test for web-accessible resources; a DNR regex test table.
- **Owner sign-off:** required. Blocking remote images by default changes rendering.

## 4. Current architecture context
A single sanitizer handles HTML, Markdown and MOBI content by stripping tags, event attributes and
styles. The EPUB path has its own anchor rewrite. Every module is web-accessible. The OCR broker
and host communicate by broadcast messages identified by a host id. Downloads and OCR fetches use
default credentials.

## 5. Proposed approach
### 5.1 Pillars / modules
- **URL scheme allow-list:** in the sanitizer and the EPUB path, for links and resource attributes, including SVG links.
- **Self-protecting export:** a restrictive content policy is embedded in every exported file.
- **Minimal exposure:** only strictly needed resources are web-accessible, with dynamic URLs; frame guards on internal pages; sender verification in the broker.
- **Remote content policy:** off by default, with an opt-in per document or globally.
- **Name neutralization:** `name` attributes are prefixed or removed like ids.
- **Fragment rewriting:** in-page links are rewritten with the same prefix as ids.
- **Path-anchored interception:** the rule matches the document extension in the path only.
- **Credentialed fetch:** for the document itself; an HTML reply to a document request offers "open original".
- **TAR long names:** GNU and PAX name records are honoured.
- **Build integrity:** a pinned digest and a timeout for bundled data.

## 6. Open questions / research items
1. **Remote images default**
   - **Question:** block by default, or allow with a notice?
   - **Status:** Open.
2. **Which pages must stay web-accessible**
   - **Question:** which resources do the page-OCR frame and viewer redirects actually need?
   - **To find out:** trace every `getURL` and iframe use.
   - **Status:** Open.

## 7. Risks
- **Narrowing web-accessible resources breaks page OCR.** Likelihood: medium. Impact: the feature stops working. Mitigation: the §6.2 trace, plus a manual page-OCR check on three sites.
- **Blocking remote images makes some EPUBs look broken.** Likelihood: medium. Impact: user confusion. Mitigation: a visible notice with a one-click allow.

## 8. User impact (docs)
Extension docs: the remote content setting.

## 9. Architecture decisions (ADR)
**ADR-1: exports must be safe without the extension's policy.** Why: a file leaves the protective context the moment it is saved.

## 10. Links to other specs
`bugfix-extension-lifecycle-leaks`, `hotfix-epub-href-containment`, `bugfix-reader-layer-and-single-page`.

## 11. Done criteria (strategic)
1. A Markdown file with a `javascript:` link, exported and opened from disk, does nothing when the link is clicked.
2. A web page cannot load the extension's OCR pages in a frame.
3. An EPUB with a remote tracking image makes no request until the user allows it.
4. A Markdown document's footnote links work.
5. `https://site/viewer?file=a.pdf` is not intercepted.

## 12. Next step
`/spec-tech 02_2026-09-24_bugfix-extension-content-security`

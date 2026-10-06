# Site addresses held outside the site

The product site is served by GitHub Pages from the repository root. Other things hold its addresses: the
GUI, the CLI splash, the converted-page navbar, the extension, READMEs, listing sources, winget manifests and
the tests. A reader who follows one of them lands on a page of this site, so the address is a promise the
site keeps. This note is the policy; `configs/site-held-addresses.jsonl` is the list;
`scripts/site-addresses.ps1` is the gate. Contract: `SITE-STRUCTURE` rule 8 (pointer in
[contracts/SITE-STRUCTURE.md](contracts/SITE-STRUCTURE.md)).

## The list

`configs/site-held-addresses.jsonl`, one JSON record per line, two kinds:

- `held` - `holder` (a repo-relative file), `surface` (`program`, `extension`, `readme`, `store-listing`,
  `manifest`, `script`, `test` or `developer-doc`) and `addresses`, every address of this site that file
  carries, written exactly as the file writes it.
- `ignore` - `path` (a glob, `*` stays in one directory, `**/` spans any depth) and a `reason` of four words
  or more. A file ignored is not read. Used for what is the site itself (the pages the host serves), what is
  generated (`sitemap.xml`), and what is dated history (`DEV/`).

Only addresses of this site are held addresses. A link out to another site, including the sibling sites of
`SITE-FAMILY-MAP`, is not listed here.

## The gate

`scripts/site-addresses.ps1` runs inside `scripts/check.ps1`. It fails when:

- a file that is not ignored carries an address of this site that its record does not list (a new holder,
  or a new address in an old one);
- a listed address no longer answers: the file it names is not in the tree, its query is not `l=en|ru|uk`,
  or the section it names is not an id of that page;
- a listed holder is gone or no longer carries the address (the entry is stale);
- an ignore matches no file, or has no reason.

`-Suggest` prints the records a new holder needs. Adding the entry is a deliberate edit; the gate never
writes the list.

## Forwarder policy

1. **A published address never changes meaning.** A page that moves or is retired leaves a forwarder at the
   old address. A forwarder is a file, so the gate sees a moved page the moment its old file is gone.
2. **Shape.** A moved page: a small page that names the new address in a visible link, in
   `<link rel="canonical">` and in a `<meta http-equiv="refresh">`, `noindex`, in the three core languages.
   A retired page with no successor: a page that says so and links the landing.
3. **Registration.** A forwarder is an html file, so it needs a record in `docs/DOCUMENT_REGISTRY.jsonl`
   with `sitemap_exclude` and its reason; it is never listed in `sitemap.xml`.
4. **A section anchor that an outside surface links is never renamed.** The gate checks every listed
   address that names a section. If a section must go, the old id stays as an empty anchor at the same spot.
5. **The held list outlives the forwarder.** A forwarder is removed only when no listed holder carries the
   old address any more and a release has shipped without it; the winget and Store manifests of released
   versions are frozen, so their addresses stay answering for as long as those versions are installable.
6. **The language query is part of the scheme.** `?l=ru` and `?l=uk` on a page that carries its languages
   in-page; the per-language landings are folders (`de/`, `it/`, ..) with one page each. No holder builds an
   address from a locale scheme today; one that starts to lists each address it can produce.

No forwarder exists today: every listed address answers with the page it meant.

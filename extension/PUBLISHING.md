# Publishing the extension (Chrome Web Store + Edge Add-ons)

How to release `doc-html-translate-extension.zip` to the stores, and how much of it is automated.

## The one rule that shapes everything

**Neither store can create the listing or set listing metadata via an API.** The Chrome Web Store V2
API and the Edge Add-ons v1.1 API only *update an existing item*. So:

- **First release on each store: by hand, once.** You create the item, upload the first ZIP, and fill
  the Store listing + Privacy/screenshots in the dashboard. This is what produces the item IDs that the
  API needs in every URL.
- **Every release after that: one command** (`npm run release:cws`) or a git tag (CI). Build -> upload
  -> publish -> poll, fully scripted.
- Human review of each submission is always out-of-band: you can *submit*, not *approve*.

The scripts are dependency-free (Node builtins only, like `build.mjs`):
[scripts/publish-cws.mjs](scripts/publish-cws.mjs), [scripts/publish-edge.mjs](scripts/publish-edge.mjs),
[scripts/bump-version.mjs](scripts/bump-version.mjs).

---

## Identifiers (don't mix them up)

| Name | What it is | Where |
|------|------------|-------|
| `CWS_PUBLISHER_ID` | your developer/publisher GUID (the GUID in the devconsole URL) | devconsole URL / Account page |
| `CWS_EXTENSION_ID` | the 32-char **item** ID, created on first upload | dashboard, after step 1 |
| `EDGE_PRODUCT_ID` | the Edge product GUID | Partner Center > Edge > Overview > Extension identity |

The publisher GUID is **not** the item ID and **not** the OAuth client_id.

---

## Chrome Web Store

### One-time: create the item (manual)

1. Sign in to <https://chrome.google.com/webstore/devconsole> as `serhii.zhyhunenko@gmail.com`.
2. **New item** -> upload `extension/dist/doc-html-translate-extension.zip` (run `npm run build` to make it).
3. Fill **Store listing** (name, description from [store/LISTING.md](store/LISTING.md), category
   Productivity, screenshots) and **Privacy practices** (privacy-policy URL =
   `https://serzhyale.github.io/doc-html-translate/extension-privacy.html`, plus a justification for
   each of `declarativeNetRequest`, `storage`, `host_permissions <all_urls>` - all drafted in LISTING.md).
4. **Submit** and let it go live once. Copy the 32-char **Item ID** -> that is `CWS_EXTENSION_ID`.

### One-time: credentials (recommended = service account)

A Google service account has no token expiry, unlike an OAuth refresh token.

1. In Google Cloud Console: create/select a project, **Enable** the *Chrome Web Store API*.
2. IAM & Admin > Service Accounts > create one (no roles needed) > create a **JSON key**, download it.
3. In the CWS Developer Dashboard > **Account** section, add the service-account email
   (`...@...iam.gserviceaccount.com`). Only **one** service account is allowed per publisher.
4. Store the JSON key as the `CWS_SERVICE_ACCOUNT_KEY` secret (or save it as `extension/cws-key.json`
   and set `CWS_SERVICE_ACCOUNT_KEY_FILE=./cws-key.json` locally - it is git-ignored).

<details><summary>OAuth refresh-token fallback (only if you skip the service account)</summary>

Configure the OAuth consent screen (Audience = **External**, a personal Gmail cannot use Internal) and
**click PUBLISH APP to move it to "In production"** - otherwise the refresh token dies after exactly
7 days. Create a Desktop-app OAuth client, then run `npx chrome-webstore-upload-keys` and sign in as the
publisher account to mint `CWS_CLIENT_ID` / `CWS_CLIENT_SECRET` / `CWS_REFRESH_TOKEN`. The unverified-app
warning is expected and does not block the write scope.
</details>

### Every release (automated)

```sh
cd extension
npm run version:bump            # 0.1.0 -> 0.1.1  (REQUIRED: re-uploading the same version is rejected)
npm run release:cws             # build (vendor + zip) then upload + publish + status
```

`scripts/publish-cws.mjs` flags: `--upload-only` (review the draft by hand), `--publish-only`,
`--percentage 10` (staged rollout, only unlocks above ~10k users), `--source path.zip`.

---

## Microsoft Edge Add-ons (optional, second store)

### One-time (manual)

1. Sign in to Partner Center (<https://partner.microsoft.com/dashboard/microsoftedge>), create the
   extension, upload the same ZIP, fill listing + screenshots + the same privacy-policy URL, submit.
2. Copy the **Product ID** GUID (Microsoft Edge > Overview > Extension identity) -> `EDGE_PRODUCT_ID`.
3. Microsoft Edge > **Publish API** > Enable (v1.1) > **Create API credentials**. Copy the **Client ID**
   and the **API key** immediately (the key is shown once) -> `EDGE_CLIENT_ID`, `EDGE_API_KEY`.

### Every release (automated)

```sh
cd extension
npm run release:edge            # build then upload + publish (same ZIP Chrome uses)
```

The Edge v1.1 API key **expires about every 72 days** - rotate it in Partner Center and update
`EDGE_API_KEY`, or the pipeline starts returning 401. Do not use the deprecated v1 OAuth flow.

---

## Local setup

```sh
cd extension
cp .env.example .env            # then fill in the IDs / credentials (.env is git-ignored)
npm install                     # dev dep: pdfjs-dist (needed by `npm run build`)
npm run release:cws
```

## CI (GitHub Actions)

Chrome and Edge publish **independently** - separate workflows, separate tags, and therefore
separate build-time versions (each run does its own `npm run build`, which stamps a fresh version,
so a tag ships only to its own store). Both run on `ubuntu-latest`:

- [.github/workflows/publish-cws.yml](../.github/workflows/publish-cws.yml) - on an `ext-cws-v*`
  tag (or manual dispatch) -> Chrome Web Store only.
- [.github/workflows/publish-edge.yml](../.github/workflows/publish-edge.yml) - on an `ext-edge-v*`
  tag (or manual dispatch) -> Edge Add-ons only.

Set these in the repo:

- **Variables** (Settings > Secrets and variables > Actions > Variables): `CWS_PUBLISHER_ID`,
  `CWS_EXTENSION_ID` (Chrome); `EDGE_PRODUCT_ID` (Edge).
- **Secrets**: `CWS_SERVICE_ACCOUNT_KEY` (or the three `CWS_CLIENT_*`/`CWS_REFRESH_TOKEN`),
  `EDGE_CLIENT_ID`, `EDGE_API_KEY`.

The tag suffix is just a label (the published version is stamped at build time), so pick any
unused suffix per store:

```sh
git tag ext-cws-v1 && git push origin ext-cws-v1     # Chrome only
git tag ext-edge-v1 && git push origin ext-edge-v1   # Edge only, independent
```

## Gotchas

- **Bump the version every time** or the Chrome `:upload` call fails. `npm run version:bump` keeps
  `manifest.json` and `package.json` in sync.
- Both APIs publish with the item's **existing visibility**; if you change visibility you must publish
  once in the dashboard before the API will publish that visibility again.
- Chrome **V1 API is sunset 2026-10-15** - these scripts use V2 (`chromewebstore.googleapis.com`) only.
  Ignore older tutorials/wrappers that hit `www.googleapis.com/chromewebstore/v1.1`.
- `host_permissions <all_urls>` + `declarativeNetRequest` draws stricter/slower review; the API can
  submit but cannot speed up or guarantee approval.

Sources: Chrome <https://developer.chrome.com/docs/webstore/using-api>,
<https://developer.chrome.com/docs/webstore/service-accounts>,
<https://developer.chrome.com/blog/cws-api-v2>; Edge
<https://learn.microsoft.com/en-us/microsoft-edge/extensions-chromium/publish/api/using-addons-api>.

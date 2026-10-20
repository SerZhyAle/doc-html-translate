# Pointer: MEDIA-CLASSIFICATION

- **Id:** `MEDIA-CLASSIFICATION`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `media-classification/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, scoped - a reader of the category names, not a classifier (read at 0.12 on 2026-10-20, ticket 120)

The header of the contract does not list this product, and the amendment of 2026-10-17 says classification does not
require a product that never browses a media collection to classify one. This product converts the file it is given;
its dispatcher (`internal/pipeline/pipeline.go`) is "what a product does with a file", which rule 1 says a category
never describes. So no classifier conformance is claimed; the matching below is the product's own and agrees with
the categories wherever they overlap.

What the dispatcher does, case-insensitively on the lowercase extension (rule 3), and what refuses a binary file:
- **`IMAGE`:** `.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.bmp`, `.tif`, `.tiff`, `.jxl` (`internal/img`). `.avif`,
  `.heic`, `.ico`, `.wmf` and `.emf` are not converted; the binary-signature refusal (`txt.LooksBinary`) turns them away.
- **`BOOK`:** `.epub`, `.pdf`, `.mobi`, `.azw3`, `.fb2`, `.cbz`, `.cbr`, `.cb7`, `.cbt` (`internal/epub`, `internal/pdf`,
  `internal/mobi`, `internal/fb2`, `internal/comic`).
- **`DOCUMENT`:** `.rtf`, `.txt`, `.md`, `.html`, `.htm` (`internal/rtf`, `internal/txt`, `internal/md`,
  `internal/htmlconv`); an unknown extension is read as plain text only after the signature check.
- **`CONTAINER`:** `.fd-sec` (unwrapped by the installed FileDO, then dispatched as its inner type - `internal/fdsec`;
  see [FDSEC.md](FDSEC.md)); `.zip`, `.rar`, `.7z`, `.tar` are refused with a name.

**Scoped record - a permanent variant** (registry section 3; design record [`../PARITY.md`](../PARITY.md) "Contract
variants (ticket 120)"): `.svg` and a text `STREAM` file such as `.m3u8` are read as text and converted as prose, and
`.tif`, `.cb7`, `.cbt`, `.azw3`, `.html` and `.htm` are converted but absent from the contract's tables. Asked of the
owner in the `media-classification` proposal file of 2026-10-20: add those extensions and say that a converter that
never browses is a scoped reader.

**Conformance.** Package extract tests across `internal/epub`, `internal/pdf`, `internal/mobi`, `internal/fb2`,
`internal/comic`, `internal/img`, `internal/rtf`, `internal/txt`, `internal/md`, `internal/htmlconv`, plus `TestIsImage`
and `TestConvertible`. No test compares the contract's table with the dispatcher.

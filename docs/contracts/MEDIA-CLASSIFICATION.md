# Pointer: MEDIA-CLASSIFICATION

- **Id:** `MEDIA-CLASSIFICATION`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `media-classification/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - input format dispatch across books, documents, comics, and images

Normalized media kind mapping and extension classification:
- **`IMAGE`:** `.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.bmp`, `.tif`, `.tiff` (`internal/img`).
- **`BOOK`:** `.epub`, `.pdf`, `.mobi`, `.azw3`, `.fb2`, `.cbz`, `.cbr`, `.cb7`, `.cbt` (`internal/epub`, `internal/pdf`, `internal/mobi`, `internal/fb2`, `internal/comic`).
- **`DOCUMENT`:** `.rtf`, `.txt`, `.md`, `.html`, `.htm` (`internal/rtf`, `internal/txt`, `internal/md`, `internal/htmlconv`).
- Matching is case-insensitive, evaluated against lowercase extension.

**Conformance.** Package extract tests across `internal/epub`, `internal/pdf`, `internal/mobi`, `internal/fb2`, `internal/comic`, `internal/img`, `internal/rtf`, `internal/txt`, `internal/md`, `internal/htmlconv`.

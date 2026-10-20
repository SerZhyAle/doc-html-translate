// product-inventory.js - the one inventory behind the options page's About destination
// (SETTINGS-NAVIGATION 2.0 section 3 rules 3 and 4): the product's own links, what the extension
// bundles and needs, and the related projects. The page renders these records; it holds no second
// list. Labels are message keys with an English fallback, resolved through i18n.js when drawn.
//
// Only addresses the repository already publishes appear here (the site pages, the two store
// listings in extension/store/LISTING.md, the GitHub repository, the support mailbox). Nothing in
// this file is fetched: About is drawn from it without a single request, and a link is followed only
// when the reader chooses it.

// Links the reader may follow. `local` ones are files inside the extension package, so they open
// without leaving the browser's own extension origin.
export const LINKS = {
  site: { href: "https://serzhyale.github.io/doc-html-translate/", key: "linkProductSite", fallback: "Product site" },
  help: { href: "https://serzhyale.github.io/doc-html-translate/extension.html", key: "popupGuide", fallback: "Guide & help" },
  privacy: { href: "https://serzhyale.github.io/doc-html-translate/extension-privacy.html", key: "aboutPrivacy", fallback: "Privacy" },
  licence: { href: "https://github.com/SerZhyAle/doc-html-translate/blob/main/LICENSE", key: "aboutLicenceLink", fallback: "Licence text" },
  repository: { href: "https://github.com/SerZhyAle/doc-html-translate", key: "aboutRepo", fallback: "Source code" },
  notices: { href: "THIRD-PARTY-NOTICES.txt", key: "aboutNotices", fallback: "Third-party notices", local: true },
  support: { href: "mailto:sza@ukr.net", key: "linkFeedback", fallback: "Feedback" },
};

// Where updates come from. The extension never checks by itself: the browser's store delivers them.
export const UPDATE_SOURCES = [
  { id: "chrome", name: "Chrome Web Store", href: "https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic" },
  { id: "edge", name: "Edge Add-ons", href: "https://microsoftedge.microsoft.com/addons/detail/document-page-translator/anokfnnfiboaccbbpfkdaphejnkkhajh" },
  { id: "releases", key: "aboutReleases", fallback: "Download page", href: "https://github.com/SerZhyAle/doc-html-translate/releases/latest" },
];

// What the extension carries or needs in order to run, with each component's own licence text as it
// ships in the package (vendor/ is copied whole by build.mjs). Paths are relative to src/.
export const RESOURCES = [
  { id: "pdfjs", name: "pdf.js", key: "resPdfjs", fallback: "Opens PDF files", licence: "../vendor/LICENSE.pdfjs" },
  { id: "tesseract", name: "Tesseract.js", key: "resOcrEngine", fallback: "Recognizes text in pictures", licence: "../vendor/tesseract/LICENSE.tesseract" },
  { id: "tessdata", name: "tessdata_fast", key: "resOcrData", fallback: "Recognition data per language: English is built in, the others download when you ask", licence: null },
  { id: "foliate", name: "foliate-js", key: "resMobi", fallback: "Opens MOBI and AZW3 books", licence: "../vendor/foliate/LICENSE.foliate" },
  { id: "marked", name: "marked", key: "resMarkdown", fallback: "Reads Markdown files", licence: "../vendor/LICENSE.marked" },
  { id: "fflate", name: "fflate", key: "resZip", fallback: "Unpacks EPUB, comic and other ZIP-based files", licence: "../vendor/LICENSE.fflate" },
  { id: "jxl", name: "JPEG XL decoder", key: "resJxl", fallback: "Shows JPEG XL pictures", licence: "../vendor/jxl/LICENSE" },
  { id: "glyphs", name: "Material Icons", key: "resGlyphs", fallback: "The interface icons", licence: "THIRD-PARTY-NOTICES.txt" },
];

// Related projects of the same author.
export const RELATED = [
  { id: "desktop", name: "doc-html-translate for Windows", key: "relDesktop", fallback: "The Windows app of this product: converts the same documents to HTML, with translation by Google or a local Ollama", href: "https://serzhyale.github.io/doc-html-translate/" },
  { id: "filedo", name: "FileDO", key: "relFiledo", fallback: "Encrypted secret files - the Windows app of this product can read them", href: "https://serzhyale.github.io/FileDO/" },
  { id: "portal", name: "SZA", key: "relPortal", fallback: "All projects of the same author", href: "https://sza.od.ua" },
];

// _fixtures.mjs - the sample documents the screenshot run feeds to the viewer.
//
// Built in memory rather than read from disk, for the same reason the desktop side's
// tools/store/make-screenshot.ps1 writes its own sample: a listing capture must need no real
// book, must touch nothing tracked, and must produce the same frame on any machine. The
// corpus in test_doc/ is gitignored and local-only, so depending on it would make this script
// unrunnable everywhere else.
//
// The passage is the public-domain opening of Alice's Adventures in Wonderland - the same text
// the desktop screenshots use, so both editions' store listings show one book.
//
// Node builtins only, matching build.mjs and _lib.mjs.

import { deflateRawSync } from "node:zlib";

// ---- the shared sample text ------------------------------------------------

export const BOOK_TITLE = "Alice's Adventures in Wonderland";
export const BOOK_AUTHOR = "Lewis Carroll";

// Chapter titles are the book's own, so the table-of-contents frame looks like a real book's
// contents rather than "Chapter 1 .. Chapter 12". Only the first three carry their real text;
// the rest reuse it, because the contents frame is about the panel, not the prose behind it.
const CHAPTER_TITLES = [
  "CHAPTER I. Down the Rabbit-Hole",
  "CHAPTER II. The Pool of Tears",
  "CHAPTER III. A Caucus-Race and a Long Tale",
  "CHAPTER IV. The Rabbit Sends in a Little Bill",
  "CHAPTER V. Advice from a Caterpillar",
  "CHAPTER VI. Pig and Pepper",
  "CHAPTER VII. A Mad Tea-Party",
  "CHAPTER VIII. The Queen's Croquet-Ground",
  "CHAPTER IX. The Mock Turtle's Story",
  "CHAPTER X. The Lobster Quadrille",
  "CHAPTER XI. Who Stole the Tarts?",
  "CHAPTER XII. Alice's Evidence",
];

const CHAPTER_I = [
  'Alice was beginning to get very tired of sitting by her sister on the bank, and of having nothing to do: once or twice she had peeped into the book her sister was reading, but it had no pictures or conversations in it, "and what is the use of a book," thought Alice, "without pictures or conversations?"',
  "So she was considering in her own mind (as well as she could, for the hot day made her feel very sleepy and stupid), whether the pleasure of making a daisy-chain would be worth the trouble of getting up and picking the daisies, when suddenly a White Rabbit with pink eyes ran close by her.",
  'There was nothing so very remarkable in that; nor did Alice think it so very much out of the way to hear the Rabbit say to itself, "Oh dear! Oh dear! I shall be late!" But when the Rabbit actually took a watch out of its waistcoat-pocket, and looked at it, and then hurried on, Alice started to her feet, for it flashed across her mind that she had never before seen a rabbit with either a waistcoat-pocket, or a watch to take out of it, and burning with curiosity, she ran across the field after it, and fortunately was just in time to see it pop down a large rabbit-hole under the hedge.',
  "In another moment down went Alice after it, never once considering how in the world she was to get out again.",
  "The rabbit-hole went straight on like a tunnel for some way, and then dipped suddenly down, so suddenly that Alice had not a moment to think about stopping herself before she found herself falling down a very deep well.",
  "Either the well was very deep, or she fell very slowly, for she had plenty of time as she went down to look about her and to wonder what was going to happen next. First, she tried to look down and make out what she was coming to, but it was too dark to see anything; then she looked at the sides of the well, and noticed that they were filled with cupboards and book-shelves: here and there she saw maps and pictures hung upon pegs.",
  'She took down a jar from one of the shelves as she passed; it was labelled "ORANGE MARMALADE," but to her great disappointment it was empty: she did not like to drop the jar for fear of killing somebody underneath, so managed to put it into one of the cupboards as she fell past it.',
  "\"Well!\" thought Alice to herself, \"after such a fall as this, I shall think nothing of tumbling down stairs! How brave they'll all think me at home! Why, I wouldn't say anything about it, even if I fell off the top of the house!\" (Which was very likely true.)",
];

const CHAPTER_II = [
  "\"Curiouser and curiouser!\" cried Alice (she was so much surprised, that for the moment she quite forgot how to speak good English); \"now I'm opening out like the largest telescope that ever was! Good-bye, feet!\" (for when she looked down at her feet, they seemed to be almost out of sight, they were getting so far off).",
  "\"Oh, my poor little feet, I wonder who will put on your shoes and stockings for you now, dears? I'm sure I shan't be able! I shall be a great deal too far off to trouble myself about you: you must manage the best way you can; but I must be kind to them,\" thought Alice, \"or perhaps they won't walk the way I want to go!\"",
];

const CHAPTER_III = [
  "They were indeed a queer-looking party that assembled on the bank - the birds with draggled feathers, the animals with their fur clinging close to them, and all dripping wet, cross, and uncomfortable.",
  "The first question of course was, how to get dry again: they had a consultation about this, and after a few minutes it seemed quite natural to Alice to find herself talking familiarly with them, as if she had known them all her life.",
];

// chapters pairs every title with the paragraphs shown under it.
function chapters() {
  const bodies = [CHAPTER_I, CHAPTER_II, CHAPTER_III];
  return CHAPTER_TITLES.map((title, i) => ({ title, paras: bodies[i % bodies.length] }));
}

// ---- PDF -------------------------------------------------------------------

const PAGE_W = 612;
const PAGE_H = 792;
const MARGIN = 72;
const FONT_SIZE = 11;
const LEADING = 15;
// Helvetica at 11 pt averages a little under 5.5 pt per character, so 468 pt of text width
// holds roughly 85 characters. Wrapping a touch short of that keeps every line inside the
// margin without measuring glyph widths.
const WRAP_COLS = 84;
const LINES_PER_PAGE = Math.floor((PAGE_H - 2 * MARGIN) / LEADING);

function wrap(text, cols) {
  const out = [];
  let line = "";
  for (const word of text.split(/\s+/)) {
    if (!line) line = word;
    else if (line.length + 1 + word.length <= cols) line += ` ${word}`;
    else {
      out.push(line);
      line = word;
    }
  }
  if (line) out.push(line);
  return out;
}

// pdfString escapes the three characters a PDF literal string cannot carry raw. The sample text
// is ASCII, so no encoding question arises with the standard Helvetica font.
function pdfString(s) {
  return s.replace(/\\/g, "\\\\").replace(/\(/g, "\\(").replace(/\)/g, "\\)");
}

// samplePdf returns a small multi-page PDF of the passage: uncompressed content streams and the
// standard Helvetica font, which needs no embedding. pdf.js reads it as it would any text PDF,
// which is the point - the frame must show the real reflow path, not a special case.
export function samplePdf() {
  const lines = [BOOK_TITLE, `by ${BOOK_AUTHOR}`, ""];
  for (const ch of chapters()) {
    lines.push(ch.title, "");
    for (const p of ch.paras) {
      lines.push(...wrap(p, WRAP_COLS), "");
    }
  }

  const pages = [];
  for (let i = 0; i < lines.length; i += LINES_PER_PAGE) {
    pages.push(lines.slice(i, i + LINES_PER_PAGE));
  }

  // Object numbering: 1 catalog, 2 page tree, 3 font, then a page and a content stream per page.
  const objects = [];
  const pageIDs = pages.map((_, i) => 4 + i * 2);
  objects[1] = "<< /Type /Catalog /Pages 2 0 R >>";
  objects[2] = `<< /Type /Pages /Kids [${pageIDs.map((id) => `${id} 0 R`).join(" ")}] /Count ${pages.length} >>`;
  objects[3] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>";

  pages.forEach((pageLines, i) => {
    const pageID = pageIDs[i];
    const contentID = pageID + 1;
    const body = [
      "BT",
      `/F1 ${FONT_SIZE} Tf`,
      `${LEADING} TL`,
      `${MARGIN} ${PAGE_H - MARGIN} Td`,
      ...pageLines.map((l) => (l ? `(${pdfString(l)}) Tj T*` : "T*")),
      "ET",
    ].join("\n");
    objects[pageID] =
      `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${PAGE_W} ${PAGE_H}] ` +
      `/Resources << /Font << /F1 3 0 R >> >> /Contents ${contentID} 0 R >>`;
    objects[contentID] = `<< /Length ${Buffer.byteLength(body, "latin1")} >>\nstream\n${body}\nendstream`;
  });

  // Serialize, recording each object's byte offset for the cross-reference table.
  const chunks = [];
  let offset = 0;
  const push = (s) => {
    const b = Buffer.from(s, "latin1");
    chunks.push(b);
    offset += b.length;
  };
  const offsets = [];

  push("%PDF-1.4\n");
  for (let id = 1; id < objects.length; id++) {
    if (!objects[id]) continue;
    offsets[id] = offset;
    push(`${id} 0 obj\n${objects[id]}\nendobj\n`);
  }
  const xrefStart = offset;
  const count = objects.length;
  let xref = `xref\n0 ${count}\n0000000000 65535 f \n`;
  for (let id = 1; id < count; id++) {
    xref += `${String(offsets[id] ?? 0).padStart(10, "0")} 00000 n \n`;
  }
  push(xref);
  push(`trailer\n<< /Size ${count} /Root 1 0 R >>\nstartxref\n${xrefStart}\n%%EOF\n`);

  return Buffer.concat(chunks);
}

// ---- EPUB ------------------------------------------------------------------

// crc32 over a buffer. A zip entry's header carries it, and readers reject a wrong one, so the
// table is not optional even for a fixture.
const CRC_TABLE = (() => {
  const t = new Int32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c;
  }
  return t;
})();

function crc32(buf) {
  let c = -1;
  for (let i = 0; i < buf.length; i++) c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
  return (c ^ -1) >>> 0;
}

// zip writes a minimal archive. Timestamps are fixed at the DOS epoch so two runs of this
// script produce byte-identical bytes - a screenshot diff should come from the UI, never from a
// clock. `store` forces method 0, which the EPUB spec requires for the mimetype entry.
function zip(entries) {
  const locals = [];
  const central = [];
  let offset = 0;

  for (const { name, data, store } of entries) {
    const nameBuf = Buffer.from(name, "utf8");
    const raw = Buffer.isBuffer(data) ? data : Buffer.from(data, "utf8");
    const body = store ? raw : deflateRawSync(raw);
    const method = store ? 0 : 8;
    const crc = crc32(raw);

    const local = Buffer.alloc(30);
    local.writeUInt32LE(0x04034b50, 0);
    local.writeUInt16LE(20, 4); // version needed
    local.writeUInt16LE(0, 6); // flags
    local.writeUInt16LE(method, 8);
    local.writeUInt16LE(0, 10); // mod time
    local.writeUInt16LE(0x0021, 12); // mod date - 1980-01-01
    local.writeUInt32LE(crc, 14);
    local.writeUInt32LE(body.length, 18);
    local.writeUInt32LE(raw.length, 22);
    local.writeUInt16LE(nameBuf.length, 26);
    local.writeUInt16LE(0, 28);
    locals.push(local, nameBuf, body);

    const dir = Buffer.alloc(46);
    dir.writeUInt32LE(0x02014b50, 0);
    dir.writeUInt16LE(20, 4); // version made by
    dir.writeUInt16LE(20, 6); // version needed
    dir.writeUInt16LE(0, 8); // flags
    dir.writeUInt16LE(method, 10);
    dir.writeUInt16LE(0, 12);
    dir.writeUInt16LE(0x0021, 14);
    dir.writeUInt32LE(crc, 16);
    dir.writeUInt32LE(body.length, 20);
    dir.writeUInt32LE(raw.length, 24);
    dir.writeUInt16LE(nameBuf.length, 28);
    dir.writeUInt32LE(offset, 42);
    central.push(dir, nameBuf);

    offset += local.length + nameBuf.length + body.length;
  }

  const dirBuf = Buffer.concat(central);
  const eocd = Buffer.alloc(22);
  eocd.writeUInt32LE(0x06054b50, 0);
  eocd.writeUInt16LE(entries.length, 8);
  eocd.writeUInt16LE(entries.length, 10);
  eocd.writeUInt32LE(dirBuf.length, 12);
  eocd.writeUInt32LE(offset, 16);

  return Buffer.concat([...locals, dirBuf, eocd]);
}

function xhtml(title, body) {
  return `<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head><title>${title}</title></head>
<body>
${body}
</body>
</html>
`;
}

function escapeXml(s) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// sampleEpub returns an EPUB 3 with a real navigation document, so the viewer's contents panel
// is populated by the same nav path a shipped book uses (epub.js prefers the manifest item with
// properties="nav"), not by a fallback.
export function sampleEpub() {
  const chs = chapters();
  const files = chs.map((ch, i) => ({
    id: `ch${String(i + 1).padStart(2, "0")}`,
    href: `ch${String(i + 1).padStart(2, "0")}.xhtml`,
    title: ch.title,
    paras: ch.paras,
  }));

  const navItems = files
    .map((f) => `      <li><a href="${f.href}">${escapeXml(f.title)}</a></li>`)
    .join("\n");

  const nav = xhtml(
    "Contents",
    `<nav xmlns:epub="http://www.idpf.org/2007/ops" epub:type="toc" id="toc">
    <h1>Contents</h1>
    <ol>
${navItems}
    </ol>
  </nav>`,
  );

  const opf = `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="bookid">urn:uuid:doc-html-translate-listing-sample</dc:identifier>
    <dc:title>${escapeXml(BOOK_TITLE)}</dc:title>
    <dc:creator>${escapeXml(BOOK_AUTHOR)}</dc:creator>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
${files.map((f) => `    <item id="${f.id}" href="${f.href}" media-type="application/xhtml+xml"/>`).join("\n")}
  </manifest>
  <spine>
${files.map((f) => `    <itemref idref="${f.id}"/>`).join("\n")}
  </spine>
</package>
`;

  const container = `<?xml version="1.0" encoding="utf-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`;

  const entries = [
    { name: "mimetype", data: "application/epub+zip", store: true },
    { name: "META-INF/container.xml", data: container },
    { name: "OEBPS/content.opf", data: opf },
    { name: "OEBPS/nav.xhtml", data: nav },
  ];

  files.forEach((f, i) => {
    const heading = i === 0 ? `<h1>${escapeXml(BOOK_TITLE)}</h1>\n<p><i>by ${escapeXml(BOOK_AUTHOR)}</i></p>\n` : "";
    const body = `${heading}<h2>${escapeXml(f.title)}</h2>\n${f.paras.map((p) => `<p>${escapeXml(p)}</p>`).join("\n")}`;
    entries.push({ name: `OEBPS/${f.href}`, data: xhtml(f.title, body) });
  });

  return zip(entries);
}

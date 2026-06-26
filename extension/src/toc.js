// toc.js - build a nested table of contents from the PDF outline, mirroring
// internal/pdf/toc.go (buildPDFTOC / bookmarksToEntries). PDF.js getOutline()
// already returns a nested {title, dest, items} tree, so the work is resolving
// each entry's destination to a 1-based page number and pruning empty nodes.
//
// The async resolver takes a `pdf` (PDFDocumentProxy or a test double exposing
// getOutline/getDestination/getPageIndex). resolveOutline is exported separately
// so the pruning/structure logic can be unit tested with a mock pdf.

// destToPageIndex resolves a PDF.js outline `dest` (named string or explicit
// array) to a 0-based page index, or null when it can't be resolved (mirrors the
// Go "unresolved destinations come back as 0" guard).
export async function destToPageIndex(pdf, dest) {
  try {
    let explicit = dest;
    if (typeof dest === "string") explicit = await pdf.getDestination(dest);
    if (!Array.isArray(explicit) || explicit.length === 0) return null;
    const ref = explicit[0];
    if (ref == null) return null;
    const idx = await pdf.getPageIndex(ref);
    return typeof idx === "number" ? idx : null;
  } catch {
    return null;
  }
}

// resolveOutline maps PDF.js outline nodes to { title, page, children }.
// page is 1-based (or null when unresolved). Nodes with no title, no page, and no
// children are dropped, exactly like bookmarksToEntries.
export async function resolveOutline(pdf, nodes) {
  const out = [];
  for (const node of nodes || []) {
    const children = node.items && node.items.length
      ? await resolveOutline(pdf, node.items)
      : [];
    const pageIndex = await destToPageIndex(pdf, node.dest);
    const title = (node.title || "").trim();
    if (title === "" && pageIndex == null && children.length === 0) continue;
    out.push({
      title,
      page: pageIndex == null ? null : pageIndex + 1,
      children,
    });
  }
  return out;
}

// buildToc reads the document outline and returns the resolved nested entries,
// or [] when the PDF has no usable bookmarks.
export async function buildToc(pdf) {
  let outline;
  try {
    outline = await pdf.getOutline();
  } catch {
    return [];
  }
  if (!outline || outline.length === 0) return [];
  return resolveOutline(pdf, outline);
}

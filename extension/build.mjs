// build.mjs - dev tooling for the extension.
//
//   node build.mjs vendor   copy the pieces of pdfjs-dist we ship into vendor/
//   node build.mjs zip      produce a store-ready dist/doc-html-translate-extension.zip
//
// We vendor the *non-minified* pdfjs build on purpose: Chrome Web Store review is
// friendlier to readable third-party code than to minified blobs, and the size cost
// (~3 MB) is irrelevant for a locally-installed extension.

import { cp, mkdir, rm, stat } from "node:fs/promises";
import { existsSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join, resolve } from "node:path";

const root = dirname(fileURLToPath(import.meta.url));
const pdfjs = join(root, "node_modules", "pdfjs-dist");
const vendor = join(root, "vendor");

async function vendorPdfjs() {
  if (!existsSync(pdfjs)) {
    console.error("pdfjs-dist not installed. Run `npm install` first.");
    process.exit(1);
  }
  await rm(vendor, { recursive: true, force: true });
  await mkdir(vendor, { recursive: true });

  // Runtime files: the main entry + the worker. ES module builds.
  await cp(join(pdfjs, "build", "pdf.mjs"), join(vendor, "pdf.mjs"));
  await cp(join(pdfjs, "build", "pdf.worker.mjs"), join(vendor, "pdf.worker.mjs"));

  // Character maps (CJK etc.) and standard-font fallbacks improve text extraction
  // quality for PDFs that don't embed their fonts. Pointed at via getDocument().
  await cp(join(pdfjs, "cmaps"), join(vendor, "cmaps"), { recursive: true });
  await cp(join(pdfjs, "standard_fonts"), join(vendor, "standard_fonts"), { recursive: true });

  // Carry the upstream licence next to the vendored code.
  await cp(join(pdfjs, "LICENSE"), join(vendor, "LICENSE.pdfjs"));

  const version = JSON.parse(
    await import("node:fs/promises").then((fs) =>
      fs.readFile(join(pdfjs, "package.json"), "utf8"),
    ),
  ).version;
  console.log(`Vendored pdfjs-dist@${version} into vendor/`);
}

// Exactly what the extension needs at runtime. An allow-list (not a deny-list) so
// dev/listing artifacts - store/, README.md, tests - can never silently leak into
// the store package.
const PACKAGE = ["manifest.json", "src", "vendor", "icons"];

async function zip() {
  // The package is broken without the vendored pdfjs build; fail loudly rather
  // than shipping a viewer whose `import ../vendor/pdf.mjs` 404s.
  if (!existsSync(join(root, "vendor", "pdf.mjs"))) {
    console.error("vendor/pdf.mjs missing - run `npm run vendor` before zipping.");
    process.exit(1);
  }
  for (const e of PACKAGE) {
    if (!existsSync(join(root, e))) {
      console.error(`required package entry missing: ${e}`);
      process.exit(1);
    }
  }

  const dist = join(root, "dist");
  await mkdir(dist, { recursive: true });
  const out = join(dist, "doc-html-translate-extension.zip");
  await rm(out, { force: true });

  // Dependency-free zip, cross-platform so CI can run on cheap Linux runners:
  //   - Windows (dev machines): PowerShell Compress-Archive.
  //   - Linux/macOS (CI, ubuntu-latest): the `zip` CLI, run from `root` so the
  //     entries land at the archive root (matching Compress-Archive's layout).
  let r;
  if (process.platform === "win32") {
    const paths = PACKAGE.map((e) => `'${resolve(root, e).replace(/'/g, "''")}'`).join(",");
    const ps = `Compress-Archive -Path ${paths} -DestinationPath '${out.replace(/'/g, "''")}' -Force`;
    r = spawnSync("powershell", ["-NoProfile", "-Command", ps], { stdio: "inherit" });
  } else {
    // -r recurse dirs, -q quiet; relative entry names preserved via cwd: root.
    r = spawnSync("zip", ["-r", "-q", out, ...PACKAGE], { cwd: root, stdio: "inherit" });
  }
  if (r.status !== 0) {
    console.error("zip failed");
    process.exit(1);
  }
  const { size } = await stat(out);
  console.log(`Wrote ${out} (${(size / 1024 / 1024).toFixed(1)} MB)`);
}

const cmd = process.argv[2];
if (cmd === "vendor") await vendorPdfjs();
else if (cmd === "zip") await zip();
else {
  console.error("usage: node build.mjs <vendor|zip>");
  process.exit(1);
}

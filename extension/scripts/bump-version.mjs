// bump-version.mjs - bump the extension version in manifest.json and package.json in
// lockstep. The Chrome Web Store :upload call is rejected unless manifest.json's
// version is incremented, so do this before every release.
//
//   node scripts/bump-version.mjs            patch  (0.1.0 -> 0.1.1)
//   node scripts/bump-version.mjs minor      0.1.0 -> 0.2.0
//   node scripts/bump-version.mjs major      0.1.0 -> 1.0.0
//   node scripts/bump-version.mjs 1.2.3      set exactly

import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const arg = process.argv[2] || "patch";

function next(version, kind) {
  if (/^\d+\.\d+\.\d+$/.test(kind)) return kind; // explicit version
  const [maj, min, pat] = version.split(".").map((n) => parseInt(n, 10) || 0);
  if (kind === "major") return `${maj + 1}.0.0`;
  if (kind === "minor") return `${maj}.${min + 1}.0`;
  if (kind === "patch") return `${maj}.${min}.${pat + 1}`;
  throw new Error(`unknown bump kind: ${kind} (use patch|minor|major|x.y.z)`);
}

async function patchFile(file) {
  const path = resolve(process.cwd(), file);
  const json = JSON.parse(await readFile(path, "utf8"));
  const updated = next(json.version, arg);
  json.version = updated;
  await writeFile(path, JSON.stringify(json, null, 2) + "\n");
  return updated;
}

const v = await patchFile("manifest.json");
await patchFile("package.json");
console.log(`Version bumped to ${v} (manifest.json + package.json).`);

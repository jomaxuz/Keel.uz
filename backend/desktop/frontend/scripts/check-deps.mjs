#!/usr/bin/env node
// Every package the till needs, listed before a build goes looking for them.
//
// ⚠️ **This exists because the same failure happened twice, one package at a
// time.** The till compiles screens that live in `frontend/src`, and those files
// import packages of their own. A build only reports the first one it cannot
// resolve, so a missing dependency costs a full build, a push and somebody
// else's afternoon — and then the next one costs the same again. This walks the
// import graph from the entry point and names all of them at once.
//
// Run it after mounting a new shared screen: `npm run check-deps`.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const shared = path.resolve(root, "../../../frontend/src");
const entry = path.join(root, "src", "main.tsx");

// The two Next modules this app supplies itself (src/shims).
const SHIMMED = new Set(["next/navigation", "next/headers"]);

const exts = [".ts", ".tsx", ".js", ".jsx", ".mjs"];
const resolveFile = (p) => {
  if (fs.existsSync(p) && fs.statSync(p).isFile()) return p;
  for (const e of exts) if (fs.existsSync(p + e)) return p + e;
  for (const e of exts) {
    const i = path.join(p, "index" + e);
    if (fs.existsSync(i)) return i;
  }
  return null;
};

const seen = new Set();
const bare = new Map(); // package -> the file that first asked for it

function walk(file) {
  if (seen.has(file)) return;
  seen.add(file);
  const src = fs.readFileSync(file, "utf8");
  const specs = [];
  for (const re of [
    /^\s*(?:import|export)\s[^;]*?\sfrom\s*["']([^"']+)["']/gm,
    /^\s*import\s*["']([^"']+)["']/gm,
    /\bimport\(\s*["']([^"']+)["']\s*\)/g,
  ]) {
    let m;
    while ((m = re.exec(src))) specs.push(m[1]);
  }

  for (const spec of specs) {
    let target = null;
    if (spec.startsWith("@/")) target = path.join(shared, spec.slice(2));
    else if (spec.startsWith(".")) target = path.resolve(path.dirname(file), spec);
    else {
      if (SHIMMED.has(spec) || spec.startsWith("node:")) continue;
      const pkg = spec.startsWith("@")
        ? spec.split("/").slice(0, 2).join("/")
        : spec.split("/")[0];
      if (!bare.has(pkg)) bare.set(pkg, path.relative(root, file));
      continue;
    }
    if (target.endsWith(".css")) continue;
    const f = resolveFile(target);
    if (f) walk(f);
  }
}

walk(entry);

const pkg = JSON.parse(fs.readFileSync(path.join(root, "package.json"), "utf8"));
const have = new Set([
  ...Object.keys(pkg.dependencies ?? {}),
  ...Object.keys(pkg.devDependencies ?? {}),
]);

const missing = [...bare].filter(([p]) => !have.has(p)).sort();
console.log(`${seen.size} ta fayl, ${bare.size} ta paket.`);
if (missing.length === 0) {
  console.log("Hammasi package.json da.");
  process.exit(0);
}
console.error("\npackage.json da YO'Q:");
for (const [p, from] of missing) console.error(`  ${p.padEnd(24)} ← ${from}`);
console.error("\nBularsiz build yiqiladi (bittalab, har safar bittasi).");
process.exit(1);

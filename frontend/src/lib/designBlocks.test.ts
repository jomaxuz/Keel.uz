import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

// The band types the site can draw, and the band types the server lets through.
//
// ⚠️ **These two lists are joined by nothing the compiler can see, and they came
// apart twice.** The first time it was `navbar`: the console could store it, the
// operator could configure it, and `DesignRenderer` had no entry — so every
// setting was saved and the page never changed. The second time it was the
// schema-driven bands (`rich-text`, `image-text`, `banner`, `banners`): the
// renderer had components for all four and `models.blockVariants` did not list
// them, so `Sanitize` dropped them out of the document on the way to the page.
//
// Both failures look identical from the operator's chair — the band is in the
// editor, it is saved, it is published, and the live site is unchanged — and
// neither produces an error anywhere. A test is the only thing that can see it,
// which is why this one reads the Go source rather than trusting a copy.

/** Every band type `Sanitize` will let through, read out of the model. */
function serverBlocks(): string[] {
  const src = readFileSync("../backend/internal/models/design.go", "utf8");
  const body = src.slice(
    src.indexOf("var blockVariants = map[string][]string{"),
    src.indexOf("\n}", src.indexOf("var blockVariants")),
  );
  const names = new Map<string, string>();
  // The map is keyed by the constants, so the constants are resolved first.
  for (const [, konst, value] of src.matchAll(
    /\n\t(Block[A-Za-z]+)\s*=\s*"([a-z-]+)"/g,
  )) {
    names.set(konst, value);
  }
  const out: string[] = [];
  for (const [, key] of body.matchAll(/\n\t(Block[A-Za-z]+):/g)) {
    const name = names.get(key);
    expect(name, `${key} has no constant`).toBeTruthy();
    out.push(name!);
  }
  return out.sort();
}

/** Every band type the site knows how to draw. */
function renderableBlocks(): string[] {
  const src = readFileSync("../frontend/src/components/design/DesignRenderer.tsx", "utf8");
  const body = src.slice(
    src.indexOf("const BLOCKS = {"),
    src.indexOf("} as const;", src.indexOf("const BLOCKS = {")),
  );
  const out: string[] = [];
  for (const [, key] of body.matchAll(/\n\s+"?([a-z-]+)"?:/g)) out.push(key);
  // ⚠️ The four that are not in that map, and each for its own reason rather
  // than by omission: the two free-drawing bands are handled before the lookup
  // (a canvas has no fixed component), and the bar and the footer are the site's
  // own shell — drawn above and below `<main>`, on every page, from the settings
  // this band carries (lib/siteChrome.ts).
  out.push("canvas", "popup", "navbar", "footer");
  return out.sort();
}

/** The settings keys the console draws into the page instead of reloading it. */
function patchedKeys(): string[] {
  const src = readFileSync("../keel-site/src/lib/designDiff.ts", "utf8");
  const body = src.slice(src.indexOf("const TEXT_KEYS"), src.indexOf("]);", src.indexOf("const TEXT_KEYS")));
  return [...body.matchAll(/"([a-zA-Z]+)"/g)].map(([, k]) => k);
}

/** Every settings key the renderer marks the printed words of. */
function markedKeys(): Set<string> {
  const out = new Set<string>();
  for (const file of [
    "../frontend/src/components/design/SchemaBlocks.tsx",
    "../frontend/src/components/design/blocks.tsx",
  ]) {
    for (const [, key] of readFileSync(file, "utf8").matchAll(/data-keel-set="([a-zA-Z]+)"/g)) {
      out.add(key);
    }
  }
  return out;
}

describe("the settings the console draws in place", () => {
  // ⚠️ **A key the console patches and the renderer does not mark is a silent
  // no-op**: the operator types a heading, the preview does not change, and
  // nothing anywhere says why. The console's list is the promise; the
  // attributes are whether it can be kept.
  it("are all marked in the page that prints them", () => {
    const marked = markedKeys();
    expect(patchedKeys().length).toBeGreaterThan(3);
    for (const key of patchedKeys()) {
      expect(marked, `settings key "${key}" is patched but never marked`).toContain(key);
    }
  });
});

describe("the bands a design may contain", () => {
  it("are the same list on the server and on the site", () => {
    expect(serverBlocks()).toEqual(renderableBlocks());
  });

  it("finds both lists at all", () => {
    // A regex that matched nothing would make the assertion above pass by
    // comparing two empty arrays, which is the way this kind of test dies.
    expect(serverBlocks().length).toBeGreaterThan(10);
    expect(renderableBlocks()).toContain("image-text");
  });
});

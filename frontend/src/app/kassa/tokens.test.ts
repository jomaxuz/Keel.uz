/**
 * Every `var(--till-…)` the screens use is actually declared.
 *
 * ⚠️ **This is here because of a bug nobody could see.** The design pass
 * renamed the till's tokens, and five selected states kept reaching for
 * `--till-action`, which no longer existed. `rgb(var(--till-action))` is not an
 * error in CSS — it is simply an invalid colour, so the background never
 * painted and `text-white` was left on a white surface. The chosen table, the
 * chosen portion size and the chosen payment method were all invisible, and
 * nothing failed: not the build, not the types, not the flow tests, which
 * assert what a control does rather than what colour it is.
 */

import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

const ROOT = join(process.cwd(), "src");
const CSS = join(ROOT, "app", "globals.css");

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    if (statSync(path).isDirectory()) return walk(path);
    // ⚠️ Tests excluded, and this one is the reason: the comment above names
    // the token that broke, and a scanner that reads its own explanation
    // reports the bug it was written to catch.
    if (/\.test\.tsx?$/.test(entry)) return [];
    return /\.(tsx?|css)$/.test(entry) ? [path] : [];
  });
}

describe("the till's design tokens", () => {
  it("are all declared in globals.css", () => {
    const css = readFileSync(CSS, "utf8");
    const declared = new Set(
      [...css.matchAll(/^\s*(--till-[a-z0-9-]+)\s*:/gim)].map((m) => m[1]),
    );
    expect(declared.size).toBeGreaterThan(5);

    const used = new Map<string, string[]>();
    for (const file of walk(ROOT)) {
      const text = readFileSync(file, "utf8");
      for (const m of text.matchAll(/var\((--till-[a-z0-9-]+)\)/gi)) {
        const name = m[1].toLowerCase();
        used.set(name, [...(used.get(name) ?? []), file]);
      }
    }
    expect(used.size).toBeGreaterThan(0);

    const missing = [...used.entries()]
      .filter(([name]) => !declared.has(name))
      .map(([name, files]) => `${name} — ${files.join(", ")}`);
    expect(missing).toEqual([]);
  });
});

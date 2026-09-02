// The three surfaces that mount a till, and the seam between them.
//
// ⚠️ **The bug this seals shipped, and nothing on the screen looked broken.**
// `kassa/layout.tsx` and `zal/layout.tsx` are Next route conventions — not
// importable — so the Windows application (backend/desktop) held a hand-written
// copy of the list of providers they mount. The day `AskProvider` was added to
// both layouts the copy fell one item behind, and every confirmation on the
// Windows till went back to `window.confirm`: the browser's own box, with the
// restaurant's domain over it, on the machine sold as a cash register. It was
// reported as "the new build is still the old one".
//
// A test rather than a comment, because the file already carried the comment.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const read = (p: string) =>
  readFileSync(fileURLToPath(new URL(p, import.meta.url)), "utf8");

/** Every place a till screen is mounted, browser and Windows alike. */
const surfaces = {
  "kassa/layout.tsx": "../../app/kassa/layout.tsx",
  "zal/layout.tsx": "../../app/zal/layout.tsx",
  // ⚠️ Outside this project on purpose: it is the copy that drifted, and a
  // test that only reads the two originals would have passed all along.
  "desktop/main.tsx": "../../../../backend/desktop/frontend/src/main.tsx",
};

describe("every till surface mounts the same shell", () => {
  for (const [name, path] of Object.entries(surfaces)) {
    const src = read(path);

    it(`${name} mounts TillShell`, () => {
      expect(src).toContain('from "@/components/till/TillShell"');
      expect(src).toMatch(/<TillShell\b/);
    });

    // ⚠️ The half that actually catches the drift. Adding a piece to TillShell
    // is safe; mounting one *beside* it is how a surface starts being a little
    // different from the other two, and the difference is only ever noticed on
    // the machine nobody develops on.
    it(`${name} does not mount the pieces itself`, () => {
      for (const piece of [
        "AskProvider",
        "TillAppliance",
        "OnScreenKeyboard",
        "StaffProvider",
        "CrashReporter",
      ]) {
        expect(src.includes(`<${piece}`)).toBe(false);
      }
    });
  }
});

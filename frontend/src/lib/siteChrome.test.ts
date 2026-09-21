import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { DEFAULT_CHROME, navActiveIndex, siteChrome } from "@/lib/siteChrome";
import type { PageDesign } from "@/lib/types";

const design = (sections: unknown[]): PageDesign =>
  ({ id: "home", brandId: "b", status: "published", sections, updatedAt: "" }) as PageDesign;

// ⚠️ **The precondition, and it is the whole reason this file has defaults at
// all.** Every site on the platform has no navbar band, or has one it never
// touched — the five built-in templates all open with an untouched one. Any of
// those reading as "a bar with no cart and no language switch" would strip the
// header of every customer we have on the day it shipped.
describe("a site that never configured its bar", () => {
  it("gets today's header when there is no design at all", () => {
    expect(siteChrome(null)).toEqual(DEFAULT_CHROME);
    expect(siteChrome(undefined)).toEqual(DEFAULT_CHROME);
  });

  it("gets today's header when the design has no navbar band", () => {
    expect(siteChrome(design([{ type: "hero", span: 12 }]))).toEqual(DEFAULT_CHROME);
  });

  it("gets today's header from an untouched navbar band", () => {
    const c = siteChrome(design([{ type: "navbar", variant: "classic", span: 12 }]));
    expect(c).toEqual(DEFAULT_CHROME);
  });

  // ⚠️ A hidden band is one an operator took out, and taking the bar's
  // *settings* out must give back the built-in bar rather than no bar.
  it("gets today's header from a hidden navbar band", () => {
    const c = siteChrome(
      design([{ type: "navbar", variant: "transparent", span: 12, hidden: true }]),
    );
    expect(c).toEqual(DEFAULT_CHROME);
  });
});

describe("a bar somebody drew", () => {
  const drawn = design([
    {
      type: "navbar",
      variant: "transparent",
      span: 12,
      style: { width: "full" },
      settings: {
        tone: "accent",
        sticky: false,
        lowercase: true,
        burger: "drawer",
        iconSearch: true,
        iconTheme: false,
      },
    },
  ]);

  it("reads every field it set", () => {
    const c = siteChrome(drawn);
    expect(c.variant).toBe("transparent");
    expect(c.tone).toBe("accent");
    expect(c.width).toBe("full");
    expect(c.sticky).toBe(false);
    expect(c.lowercase).toBe(true);
    expect(c.burger).toBe("drawer");
    expect(c.icons.search).toBe(true);
    expect(c.icons.theme).toBe(false);
  });

  // ⚠️ Fields it did *not* set keep the built-in answer rather than going
  // false: a design that switched the theme toggle off must not also, silently,
  // take away the cart.
  it("leaves the fields it did not set alone", () => {
    const c = siteChrome(drawn);
    expect(c.icons.cart).toBe(true);
    expect(c.icons.account).toBe(true);
    expect(c.icons.lang).toBe(true);
  });
});

// ⚠️ A band written by a newer console must cost the tailoring, never the
// header: a site whose bar failed to render is a site with no way to reach the
// catalogue. Same rule `known()` follows for business types.
describe("a bar from a newer console", () => {
  it("falls back value by value rather than refusing", () => {
    const c = siteChrome(
      design([
        {
          type: "navbar",
          variant: "kaleidoscope",
          span: 12,
          style: { width: "1400px" },
          settings: { tone: "neon", burger: "teleport" },
        },
      ]),
    );
    expect(c.variant).toBe(DEFAULT_CHROME.variant);
    expect(c.tone).toBe("");
    expect(c.width).toBe("");
    expect(c.burger).toBe(DEFAULT_CHROME.burger);
  });
});

// ⚠️ **The console writes these keys and this file reads them, and nothing the
// compiler can see joins the two.** A key renamed in the schema and forgotten
// here is a setting an operator changes, saves, publishes and watches do
// nothing — which is exactly what the whole navbar band did before it was
// wired up at all.
describe("the settings the console offers", () => {
  it("are all keys this file reads", () => {
    const schema = JSON.parse(
      readFileSync("../control/internal/handlers/templates/schema.json", "utf8"),
    ) as { sections: { type: string; settings: { key: string }[] }[] };
    const navbar = schema.sections.find((s) => s.type === "navbar");
    expect(navbar, "the navbar section left the schema").toBeTruthy();

    const src = readFileSync("src/lib/siteChrome.ts", "utf8");
    for (const f of navbar!.settings) {
      // `variant` lives on the band itself, not in its settings bag.
      if (f.key === "variant") continue;
      // Either spelling counts: a key read through the `bool` helper appears
      // quoted, one read directly appears as a property.
      const read = src.includes(`"${f.key}"`) || src.includes(`s.${f.key}`);
      expect(read, `the schema offers "${f.key}" and nothing reads it`).toBe(true);
    }
  });
});

describe("which navigation link is marked as the one you are on", () => {
  // ⚠️ **A live bug on ecom.keel.uz, and the reason this rule exists.** A shop
  // divides one page into sections, so its bar holds four links to `/menu` —
  // and the old rule, which compared the path alone, lit all four on the
  // catalogue page. Every item except "Biz haqimizda" was highlighted, which is
  // the same as none being highlighted, except it also hides where you are.
  const shopBar = [
    { href: "/menu" },
    { href: "/menu#cat-erkaklar" },
    { href: "/menu#cat-ayollar" },
    { href: "/menu#cat-sport" },
    { href: "/about" },
  ];

  it("marks exactly one item on the catalogue page", () => {
    expect(navActiveIndex(shopBar, "/menu", "")).toBe(0);
  });

  it("moves to the section the guest jumped to", () => {
    expect(navActiveIndex(shopBar, "/menu", "#cat-ayollar")).toBe(2);
    // A fragment nobody in the bar owns leaves the page itself marked.
    expect(navActiveIndex(shopBar, "/menu", "#something-else")).toBe(0);
  });

  it("marks a deeper page by its own link", () => {
    expect(navActiveIndex(shopBar, "/about", "")).toBe(4);
    // A dish page is still "the catalogue" — that is what `startsWith` is for.
    expect(navActiveIndex(shopBar, "/menu/abc", "")).toBe(0);
  });

  it("marks nothing where the bar leads nowhere", () => {
    expect(navActiveIndex(shopBar, "/cart", "")).toBe(-1);
  });

  it("never marks an address that leaves the site", () => {
    const bar = [{ href: "https://t.me/spike", external: true }, { href: "/menu" }];
    expect(navActiveIndex(bar, "/menu", "")).toBe(1);
    // ⚠️ Not by accident: `startsWith` on "https://…" is false anyway, and a
    // rule that is right for the wrong reason stops being right when it moves.
    expect(navActiveIndex(bar, "https://t.me/spike", "")).toBe(-1);
  });

  it("marks home only at home", () => {
    const bar = [{ href: "/" }, { href: "/menu" }];
    expect(navActiveIndex(bar, "/", "")).toBe(0);
    expect(navActiveIndex(bar, "/menu", "")).toBe(1);
  });
});

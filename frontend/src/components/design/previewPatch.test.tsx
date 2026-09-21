import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import CanvasBlock from "./CanvasBlock";
import { LangProvider } from "@/lib/i18n/client";
import { applyBand, applyElement } from "./previewPatch";
import type { DesignElement, DesignSection } from "@/lib/types";

// The console draws an edit into the live preview instead of reloading the
// customer's site (see lib/designDiff.ts in the console for which edits those
// are). That only works while this file and the renderer agree about the markup,
// and **nothing else can notice when they stop agreeing**: a patch that misses
// its node throws nothing, logs nothing and leaves a preview that quietly shows
// the previous value until something forces a reload.
//
// So these tests render the real band, patch it, and read the DOM back.

function band(elements: DesignElement[], over: Partial<DesignSection> = {}) {
  return {
    type: "canvas",
    variant: "free",
    span: 12,
    canvas: { height: 60, elements },
    ...over,
  } as DesignSection;
}

function draw(section: DesignSection) {
  return render(
    <LangProvider initial="uz">
      <CanvasBlock canvas={section.canvas} bandIndex={0} lang="uz" />
    </LangProvider>,
  );
}

/** The desktop surface: the one that places its elements itself. */
function desktop(container: HTMLElement) {
  return container.querySelector<HTMLElement>('[data-keel-band="0"].lg\\:block')!;
}

describe("drawing an edit into a rendered band", () => {
  it("moves a box without touching anything else", () => {
    const el: DesignElement = {
      type: "text",
      box: { x: 1, y: 2, w: 30, h: 10, z: 1 },
      text: { uz: "Salom", ru: "", en: "" },
    };
    const { container } = draw(band([el]));
    const node = desktop(container).querySelector<HTMLElement>('[data-keel-el="0"]')!;
    expect(node.style.left).toBe("1%");

    applyElement(container, { band: 0, index: 0, el: { ...el, box: { x: 40, y: 12, w: 25, h: 8, z: 3 } } });
    expect([node.style.left, node.style.top, node.style.width, node.style.height]).toEqual(
      ["40%", "12%", "25%", "8%"],
    );
    expect(node.style.zIndex).toBe("3");
    expect(node.textContent).toContain("Salom");
  });

  it("restyles an element the way the server would have", () => {
    const el: DesignElement = {
      type: "text",
      box: { x: 0, y: 0, w: 30, h: 10 },
      text: { uz: "Salom", ru: "", en: "" },
      style: { size: 1 },
    };
    const { container } = draw(band([el]));
    const node = desktop(container).querySelector<HTMLElement>('[data-keel-el="0"]')!;

    const restyled = { ...el, style: { size: 6, weight: "black", color: "brand", align: "center" } };
    applyElement(container, { band: 0, index: 0, el: restyled });

    // ⚠️ Compared against a fresh render of the same element rather than against
    // a class list typed into the test: what this has to match is the server,
    // and a hand-written expectation would pass while both were wrong.
    const fresh = draw(band([restyled]));
    const expected = desktop(fresh.container).querySelector<HTMLElement>('[data-keel-el="0"]')!;
    expect(node.className.split(" ").sort()).toEqual(expected.className.split(" ").sort());
  });

  it("writes a word into every surface that renders it", () => {
    const el: DesignElement = {
      type: "button",
      box: { x: 0, y: 0, w: 20, h: 8 },
      text: { uz: "Eski", ru: "", en: "" },
      link: "/menu",
    };
    const { container } = draw(band([el]));
    applyElement(container, { band: 0, index: 0, el: { ...el, text: { uz: "Yangi", ru: "", en: "" } } });
    // Two renderings of the same band: the absolute one and the stacked phone
    // fallback. Both carry the button, and both must say the new word.
    const said = [...container.querySelectorAll('[data-keel-el="0"]')].map((n) => n.textContent);
    expect(said.length).toBeGreaterThan(1);
    for (const s of said) expect(s).toContain("Yangi");
    // ⚠️ And it is still a link: writing the word into the outer box would have
    // replaced the working button with a bare word.
    expect(container.querySelector('[data-keel-el="0"] a')).not.toBeNull();
  });

  it("keeps a quote's typographic quotes and a stat's second line", () => {
    const quote: DesignElement = {
      type: "quote",
      box: { x: 0, y: 0, w: 30, h: 20 },
      text: { uz: "Eski", ru: "", en: "" },
      subtext: { uz: "Kim", ru: "", en: "" },
    };
    const { container } = draw(band([quote]));
    applyElement(container, {
      band: 0, index: 0,
      el: { ...quote, text: { uz: "Yangi", ru: "", en: "" }, subtext: { uz: "Sardor", ru: "", en: "" } },
    });
    const node = desktop(container).querySelector<HTMLElement>('[data-keel-el="0"]')!;
    expect(node.textContent).toContain("“Yangi”");
    expect(node.textContent).toContain("— Sardor");
  });

  it("repaints a band's tone without stacking the old one", () => {
    const section = band([{ type: "box", box: { x: 0, y: 0, w: 10, h: 10 } }]);
    section.canvas!.background = "surface";
    const { container } = draw(section);
    const node = desktop(container);
    expect(node.className).toContain("bg-surface");

    applyBand(container, {
      band: 0,
      section: { ...section, canvas: { ...section.canvas!, background: "charcoal", height: 90 } },
    });
    expect(node.className).toContain("bg-charcoal");
    expect(node.className).not.toContain("bg-surface");
    expect(node.style.height).toBe("90vh");
  });
});

import { describe, expect, it } from "vitest";

import { diffDesign } from "./designDiff";
import type { DesignSection } from "./api";

// ⚠️ **The rule this file protects is "when in doubt, reload".**
//
// A change wrongly called drawable leaves the operator looking at a page that
// does not match the document they are editing — and they find out by
// publishing. A change wrongly called structural costs a flash. So the tests
// below are mostly about the second column: the things that must *not* be
// patched.

const band = (over: Partial<DesignSection> = {}): DesignSection => ({
  type: "canvas",
  variant: "free",
  span: 12,
  canvas: {
    height: 60,
    elements: [
      { type: "text", box: { x: 1, y: 2, w: 30, h: 10 }, text: { uz: "Salom", ru: "", en: "" } },
    ],
  },
  ...over,
});

/** A deep copy, because the diff is allowed to compare by reference first. */
const copy = (s: DesignSection[]): DesignSection[] => JSON.parse(JSON.stringify(s));

describe("what a change to a design costs", () => {
  it("draws a moved box", () => {
    const a = [band()];
    const b = copy(a);
    b[0].canvas!.elements![0].box.x = 40;
    expect(diffDesign(a, b)).toEqual({ kind: "patch", elements: [{ band: 0, index: 0 }], bands: [] });
  });

  it("draws a restyled element", () => {
    const a = [band()];
    const b = copy(a);
    b[0].canvas!.elements![0].style = { size: 5, color: "brand" };
    expect(diffDesign(a, b).kind).toBe("patch");
  });

  it("draws a band's own tone and height", () => {
    const a = [band()];
    const b = copy(a);
    b[0].style = { tone: "ink" };
    b[0].canvas!.height = 80;
    expect(diffDesign(a, b)).toEqual({ kind: "patch", elements: [], bands: [0] });
  });

  it("draws a heading typed into a fixed band", () => {
    const a = [band({ type: "cta", variant: "banner", canvas: null, settings: { heading: { uz: "Eski", ru: "", en: "" } } })];
    const b = copy(a);
    b[0].settings = { heading: { uz: "Yangi", ru: "", en: "" } };
    expect(diffDesign(a, b)).toEqual({ kind: "patch", elements: [], bands: [0] });
  });

  it("reloads when a heading is cleared, because the page decides the fallback", () => {
    const a = [band({ type: "cta", variant: "banner", canvas: null, settings: { heading: { uz: "Eski", ru: "", en: "" } } })];
    const b = copy(a);
    b[0].settings = { heading: { uz: "", ru: "", en: "" } };
    expect(diffDesign(a, b).kind).toBe("reload");
  });

  it("reloads for a setting that changes what the band does", () => {
    const a = [band({ type: "menu-grid", variant: "cards", canvas: null, settings: { limit: 8 } })];
    const b = copy(a);
    b[0].settings = { limit: 12 };
    expect(diffDesign(a, b).kind).toBe("reload");
  });

  it.each([
    ["a band added", (b: DesignSection[]) => b.push(band())],
    ["an element added", (b: DesignSection[]) => b[0].canvas!.elements!.push({ type: "box", box: { x: 0, y: 0, w: 10, h: 10 } })],
    ["an element removed", (b: DesignSection[]) => b[0].canvas!.elements!.pop()],
    ["a variant changed", (b: DesignSection[]) => (b[0].variant = "center")],
    ["a band hidden", (b: DesignSection[]) => (b[0].hidden = true)],
    ["an element hidden", (b: DesignSection[]) => (b[0].canvas!.elements![0].hidden = true)],
    ["an element hidden on the phone", (b: DesignSection[]) => (b[0].canvas!.elements![0].hiddenMobile = true)],
    ["an element's type", (b: DesignSection[]) => (b[0].canvas!.elements![0].type = "button")],
    ["an icon swapped", (b: DesignSection[]) => (b[0].canvas!.elements![0].icon = "star")],
    ["a carousel's photographs", (b: DesignSection[]) => (b[0].canvas!.elements![0].images = ["/uploads/a.jpg"])],
    ["a binding", (b: DesignSection[]) => (b[0].binding = { limit: 4 })],
  ])("reloads for %s", (_name, mutate) => {
    const a = [band()];
    const b = copy(a);
    mutate(b);
    expect(diffDesign(a, b).kind).toBe("reload");
  });

  it("says nothing changed when nothing did", () => {
    const a = [band()];
    expect(diffDesign(a, copy(a))).toEqual({ kind: "none" });
  });

  it("reports every element that moved, once per band", () => {
    const a = [band(), band()];
    const b = copy(a);
    b[0].canvas!.elements![0].box.y = 9;
    b[1].canvas!.elements![0].box.y = 9;
    b[1].canvas!.background = "ink";
    const out = diffDesign(a, b);
    expect(out).toEqual({
      kind: "patch",
      elements: [{ band: 0, index: 0 }, { band: 1, index: 0 }],
      bands: [1],
    });
  });
});

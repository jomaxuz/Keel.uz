import { act, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useHash } from "./useHash";

// ⚠️ **This is a bug found by clicking, not by reading.** The bar's marker
// followed a click to «Erkaklar» correctly and then stayed there when the guest
// clicked «Katalog» again: the router had cleared the fragment from the address
// bar with `pushState`, which fires no event, so React never heard that the page
// had moved. The rule in siteChrome was right the whole time; what it was told
// about the browser was stale.

function Probe() {
  const hash = useHash();
  return <output data-testid="hash">{hash || "(none)"}</output>;
}

const shown = () => screen.getByTestId("hash").textContent;

describe("the fragment the bar is looking at", () => {
  it("starts empty, so the first render matches the server's", () => {
    window.history.replaceState({}, "", "/menu");
    render(<Probe />);
    expect(shown()).toBe("(none)");
  });

  it("follows a pushState into a section", async () => {
    window.history.replaceState({}, "", "/menu");
    render(<Probe />);
    await act(async () => {
      window.history.pushState({}, "", "/menu#cat-erkaklar");
    });
    expect(shown()).toBe("#cat-erkaklar");
  });

  it("follows a pushState back out of one — the case that was broken", async () => {
    window.history.replaceState({}, "", "/menu#cat-erkaklar");
    render(<Probe />);
    await act(async () => {
      window.history.pushState({}, "", "/menu#cat-erkaklar");
    });
    expect(shown()).toBe("#cat-erkaklar");
    await act(async () => {
      // Clicking «Katalog» from a section: same path, no fragment.
      window.history.pushState({}, "", "/menu");
    });
    expect(shown()).toBe("(none)");
  });

  it("hears the back button", async () => {
    window.history.replaceState({}, "", "/menu");
    render(<Probe />);
    await act(async () => {
      window.history.replaceState({}, "", "/menu#cat-sport");
      window.dispatchEvent(new Event("popstate"));
    });
    expect(shown()).toBe("#cat-sport");
  });

  it("wraps the history methods once, however many bars are on the page", async () => {
    render(<Probe />);
    const first = window.history.pushState;
    render(<Probe />);
    expect(window.history.pushState).toBe(first);
    // And the wrapper still does what history does.
    await act(async () => {
      window.history.pushState({}, "", "/menu#cat-ayollar");
    });
    expect(window.location.hash).toBe("#cat-ayollar");
  });
});

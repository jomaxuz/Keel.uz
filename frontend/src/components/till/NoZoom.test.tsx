import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import NoZoom from "./NoZoom";

// ⚠️ Pinch-zoom is asked of every element up to the document, so the root has
// to refuse it too — and give the website its zoom back when the till leaves.
describe("NoZoom", () => {
  it("forbids pinch on the root while mounted, and restores it after", () => {
    const root = document.documentElement;
    root.style.touchAction = "";
    const { unmount } = render(<NoZoom />);
    expect(root.style.touchAction).toBe("pan-x pan-y");
    unmount();
    expect(root.style.touchAction).toBe("");
  });
});

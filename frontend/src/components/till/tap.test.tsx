import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import { tapProps, tapScroll } from "./tap";

/**
 * ⚠️ **The bug this file exists for is "it does not register when I tap fast".**
 *
 * A `click` needs press and release on the same element and arrives only after
 * the browser has finished waiting for a possible double-tap — so quick repeats
 * on a monoblock are exactly the case that drops. The PIN pad was fixed by
 * moving to `pointerdown`; the menu grid was not, and reported the same
 * symptom months later. These tests hold the rule in one place.
 */
describe("the till's tap rule", () => {
  function Tile({ onTap, disabled }: { onTap: () => void; disabled?: boolean }) {
    return (
      <button {...tapProps(onTap, disabled)} disabled={disabled}>
        tile
      </button>
    );
  }

  it("fires on contact, not on release", () => {
    const onTap = vi.fn();
    render(<Tile onTap={onTap} />);
    fireEvent.pointerDown(screen.getByText("tile"));
    expect(onTap).toHaveBeenCalledTimes(1);
  });

  // The real complaint: four taps in half a second must be four.
  it("counts every tap in a fast run", () => {
    const onTap = vi.fn();
    render(<Tile onTap={onTap} />);
    const el = screen.getByText("tile");
    for (let i = 0; i < 4; i++) fireEvent.pointerDown(el);
    expect(onTap).toHaveBeenCalledTimes(4);
  });

  // ⚠️ The browser synthesises a click from the same touch. Without the
  // preventDefault in `tapProps` a dish would be added twice.
  it("does not also answer the click the touch synthesises", () => {
    const onTap = vi.fn();
    render(<Tile onTap={onTap} />);
    const el = screen.getByText("tile");
    fireEvent.pointerDown(el);
    fireEvent.click(el);
    expect(onTap).toHaveBeenCalledTimes(1);
  });

  it("stays quiet when the tile is off", () => {
    const onTap = vi.fn();
    render(<Tile onTap={onTap} disabled />);
    fireEvent.pointerDown(screen.getByText("tile"));
    expect(onTap).not.toHaveBeenCalled();
  });
});

describe("tapScroll, for chips inside something draggable", () => {
  function Chip({ onTap }: { onTap: () => void }) {
    return <button {...tapScroll(onTap)}>chip</button>;
  }

  it("answers a tap that stayed put", () => {
    const onTap = vi.fn();
    render(<Chip onTap={onTap} />);
    const el = screen.getByText("chip");
    fireEvent.pointerDown(el, { clientX: 100, clientY: 100 });
    fireEvent.pointerUp(el, { clientX: 102, clientY: 101 });
    expect(onTap).toHaveBeenCalledTimes(1);
  });

  // ⚠️ Scrolling the strip must not change the category under the finger.
  it("stays quiet when the finger dragged", () => {
    const onTap = vi.fn();
    render(<Chip onTap={onTap} />);
    const el = screen.getByText("chip");
    fireEvent.pointerDown(el, { clientX: 100, clientY: 100 });
    fireEvent.pointerMove(el, { clientX: 160, clientY: 104 });
    fireEvent.pointerUp(el, { clientX: 160, clientY: 104 });
    expect(onTap).not.toHaveBeenCalled();
  });
});

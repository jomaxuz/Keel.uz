import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import TillAppliance from "./TillAppliance";
import { tapProps } from "./tap";

/**
 * ⚠️ **These tests are about the screens nobody fixed.**
 *
 * The PIN pad was fixed, then the menu grid, and the complaint came back each
 * time about a different surface — because the fix was a habit rather than a
 * default. This layer is the default, so what it must and must not do is
 * pinned: a press that landed anywhere registers, a flick scrolls, a component
 * that already handles its own pointers is left alone, and nothing fires twice.
 */
function touch(el: Element, from: [number, number], to = from) {
  fireEvent.pointerDown(el, {
    pointerType: "touch",
    isPrimary: true,
    button: 0,
    clientX: from[0],
    clientY: from[1],
  });
  fireEvent.pointerUp(el, {
    pointerType: "touch",
    isPrimary: true,
    clientX: to[0],
    clientY: to[1],
  });
}

describe("the till's global tap layer", () => {
  it("answers a tap that stayed put", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>pay</button>
      </>,
    );
    touch(screen.getByText("pay"), [100, 100], [104, 98]);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  // The complaint itself: four presses in half a second must be four sales.
  it("counts every press in a fast run", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>7</button>
      </>,
    );
    const el = screen.getByText("7");
    for (let i = 0; i < 5; i++) touch(el, [50, 50]);
    expect(onClick).toHaveBeenCalledTimes(5);
  });

  // ⚠️ The browser follows a touch with a click of its own. Without the capture
  // listener that swallows it, every button on the till would fire twice — a
  // dish added twice, a void confirmed twice.
  it("does not also answer the click the touch synthesises", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>void</button>
      </>,
    );
    const el = screen.getByText("void");
    touch(el, [10, 10]);
    fireEvent.click(el);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  // ⚠️ **Filmed on a monoblock.** "Back" closes the new-check dialog, so by the
  // time the browser's own click arrives the button is gone and the click is
  // hit-tested onto the floor plan underneath — opening the dialog again, for
  // whichever table was drawn behind the button.
  it("does not let the echo fall through to what was under a closed dialog", () => {
    const openTable = vi.fn();
    function Floor() {
      const [open, setOpen] = useState(true);
      return (
        <>
          <button onClick={openTable}>table 10</button>
          {open && <button onClick={() => setOpen(false)}>back</button>}
        </>
      );
    }
    render(
      <>
        <TillAppliance />
        <Floor />
      </>,
    );
    touch(screen.getByText("back"), [300, 500]);
    expect(screen.queryByText("back")).toBeNull();
    // The browser's click from the same touch, landing on the table beneath.
    fireEvent.click(screen.getByText("table 10"), { clientX: 301, clientY: 502 });
    expect(openTable).not.toHaveBeenCalled();
  });

  // The same fall-through from a control that answers on contact: a keypad key
  // that closes the keypad.
  it("catches the echo of a tapProps control too", () => {
    const underneath = vi.fn();
    render(
      <>
        <TillAppliance />
        <button {...tapProps(() => {})}>done</button>
        <button onClick={underneath}>under</button>
      </>,
    );
    touch(screen.getByText("done"), [40, 700]);
    fireEvent.click(screen.getByText("under"), { clientX: 40, clientY: 700 });
    expect(underneath).not.toHaveBeenCalled();
  });

  // ⚠️ And only the echo: a real press somewhere else, or the next touch, is
  // never eaten by a guard left over from the last one.
  it("does not swallow a click that is not the echo", () => {
    const other = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={() => {}}>first</button>
        <button onClick={other}>second</button>
      </>,
    );
    touch(screen.getByText("first"), [10, 10]);
    fireEvent.click(screen.getByText("second"), { clientX: 600, clientY: 400 });
    expect(other).toHaveBeenCalledTimes(1);
  });

  // ⚠️ **The check this whole design turns on.** A till is full of scrolling
  // lists; firing on contact would open whichever row a flick began on, which
  // is a worse fault than the one being fixed.
  it("stays quiet when the finger scrolled", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>check 214</button>
      </>,
    );
    touch(screen.getByText("check 214"), [100, 300], [104, 120]);
    expect(onClick).not.toHaveBeenCalled();
  });

  // ⚠️ A component that already handles its own pointers keeps them. `tapProps`
  // fires on contact for the keypad and the dish grid, and a second activation
  // from here would double every one of them.
  it("leaves a component that handles its own pointers alone", () => {
    const onTap = vi.fn();
    render(
      <>
        <TillAppliance />
        <button {...tapProps(onTap)}>tile</button>
      </>,
    );
    touch(screen.getByText("tile"), [20, 20]);
    expect(onTap).toHaveBeenCalledTimes(1);
  });

  // A disabled button is not a quiet button that happens to look grey.
  it("stays quiet on a disabled control", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick} disabled>
          pay
        </button>
      </>,
    );
    touch(screen.getByText("pay"), [20, 20]);
    expect(onClick).not.toHaveBeenCalled();
  });

  // ⚠️ **A press that landed on the button counts even if the finger drifted
  // off it.** This is the failure that reads as "I pressed it and nothing
  // happened": a `click` needs press *and* release on the same element, and on
  // a tilted monoblock two pixels of roll is enough to lose it.
  it("answers a press that started on the control and slid slightly off", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>pay</button>
        <div data-testid="gap">gap</div>
      </>,
    );
    fireEvent.pointerDown(screen.getByText("pay"), {
      pointerType: "touch",
      isPrimary: true,
      button: 0,
      clientX: 100,
      clientY: 100,
    });
    fireEvent.pointerUp(screen.getByTestId("gap"), {
      pointerType: "touch",
      isPrimary: true,
      clientX: 106,
      clientY: 103,
    });
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  // ⚠️ A mouse is left entirely alone: its click is already instant and already
  // lands where it was aimed, and intercepting it breaks the drag and the
  // context menu a desk machine still has.
  it("does not touch a mouse click", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <button onClick={onClick}>pay</button>
      </>,
    );
    const el = screen.getByText("pay");
    fireEvent.pointerDown(el, { pointerType: "mouse", isPrimary: true, button: 0 });
    fireEvent.pointerUp(el, { pointerType: "mouse", isPrimary: true });
    fireEvent.click(el);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  // ⚠️ Form fields keep the browser's own behaviour, all of it: a caret has to
  // be placeable, and a checkbox toggled twice is a discount nobody agreed to.
  // ⚠️ **The floor plan's tables are SVG**, and `click()` is a method of
  // HTMLElement alone. Activating one used to throw, after the echo guard had
  // already been armed — so the browser's own click was swallowed on its way
  // past and a table could not be opened by touch at all. It opened fine with a
  // mouse, which is why this was only ever reported from the monoblocks.
  it("opens a table drawn in SVG", () => {
    const onClick = vi.fn();
    render(
      <>
        <TillAppliance />
        <svg>
          <g role="button" aria-label="7-stol" onClick={onClick}>
            <rect width="40" height="40" />
          </g>
        </svg>
      </>,
    );
    touch(screen.getByLabelText("7-stol"), [60, 60], [63, 61]);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("keeps a checkbox's own behaviour", () => {
    const onChange = vi.fn();
    render(
      <>
        <TillAppliance />
        <input type="checkbox" aria-label="open" onChange={onChange} />
      </>,
    );
    const el = screen.getByLabelText("open");
    touch(el, [20, 20]);
    fireEvent.click(el);
    expect(onChange).toHaveBeenCalledTimes(1);
  });
});

describe("nothing leaves the screen", () => {
  it("refuses the long-press menu and a copy outside a field", () => {
    render(
      <>
        <TillAppliance />
        <p>Lag&apos;mon 32 000</p>
      </>,
    );
    const menu = fireEvent.contextMenu(screen.getByText(/Lag/));
    // fireEvent returns false when a listener called preventDefault.
    expect(menu).toBe(false);
  });

  // ⚠️ Typing is exempt, all of it. A till where a typo can only be fixed by
  // clearing the whole field is a slower till, not a safer one.
  it("leaves a field's own menu alone", () => {
    render(
      <>
        <TillAppliance />
        <input aria-label="discount" defaultValue="5000" />
      </>,
    );
    expect(fireEvent.contextMenu(screen.getByLabelText("discount"))).toBe(true);
  });
});

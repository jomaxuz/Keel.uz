/**
 * How every key, tile and chip on the till answers a finger.
 *
 * ⚠️ **`pointerdown`, not `click`, and this is one rule rather than three
 * copies of it.** A `click` needs the press *and* the release on the same
 * element, so a finger that shifts two pixels on a tilted monoblock produces
 * nothing at all — felt as "I pressed it and it did not register". It also
 * arrives after the browser has waited to see whether the tap was the first
 * half of a double-tap, which is why *fast* tapping is the case that fails:
 * the repeats land inside that window, on a surface a finger is rolling across.
 *
 * The PIN pad was fixed first and the fix stayed there, so the menu grid — the
 * surface a cashier taps hardest and fastest, mid-service, with a queue — kept
 * dropping presses for exactly the same reason. Written down once so the next
 * pressable thing on this screen starts correct.
 *
 * ⚠️ **The wrong half of the trade cannot happen here.** A key registering when
 * somebody meant to swipe is the usual cost of pointer-down; there is nothing
 * on the till to swipe, and the category strip that does scroll gets
 * `tapScroll` below, which waits to see whether the finger moved.
 */
export function tapProps(onTap: () => void, disabled?: boolean) {
  return {
    onPointerDown: (e: React.PointerEvent) => {
      // ⚠️ Stops the browser also synthesising a click from the same touch,
      // which would fire the handler twice — a dish added twice, a digit
      // entered twice.
      e.preventDefault();
      if (!disabled) onTap();
    },
  };
}

/** How far a finger may travel and still count as a tap, in CSS pixels. */
const SLOP = 12;

/**
 * For things that sit inside something scrollable.
 *
 * ⚠️ **A horizontal category strip is the one place pointer-down is wrong.**
 * Fired on contact, dragging the strip sideways also selects whichever chip the
 * finger started on — so scrolling the menu changes the category under you. So
 * this one waits for the release and checks the finger stayed put: still ahead
 * of `click` (no double-tap wait, no requirement that press and release share
 * an element), without stealing the drag.
 */
export function tapScroll(onTap: () => void, disabled?: boolean) {
  let x = 0;
  let y = 0;
  let moved = false;
  return {
    onPointerDown: (e: React.PointerEvent) => {
      x = e.clientX;
      y = e.clientY;
      moved = false;
    },
    onPointerMove: (e: React.PointerEvent) => {
      if (Math.abs(e.clientX - x) > SLOP || Math.abs(e.clientY - y) > SLOP) {
        moved = true;
      }
    },
    onPointerUp: (e: React.PointerEvent) => {
      e.preventDefault();
      if (!moved && !disabled) onTap();
    },
  };
}

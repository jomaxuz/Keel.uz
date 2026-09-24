"use client";

/**
 * The rules that make a till behave like an appliance, in one place.
 *
 * ⚠️ **This exists because the fixes kept being local.** The PIN pad was fixed
 * first, then the menu grid — the two surfaces somebody complained about — and
 * the fix stayed where it was made both times. Everything else on the till went
 * on dropping presses for the same reason, and the report came back as "it
 * still freezes", about a different screen each time. There is no version of
 * this that works as a habit; the default has to be correct.
 *
 * Three guarantees, and they fail for three unrelated reasons:
 *
 *   1. **A press registers.** See `activate` below.
 *   2. **Nothing zooms.** See NoZoom, which this mounts.
 *   3. **Nothing can be selected, copied or dragged off the screen.**
 *
 * ⚠️ **It does not replace `tapProps`** (components/till/tap.ts) and must not.
 * That fires on *pointer-down*, which is right for a keypad and a dish grid —
 * surfaces where the fastest possible response is worth activating something
 * the finger merely brushed. This is the global default, so it takes the
 * scroll-safe half of the same reasoning: a till is full of scrolling lists,
 * and a check list that opened whichever row a flick started on would be a
 * worse bug than the one being fixed. A component that already handles its own
 * pointer events is left alone (`defaultPrevented`).
 */

import { useEffect } from "react";

import NoZoom from "./NoZoom";

/** How far a finger may travel and still count as a tap, in CSS pixels.
 *  The same figure tap.ts uses — a monoblock is tilted and fingers roll. */
const SLOP = 12;

/** How long a tap may take before it stops being one. A finger held on a
 *  button while somebody thinks, then lifted, is a press; a finger held for
 *  three seconds is a long-press, and firing on its release is how a cashier
 *  ends up voiding a line they were only pointing at. */
const HOLD_MS = 900;

/** How long after our own activation a browser-synthesised click is swallowed.
 *  Generous, because the wait we are avoiding is exactly the one that varies. */
const ECHO_MS = 1200;

export default function TillAppliance() {
  useEffect(() => {
    // ---- 1. A press registers ----
    //
    // ⚠️ **The start point decides, not the release point.** A `click` needs
    // press and release on the same element, so a finger that shifts two pixels
    // across the gap between two buttons produces nothing at all — felt as "I
    // pressed it and it did not register", and felt most often by whoever is
    // fastest. Here the element under the finger when it landed is the one that
    // is activated, and the release only has to be near it.
    let startEl: Element | null = null;
    let startX = 0;
    let startY = 0;
    let startAt = 0;
    // The element whose next native click is ours to swallow, and until when.
    let echoEl: Element | null = null;
    let echoUntil = 0;
    // …and **where** the finger was lifted. See `onClickCapture`.
    let echoX = 0;
    let echoY = 0;
    // The press began on a component that handles its own pointers
    // (`tapProps`, which fires on contact). Left alone — but its echo is
    // still ours to catch, see `onUp`.
    let ownStart = false;
    // True while we are dispatching a click ourselves, so the capture listener
    // below can tell our own event from the browser's echo of it.
    let dispatching = false;

    /** The pressable thing under a point, or null.
     *
     *  ⚠️ **An Element, not an HTMLElement, and that difference was a bug in a
     *  restaurant.** The floor plan is drawn in SVG and its tables are
     *  `<g role="button">` — which matches the selector below perfectly well,
     *  so a finger landing on table 7 was recorded here and then activated by
     *  `el.click()`. `click()` is defined on HTMLElement and on nothing else:
     *  an SVG group does not have it, so the call threw, the echo guard had
     *  already been armed a line earlier, and the browser's own click — the one
     *  that would have worked — was swallowed on its way past. The result was a
     *  floor plan where tables could not be opened by touch and opened
     *  perfectly with a mouse, which is why it was only ever reported from the
     *  monoblocks. See `activate`. */
    const pressable = (target: EventTarget | null): Element | null => {
      if (!(target instanceof Element)) return null;
      // ⚠️ Form fields keep the browser's own behaviour, all of it: a caret
      // has to be placeable, a select has to open, and a checkbox toggled
      // twice by a helpful synthetic click is a discount nobody agreed to.
      if (target.closest("input, textarea, select, label, [contenteditable]")) {
        return null;
      }
      const el = target.closest('button, [role="button"], a[href]');
      if (!el) return null;
      if (el.hasAttribute("disabled") || el.getAttribute("aria-disabled") === "true") {
        return null;
      }
      // The opt-out, for anything that genuinely wants the browser's own
      // sequence. Nothing uses it today; it exists so the next thing that needs
      // to can say so rather than fight this.
      if (el.getAttribute("data-tap") === "native") return null;
      return el;
    };

    /** Press it, whatever kind of element it is.
     *
     *  ⚠️ **`click()` is an HTMLElement method.** Everything else — an SVG
     *  table on the floor plan, an SVG zone on a delivery map — gets the event
     *  dispatched by hand: bubbling and cancellable, which is all React needs
     *  to route it to the handler on the group. ⚠️ No `view` — the element is
     *  what matters here, and passing a window belonging to another realm is a
     *  thrown constructor rather than a wrong view. */
    const activate = (el: Element) => {
      if (typeof (el as HTMLElement).click === "function") {
        (el as HTMLElement).click();
        return;
      }
      el.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
    };

    const onDown = (e: PointerEvent) => {
      startEl = null;
      ownStart = false;
      // ⚠️ Mouse is left entirely alone. A click from a mouse is already
      // instant and already lands where it was aimed — the whole problem here
      // is a fingertip on a panel — and intercepting it would break the drag,
      // the context menu and the double-click that a desk machine still has.
      if (e.pointerType === "mouse" || !e.isPrimary || e.button !== 0) return;
      // A new touch: whatever echo the last one owed has long arrived. A guard
      // left armed past this point could only eat a click that is not an echo.
      echoEl = null;
      echoUntil = 0;
      startX = e.clientX;
      startY = e.clientY;
      startAt = Date.now();
      if (e.defaultPrevented) {
        ownStart = true;
        return;
      }
      const el = pressable(e.target);
      if (!el) return;
      startEl = el;
    };

    /** Swallow the browser's click from this touch — on this element, or
     *  anywhere near where the finger came off. */
    const arm = (el: Element | null, e: PointerEvent) => {
      echoEl = el;
      echoUntil = Date.now() + ECHO_MS;
      echoX = e.clientX;
      echoY = e.clientY;
    };

    const onUp = (e: PointerEvent) => {
      const el = startEl;
      startEl = null;
      if (e.pointerType === "mouse" || !e.isPrimary) return;
      // ⚠️ **A `tapProps` control already answered on contact — and its echo
      // is the same fall-through as ours.** Pressing a keypad key that closes
      // the keypad leaves the browser's click to land on whatever was drawn
      // underneath it. Caught here by position, and nothing is activated.
      if (ownStart) {
        ownStart = false;
        if (
          Math.abs(e.clientX - startX) <= SLOP &&
          Math.abs(e.clientY - startY) <= SLOP
        ) {
          arm(null, e);
        }
        return;
      }
      if (!el) return;
      // Scrolled, not tapped. ⚠️ The check this whole design turns on: without
      // it, flicking a list of open checks opens whichever row the flick began
      // on, which is a worse fault than the one being fixed.
      if (
        Math.abs(e.clientX - startX) > SLOP ||
        Math.abs(e.clientY - startY) > SLOP ||
        Date.now() - startAt > HOLD_MS
      ) {
        return;
      }
      // The element may have gone: React re-renders under a finger all the
      // time, and clicking a detached node does nothing while still swallowing
      // the browser's click.
      if (!el.isConnected) return;

      // ⚠️ **Armed before the dispatch, not after.** The click below runs
      // synchronously and bubbles all the way out; a flag set afterwards would
      // arrive too late to tell our own event from the browser's echo.
      arm(el, e);
      dispatching = true;
      try {
        activate(el);
      } finally {
        dispatching = false;
      }
      // The browser will follow with its own click from the same touch. It is
      // swallowed below; preventing the default here does not stop it.
      e.preventDefault();
    };

    // ⚠️ **Capture, on the document, because React listens lower down.** React
    // attaches its handlers to the root container, which is inside the
    // document — so a bubble-phase listener here would run *after* the handler
    // it is trying to stop, and every button would fire twice. A dish added
    // twice, a digit entered twice, a void confirmed twice.
    //
    // ⚠️ **Matched by position as well as by element, and that is a fix from a
    // restaurant's floor.** The echo is hit-tested when it arrives, not when
    // the finger landed — and our activation has run by then. A "Back" button
    // that closes the new-check dialog is gone before its own click comes, so
    // the click lands on the floor plan underneath, on whichever table was
    // drawn behind the button, and opens the dialog again for *that* table.
    // Filmed: the dialog flickered shut and back open with a different table
    // chosen, and it looked like lag. Matching only the element let every such
    // echo through — which is every button that closes the thing it sits on.
    //
    // Once, and only until the next touch starts (`onDown`): a click that is
    // not an echo is never ours to eat.
    const onClickCapture = (e: MouseEvent) => {
      if (dispatching) return;
      if (Date.now() > echoUntil) {
        echoEl = null;
        echoUntil = 0;
        return;
      }
      const onElement =
        echoEl !== null &&
        e.target instanceof Node &&
        (echoEl === e.target || echoEl.contains(e.target));
      const nearLift =
        Math.abs(e.clientX - echoX) <= SLOP * 2 &&
        Math.abs(e.clientY - echoY) <= SLOP * 2;
      if (onElement || nearLift) {
        echoEl = null;
        echoUntil = 0;
        e.stopPropagation();
        e.preventDefault();
      }
    };

    const onCancel = () => {
      startEl = null;
      ownStart = false;
    };

    // ---- 3. Nothing leaves the screen ----
    //
    // ⚠️ **A selection is not only untidy, it is the freeze.** Once a fast
    // repeated tap starts one, the taps that follow go to the selection rather
    // than to the button, and what the cashier sees is a screen that has
    // stopped responding. The CSS in globals.css says `user-select: none`;
    // these listeners are the half the CSS cannot do — the long-press callout,
    // the drag, and the keyboard.
    const inField = (t: EventTarget | null) =>
      t instanceof Element &&
      t.closest("input, textarea, [contenteditable]") !== null;

    const stopOutsideFields = (e: Event) => {
      // ⚠️ Form fields keep all of it. A cashier pasting a scanned code, or
      // selecting a wrong digit to retype it, is doing the ordinary work this
      // screen exists for — and a till where a typo can only be fixed by
      // clearing the whole field is a slower till, not a safer one.
      if (inField(e.target)) return;
      e.preventDefault();
    };

    const onKey = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return;
      // Copy, cut and select-all off the screen itself. ⚠️ Paste is untouched:
      // it only ever lands in a field, and blocking it would stop a scanner
      // configured to paste rather than type.
      if (["c", "x", "a", "C", "X", "A"].includes(e.key) && !inField(e.target)) {
        e.preventDefault();
      }
    };

    const opts = { passive: false } as const;
    document.addEventListener("pointerdown", onDown, opts);
    document.addEventListener("pointerup", onUp, opts);
    document.addEventListener("pointercancel", onCancel);
    document.addEventListener("click", onClickCapture, true);
    document.addEventListener("contextmenu", stopOutsideFields, opts);
    document.addEventListener("selectstart", stopOutsideFields, opts);
    document.addEventListener("dragstart", stopOutsideFields, opts);
    document.addEventListener("copy", stopOutsideFields, opts);
    document.addEventListener("cut", stopOutsideFields, opts);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("pointerup", onUp);
      document.removeEventListener("pointercancel", onCancel);
      document.removeEventListener("click", onClickCapture, true);
      document.removeEventListener("contextmenu", stopOutsideFields);
      document.removeEventListener("selectstart", stopOutsideFields);
      document.removeEventListener("dragstart", stopOutsideFields);
      document.removeEventListener("copy", stopOutsideFields);
      document.removeEventListener("cut", stopOutsideFields);
      window.removeEventListener("keydown", onKey);
    };
  }, []);

  return <NoZoom />;
}

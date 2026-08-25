"use client";

import { useEffect, useRef, useState } from "react";

import { normalizeQty, qtyNumber, qtyText } from "@/lib/qty";

/**
 * A field for a quantity — kilograms on a shelf, litres of oil, portions.
 *
 * ⚠️ **Text, not `type="number"`, and that is the point.** A number field
 * refuses a comma: the browser hands us an empty string for "9,4" before any
 * of our code runs, so the digits disappear as the separator is pressed. It is
 * also what kept a comma off the till's own keypad, since a key typing a
 * character the field drops teaches the cashier the pad is lying. The rules
 * that a number field gave for free now live in `lib/qty.ts`, where they can
 * be read and are tested.
 *
 * ⚠️ **It holds its own draft.** The parent stores a number, and a parent that
 * re-prints that number on every keystroke turns "9." into "9" the moment the
 * point is pressed — the fraction could then never be typed at all. So what is
 * shown belongs to the field until it disagrees with the parent *as a number*,
 * which is what happens when a form is reset or a row is replaced.
 */
export function QtyInput({
  value,
  onValue,
  className = "input",
  placeholder,
  disabled,
  "aria-label": ariaLabel,
}: {
  /** What the parent holds. A number for a parsed field, a string for one that
   *  keeps what was typed (a stocktake sheet: "counted nothing" and "not
   *  counted yet" are different answers and only text tells them apart). */
  value: number | string;
  /** The cleaned text. `qtyNumber()` turns it into what the server is sent. */
  onValue: (text: string) => void;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
  "aria-label"?: string;
}) {
  const [draft, setDraft] = useState(() => qtyText(value));
  // What we last sent up, so an echo of our own value is not mistaken for the
  // parent changing its mind.
  const sent = useRef(draft);

  useEffect(() => {
    const incoming = qtyText(value);
    if (qtyNumber(incoming) === qtyNumber(sent.current)) return;
    sent.current = incoming;
    setDraft(incoming);
  }, [value]);

  return (
    <input
      type="text"
      // The till's keypad reads this to decide it needs a separator key, and a
      // phone keyboard reads it to show a numeric layout. It is the one
      // declaration that survived the field becoming text.
      inputMode="decimal"
      autoComplete="off"
      className={className}
      placeholder={placeholder}
      disabled={disabled}
      aria-label={ariaLabel}
      value={draft}
      onChange={(e) => {
        const next = normalizeQty(e.target.value);
        setDraft(next);
        sent.current = next;
        onValue(next);
      }}
    />
  );
}

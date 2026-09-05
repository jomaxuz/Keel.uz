"use client";

// How much of it is there? — asked once, in one place.
//
// ⚠️ **Both ways of putting weighed goods on a check come through here.** A
// packet can arrive by scan or by a tap on its card when the scanner is dead,
// and the second path is the one that matters: it is used on the day something
// is already broken. Two copies of this dialog would drift, and the half that
// drifted would be the half nobody exercises until then.
//
// ⚠️ **The reading is shown and confirmed, never added straight to the check.**
// That is what makes a connected scale safe in a way a printed label is not: a
// misread is caught by the person holding the goods. The button fills the box;
// the cashier still presses add.

import { useState } from "react";

import type { MenuItem } from "@/lib/types";
import { useAdminT } from "@/lib/i18n/admin";
import { canWeigh, weigh } from "@/lib/tillBridge";

export default function WeightDialog({
  item,
  scalePort,
  onCancel,
  onConfirm,
}: {
  item: MenuItem;
  /** The serial port a counter scale is wired to, when there is one. */
  scalePort?: string;
  onCancel: () => void;
  onConfirm: (kg: number) => void;
}) {
  const t = useAdminT();
  const [kg, setKg] = useState("");
  const [reading, setReading] = useState(false);
  const [error, setError] = useState("");

  function confirm() {
    const n = weightOf(kg);
    if (!n) return;
    onConfirm(n);
  }

  return (
    // ⚠️ A number pad rather than a free field — this is entered with a thumb,
    // at speed, and a stray letter here is a line that cannot be priced.
    <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-6">
      <div className="card w-full max-w-sm space-y-3 p-5">
        <div className="text-base font-semibold">{item.name}</div>
        {canWeigh() && !!scalePort && (
          <button
            className="till-btn w-full"
            disabled={reading}
            onClick={async () => {
              setReading(true);
              setError("");
              try {
                setKg(String(await weigh(scalePort)));
              } catch (e) {
                // The binary's own words: "the scale did not answer", "there is
                // no weight on it" — each sends somebody somewhere different.
                setError(e instanceof Error ? e.message : t.till.barcodeFailed);
              } finally {
                setReading(false);
              }
            }}
          >
            {t.till.weighRead}
          </button>
        )}
        {error && <p className="text-warn text-xs">{error}</p>}
        <input
          className="till-input h-12 w-full text-lg"
          autoFocus
          inputMode="decimal"
          placeholder={t.till.kgUnit}
          value={kg}
          onChange={(e) => setKg(e.target.value.replace(/[^0-9.,]/g, ""))}
          onKeyDown={(e) => {
            if (e.key === "Enter") confirm();
          }}
        />
        <div className="flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn till-btn-primary flex-1"
            disabled={!weightOf(kg)}
            onClick={confirm}
          >
            {t.till.barcodeAdd}
          </button>
        </div>
      </div>
    </div>
  );
}

/** The typed weight as a number, or 0 when it is not one.
 *
 *  ⚠️ **A comma is a decimal point here.** Every Uzbek keyboard offers one and
 *  half the scales print one; `parseFloat("1,5")` is 1, which would sell a kilo
 *  and a half of meat as one kilo without a single error anywhere. */
export function weightOf(s: string): number {
  const n = parseFloat(s.replace(",", "."));
  return Number.isFinite(n) && n > 0 ? n : 0;
}

/** Whether this product is sold by weight.
 *
 *  ⚠️ Read from the fiscal measure code, which is the one place this is already
 *  recorded — a second flag would be a second answer to the same question, and
 *  the receipt would eventually disagree with the scale. 10 = gram,
 *  11 = kilogram, 41 = litre. */
export function byWeight(item: MenuItem): boolean {
  return item.unitCode === 10 || item.unitCode === 11 || item.unitCode === 41;
}

"use client";

// The counter of a shop, which is one action repeated.
//
// ⚠️ **A restaurant is tapped; a shop is scanned.** A waiter knows the menu and
// reaches for a tile. A cashier faces three thousand packets nobody has
// memorised, and their whole job is beep, beep, total — so this screen has no
// grid, no categories and nothing to browse. What it has is a field that is
// always listening and a list of what has gone in.
//
// ⚠️ **The scanner is a keyboard.** Almost every reader sold here is a
// keyboard wedge: it types the digits and presses Enter. That is why there is a
// real input rather than a camera, why it takes the focus back whenever it loses
// it, and why Enter is the only submit — a cashier never touches this field
// deliberately, and a field that has quietly lost focus is a scan that goes
// nowhere while somebody keeps scanning.

import { useCallback, useEffect, useRef, useState } from "react";

import { api, ApiError } from "@/lib/api";
import type { MenuItem } from "@/lib/types";
import { useAdminT } from "@/lib/i18n/admin";
import { formatPrice } from "@/lib/format";
import { canWeigh, weigh } from "@/lib/tillBridge";

/** What one scan turned into, for the row the cashier reads back. */
type Scanned = {
  item: MenuItem;
  /** Kilograms when the goods are weighed, otherwise a count.
   *
   *  ⚠️ **Always a quantity, never a price.** A scale that printed money has
   *  already been divided by the catalogue price on the server — see
   *  `api.tillScan`. One concept reaches the check. */
  qty: number;
};

export default function ScanPanel({
  onAdd,
  weighs,
  scalePort,
}: {
  /** Put this on the check. ⚠️ The panel never writes to the check itself: the
   *  check is the till's, and two things writing to one check is how a line
   *  appears twice. */
  onAdd: (item: MenuItem, qty: number) => Promise<void>;
  /** Whether a manual weight box is worth offering. A shop with no scales
   *  never wants one. */
  weighs: boolean;
  /** The serial port a counter scale is wired to, if any. */
  scalePort?: string;
}) {
  const t = useAdminT();
  const box = useRef<HTMLInputElement>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [last, setLast] = useState<Scanned | null>(null);
  /** A weighed product that arrived without a weight — mode three. */
  const [asking, setAsking] = useState<MenuItem | null>(null);
  const [kg, setKg] = useState("");
  const [reading, setReading] = useState(false);
  const [weighError, setWeighError] = useState("");

  // ⚠️ **The field takes the focus back, and this is not a nicety.** A cashier
  // who taps the check to correct a line leaves this input; the next scan then
  // types its digits into nothing and ends with an Enter that does nothing at
  // all. They scan again, harder, and report that the scanner is broken.
  const grab = useCallback(() => box.current?.focus(), []);
  useEffect(() => {
    grab();
    const t = setInterval(grab, 1500);
    return () => clearInterval(t);
  }, [grab]);

  async function submit(raw: string) {
    const value = raw.trim();
    if (!value || busy) return;
    setBusy(true);
    setError("");
    try {
      const res = await api.tillScan(value);
      setCode("");
      if (!res.found || !res.item) {
        // ⚠️ The code is shown because it is what somebody will be asked for,
        // and because reading it off the screen is faster than reading it off a
        // crumpled sticker.
        setError(t.till.barcodeUnknown(res.code ?? value));
        return;
      }
      if (res.soldOut) {
        // ⚠️ Said before it is added, not after: the cashier is still holding
        // the packet and has not yet told the customer a price.
        setError(t.till.barcodeSoldOut(res.item.name));
        return;
      }

      // ⚠️ The sticker and the catalogue disagree — said before the line goes
      // on, because the customer is holding the sticker.
      if (res.priceMismatch) {
        setError(t.till.barcodeStale(res.priceMismatch));
        return;
      }
      // A scale label, whether it printed grams or money: the server has
      // resolved both to kilograms.
      if (res.weighed && res.kg) {
        await put({ item: res.item, qty: res.kg });
        return;
      }
      // ⚠️ **A weighed product with no weight is the third mode, not a
      // failure.** It happens when the goods are picked from the shelf and
      // weighed at the counter, or when a connected scale has not been wired up
      // yet. Asking is the honest answer; guessing one kilogram is not.
      if (weighs && byWeight(res.item)) {
        setAsking(res.item);
        setKg("");
        return;
      }
      await put({ item: res.item, qty: 1 });
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.till.barcodeFailed);
    } finally {
      setBusy(false);
      grab();
    }
  }

  async function put(s: Scanned) {
    await onAdd(s.item, s.qty);
    setLast(s);
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b border-line bg-surface px-3 py-2.5">
        <input
          ref={box}
          className="till-input h-11 flex-1 text-lg"
          placeholder={t.till.barcodePlaceholder}
          value={code}
          onChange={(e) => setCode(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              void submit(code);
            }
          }}
          // ⚠️ Off, all of it: a barcode is not a word. Autocorrect on an
          // Android tablet turns a digit string into something else entirely,
          // and the failure looks like the scanner misreading.
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck={false}
          inputMode="numeric"
        />
        <button
          className="till-btn h-11 shrink-0 px-4"
          disabled={!code.trim() || busy}
          onClick={() => void submit(code)}
        >
          {t.till.barcodeAdd}
        </button>
      </div>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
        {error && (
          <div className="rounded-2xl border border-warn/40 bg-warn/10 p-4 text-sm">
            {error}
          </div>
        )}

        {/* ⚠️ **The last thing scanned, large.** A cashier scanning at speed
            never reads the whole check — they glance once to confirm the beep
            was the right packet, and that glance has to land on a name and a
            number without being aimed. */}
        {last && !error && (
          <div className="rounded-2xl border border-line bg-raised p-4">
            <div className="text-base font-semibold">{last.item.name}</div>
            <div className="text-ink-muted text-sm">
              {byWeight(last.item)
                ? `${last.qty} ${t.till.kgUnit}`
                : `× ${last.qty}`}{" "}
              ·{" "}
              {formatPrice(Math.round(last.item.price * last.qty))}
            </div>
          </div>
        )}

        {!last && !error && (
          <p className="text-ink-muted p-6 text-center text-sm">
            {t.till.barcodeHint}
          </p>
        )}
      </div>

      {/* Mode three: the goods are on the counter and somebody types the
          weight. ⚠️ A number pad rather than a free field — this is entered
          with a thumb, at speed, and a stray letter here is a line that cannot
          be priced. */}
      {asking && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-6">
          <div className="card w-full max-w-sm space-y-3 p-5">
            <div className="text-base font-semibold">{asking.name}</div>
            {/* ⚠️ **The reading is shown and confirmed, never added straight
                to the check.** That is what makes a connected scale safe in a
                way a printed label is not: a misread is caught by the person
                holding the goods. The button fills the box; the cashier still
                presses add. */}
            {canWeigh() && !!scalePort && (
              <button
                className="till-btn w-full"
                disabled={reading}
                onClick={async () => {
                  setReading(true);
                  setWeighError("");
                  try {
                    const kg = await weigh(scalePort);
                    setKg(String(kg));
                  } catch (e) {
                    // The binary's own words: "the scale did not answer",
                    // "there is no weight on it" — each sends somebody
                    // somewhere different.
                    setWeighError(
                      e instanceof Error ? e.message : t.till.barcodeFailed,
                    );
                  } finally {
                    setReading(false);
                  }
                }}
              >
                {t.till.weighRead}
              </button>
            )}
            {weighError && (
              <p className="text-warn text-xs">{weighError}</p>
            )}
            <input
              className="till-input h-12 w-full text-lg"
              autoFocus
              inputMode="decimal"
              placeholder={t.till.kgUnit}
              value={kg}
              onChange={(e) => setKg(e.target.value.replace(/[^0-9.,]/g, ""))}
              onKeyDown={(e) => {
                if (e.key === "Enter") confirmWeight();
              }}
            />
            <div className="flex gap-2">
              <button
                className="till-btn flex-1"
                onClick={() => {
                  setAsking(null);
                  grab();
                }}
              >
                {t.till.back}
              </button>
              <button
                className="till-btn till-btn-primary flex-1"
                disabled={!weightOf(kg)}
                onClick={confirmWeight}
              >
                {t.till.barcodeAdd}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );

  function confirmWeight() {
    const n = weightOf(kg);
    if (!asking || !n) return;
    const item = asking;
    setAsking(null);
    void put({ item, qty: n }).finally(grab);
  }
}

/** ⚠️ A comma is a decimal point here, and a keypad on an Uzbek phone sends
 *  one. Read as nothing, "1,5" becomes an empty weight and the cashier retypes
 *  it wondering what they did wrong. */
function weightOf(raw: string): number {
  const n = Number(raw.replace(",", "."));
  return Number.isFinite(n) && n > 0 ? n : 0;
}

/** Whether this product is sold by weight.
 *
 *  ⚠️ Read from the fiscal measure code, which is the one place this is already
 *  recorded — a second flag would be a second answer to the same question, and
 *  the receipt would eventually disagree with the scale. 10 = gram,
 *  11 = kilogram, 41 = litre. */
function byWeight(item: MenuItem): boolean {
  return item.unitCode === 10 || item.unitCode === 11 || item.unitCode === 41;
}

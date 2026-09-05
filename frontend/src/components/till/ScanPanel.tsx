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
import WeightDialog, { byWeight } from "./WeightDialog";

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
  scalePort,
}: {
  /** Put this on the check. ⚠️ The panel never writes to the check itself: the
   *  check is the till's, and two things writing to one check is how a line
   *  appears twice. */
  onAdd: (item: MenuItem, qty: number) => Promise<void>;
  /** Whether a manual weight box is worth offering. A shop with no scales
   *  never wants one. */
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
  /** Whether the cashier asked to type the digits themselves.
   *
   *  ⚠️ **Off by default, and that is what keeps the on-screen keyboard down.**
   *  The field must hold the focus — a scanner types into whatever has it — but
   *  a focused numeric input on a touchscreen summons the software keyboard,
   *  and this one takes the focus back every second and a half. The keyboard
   *  therefore reappeared over the check for the whole shift, however many
   *  times it was dismissed. `inputMode: "none"` keeps the focus and refuses
   *  the keyboard; a hardware scanner is unaffected, because it is a keyboard
   *  itself. */
  const [typing, setTyping] = useState(false);

  // ⚠️ **The field takes the focus back, and this is not a nicety.** A cashier
  // who taps the check to correct a line leaves this input; the next scan then
  // types its digits into nothing and ends with an Enter that does nothing at
  // all. They scan again, harder, and report that the scanner is broken.
  // ⚠️ **Never out of another field somebody is typing in.** Taking it back
  // from a *button* is the whole point — that is where the focus lands after
  // every tap on the screen, and the next scan has to reach this box. Taking it
  // back from an input is the opposite: "1.5" typed into the weight dialog
  // arrived as "1" there and ".5" here, and the search box above the cards
  // could not be typed into at all. The cashier saw their own digits jump
  // between two fields and had no way to describe it.
  const grab = useCallback(() => {
    const el = document.activeElement as HTMLElement | null;
    const busyElsewhere =
      !!el &&
      el !== box.current &&
      (el.tagName === "INPUT" ||
        el.tagName === "TEXTAREA" ||
        el.isContentEditable);
    if (busyElsewhere) return;
    box.current?.focus();
  }, []);
  useEffect(() => {
    grab();
    const t = setInterval(grab, 1500);
    return () => clearInterval(t);
  }, [grab]);

  /** Codes waiting their turn, so none is lost and none overtakes another.
   *
   *  ⚠️ **A queue, because refusing while busy loses the packet.** A scanner
   *  sends the next code the moment it is pointed at the next label, which is
   *  well inside the round trip for the one before it. Dropping that scan is
   *  the worst available outcome: the beep sounds, the cashier moves on, and
   *  the item is simply not on the bill. Nobody finds out at the counter —
   *  it turns up as a shortfall at the next stocktake, with nothing to tie it
   *  to.
   *
   *  ⚠️ **In order, one at a time.** They must not run together either: the
   *  first scan of a sale opens the check, and a second one racing it would
   *  ask for a second check. */
  const queue = useRef<Promise<void>>(Promise.resolve());

  function enqueue(raw: string) {
    const value = raw.trim();
    if (!value) return;
    // ⚠️ Cleared here rather than after the round trip: the next code is
    // already being typed into this box by the scanner.
    setCode("");
    queue.current = queue.current.then(() => submit(value));
  }

  /** Ask for a keyboard.
   *
   *  ⚠️ **Re-focused, not merely focused.** The pad opens on `focusin`, and
   *  this field is already focused — it never lets go. Without letting it go
   *  and taking it back, switching the field to a typable mode changes nothing
   *  a person can see, and the button looks broken. */
  function startTyping() {
    if (typing) return;
    setTyping(true);
    setTimeout(() => {
      box.current?.blur();
      box.current?.focus();
    }, 0);
  }

  /** Put it away and go back to listening for the scanner. */
  function stopTyping() {
    setTyping(false);
    setTimeout(() => {
      box.current?.blur();
      box.current?.focus();
    }, 0);
  }

  async function submit(value: string) {
    setBusy(true);
    setError("");
    try {
      const res = await api.tillScan(value);
      // ⚠️ **Back to scanning after a code goes through.** Typing one in by
      // hand is the exception — a torn label — and leaving the keyboard up
      // afterwards would put it back over the check for the rest of the queue.
      stopTyping();
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
      // ⚠️ **Asked whenever the catalogue says the packet is weighed**, and
      // not gated on the branch's scale settings. Those describe the equipment;
      // this describes the goods. A shop whose scales are not set up yet would
      // otherwise ring a kilogram of anything as one — quietly, on the receipt,
      // at the price of one unit.
      if (byWeight(res.item)) {
        setAsking(res.item);
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

  // ⚠️ **A band across the top, not the whole pane.** The counter keeps the
  // product cards underneath it: a scanner that has stopped reading is an
  // ordinary morning — a cable, a dead battery, a label the freezer rubbed off
  // — and a screen with nothing but a dead input on it is a shop that cannot
  // sell anything until somebody arrives with a new one.
  return (
    <div className="flex shrink-0 flex-col border-b border-line">
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
              enqueue(code);
            }
          }}
          // ⚠️ Off, all of it: a barcode is not a word. Autocorrect on an
          // Android tablet turns a digit string into something else entirely,
          // and the failure looks like the scanner misreading.
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck={false}
          // ⚠️ **"none" until somebody asks to type.** See `typing` above: this
          // is the difference between a till that holds the scanner's focus and
          // a till whose keyboard covers the check all day.
          inputMode={typing ? "numeric" : "none"}
          // ⚠️ **And the till's own pad has to be told separately.** It opens on
          // `focusin` for any editable field, and this field holds the focus
          // permanently so a scan never lands in nothing — which meant the pad
          // stood open over the whole counter, all day, on every screen. The
          // opt-out already existed for exactly this case; `inputMode="none"`
          // keeps the operating system's pad away and this keeps ours away, and
          // both are lifted the moment somebody asks to type.
          data-osk={typing ? undefined : "off"}
          // Tapping the field is asking for the keyboard — the gesture anybody
          // would try first, and the one the ⌨ button exists to make findable.
          onPointerDown={() => startTyping()}
        />
        {/* The way in for a code that will not scan — a torn label, a packet
            whose barcode is under the fold. ⚠️ A button rather than "just tap
            the field": tapping the field is what the cashier does by accident
            all day, and if that opened the keyboard nothing above would have
            changed. */}
        <button
          className="till-btn h-11 w-11 shrink-0 px-0 text-base"
          aria-pressed={typing}
          title={t.till.barcodeType}
          aria-label={t.till.barcodeType}
          onClick={() => (typing ? stopTyping() : startTyping())}
        >
          ⌨
        </button>
        <button
          className="till-btn h-11 shrink-0 px-4"
          disabled={!code.trim() || busy}
          onClick={() => enqueue(code)}
        >
          {t.till.barcodeAdd}
        </button>
      </div>

      <div className="space-y-2 px-3 py-2">
        {error && (
          <div className="rounded-xl border border-warn/40 bg-warn/10 px-3 py-2 text-sm">
            {error}
          </div>
        )}

        {/* ⚠️ **The last thing scanned, large.** A cashier scanning at speed
            never reads the whole check — they glance once to confirm the beep
            was the right packet, and that glance has to land on a name and a
            number without being aimed. */}
        {last && !error && (
          <div className="rounded-xl border border-line bg-raised px-3 py-2">
            <div className="text-sm font-semibold">{last.item.name}</div>
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
          <p className="text-ink-muted text-center text-xs">
            {t.till.barcodeHint}
          </p>
        )}
      </div>

      {asking && (
        <WeightDialog
          item={asking}
          scalePort={scalePort}
          onCancel={() => {
            setAsking(null);
            grab();
          }}
          onConfirm={(kg) => {
            const item = asking;
            setAsking(null);
            void put({ item, qty: kg }).finally(grab);
          }}
        />
      )}
    </div>
  );

}



"use client";

import { useEffect, useState } from "react";

import { api, ApiError, setTillToken } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import LangSwitch from "@/components/site/LangSwitch";
import type { TillPerson } from "@/lib/types";

import KeelMark from "./KeelMark";

/**
 * The lock screen: who is standing at this till.
 *
 * ⚠️ **A till is a shared screen**, and until this existed one account stayed
 * signed in all evening — so every void, every discount and every closed check
 * was recorded against whoever unlocked it at six. Those records exist to
 * answer one question, and the login model was quietly answering it wrong.
 *
 * ⚠️ **The PIN is not the authentication.** The monoblock already holds a token
 * for its branch, obtained once with a real username and password. The four
 * digits only say *who*. That is why they can be four digits at all — and why
 * the token they buy is weaker than the one behind it.
 */
/** How many digits a till code has. Fixed, so the pad can draw exactly that
 *  many dots and submit by itself — see pinDigits on the server. */
const PIN_DIGITS = 4;

export default function PinPad({
  onUnlock,
}: {
  onUnlock: (person: TillPerson) => void;
}) {
  const t = useAdminT();
  const [pin, setPin] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // ⚠️ **Submits itself on the fourth digit.** A confirming tap after every
  // code is a tap added to the busiest screen in the building, and the pad
  // knows exactly when the code is complete because the length is fixed.
  useEffect(() => {
    if (pin.length === PIN_DIGITS) void submit(pin);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pin]);

  async function submit(code: string) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const res = await api.tillUnlock(code);
      setTillToken(res.token);
      setPin("");
      onUnlock(res.staff);
    } catch (e) {
      // ⚠️ The server's own words. It distinguishes "wrong code" from "too many
      // tries, wait N seconds", and a cashier who cannot tell those apart
      // retypes the same PIN and extends their own lockout.
      setError(e instanceof ApiError ? e.message : t.till.retry);
      setPin("");
    } finally {
      setBusy(false);
    }
  }

  function tap(d: string) {
    if (busy) return;
    setError("");
    setPin((p) => (p.length >= PIN_DIGITS ? p : p + d));
  }

  return (
    // ⚠️ `overflow-y-auto` as a floor, not a design: a monoblock is 768px tall
    // and this fits, but the same screen is opened on a phone to check
    // something, and a lock screen whose last row of keys is off the edge
    // cannot be got past at all.
    <div className="till relative flex min-h-dvh flex-col items-center justify-center overflow-y-auto bg-cream px-6 py-6">
      {/* ⚠️ **The language switch belongs on this screen, not behind it.** The
          till is a shared machine and the person who set it up is not the
          person standing at it now; a cashier who reads Russian met an
          Uzbek-only lock screen and had no way past it to the switch that would
          have fixed the whole shift. */}
      <div className="absolute right-4 top-4">
        <LangSwitch />
      </div>

      {/* ---- The code ----

          ⚠️ **Low on the screen on purpose.** A monoblock stands upright on a
          counter and the pad is reached across it, so the digits belong near
          the bottom edge where a hand already rests — a pad centred on a 15"
          panel is typed at with a raised arm, hundreds of times a shift. */}
      {/* ⚠️ Centred, and the container keeps `overflow-y-auto` behind it: on a
          short viewport a centred block that outgrows the screen loses **both**
          ends, and the end it loses at the bottom is the row with the
          backspace. Centring decides where the group sits; the scroll decides
          that it can always be reached. */}
      <div className="flex flex-col items-center">
        {/* ---- Whose machine this is ----

            ⚠️ **Grouped with the pad, not floated at the top.** The mark and
            the code are one thing the eye lands on; separated by half a screen
            they read as two, and the empty band between them was the largest
            shape on the display. The gap here is one line — enough to say they
            are different, not enough to make you look twice.

            ⚠️ **Our colour, not the restaurant's.** `text-brand` would draw the
            Keel mark in whatever accent the owner picked, which is a different
            logo in every install. */}
        <div className="mb-7 flex items-center gap-3">
          <KeelMark className="h-14 w-14 text-keel-deep" />
          {/* ⚠️ Our own type, not the restaurant's. The theme fonts dress the
              restaurant — its menu, its site, its receipts — and letting them
              reset our name would make the wordmark different in every
              install. */}
          <span className="font-poppins text-4xl font-semibold tracking-tight text-ink">
            Keel
          </span>
        </div>

        <h1 className="text-base font-semibold">{t.till.pinTitle}</h1>
        <p className="mt-0.5 text-sm text-ink-muted">{t.till.pinHint}</p>

        {/* Dots rather than digits: the pad is at head height in a room with
            guests and colleagues in it. */}
        <div className="mt-4 flex justify-center gap-3.5">
          {Array.from({ length: PIN_DIGITS }, (_, i) => (
            <span
              key={i}
              className={`h-3.5 w-3.5 rounded-full transition-colors ${
                i < pin.length ? "bg-brand" : "bg-ink/15"
              }`}
            />
          ))}
        </div>

        {/* ⚠️ The row is always there, empty or not: a message that appears
            pushes the whole pad down, and the key under the finger changes
            between the tap that failed and the retry. */}
        <p className="mt-2 h-5 text-sm text-danger">{error}</p>

        {/* Big targets: this is tapped hundreds of times a day, often with a
            wet hand, on a screen at arm's length. */}
        <div className="mt-1 grid w-[17rem] grid-cols-3 gap-2.5">
          {["1", "2", "3", "4", "5", "6", "7", "8", "9"].map((d) => (
            <PadKey key={d} onClick={() => tap(d)}>
              {d}
            </PadKey>
          ))}
          {/* No confirm key: the pad submits on the fourth digit, so a tick
              would be a control that is never the right thing to press. */}
          <span />
          <PadKey onClick={() => tap("0")}>0</PadKey>
          <PadKey onClick={() => setPin("")} disabled={pin.length === 0}>
            ⌫
          </PadKey>
        </div>
      </div>
    </div>
  );
}

function PadKey({
  children,
  onClick,
  disabled,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="rounded-[12px] border border-line bg-surface py-3.5 font-display text-2xl font-bold text-ink shadow-sm transition active:scale-[0.96] active:bg-ink/10 disabled:opacity-30"
    >
      {children}
    </button>
  );
}

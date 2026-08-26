"use client";

import { useEffect, useState } from "react";

import { LuKeyRound } from "react-icons/lu";

import { useAdminT } from "@/lib/i18n/admin";

/**
 * "Fetch somebody who may."
 *
 * ⚠️ **This dialog is what makes restrictive permissions survive a Friday
 * evening.** A waiter who needs to write off a burnt steak and is refused does
 * not go and find the manager — within a fortnight the manager's PIN is known
 * to the whole room, and after that every void in the journal carries one name,
 * always the same one. The permission system would have destroyed the
 * attribution it exists to protect.
 *
 * So the refusal is not a dead end. The manager is standing in the room; they
 * tap four digits on the screen already in front of the waiter, the action goes
 * through, and **both names are recorded**.
 *
 * ⚠️ **It asks for a code, never for a decision.** The dialog hands the PIN back
 * and the server decides — a screen that could report "a manager approved this"
 * would be a screen that can approve anything.
 */
export default function OverrideDialog({
  permissionName,
  busy,
  error,
  onCancel,
  onSubmit,
}: {
  /** What the action needed, in words: "Chegirma berish". */
  permissionName: string;
  busy: boolean;
  /** The server's answer to the last attempt, if it refused again. */
  error: string;
  onCancel: () => void;
  onSubmit: (pin: string) => void;
}) {
  const t = useAdminT();
  const [pin, setPin] = useState("");

  // Submits itself on the fourth digit, exactly as the lock screen does — the
  // two pads must not behave differently, or the muscle memory that makes one
  // fast makes the other wrong.
  useEffect(() => {
    if (pin.length === PIN_DIGITS) {
      onSubmit(pin);
      setPin("");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pin]);

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-ink/50 p-4">
      <div className="till-dialog w-full max-w-xs p-4 text-center">
        <h2 className="flex items-center gap-2 font-display text-lg font-bold">
          <LuKeyRound className="text-ink-muted" aria-hidden />
          {t.till.overrideTitle}
        </h2>
        {/* ⚠️ Names the permission, not the failure. "You may not" tells the
            waiter nothing they can act on; "Chegirma berish uchun ruxsat
            kerak" tells them exactly what to ask for. */}
        <p className="mt-1 text-sm text-ink-muted">
          {permissionName} — {t.till.overrideHint}
        </p>

        <div className="mt-5 flex justify-center gap-4">
          {Array.from({ length: PIN_DIGITS }, (_, i) => (
            <span
              key={i}
              className={`h-4 w-4 rounded-full ${
                i < pin.length ? "bg-keel-deep" : "bg-ink/20"
              }`}
            />
          ))}
        </div>

        <p className="mt-3 h-5 text-sm text-danger">{error}</p>

        <div className="mt-2 grid grid-cols-3 gap-2">
          {["1", "2", "3", "4", "5", "6", "7", "8", "9"].map((d) => (
            <Key key={d} onClick={() => !busy && setPin((p) => p + d)}>
              {d}
            </Key>
          ))}
          <Key onClick={onCancel}>✕</Key>
          <Key onClick={() => !busy && setPin((p) => p + "0")}>0</Key>
          <Key onClick={() => setPin("")} disabled={pin.length === 0}>
            ⌫
          </Key>
        </div>
      </div>
    </div>
  );
}

const PIN_DIGITS = 4;

function Key({
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
      className="rounded-[12px] border border-line bg-surface py-3.5 font-display text-xl font-bold transition active:scale-[0.96] active:bg-ink/10 disabled:opacity-40"
    >
      {children}
    </button>
  );
}

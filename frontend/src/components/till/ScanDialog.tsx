"use client";

// The scan a marked bottle cannot be sold without.
//
// ⚠️ **The scanner is a keyboard, so this dialog is a focused input and
// nothing else.** Pistol scanners run in HID mode: no driver, no permission
// prompt, no device API — the code arrives as typed text ending in Enter.
// Anything cleverer would be a second way for the hardware to be wrong.
//
// ⚠️ **It refuses here, where the bottle is in somebody's hand.** The tax
// register refuses a bad code too, but it does so during the payment, in front
// of a guest, when the only remedy is to scan again — which is the remedy that
// should have happened before the sale.
//
// ⚠️ **One code, one bottle.** There is no quantity: two bottles are two scans,
// because a line of two behind one code files one and hands over two.

import { useEffect, useRef, useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import { checkMark, isDuplicateMark, normalizeMark } from "@/lib/marking";
import type { MenuItem } from "@/lib/types";

export default function ScanDialog({
  item,
  existing,
  busy,
  onCancel,
  onScanned,
}: {
  item: MenuItem;
  /** Codes already on this check, so the same bottle cannot be counted twice. */
  existing: string[];
  busy: boolean;
  onCancel: () => void;
  onScanned: (code: string) => void;
}) {
  const t = useAdminT();
  const [raw, setRaw] = useState("");
  const [problem, setProblem] = useState<string | null>(null);
  const field = useRef<HTMLInputElement>(null);

  // ⚠️ Focused on open and kept focused: the scanner types into whatever has
  // the caret, so a dialog that opens unfocused sends the code into the check's
  // comment box — or nowhere.
  useEffect(() => {
    field.current?.focus();
  }, []);

  function submit(value: string) {
    const code = normalizeMark(value);
    const bad = checkMark(code);
    if (bad) {
      setProblem(bad === "empty" ? t.till.scanEmpty : t.till.scanShape);
      setRaw("");
      field.current?.focus();
      return;
    }
    if (isDuplicateMark(code, existing)) {
      setProblem(t.till.scanDuplicate);
      setRaw("");
      field.current?.focus();
      return;
    }
    onScanned(code);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-6">
      <div className="card w-full max-w-md space-y-3 p-5">
        <h2 className="text-lg font-semibold">{t.till.scanTitle}</h2>
        <p className="text-sm text-ink-muted">
          {item.name} — {t.till.scanBody}
        </p>
        <input
          ref={field}
          className="till-input w-full font-mono text-sm"
          // ⚠️ The till's own keypad must not open over this: the code comes
          // from a pistol, and a cashier typing 60 characters by hand at a
          // counter is not the flow this exists for.
          inputMode="none"
          autoComplete="off"
          value={raw}
          placeholder={t.till.scanPlaceholder}
          onChange={(e) => {
            setProblem(null);
            setRaw(e.target.value);
          }}
          onKeyDown={(e) => {
            // The Enter the scanner sends at the end of the code.
            if (e.key === "Enter") {
              e.preventDefault();
              submit(raw);
            }
          }}
        />
        {problem && <p className="text-sm text-danger">{problem}</p>}
        <div className="flex justify-end gap-2">
          <button type="button" className="btn-ghost" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            type="button"
            className="btn-primary"
            disabled={busy || raw.trim() === ""}
            onClick={() => submit(raw)}
          >
            {t.till.scanAdd}
          </button>
        </div>
      </div>
    </div>
  );
}

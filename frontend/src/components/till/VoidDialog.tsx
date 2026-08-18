"use client";

import { useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import type { CheckLine } from "@/lib/types";

/**
 * Asking why.
 *
 * Used for the two things on this screen that cost the restaurant food or money
 * and cannot be undone: taking a cooked dish off a check, and cancelling a whole
 * check. Both refuse an empty reason — here and again on the server, because a
 * reason nobody can be made to give is a column full of blanks by the second
 * week.
 *
 * The "thrown away" box appears only for a dish, and only because it is the sole
 * input the waste figures have: whether the kitchen caught it in time is
 * something only the person at the pass knows, and it is unrecoverable one
 * minute later.
 */
export default function VoidDialog({
  line,
  title,
  label,
  confirmLabel,
  onCancel,
  onConfirm,
}: {
  line?: CheckLine;
  title?: string;
  label?: string;
  confirmLabel?: string;
  onCancel: () => void;
  onConfirm: (reason: string, wasted: boolean) => void | Promise<void>;
}) {
  const t = useAdminT();
  const [reason, setReason] = useState("");
  const [wasted, setWasted] = useState(true);
  const ok = reason.trim().length > 0;

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="font-display text-lg font-bold">
          {title ?? t.till.voidTitle}
        </h2>
        {line && (
          <>
            <p className="mt-1 text-sm text-ink-soft">
              {line.qty} × {line.name}
            </p>
            <p className="mt-2 text-xs text-ink-muted">{t.till.voidHint}</p>
          </>
        )}

        <label className="mt-4 block text-sm">
          <span className="text-ink-muted">{label ?? t.till.voidReason}</span>
          <input
            className="till-input mt-1"
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        </label>

        {line && (
          <label className="mt-3 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={wasted}
              onChange={(e) => setWasted(e.target.checked)}
            />
            <span>{t.till.voidWasted}</span>
          </label>
        )}

        <div className="mt-5 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.till.back}
          </button>
          <button
            className="till-btn-primary flex-1"
            disabled={!ok}
            onClick={() => void onConfirm(reason.trim(), wasted)}
          >
            {confirmLabel ?? t.till.remove}
          </button>
        </div>
        {!ok && (
          <p className="mt-2 text-center text-xs text-ink-muted">
            {t.till.reasonRequired}
          </p>
        )}
      </div>
    </div>
  );
}

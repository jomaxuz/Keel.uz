"use client";

// The note that goes to the kitchen with one dish.
//
// ⚠️ **Shared by both screens on purpose.** It lived inside the floor's order
// panel, and the till had no way to write one at all — a cashier taking a
// counter order for a guest saying "no onions" had to remember it and tell the
// kitchen out loud. Two copies of this dialog would drift, and the drift would
// be a note that reaches the pass from one screen and not the other.

import { useState } from "react";

import { LuMessageSquare } from "react-icons/lu";

import { useAdminT } from "@/lib/i18n/admin";
import type { CheckLine } from "@/lib/types";

export default function CommentDialog({
  line,
  onCancel,
  onSave,
}: {
  line: CheckLine;
  onCancel: () => void;
  onSave: (comment: string) => void | Promise<void>;
}) {
  const t = useAdminT();
  const [text, setText] = useState(line.comment ?? "");
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="flex items-center gap-2 font-display text-lg font-bold">
          <LuMessageSquare className="text-ink-muted" aria-hidden />
          {t.till.commentTitle}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">{line.name}</p>
        <input
          className="till-input mt-3 h-12"
          autoFocus
          placeholder={t.till.commentPh}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <div className="mt-4 flex gap-2">
          <button className="till-btn flex-1" onClick={onCancel}>
            {t.common.cancel}
          </button>
          <button
            className="till-btn-primary flex-1"
            onClick={() => void onSave(text.trim())}
          >
            {t.common.save}
          </button>
        </div>
      </div>
    </div>
  );
}

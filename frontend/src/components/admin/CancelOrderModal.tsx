"use client";

// Cancelling an order always asks why. The customer sees the reason on the
// tracking page, so "why was my order cancelled?" never becomes a phone call —
// which is also why the reason is required, not optional.

import { useState } from "react";
import Modal from "@/components/admin/Modal";
import { useAdminT } from "@/lib/i18n/admin";
import { usePanelWords } from "@/lib/panelWords";
import type { Order } from "@/lib/types";

export default function CancelOrderModal({
  order,
  onClose,
  onConfirm,
}: {
  order: Order;
  onClose: () => void;
  // Resolves once the status change has been sent.
  onConfirm: (reason: string) => Promise<void> | void;
}) {
  const t = useAdminT();
  const w = usePanelWords();
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const trimmed = reason.trim();

  async function submit() {
    if (!trimmed) return;
    setBusy(true);
    try {
      await onConfirm(trimmed);
      onClose();
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal onClose={onClose}>
      <h3 className="font-display text-lg font-bold">
        {t.orders.cancelTitle(order.number)}
      </h3>
      <p className="mt-1 text-sm text-ink-muted">{t.orders.cancelHint}</p>

      {/* The four reasons that cover almost every cancellation — one tap, then
          edit if the case is unusual. */}
      <div className="mt-4 flex flex-wrap gap-2">
        {/* ⚠️ The second preset is the one that names what was sold — "Taom
            qolmagan" in a kitchen, "Tovar qolmagan" in a shop. The other three
            are about the customer and read the same either way. */}
        {t.orders.cancelPresets
          .map((preset, i) => (i === 1 ? w.cancelPresetOut : preset))
          .map((preset) => (
          <button
            key={preset}
            type="button"
            onClick={() => setReason(preset)}
            className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors ${
              reason === preset
                ? "border-brand bg-brand-tint/50 text-brand-dark"
                : "border-line-strong text-ink-soft hover:border-brand"
            }`}
          >
            {preset}
          </button>
        ))}
      </div>

      <label className="mt-3 block text-sm">
        <span className="font-medium">{t.orders.cancelReason}</span>
        <textarea
          className="mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
          rows={3}
          maxLength={300}
          autoFocus
          placeholder={w.cancelReasonPh}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </label>

      <div className="mt-6 flex justify-end gap-3">
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2">
          {t.common.cancel}
        </button>
        <button
          type="button"
          disabled={busy || !trimmed}
          onClick={submit}
          className="btn-primary px-5 py-2 disabled:opacity-50"
        >
          {busy ? "..." : t.orders.cancelConfirm}
        </button>
      </div>
    </Modal>
  );
}

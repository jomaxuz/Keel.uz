"use client";

// Taking an order over the phone, from the orders screen.
//
// The machinery for this already existed on `/admin/calls`, where the call log
// lives. But an operator with a customer on the line is usually looking at the
// **orders** screen — it is the one that tells them what is happening in the
// kitchen right now — and from there the only route to a phone order was to
// leave the page they were watching.
//
// ⚠️ **The number comes first, and everything else follows from it.** That is not
// a form-design preference: it is the order a call actually happens in. The
// operator hears the number before anything else, and it is what the account, the
// saved addresses, the usual dishes and the unanswered complaint all hang off. A
// form that asked for a name first would be asking the operator to interview
// somebody they can already identify.
//
// It reuses `OperatorOrderModal` unchanged — the same pipeline the call centre
// and the site both run, so a phone order cannot end up priced differently from
// the same basket typed on the site.

import { useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { CallerLookup, Order, SiteUser } from "@/lib/types";
import OperatorOrderModal from "@/components/admin/OperatorOrderModal";

export default function PhoneOrderButton({
  onCreated,
}: {
  onCreated: (order: Order) => void;
}) {
  const t = useAdminT();
  const [open, setOpen] = useState(false);
  const [phone, setPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [found, setFound] = useState<{ phone: string; user: SiteUser | null } | null>(
    null,
  );

  async function find() {
    const digits = phone.replace(/\D/g, "");
    if (digits.length < 9) {
      setError(t.calls.phoneOrderHint);
      return;
    }
    setBusy(true);
    setError("");
    try {
      const res: CallerLookup = await api.adminLookup(digits);
      // A caller with no account is the normal case, not an error: the server
      // creates one from the number as the order is placed.
      setFound({ phone: res.phone || digits, user: res.user ?? null });
    } catch (e) {
      // ⚠️ A failed lookup must not block the order. The operator is mid-call,
      // and everything the lookup adds is context — the order itself only needs
      // the number.
      setFound({ phone: digits, user: null });
      void e;
    } finally {
      setBusy(false);
    }
  }

  if (found) {
    return (
      <OperatorOrderModal
        phone={found.phone}
        customer={found.user}
        onClose={() => {
          setFound(null);
          setOpen(false);
          setPhone("");
        }}
        onCreated={(order) => {
          setFound(null);
          setOpen(false);
          setPhone("");
          onCreated(order);
        }}
      />
    );
  }

  if (!open) {
    return (
      <button type="button" onClick={() => setOpen(true)} className="btn btn-dark">
        {t.calls.phoneOrder}
      </button>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <input
        autoFocus
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && void find()}
        placeholder="998901234567"
        inputMode="tel"
        className="input h-10 w-48"
      />
      <button
        type="button"
        onClick={() => void find()}
        disabled={busy}
        className="btn btn-dark disabled:opacity-40"
      >
        {busy ? t.common.loading : t.calls.phoneOrderFind}
      </button>
      <button
        type="button"
        onClick={() => {
          setOpen(false);
          setPhone("");
          setError("");
        }}
        className="btn btn-ghost"
      >
        {t.common.cancel}
      </button>
      {error && <span className="text-xs text-brand">{error}</span>}
    </div>
  );
}

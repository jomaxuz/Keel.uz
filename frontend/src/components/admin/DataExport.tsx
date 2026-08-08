"use client";

// "Take my data with me."
//
// The whole business as one zip: menu, orders, customers, bookings, staff,
// cash, images. It is the customer's own data and they must be able to leave
// with it — a restaurant that cannot is one held by the exit cost rather than
// by the product.
//
// ⚠️ **This section does not exist until the platform opens a window for it.**
// Not because the data is ours (it is not) but because of what the file is: one
// archive holding every one of their guests' names, phone numbers and
// addresses. A permanent button turns any borrowed panel session into a silent
// complete copy of the business, and the account that gets shared is exactly
// the one that would press it. So the archive is available for a few days, on
// request, with somebody's name against the decision — and it says so on
// screen, because an owner who finds a new button reasonably asks who put it
// there.
//
// The button being hidden is a courtesy, never the control: the server checks
// the grant, the clock and the owner role on the request itself.

import { useEffect, useState } from "react";
import { api, downloadDataArchive } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDate } from "@/lib/format";

export default function DataExport() {
  const t = useAdminT();
  const [status, setStatus] = useState<{
    allowed: boolean;
    reason?: string;
    expiresAt?: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);

  useEffect(() => {
    // A manager gets 403 here, and that is not an error worth showing them —
    // the section simply does not appear.
    api
      .adminExportStatus()
      .then(setStatus)
      .catch(() => setStatus(null));
  }, []);

  // Nothing at all — not an empty card, not a heading. The section has to be
  // invisible in the ordinary case, because its ordinary case is "closed".
  if (!status?.allowed) return null;

  async function download() {
    setBusy(true);
    setError("");
    setDone(false);
    try {
      await downloadDataArchive();
      setDone(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="mt-6 rounded-3xl border border-line bg-surface p-6 shadow-card">
      <h2 className="mb-4 text-lg font-bold">{t.settings.exportTitle}</h2>
      <p className="text-sm text-ink-soft">{t.settings.exportHint}</p>
      <p className="mt-2 text-xs text-ink-muted">
        {status.expiresAt
          ? t.settings.exportUntil(formatDate(status.expiresAt))
          : ""}
      </p>

      <button
        type="button"
        onClick={download}
        disabled={busy}
        className="btn btn-primary mt-4 disabled:opacity-40"
      >
        {busy ? t.settings.exportBusy : t.settings.exportBtn}
      </button>

      {/* Said before they press it, not after: a large archive takes a while
          and a page that looks frozen is a page people reload — which starts
          the whole download again. */}
      <p className="mt-2 text-xs text-ink-muted">{t.settings.exportWait}</p>

      {done && (
        <p className="mt-3 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
          {t.settings.exportDone}
        </p>
      )}
      {error && <p className="mt-3 text-sm text-brand">{error}</p>}

      <p className="mt-4 rounded-xl border border-line px-3 py-2 text-xs text-ink-soft">
        {t.settings.exportNoKeys}
      </p>
    </section>
  );
}

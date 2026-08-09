"use client";

// Choosing one customer to write to.
//
// ⚠️ **Search and the segment filters, not a scrolling list of everybody.** A
// restaurant with two thousand guests has a list nobody can find anybody in, and the
// question an operator actually arrives with is one of two: "the person who just rang"
// (a number or a name) or "one of the ones who are like this" (a segment). Both are
// here, and nothing else is.
//
// It shows what makes a name choosable — orders, spend, and whether they can be
// reached at all. ⚠️ That last one matters most: a guest who opted out or has no bot
// link is picked, and then the send is refused, and the refusal reads as a bug. Said
// here instead, before the choice.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatPrice, formatUzPhone } from "@/lib/format";
import type { AdminUserRow } from "@/lib/types";

const SEGMENTS = ["", "new", "regular", "vip", "sleeping", "lost", "birthday", "unhappy"];

export default function CustomerPicker({
  channel,
  onPick,
  onClose,
}: {
  /** Which channel the message goes by, so "cannot be reached" is accurate. */
  channel: string;
  onPick: (user: AdminUserRow) => void;
  onClose: () => void;
}) {
  const t = useAdminT();
  const [q, setQ] = useState("");
  const [segment, setSegment] = useState("");
  const [rows, setRows] = useState<AdminUserRow[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    // Debounced: the field is typed into a character at a time and each keystroke
    // would otherwise be a query against every customer.
    const timer = window.setTimeout(() => {
      setLoading(true);
      api
        .adminUsers({ q: q.trim() || undefined, segment: segment || undefined })
        .then((r) => !cancelled && setRows(r))
        .catch(() => !cancelled && setRows([]))
        .finally(() => !cancelled && setLoading(false));
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [q, segment]);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-ink/40 p-4 sm:p-8">
      <div className="card flex max-h-full w-full max-w-2xl flex-col overflow-hidden p-0">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <h3 className="font-semibold">{t.campaigns.pickCustomer}</h3>
          <button type="button" onClick={onClose} className="btn btn-ghost px-3 py-1.5 text-xs">
            {t.common.cancel}
          </button>
        </div>

        <div className="space-y-2 border-b border-line p-4">
          <input
            autoFocus
            className="input"
            value={q}
            placeholder={t.campaigns.searchPh}
            onChange={(e) => setQ(e.target.value)}
          />
          <div className="flex flex-wrap gap-1.5">
            {SEGMENTS.map((sg) => (
              <button
                key={sg}
                type="button"
                onClick={() => setSegment(sg)}
                className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold ${
                  segment === sg
                    ? "border-brand bg-brand-tint text-brand-dark"
                    : "border-line text-ink-soft"
                }`}
              >
                {sg === "" ? t.users.allCustomers : (t.users.segment[sg as keyof typeof t.users.segment] ?? sg)}
              </button>
            ))}
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-2">
          {loading && <p className="p-3 text-xs text-ink-muted">{t.common.loading}</p>}
          {!loading && rows.length === 0 && (
            <p className="p-3 text-xs text-ink-muted">{t.campaigns.noneFound}</p>
          )}
          <ul className="divide-y divide-line">
            {rows.slice(0, 100).map((u) => {
              // The reason a pick would be refused, before it is made.
              const blocked = u.noMarketing
                ? t.campaigns.blockedOptOut
                : channel === "telegram" && !u.telegramId
                  ? t.campaigns.blockedNoBot
                  : !u.phone
                    ? t.campaigns.blockedNoPhone
                    : "";
              return (
                <li key={u.id}>
                  <button
                    type="button"
                    disabled={!!blocked}
                    onClick={() => onPick(u)}
                    className="flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left hover:bg-ink/5 disabled:opacity-45"
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-semibold">
                        {u.firstName || formatUzPhone(u.phone)}
                      </span>
                      <span className="block truncate text-xs text-ink-muted">
                        {formatUzPhone(u.phone)}
                        {u.ordersCount > 0 && ` · ${u.ordersCount}`}
                        {u.ordersTotal > 0 &&
                          ` · ${formatPrice(u.ordersTotal, "UZS", "uz")}`}
                      </span>
                    </span>
                    {blocked && (
                      <span className="shrink-0 text-[11px] text-brand">{blocked}</span>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      </div>
    </div>
  );
}

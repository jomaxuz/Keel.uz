"use client";

// Who the food comes from, what they cost, and what is still owed.
//
// ⚠️ **It was free text, and free text cannot be asked a question.** A delivery
// carried whoever typed it, so "Makro", "makro" and "Makro MCHJ" were one
// company and three rows — and "how much did we buy from Makro this quarter"
// had no answer at all, not a wrong one. The text field survives, and naming a
// supplier stays optional: a market run has no supplier, and demanding one
// would stop deliveries being recorded, which costs more than an ungrouped row.
//
// ⚠️ **What is owed ignores the period.** Everything else here is measured over
// the window; a March invoice is still a debt in May, and a figure that clears
// itself when the month rolls over is not a debt — the same rule the courier's
// cash in hand follows.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type { Supplier, SupplierTotal } from "@/lib/types";

function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function monthStart() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`;
}

const BLANK = { name: "", phone: "", note: "" };

export default function SuppliersPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Supplier[]>([]);
  const [report, setReport] = useState<SupplierTotal[]>([]);
  const [spent, setSpent] = useState(0);
  const [owed, setOwed] = useState(0);
  const [from, setFrom] = useState(monthStart);
  const [to, setTo] = useState(today);
  const [draft, setDraft] = useState<typeof BLANK & { id?: string }>(BLANK);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminSuppliers()
      .then((d) => setRows(d.suppliers))
      .catch(() => setError(t.common.loadFailed));
    api
      .adminSupplierReport({ from, to })
      .then((d) => {
        setReport(d.rows);
        setSpent(d.spent);
        setOwed(d.owed);
      })
      .catch(() => setReport([]));
  }, [t.common.loadFailed, from, to]);

  useEffect(load, [load, scope.scopeKey]);

  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.adminSaveSupplier({
        id: draft.id,
        name: draft.name.trim(),
        phone: draft.phone.trim(),
        note: draft.note.trim(),
      });
      setDraft(BLANK);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.suppliers.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.suppliers.intro}</p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="card p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.suppliers.name}</span>
            <input
              className="input mt-1 w-full"
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            {/* ⚠️ The most useful field here: a number in somebody's contacts
                becomes a fact the business owns rather than one that leaves
                with them. */}
            <span className="text-xs text-ink-muted">{t.suppliers.phone}</span>
            <input
              className="input mt-1 w-40"
              value={draft.phone}
              onChange={(e) => setDraft({ ...draft, phone: e.target.value })}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.suppliers.note}</span>
            <input
              className="input mt-1 w-full"
              placeholder={t.suppliers.notePlaceholder}
              value={draft.note}
              onChange={(e) => setDraft({ ...draft, note: e.target.value })}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !draft.name.trim()}
            onClick={save}
          >
            {draft.id ? t.common.save : t.common.add}
          </button>
          {draft.id && (
            <button
              className="btn-ghost px-3 py-2"
              onClick={() => setDraft(BLANK)}
            >
              {t.common.cancel}
            </button>
          )}
        </div>
      </div>

      <div className="card p-0">
        <ListScroll>
          <ul className="divide-y divide-line">
            {rows.map((s) => (
              <li
                key={s.id}
                className={`flex items-center gap-3 px-3 py-2 text-sm ${
                  s.isActive ? "" : "opacity-50"
                }`}
              >
                <div className="min-w-0 flex-1">
                  <div className="truncate font-medium">{s.name}</div>
                  {(s.phone || s.note) && (
                    <div className="truncate text-xs text-ink-muted">
                      {[s.phone, s.note].filter(Boolean).join(" · ")}
                    </div>
                  )}
                </div>
                <button
                  className="btn-ghost px-2 py-1 text-xs"
                  onClick={() =>
                    setDraft({
                      id: s.id,
                      name: s.name,
                      phone: s.phone ?? "",
                      note: s.note ?? "",
                    })
                  }
                >
                  {t.common.edit}
                </button>
                {s.isActive && (
                  <button
                    className="btn-ghost px-2 py-1 text-xs text-danger"
                    disabled={busy}
                    onClick={async () => {
                      setBusy(true);
                      try {
                        await api.adminDeleteSupplier(s.id);
                        load();
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    {/* ⚠️ Deactivated, never deleted: every delivery ever
                        entered points here, and removing one takes the meaning
                        of a year of invoices with it. */}
                    {t.suppliers.deactivate}
                  </button>
                )}
              </li>
            ))}
          </ul>
          {rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.suppliers.empty}
            </p>
          )}
        </ListScroll>
      </div>

      {/* ---- What each one costs ---- */}
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="text-xs text-ink-muted">{t.suppliers.period}</span>
        <input
          type="date"
          className="input w-auto"
          value={from}
          onChange={(e) => setFrom(e.target.value)}
        />
        <span className="text-ink-muted">—</span>
        <input
          type="date"
          className="input w-auto"
          value={to}
          onChange={(e) => setTo(e.target.value)}
        />
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.suppliers.name}</th>
                <th className="px-3 py-2 text-right">
                  {t.suppliers.deliveries}
                </th>
                <th className="px-3 py-2 text-right">{t.suppliers.spent}</th>
                <th className="px-3 py-2 text-right">{t.suppliers.owed}</th>
              </tr>
            </thead>
            <tbody>
              {report.map((s) => (
                <tr key={s.supplierId} className="border-t border-line">
                  <td className="px-3 py-2">
                    {s.name || t.suppliers.unnamed}
                    {s.phone && (
                      <span className="ml-1.5 text-xs text-ink-muted">
                        {s.phone}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {s.count}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {formatPrice(s.spent)}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {s.owed > 0 ? (
                      <span className="font-medium text-amber-700 dark:text-amber-300">
                        {formatPrice(s.owed)}
                        <span className="ml-1 text-xs text-ink-muted">
                          ×{s.owedCount}
                        </span>
                      </span>
                    ) : (
                      <span className="text-ink-muted">—</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {report.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.suppliers.noDeliveries}
            </p>
          )}
        </ListScroll>
        {report.length > 0 && (
          <div className="flex justify-between border-t border-line px-3 py-2 text-sm">
            <span>
              {t.suppliers.spent}:{" "}
              <span className="font-medium tabular-nums">
                {formatPrice(spent)}
              </span>
            </span>
            {/* ⚠️ Named separately and never added to the spend: one is what
                left the till this period, the other is what has not left it
                yet. A single total would be neither. */}
            <span>
              {t.suppliers.owed}:{" "}
              <span className="font-medium tabular-nums">
                {formatPrice(owed)}
              </span>
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

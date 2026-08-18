"use client";

// Dining room and counter sales.
//
// ⚠️ **This screen exists because of an absence, not a gap in the data.** The
// orders board leaves till checks out on purpose (backend: AdminListOrders) —
// a room full of open tables would bury the deliveries somebody has to accept —
// and the money was always in the statistics and the reports. What was missing
// was the list in between: an owner could read that Tuesday took 4.2M and could
// not see *which sales those were*, which is the question every argument about
// a shift starts with.
//
// ⚠️ So the first line on the page says where the other half is. Two screens
// showing two different subsets of the same word ("orders") is how somebody
// concludes one of them is wrong about the money, and they would go looking for
// the bug rather than for the tab.
//
// The totals come from the server and cover the whole filtered period, not the
// rows on screen: a footer that adds up the page changes when you press "next",
// and it is the number that gets copied into a message.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice, formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll, Pager } from "@/components/admin/PagedList";
import type { CheckRow, ChecksPage } from "@/lib/types";

const PAGE = 50;

/** Today, in the restaurant's own day rather than the browser's UTC one. */
function today() {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export default function AdminChecksPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  const scope = useAdminScope();

  const [from, setFrom] = useState(today);
  const [to, setTo] = useState(today);
  const [state, setState] = useState<"all" | "open" | "closed">("all");
  const [place, setPlace] = useState<"all" | "hall" | "counter">("all");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(0);

  const [data, setData] = useState<ChecksPage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const money = (n: number) => formatPrice(n, "UZS", lang);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminChecks({
        from,
        to,
        state,
        place,
        q,
        limit: PAGE,
        skip: page * PAGE,
      })
      .then((d) => {
        setData(d);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [from, to, state, place, q, page, t.common.loadFailed]);

  // The scope key is in the dependencies for the reason every panel screen has
  // it: switching branch has to re-read, not re-label.
  useEffect(load, [load, scope.scopeKey]);

  // Any filter change starts at the first page. Staying on page 4 of a result
  // that now has two rows shows an empty screen that looks like "no sales".
  function filtered<T>(set: (v: T) => void) {
    return (v: T) => {
      setPage(0);
      set(v);
    };
  }

  const rows = data?.rows ?? [];
  const totals = data?.totals;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.sales.title}</h1>
        <p className="mt-1 text-sm text-ink-soft">{t.sales.note}</p>
      </div>

      <div className="card space-y-3 p-3">
        <div className="flex flex-wrap items-center gap-2">
          <input
            type="date"
            className="input w-auto"
            value={from}
            onChange={(e) => filtered(setFrom)(e.target.value)}
          />
          <span className="text-ink-muted">—</span>
          <input
            type="date"
            className="input w-auto"
            value={to}
            onChange={(e) => filtered(setTo)(e.target.value)}
          />
          <input
            className="input min-w-[220px] flex-1"
            placeholder={t.sales.search}
            value={q}
            onChange={(e) => filtered(setQ)(e.target.value)}
          />
        </div>
        {/* ⚠️ Both groups are captioned. Two segmented controls side by side
            whose first option is the same word ("Hammasi") read as one broken
            control — the screen shows two things highlighted and no clue what
            either of them is about. */}
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <Segment
            caption={t.sales.stateLabel}
            value={state}
            onChange={filtered(setState)}
            options={[
              ["all", t.sales.all],
              ["open", t.sales.open],
              ["closed", t.sales.closed],
            ]}
          />
          <Segment
            caption={t.sales.placeLabel}
            value={place}
            onChange={filtered(setPlace)}
            options={[
              ["all", t.sales.all],
              ["hall", t.sales.hall],
              ["counter", t.sales.counter],
            ]}
          />
        </div>
      </div>

      {totals && (
        <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
          <Kpi label={t.sales.sales} value={money(totals.sales)} big />
          <Kpi
            label={t.sales.checks}
            value={String(totals.checks)}
            hint={`${t.sales.hall}: ${totals.hall} · ${t.sales.counter}: ${totals.counter}`}
          />
          <Kpi
            label={t.sales.avgCheck}
            value={money(totals.avgCheck)}
            // Zero guests is not "0 so'm per guest" — it is nobody counted
            // heads, and a figure there would be read as a real average.
            hint={
              totals.avgGuest > 0
                ? `${t.sales.avgGuest}: ${money(totals.avgGuest)}`
                : undefined
            }
          />
          {/* Cash on the face, card underneath: two sums of money in one line
              wrap on a 1024-wide monoblock, which is the screen this panel is
              actually read on. */}
          <Kpi
            label={t.sales.cash}
            value={money(totals.cash)}
            hint={`${t.sales.card}: ${money(totals.card)}${
              totals.open > 0 ? ` · ${t.sales.openNow}: ${totals.open}` : ""
            }`}
          />
        </div>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="card p-0">
        <ListScroll>
          {/* ⚠️ The table scrolls inside its own box rather than squeezing.
              Eight columns of times, names and sums do not fit a 1024-wide
              till monoblock, and columns that overlap are the failure this
              panel already had once. */}
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] text-sm">
              <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
                <tr>
                  <th className="px-3 py-2">{t.sales.number}</th>
                  <th className="px-3 py-2">{t.sales.place}</th>
                  <th className="px-3 py-2">{t.sales.server}</th>
                  <th className="px-3 py-2">{t.sales.openedAt}</th>
                  <th className="px-3 py-2">{t.sales.closedAt}</th>
                  <th className="px-3 py-2 text-right">{t.sales.items}</th>
                  <th className="px-3 py-2 text-right">{t.sales.total}</th>
                  <th className="px-3 py-2">{t.sales.method}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <Row key={row.id} row={row} money={money} />
                ))}
              </tbody>
            </table>
          </div>
          {!loading && rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.sales.empty}
            </p>
          )}
        </ListScroll>
        {data && data.total > PAGE && (
          <Pager
            className="border-t border-line px-3 py-2"
            page={page}
            pageCount={Math.ceil(data.total / PAGE)}
            from={page * PAGE + 1}
            to={Math.min((page + 1) * PAGE, data.total)}
            total={data.total}
            onPage={setPage}
          />
        )}
      </div>
    </div>
  );
}

function Row({ row, money }: { row: CheckRow; money: (n: number) => string }) {
  const t = useAdminT();
  return (
    <tr className="border-t border-line">
      <td className="px-3 py-2 font-medium">
        {row.number}
        {/* An unfiled sale is the one thing on this row an owner has to act on,
            so it is named rather than coloured. */}
        {row.fiscal === "error" || row.fiscal === "pending" ? (
          <span className="ml-1.5 text-xs text-danger">{t.sales.unfiled}</span>
        ) : null}
      </td>
      <td className="px-3 py-2">
        {row.table ? t.sales.table(row.table) : t.sales.counter}
        {row.guests ? (
          <span className="text-ink-muted"> · {row.guests}</span>
        ) : null}
      </td>
      <td className="px-3 py-2 text-ink-soft">{row.server || "—"}</td>
      <td className="px-3 py-2 text-ink-soft">{formatTime(row.openedAt)}</td>
      <td className="px-3 py-2 text-ink-soft">
        {row.closedAt ? (
          formatTime(row.closedAt)
        ) : (
          <span className="text-ink-muted">{t.sales.stillOpen}</span>
        )}
      </td>
      <td className="px-3 py-2 text-right tabular-nums">{row.items}</td>
      <td className="px-3 py-2 text-right font-medium tabular-nums">
        {money(row.total)}
        {row.discount ? (
          <span className="block text-xs text-ink-muted">
            −{money(row.discount)}
          </span>
        ) : null}
      </td>
      <td className="px-3 py-2 text-ink-soft">
        {row.open ? "—" : methodLabel(row.paymentMethod, t)}
      </td>
    </tr>
  );
}

function methodLabel(
  m: string | undefined,
  t: ReturnType<typeof useAdminT>,
): string {
  if (m === "cash") return t.sales.cash;
  if (m === "card") return t.sales.card;
  return m || "—";
}

function Kpi({
  label,
  value,
  hint,
  big,
}: {
  label: string;
  value: string;
  hint?: string;
  big?: boolean;
}) {
  return (
    <div className="card p-3">
      <div className="text-xs text-ink-muted">{label}</div>
      <div
        className={`mt-0.5 font-display tabular-nums ${big ? "text-2xl" : "text-lg"}`}
      >
        {value}
      </div>
      {hint && <div className="mt-0.5 text-xs text-ink-muted">{hint}</div>}
    </div>
  );
}

/** ⚠️ Both states on screen at once, like the floor screen's own switch: a
 *  control labelled with the state you are *not* in is read wrong by half the
 *  people who press it. */
function Segment<T extends string>({
  caption,
  value,
  onChange,
  options,
}: {
  caption: string;
  value: T;
  onChange: (v: T) => void;
  options: [T, string][];
}) {
  return (
    <div className="inline-flex items-center gap-2">
      <span className="text-xs text-ink-muted whitespace-nowrap">
        {caption}
      </span>
      <div className="inline-flex rounded-xl border border-line p-0.5">
        {options.map(([v, label]) => (
          <button
            key={v}
            onClick={() => onChange(v)}
            className={`rounded-lg px-3 py-1.5 text-sm whitespace-nowrap ${
              value === v ? "bg-brand text-white" : "text-ink-soft"
            }`}
          >
            {label}
          </button>
        ))}
      </div>
    </div>
  );
}

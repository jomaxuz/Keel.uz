"use client";

// The platform, split by what our customers actually are.
//
// ⚠️ **A section of the overview rather than a page of its own.** It answers
// the same question the rest of that screen answers — how is the platform
// doing — only cut a different way, and a separate tab meant a number nobody
// arrived at unless they already suspected it. The tab was removed; this is
// where it belongs.
//
// The platform, split by what our customers actually are.
//
// ⚠️ **A restaurant and a shop must not be one row.** They are sold ladders at
// a third of each other's price and they trade differently — a dining room's
// money arrives in checks, a grocery's in a few hundred scans an hour. Averaged
// together the only question worth asking of the pair has no answer: one figure
// is the mean of two populations and describes neither, which is how a platform
// decides to sell more of the wrong thing.

import { useCallback, useEffect, useState } from "react";

import { business, type BizRow, type BizTenant } from "@/lib/api";
import { BIZ_TYPES, bizLabel } from "@/lib/biz";
import { useT } from "@/lib/i18n/client";

/** The three the screen is usually read at. ⚠️ Shortest first, and the typed
 *  range last: the common press is a week or a month, and a row that opens on
 *  two date boxes makes the frequent answer the furthest to reach. */
const WINDOWS = [7, 30, 90];

export default function BusinessBreakdown() {
  const { t } = useT();
  const [rows, setRows] = useState<BizRow[]>([]);
  const [days, setDays] = useState(30);
  /** Whether the two boxes are the question being asked. ⚠️ Held apart from the
   *  dates themselves so that switching to a shorthand and back does not wipe
   *  what was typed. */
  const [custom, setCustom] = useState(false);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  /** What the server says it answered — printed rather than assumed, because a
   *  typed range and the label beside the money have to be the same window. */
  const [covered, setCovered] = useState({ from: "", to: "", days: 30 });
  const [now, setNow] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(
    (q: { days?: number; from?: string; to?: string }) => {
      setLoading(true);
      business(q)
        .then((d) => {
          setRows(d.rows);
          setNow(d.now);
          setCovered({ from: d.from, to: d.to, days: d.days });
          setError("");
        })
        .catch(() => setError(t.dash.loadFailed))
        .finally(() => setLoading(false));
    },
    [t.dash.loadFailed],
  );

  // ⚠️ Only the shorthand buttons reload on their own. A typed range waits for
  // the button: refetching on every keystroke in a date input means a request
  // for the year 0002 while somebody types 2026.
  useEffect(() => {
    if (!custom) load({ days });
  }, [custom, days, load]);

  // ⚠️ **A block for every kind we sell to, not only for the kinds that turned
  // up in the answer.** A type with no customer is itself the finding — we
  // decided to sell to bakeries and there are none — and a screen that simply
  // omits the row leaves nobody to notice. The server orders what it has by
  // what it earns; the empty ones follow, in the order the console offers them.
  const seen = new Set(rows.map((r) => r.type));
  const blocks: BizRow[] = [
    ...rows,
    ...BIZ_TYPES.filter((b) => !seen.has(b)).map((type) => emptyRow(type)),
  ];

  return (
    <div>
      <h2 className="font-display text-lg font-semibold text-ink">
        {t.biz.title}
      </h2>
      <p className="mt-1 max-w-3xl text-sm text-ink-muted">{t.biz.intro}</p>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        {WINDOWS.map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => {
              setCustom(false);
              setDays(d);
            }}
            className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
              !custom && d === days
                ? "border-signal-500 bg-signal-500/10"
                : "border-line"
            }`}
          >
            {t.biz.window(d)}
          </button>
        ))}
        <button
          type="button"
          onClick={() => setCustom(true)}
          className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
            custom ? "border-signal-500 bg-signal-500/10" : "border-line"
          }`}
        >
          {t.dash.ovRangeCustom}
        </button>
        {/* ⚠️ The window the numbers actually cover, printed beside the
            buttons: a typed range and a shorthand look identical once the page
            has redrawn, and this row is what makes a figure quotable. */}
        {covered.from && (
          <span className="text-xs text-ink-muted">
            {covered.from} — {covered.to} · {t.biz.window(covered.days)}
          </span>
        )}
      </div>

      {custom && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <label className="text-xs text-ink-muted">{t.biz.from}</label>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="rounded-lg border border-line bg-surface px-3 py-1.5 text-sm text-ink"
          />
          <label className="text-xs text-ink-muted">{t.biz.to}</label>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="rounded-lg border border-line bg-surface px-3 py-1.5 text-sm text-ink"
          />
          <button
            type="button"
            // ⚠️ Disabled until both are filled. One date is not a window, and
            // the server would answer a question nobody asked.
            disabled={!from || !to}
            onClick={() => load({ from, to })}
            className="rounded-lg bg-ink px-3 py-1.5 text-sm font-semibold text-page disabled:opacity-40"
          >
            {t.dash.ovApply}
          </button>
        </div>
      )}

      {error && <p className="mt-4 text-sm text-rose-600">{error}</p>}
      {loading && (
        <p className="mt-6 text-sm text-ink-muted">{t.dash.loading}</p>
      )}
      {!loading && rows.length === 0 && (
        <p className="mt-6 text-sm text-ink-muted">{t.biz.empty}</p>
      )}

      <div className="mt-6 space-y-4">
        {blocks.map((r) => (
          <section
            key={r.type}
            className="rounded-3xl border border-line bg-surface p-5"
          >
            <div className="flex flex-wrap items-baseline justify-between gap-3">
              <h2 className="font-display text-lg font-semibold">
                {bizLabel(t, r.type)}
              </h2>
              {/* ⚠️ **Two figures side by side, never one.** The till is a
                  monthly subscription and the website is billed per order;
                  added together they would be a number matching no invoice we
                  have ever issued. */}
              {/* ⚠️ Not drawn for a kind with nobody in it: "Bizga: 0 oyiga +
                  0" is a sum of nothing, and a zero beside money reads as a
                  figure somebody measured. */}
              {r.tenants > 0 && (
                <p className="text-sm text-ink-muted">
                  {t.biz.ours}:{" "}
                  <span className="font-semibold text-ink">
                    {money(r.subscription)}
                  </span>{" "}
                  {t.biz.perMonth} +{" "}
                  <span className="font-semibold text-ink">
                    {money(r.perOrder)}
                  </span>{" "}
                  {t.biz.perOrder(covered.days)}
                </p>
              )}
            </div>

            {/* ⚠️ **A kind we sell to and have nobody in says so in words.** Six
                zeroed statistics and two empty "best and worst" cards read as a
                loading failure; one sentence reads as the finding it is. */}
            {r.tenants === 0 ? (
              <p className="mt-3 text-sm text-ink-muted">{t.biz.none}</p>
            ) : (
              <>
                <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
                  <Stat label={t.biz.tenants} value={String(r.tenants)} />
                  {/* ⚠️ Unknown is not "down": Docker unreachable means nothing is
                  known about anybody, and a dash says that where a 0 would lie. */}
                  <Stat
                    label={t.biz.online}
                    value={r.onlineKnown ? `${r.online} / ${r.tenants}` : "—"}
                  />
                  <Stat
                    label={t.biz.idle}
                    value={String(r.idle)}
                    tone={r.idle > 0 ? "warn" : ""}
                  />
                  <Stat label={t.biz.ops} value={String(r.ops)} />
                  <Stat label={t.biz.revenue} value={money(r.revenue)} />
                  <Stat
                    label={t.biz.lastSale}
                    value={ago(r.lastSale, now, t)}
                  />
                </div>

                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                  <Best row={r.top} label={t.biz.top} t={t} />
                  <Best row={r.bottom} label={t.biz.bottom} t={t} />
                </div>

                <p className="mt-3 text-xs text-ink-muted">
                  {t.biz.split(r.orders, r.tillChecks, r.visitors)}
                </p>
              </>
            )}
          </section>
        ))}
      </div>
    </div>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: string;
}) {
  return (
    <div>
      <p className="text-xs text-ink-muted">{label}</p>
      <p
        className={`font-display text-lg tabular-nums ${
          tone === "warn" ? "text-amber-600" : "text-ink"
        }`}
      >
        {value}
      </p>
    </div>
  );
}

/** ⚠️ **Picked among customers that traded.** A shop that opened yesterday
 *  would otherwise always be "the worst", which is a fact about its age rather
 *  than its trade — and this row exists to start a phone call. */
function Best({
  row,
  label,
  t,
}: {
  row?: BizTenant;
  label: string;
  t: ReturnType<typeof useT>["t"];
}) {
  if (!row) {
    return (
      <div className="rounded-2xl border border-line px-4 py-3 text-sm text-ink-muted">
        {label}: {t.biz.noneTraded}
      </div>
    );
  }
  return (
    <div className="rounded-2xl border border-line px-4 py-3">
      <p className="text-xs text-ink-muted">{label}</p>
      <p className="font-display text-base font-semibold">{row.name}</p>
      <p className="text-sm text-ink-muted tabular-nums">
        {money(row.revenue)} · {row.ops} {t.biz.opsShort}
      </p>
    </div>
  );
}

/** Whole so'm, grouped. ⚠️ Not a locale formatter, for the reason every other
 *  money on this platform is not: two spellings of one number on two screens
 *  read as two numbers. */
function money(n: number): string {
  const s = Math.round(n).toString();
  let out = "";
  for (let i = 0; i < s.length; i++) {
    if (i > 0 && (s.length - i) % 3 === 0) out += " ";
    out += s[i];
  }
  return out;
}

/** How long ago, from the server's clock.
 *
 *  ⚠️ **The server sends "now".** A console open on a machine whose clock is a
 *  day out would shift every "how long ago" on the screen, and the number it
 *  shifted is the one somebody rings a customer about. */
function ago(
  date: string | undefined,
  now: string,
  t: ReturnType<typeof useT>["t"],
): string {
  if (!date) return t.biz.never;
  const then = new Date(`${date}T00:00:00`);
  const today = new Date(now);
  const days = Math.floor(
    (Date.UTC(today.getFullYear(), today.getMonth(), today.getDate()) -
      Date.UTC(then.getFullYear(), then.getMonth(), then.getDate())) /
      86400000,
  );
  if (days <= 0) return t.biz.today;
  return t.biz.daysAgo(days);
}

/** A kind of business we sell to and have nobody in yet.
 *
 *  ⚠️ **Built here rather than sent by the server**, because the server reports
 *  what it found and this is a fact about what we decided to sell. `onlineKnown`
 *  is false: nothing is known about nobody, and a "0 / 0 answering" would be a
 *  claim rather than a reading. */
function emptyRow(type: string): BizRow {
  return {
    type,
    tenants: 0,
    online: 0,
    onlineKnown: false,
    idle: 0,
    orders: 0,
    tillChecks: 0,
    ops: 0,
    revenue: 0,
    visitors: 0,
    subscription: 0,
    perOrder: 0,
  };
}

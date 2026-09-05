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
import { useT } from "@/lib/i18n/client";

const WINDOWS = [7, 30, 90];

export default function BusinessBreakdown() {
  const { t } = useT();
  const [rows, setRows] = useState<BizRow[]>([]);
  const [days, setDays] = useState(30);
  const [now, setNow] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    business(days)
      .then((d) => {
        setRows(d.rows);
        setNow(d.now);
        setError("");
      })
      .catch(() => setError(t.dash.loadFailed))
      .finally(() => setLoading(false));
  }, [days, t.dash.loadFailed]);

  useEffect(load, [load]);

  const label = (type: string) =>
    t.biz.types[type as keyof typeof t.biz.types] ?? type;

  return (
    <div>
      <h2 className="font-display text-lg font-semibold text-ink">{t.biz.title}</h2>
      <p className="mt-1 max-w-3xl text-sm text-ink-muted">{t.biz.intro}</p>

      <div className="mt-4 flex flex-wrap gap-2">
        {WINDOWS.map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => setDays(d)}
            className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
              d === days ? "border-signal-500 bg-signal-500/10" : "border-line"
            }`}
          >
            {t.biz.window(d)}
          </button>
        ))}
      </div>

      {error && <p className="mt-4 text-sm text-rose-600">{error}</p>}
      {loading && <p className="mt-6 text-sm text-ink-muted">{t.dash.loading}</p>}

      {!loading && rows.length === 0 && (
        <p className="mt-6 text-sm text-ink-muted">{t.biz.empty}</p>
      )}

      <div className="mt-6 space-y-4">
        {rows.map((r) => (
          <section
            key={r.type}
            className="rounded-3xl border border-line bg-surface p-5"
          >
            <div className="flex flex-wrap items-baseline justify-between gap-3">
              <h2 className="font-display text-lg font-semibold">{label(r.type)}</h2>
              {/* ⚠️ **Two figures side by side, never one.** The till is a
                  monthly subscription and the website is billed per order;
                  added together they would be a number matching no invoice we
                  have ever issued. */}
              <p className="text-sm text-ink-muted">
                {t.biz.ours}:{" "}
                <span className="font-semibold text-ink">
                  {money(r.subscription)}
                </span>{" "}
                {t.biz.perMonth} + <span className="font-semibold text-ink">
                  {money(r.perOrder)}
                </span>{" "}
                {t.biz.perOrder(days)}
              </p>
            </div>

            <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
              <Stat label={t.biz.tenants} value={String(r.tenants)} />
              {/* ⚠️ Unknown is not "down": Docker unreachable means nothing is
                  known about anybody, and a dash says that where a 0 would lie. */}
              <Stat
                label={t.biz.online}
                value={r.onlineKnown ? `${r.online} / ${r.tenants}` : "—"}
              />
              <Stat label={t.biz.idle} value={String(r.idle)} tone={r.idle > 0 ? "warn" : ""} />
              <Stat label={t.biz.ops} value={String(r.ops)} />
              <Stat label={t.biz.revenue} value={money(r.revenue)} />
              <Stat label={t.biz.lastSale} value={ago(r.lastSale, now, t)} />
            </div>

            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              <Best row={r.top} label={t.biz.top} t={t} />
              <Best row={r.bottom} label={t.biz.bottom} t={t} />
            </div>

            <p className="mt-3 text-xs text-ink-muted">
              {t.biz.split(r.orders, r.tillChecks, r.visitors)}
            </p>
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
function ago(date: string | undefined, now: string, t: ReturnType<typeof useT>["t"]): string {
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

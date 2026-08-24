"use client";

// The whole platform over a window somebody chooses.
//
// ⚠️ **A different question from the tiles above it, which is why it is its own
// section rather than more cards.** Those answer the billing month — a period
// with a start nobody picks and an invoice at the end. This answers "is this
// growing", and growth is not visible in a single number: it needs a yesterday,
// a week and a year that can be put beside each other.
//
// ⚠️ **Loaded separately from `/stats`.** The front page must render at its
// usual speed; this reads a wider window and the operator asked for it by
// pressing a button. Folding it into the same request would make the default
// page slower for everybody to serve a question most visits do not ask.

import { useCallback, useEffect, useState } from "react";

import { MultiTrendChart, TrendChart } from "@/components/Charts";
import { useT } from "@/lib/i18n/client";
import { money, overview, type Overview, type OverviewRange } from "@/lib/api";

/** The buttons, in the order they are drawn. ⚠️ Shortest first: the common
 *  press is "today" or "7 days", and a row that starts at a year makes the
 *  frequent answer the furthest to reach. */
const RANGES: { id: OverviewRange; key: keyof RangeLabels }[] = [
  { id: "1d", key: "ovRange1d" },
  { id: "7d", key: "ovRange7d" },
  { id: "30d", key: "ovRange30d" },
  { id: "90d", key: "ovRange90d" },
  { id: "1y", key: "ovRange1y" },
];

type RangeLabels = {
  ovRange1d: string;
  ovRange7d: string;
  ovRange30d: string;
  ovRange90d: string;
  ovRange1y: string;
};

export default function PlatformOverview() {
  const { t } = useT();
  const [range, setRange] = useState<OverviewRange>("30d");
  // The two boxes. Held even while a shorthand is selected, so switching to
  // "custom" and back does not wipe what was typed.
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(
    (q: { range?: OverviewRange; from?: string; to?: string }) => {
      setLoading(true);
      setError("");
      overview(q)
        .then(setData)
        .catch(() => setError(t.dash.loadFailed))
        .finally(() => setLoading(false));
    },
    [t],
  );

  // ⚠️ Only the shorthand buttons reload on click. A custom range waits for
  // the button: refetching on every keystroke in a date input means a request
  // for the year 0002 while somebody types 2026.
  useEffect(() => {
    if (range !== "custom") load({ range });
  }, [range, load]);

  const grain =
    data?.bucket === "month"
      ? t.dash.ovBucketMonth
      : data?.bucket === "week"
        ? t.dash.ovBucketWeek
        : t.dash.ovBucketDay;

  const labels = data?.series.map((p) => p.date) ?? [];
  const tot = data?.total;
  // Drawn only when the counter has sold something. A permanent row of zeroed
  // till figures is decoration within a week — the same rule the attention
  // strip and the tenant card both follow.
  const hasTill = (tot?.tillChecks ?? 0) > 0 || (tot?.tillRevenue ?? 0) > 0;

  return (
    <section className="card">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">{t.dash.ovTitle}</p>
        {data && (
          // ⚠️ **The grain is written down, not left to the bar count.** Daily
          // and monthly bars look identical and mean numbers thirty times
          // apart; the dates beside it are what makes the chart quotable.
          <span className="text-xs text-ink-muted">
            {t.dash.ovBucketNote(grain, data.from, data.to)}
          </span>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        {RANGES.map((r) => (
          <button
            key={r.id}
            onClick={() => setRange(r.id)}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
              range === r.id
                ? "bg-ink text-page"
                : "bg-ink/[0.05] text-ink-soft hover:bg-ink/[0.09]"
            }`}
          >
            {t.dash[r.key]}
          </button>
        ))}
        <button
          onClick={() => setRange("custom")}
          className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
            range === "custom"
              ? "bg-ink text-page"
              : "bg-ink/[0.05] text-ink-soft hover:bg-ink/[0.09]"
          }`}
        >
          {t.dash.ovRangeCustom}
        </button>
      </div>

      {range === "custom" && (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="rounded-lg border border-line bg-surface px-3 py-1.5 text-sm text-ink"
          />
          <span className="text-ink-muted">—</span>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="rounded-lg border border-line bg-surface px-3 py-1.5 text-sm text-ink"
          />
          <button
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

      {error && (
        <p className="mt-4 text-sm text-rose-600 dark:text-rose-400">{error}</p>
      )}
      {loading && !data && (
        <p className="mt-4 text-sm text-ink-muted">{t.dash.loading}</p>
      )}

      {tot && (
        <>
          <div className="mt-4 grid gap-px overflow-hidden rounded-2xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
            {/* ⚠️ Traffic first, and orders after it. This is the order the
                funnel actually runs in, and the pair is the only thing that
                separates "nobody wants this" from "nobody can find it" — a
                window with visits and no orders is a broken checkout, one with
                neither is a marketing problem. */}
            <Cell label={t.dash.ovVisitors} value={String(tot.visitors)} />
            <Cell label={t.dash.ovViews} value={String(tot.views)} />
            <Cell label={t.dash.ovOrders} value={String(tot.orders)} />
            <Cell label={t.dash.ovRevenue} value={money(tot.revenue)} />
            {hasTill && (
              <>
                <Cell label={t.dash.ovTillChecks} value={String(tot.tillChecks)} />
                <Cell label={t.dash.ovTillRevenue} value={money(tot.tillRevenue)} />
                <Cell label={t.dash.ovTillGuests} value={String(tot.tillGuests)} />
              </>
            )}
            {/* Only when it happened: a permanent zero is a line nobody reads
                by the second week, and this is one that has to be read the
                first time it is not zero. */}
            {tot.cancelled > 0 && (
              <Cell label={t.dash.ovCancelled} value={String(tot.cancelled)} />
            )}
            <Cell label={t.dash.ovBillable} value={money(tot.billable)} />
            {/* ⚠️ The denominator, and it belongs on this strip rather than in
                a footnote: "8 000 orders" is a different platform at three
                customers than at eighty, and the figure alone cannot say
                which. */}
            <Cell
              label={t.dash.ovActive}
              value={String(data?.activeTenants ?? 0)}
            />
          </div>

          <div className="mt-5 grid gap-5 lg:grid-cols-2">
            <div>
              <p className="text-sm font-semibold text-ink">
                {t.dash.ovChartOrders}
              </p>
              {/* ⚠️ Online orders and till checks on **one** axis: both are one
                  sale, so the comparison is honest — and it is the comparison
                  that shows the platform's shape changing as counters are
                  sold. Money is never on this frame. */}
              <MultiTrendChart
                labels={labels}
                series={[
                  {
                    label: t.dash.ovOrders,
                    data: data?.series.map((p) => p.orders) ?? [],
                  },
                  ...(hasTill
                    ? [
                        {
                          label: t.dash.ovTillChecks,
                          data: data?.series.map((p) => p.tillChecks) ?? [],
                        },
                      ]
                    : []),
                ]}
              />
            </div>

            <div>
              <p className="text-sm font-semibold text-ink">
                {t.dash.ovChartRevenue}
              </p>
              <MultiTrendChart
                labels={labels}
                money
                series={[
                  {
                    label: t.dash.ovRevenue,
                    data: data?.series.map((p) => p.revenue) ?? [],
                  },
                  ...(hasTill
                    ? [
                        {
                          label: t.dash.ovTillRevenue,
                          data: data?.series.map((p) => p.tillRevenue) ?? [],
                        },
                      ]
                    : []),
                ]}
              />
            </div>
          </div>

          <div className="mt-5">
            <p className="text-sm font-semibold text-ink">
              {t.dash.ovChartVisits}
            </p>
            {/* One series, so no legend: the title names it. Its own frame
                because traffic is neither a sale nor a so'm. */}
            <TrendChart
              labels={labels}
              data={data?.series.map((p) => p.visitors) ?? []}
              label={t.dash.ovVisitors}
              height={200}
            />
          </div>
        </>
      )}
    </section>
  );
}

function Cell({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-surface px-4 py-3">
      <p className="text-xs text-ink-muted">{label}</p>
      <p className="mt-0.5 text-lg font-bold tabular-nums text-ink">{value}</p>
    </div>
  );
}

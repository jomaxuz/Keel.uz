"use client";

// Menu analysis: which dishes earn the money, and which of them can be planned.
//
// The two letters answer different questions and are shown crossed rather than
// separately, because neither decides anything alone:
//
//   • **ABC** — share of takings. A dish can be an A on four sales a month if
//     it is expensive, and a C on four hundred if it is a cup of tea.
//   • **XYZ** — steadiness of daily demand. Says nothing about money; says
//     everything about whether the kitchen can plan for it.
//
// So the useful reading is the pair: AX is what must never run out, AZ is what
// earns well but arrives in waves, CZ is what could leave the menu tomorrow.
//
// ⚠️ Counted on dishes **sold**, not money **collected** — a dish in a
// confirmed order has sold even if the courier is still out with the cash. The
// dashboard's revenue figures use the other basis on purpose, and mixing the
// two is the mistake this panel already made once.

import { useCallback, useEffect, useState } from "react";
import { api, downloadReport } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { usePanelWords } from "@/lib/panelWords";
import { ListScroll } from "@/components/admin/PagedList";
import type { AbcXyzResponse, AbcXyzRow } from "@/lib/types";

type Range = { from?: string; to?: string };

const ABC_TONE: Record<string, string> = {
  A: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  B: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  C: "bg-ink/10 text-ink-soft",
};

const XYZ_TONE: Record<string, string> = {
  X: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  Y: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  Z: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
};

export default function MenuAnalysis({
  range: period,
  shortPeriod,
}: {
  range: Range;
  /** True when the chosen window is too short for XYZ to mean anything. */
  shortPeriod: boolean;
}) {
  const t = useAdminT();
  const w = usePanelWords();
  const [data, setData] = useState<AbcXyzResponse | null>(null);
  const [abcFilter, setAbcFilter] = useState("");
  const [xyzFilter, setXyzFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const range = useCallback(() => ({ ...period }), [period]);

  useEffect(() => {
    setLoading(true);
    api
      .abcXyz(range())
      .then(setData)
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [range, t]);

  async function download() {
    setBusy(true);
    setError("");
    try {
      await downloadReport("/admin/reports/abc-xyz", range());
    } catch {
      setError(t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  const rows = (data?.items ?? []).filter(
    (r) =>
      (!abcFilter || r.abc === abcFilter) &&
      (!xyzFilter || r.xyz === xyzFilter),
  );
  // The filtered slice's own share, so selecting "A" answers "how much of the
  // takings is this group" rather than leaving the reader to add a column up.
  const shown = rows.reduce((sum, r) => sum + r.revenue, 0);
  // ⚠️ Decided over **every** dish in the period, not the filtered rows: the
  // columns must not appear and disappear as somebody clicks through the
  // classes, which reads as the table breaking.
  const costed = (data?.items ?? []).some((r) => r.costed);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-ink">{t.reports.title}</h2>
        <button
          type="button"
          onClick={download}
          disabled={busy || !data?.items.length}
          className="btn btn-primary disabled:opacity-60"
        >
          {busy ? t.common.loading : t.reports.excel}
        </button>
      </div>

      <p className="text-sm text-ink-soft">{t.reports.intro}</p>

      {/* ⚠️ The warning stays with the analysis, not with the period control:
          seven days is short **for XYZ** and perfectly reasonable for the sales
          and channel tabs, which share the same picker. A caveat shown next to
          the buttons would be wrong on three screens out of four. */}
      {shortPeriod && (
        <p className="text-xs text-amber-700 dark:text-amber-300">
          {t.reports.shortPeriodWarning}
        </p>
      )}

      {/* The two axes, filtered independently: "show me the A dishes" and
          "show me everything erratic" are both real questions, and so is their
          intersection. */}
      <div className="flex flex-wrap items-center gap-4">
        <Filter
          label="ABC"
          options={["A", "B", "C"]}
          value={abcFilter}
          onChange={setAbcFilter}
          tone={ABC_TONE}
          allLabel={t.reports.all}
        />
        <Filter
          label="XYZ"
          options={["X", "Y", "Z"]}
          value={xyzFilter}
          onChange={setXyzFilter}
          tone={XYZ_TONE}
          allLabel={t.reports.all}
        />
        {(abcFilter || xyzFilter) && (
          <button
            type="button"
            onClick={() => {
              setAbcFilter("");
              setXyzFilter("");
            }}
            className="text-xs text-ink-muted hover:text-ink"
          >
            {t.reports.clear}
          </button>
        )}
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : !data?.items.length ? (
        <p className="py-10 text-center text-ink-muted/70">{t.reports.empty}</p>
      ) : (
        <>
          <p className="text-sm text-ink-soft">
            {t.reports.summary(rows.length, formatPrice(shown), data.note)}
          </p>

          <ListScroll className="max-h-[60vh]">
            <table className="w-full min-w-[820px] text-sm">
              <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
                <tr>
                  <th className="px-3 py-2">{w.reportItem}</th>
                  <th className="px-3 py-2 text-right">{t.reports.sold}</th>
                  <th className="px-3 py-2 text-right">{t.reports.revenue}</th>
                  {/* ⚠️ The two columns exist only when the menu carries any
                      cost at all. On a restaurant that never fills it in they
                      would be two empty money columns on every report — and an
                      empty money column is read as zero. */}
                  {costed && (
                    <>
                      <th className="px-3 py-2 text-right">{t.reports.cost}</th>
                      <th className="px-3 py-2 text-right">
                        {t.reports.margin}
                      </th>
                    </>
                  )}
                  <th className="px-3 py-2 text-right">{t.reports.share}</th>
                  {/* The column that turns "this dish is 3%" into "these nine
                      dishes are 80%" — the whole point of a Pareto cut. */}
                  <th className="px-3 py-2 text-right">
                    {t.reports.cumulative}
                  </th>
                  <th className="px-3 py-2 text-right">
                    {t.reports.variation}
                  </th>
                  <th className="px-3 py-2">{t.reports.klass}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {rows.map((r) => (
                  <Row key={r.name} r={r} t={t} costed={costed} />
                ))}
              </tbody>
            </table>
          </ListScroll>
        </>
      )}
    </div>
  );
}

function Row({
  r,
  t,
  costed,
}: {
  r: AbcXyzRow;
  t: ReturnType<typeof useAdminT>;
  costed: boolean;
}) {
  return (
    <tr className="hover:bg-raised/60">
      <td className="px-3 py-2 font-medium text-ink">{r.name}</td>
      <td className="px-3 py-2 text-right tabular-nums">{r.qty}</td>
      <td className="px-3 py-2 text-right tabular-nums">
        {formatPrice(r.revenue)}
      </td>
      {costed && (
        <>
          {/* A dash, not a zero: this dish has no cost typed in, and the
              difference is the whole point. */}
          <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
            {r.costed ? formatPrice(r.cost ?? 0) : "—"}
          </td>
          <td className="px-3 py-2 text-right font-medium tabular-nums">
            {r.costed ? formatPrice(r.margin ?? 0) : "—"}
          </td>
        </>
      )}
      <td className="px-3 py-2 text-right tabular-nums">
        {r.share.toFixed(1)}%
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-muted">
        {r.cumulative.toFixed(1)}%
      </td>
      <td className="px-3 py-2 text-right tabular-nums">
        {/* ⚠️ The variation is shown next to the number of days it rests on.
            A dish sold on two days out of thirty has a coefficient that is
            arithmetically true and worth nothing, and without the second
            number nobody can tell those apart. */}
        {r.variation.toFixed(0)}%
        <span className="ml-1 text-xs text-ink-muted">
          {t.reports.onDays(r.days)}
        </span>
      </td>
      <td className="px-3 py-2">
        <span
          className={`rounded-lg px-2 py-1 text-xs font-bold ${ABC_TONE[r.abc]}`}
        >
          {r.abc}
        </span>
        <span
          className={`ml-1 rounded-lg px-2 py-1 text-xs font-bold ${XYZ_TONE[r.xyz]}`}
        >
          {r.xyz}
        </span>
      </td>
    </tr>
  );
}

function Filter({
  label,
  options,
  value,
  onChange,
  tone,
  allLabel,
}: {
  label: string;
  options: string[];
  value: string;
  onChange: (v: string) => void;
  tone: Record<string, string>;
  allLabel: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs font-semibold text-ink-muted">{label}</span>
      <button
        type="button"
        onClick={() => onChange("")}
        className={
          value === ""
            ? "rounded-lg bg-ink px-2 py-1 text-xs font-bold text-surface"
            : "rounded-lg border border-line px-2 py-1 text-xs font-bold text-ink-soft"
        }
      >
        {allLabel}
      </button>
      {options.map((o) => (
        <button
          key={o}
          type="button"
          // Pressing the active one clears it: a filter that can only be
          // changed and never removed traps the reader in a subset.
          onClick={() => onChange(value === o ? "" : o)}
          className={`rounded-lg px-2 py-1 text-xs font-bold ${
            value === o ? "ring-2 ring-ink " : ""
          }${tone[o]}`}
        >
          {o}
        </button>
      ))}
    </div>
  );
}

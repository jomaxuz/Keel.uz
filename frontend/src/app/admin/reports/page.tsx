"use client";

// The analysis section: four questions that share one period.
//
// Tabs rather than four entries in the sidebar, and the reason is the period
// control above them. Every one of these screens is read by comparing it with
// something — last month, the other channel, the other courier — and a period
// chosen on one screen and silently reset on the next is why nobody compares
// anything. One picker, four answers.
//
// The four:
//
//   • **Menyu** — which dishes earn the money and which can be planned for.
//   • **Savdo** — how the period went, and whether that is up or down.
//   • **Kanallar** — which door the orders came in through.
//   • **Jamoa** — who did the work.
//   • **Kassa** — what the drawer was counted at, and where it did not match.

import { useMemo, useState } from "react";
import { useAdminT } from "@/lib/i18n/admin";
import MenuAnalysis from "@/components/admin/reports/MenuAnalysis";
import SalesReport from "@/components/admin/reports/SalesReport";
import ChannelReport from "@/components/admin/reports/ChannelReport";
import TeamReport from "@/components/admin/reports/TeamReport";
import CashReport from "@/components/admin/reports/CashReport";
import FinanceReport from "@/components/admin/reports/FinanceReport";
import StockReport from "@/components/admin/reports/StockReport";

type Tab =
  "menu" | "sales" | "channels" | "team" | "cash" | "finance" | "stock";
type Preset = "week" | "month" | "quarter" | "all";

/** The period presets, in days. `all` sends no bounds at all. */
const PRESET_DAYS: Record<Exclude<Preset, "all">, number> = {
  week: 7,
  month: 30,
  quarter: 90,
};

function isoDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
}

function daysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return isoDate(d);
}

export default function ReportsPage() {
  const t = useAdminT();
  const [tab, setTab] = useState<Tab>("sales");
  const [preset, setPreset] = useState<Preset>("month");

  // ⚠️ Both bounds are sent, not just `from`. The sales report compares with
  // the period of the same length immediately before this one, and it can only
  // work out "the same length" when it knows where this one ends. An open
  // upper bound would silently drop the comparison — the arrows would simply
  // never appear, which reads as "nothing changed".
  const range = useMemo(
    () =>
      preset === "all"
        ? {}
        : { from: daysAgo(PRESET_DAYS[preset]), to: isoDate(new Date()) },
    [preset],
  );

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold">{t.nav.reports}</h1>

      {/* Period first, and above the tabs: it applies to all of them, and a
          control that changes every screen must not look like it belongs to
          the one currently open. */}
      <div className="flex flex-wrap items-center gap-2">
        {(["week", "month", "quarter", "all"] as Preset[]).map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => setPreset(p)}
            className={
              preset === p
                ? "rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface"
                : "rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-ink/5"
            }
          >
            {t.reports.periods[p]}
          </button>
        ))}
      </div>

      <div className="flex flex-wrap gap-1 border-b border-line">
        {(
          ["sales", "menu", "channels", "team", "cash", "finance"] as Tab[]
        ).map((x) => (
          <button
            key={x}
            type="button"
            onClick={() => setTab(x)}
            className={`-mb-px border-b-2 px-3 py-2 text-sm font-semibold transition ${
              tab === x
                ? "border-brand text-ink"
                : "border-transparent text-ink-muted hover:text-ink-soft"
            }`}
          >
            {t.reports.tabs[x]}
          </button>
        ))}
      </div>

      {/* Keyed on the period so a tab remounts when it changes: each of these
          fetches on mount, and a stale table under a fresh period label is the
          one failure the reader cannot see. */}
      {tab === "sales" && <SalesReport key={`s-${preset}`} range={range} />}
      {tab === "menu" && (
        <MenuAnalysis
          key={`m-${preset}`}
          range={range}
          shortPeriod={preset === "week"}
        />
      )}
      {tab === "channels" && (
        <ChannelReport key={`c-${preset}`} range={range} />
      )}
      {tab === "team" && <TeamReport key={`t-${preset}`} range={range} />}
      {/* Last, because it is the only tab read backwards — the others answer
          "how did we do", this one answers "did anything go missing", and that
          question is asked after the others rather than instead of them. */}
      {tab === "cash" && <CashReport key={`k-${preset}`} range={range} />}
      {/* ⚠️ Last, and it existed for months with no screen at all — reachable
          only as a spreadsheet, which meant it was read without the sentence
          that says it is not a profit report. */}
      {tab === "finance" && <FinanceReport key={`f-${preset}`} range={range} />}
      {/* ⚠️ Last, and a flow rather than a balance: there is no opening count
          in this system, and a "remaining" column would be believed. */}
      {tab === "stock" && <StockReport key={`st-${preset}`} range={range} />}
    </div>
  );
}

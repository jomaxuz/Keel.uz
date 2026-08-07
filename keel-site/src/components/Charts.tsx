"use client";

// The console's charts.
//
// ⚠️ **Copied from the tenant panel rather than imported.** keel.uz and the
// restaurant app are two separate builds with two separate palettes, and a
// shared package for a few chart wrappers is a dependency somebody has to
// reason about forever — the same call already made for the logo mark. When
// the shapes change they change in both.
//
// Three rules decided here rather than at each call site, because getting any
// of them wrong is invisible until somebody misreads a number:
//
//  1. **The series palette is fixed and validated**, not taken from the
//     surrounding theme. These five hues are checked for colour-vision
//     deficiency against both the light and the dark surface.
//  2. **One axis, always.** Orders and money are never drawn on the same chart
//     with two scales: the shape of such a chart is decided by where the two
//     axes happen to be zeroed, which is to say by nothing.
//  3. **Values are labelled, not just coloured.** The light palette sits below
//     3:1 against the page, which is fine for a fill and not fine as the only
//     way to read a quantity.

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  PointElement,
  LineElement,
  Filler,
  Tooltip,
  Legend,
  type ChartOptions,
} from "chart.js";
import { Bar, Line } from "react-chartjs-2";
import { useEffect, useState } from "react";
import { money as formatPrice, moneyShort } from "@/lib/api";

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  PointElement,
  LineElement,
  Filler,
  Tooltip,
  Legend,
);

/** Categorical slots, in fixed order. Never cycled: a sixth category folds into
 *  "boshqa" rather than reusing slot 1, which would make two different things
 *  the same colour on one chart. */
const SERIES_LIGHT = ["#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4"];
const SERIES_DARK = ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181"];

/** Reads the panel's current theme. The chart is canvas, so it cannot inherit
 *  CSS variables — the colours have to be resolved in JS and redrawn when the
 *  theme toggle moves. */
function useDark(): boolean {
  const [dark, setDark] = useState(false);
  useEffect(() => {
    const el = document.documentElement;
    const read = () => setDark(el.classList.contains("dark"));
    read();
    const observer = new MutationObserver(read);
    observer.observe(el, { attributes: true, attributeFilter: ["class"] });
    return () => observer.disconnect();
  }, []);
  return dark;
}

function palette(dark: boolean) {
  return {
    series: dark ? SERIES_DARK : SERIES_LIGHT,
    // Grid and axes stay recessive: they are scaffolding, not data.
    grid: dark ? "rgba(255,255,255,0.08)" : "rgba(0,0,0,0.07)",
    text: dark ? "#a9a9a4" : "#6b6b66",
    ink: dark ? "#ececea" : "#1a1a19",
    surface: dark ? "#1a1a19" : "#ffffff",
  };
}

function baseOptions(dark: boolean, money: boolean): ChartOptions<"bar" | "line"> {
  const p = palette(dark);
  return {
    responsive: true,
    maintainAspectRatio: false,
    // The hover layer is the default, not an extra: a chart on a screen is
    // interactive, and the exact value is the first thing anyone wants.
    interaction: { mode: "index", intersect: false },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: p.surface,
        titleColor: p.ink,
        bodyColor: p.ink,
        borderColor: p.grid,
        borderWidth: 1,
        padding: 10,
        displayColors: true,
        callbacks: {
          label: (ctx) => {
            const v = Number(ctx.parsed.y ?? ctx.parsed.x ?? 0);
            return ` ${ctx.dataset.label ?? ""} ${money ? formatPrice(v) : v}`.trim();
          },
        },
      },
    },
    scales: {
      x: {
        grid: { display: false },
        ticks: { color: p.text, maxRotation: 0, autoSkipPadding: 16 },
        border: { color: p.grid },
      },
      y: {
        beginAtZero: true,
        grid: { color: p.grid },
        border: { display: false },
        ticks: {
          color: p.text,
          // ⚠️ **Whole numbers only when the data is a count.** Chart.js picks
          // tick steps from the range, so a chart topping out at four orders
          // is labelled 0.5, 1.5, 2.5 — and half an order does not exist.
          // Caught by looking at the rendered chart, which no palette check
          // would ever have found.
          precision: money ? undefined : 0,
          // Money on an axis is shortened; the exact figure lives in the
          // tooltip. A y-axis of nine-digit so'm is unreadable at any size.
          callback: (v) => (money ? shortMoney(Number(v)) : v),
        },
      },
    },
  };
}

/** Axis labels use the console's own short money format, so the chart and the
 *  table beside it shorten the same number the same way. */
const shortMoney = moneyShort;

/** Trend over time, one series. No legend: the title names it. */
export function TrendChart({
  labels,
  data,
  label,
  money = false,
  height = 220,
}: {
  labels: string[];
  data: number[];
  label: string;
  money?: boolean;
  height?: number;
}) {
  const dark = useDark();
  const p = palette(dark);
  const accent = p.series[0];
  return (
    <div style={{ height }}>
      <Line
        options={baseOptions(dark, money) as ChartOptions<"line">}
        data={{
          labels,
          datasets: [
            {
              label,
              data,
              borderColor: accent,
              // A wash rather than a solid fill: the line is the data, the
              // area only says "this is one series over time".
              backgroundColor: dark ? "rgba(57,135,229,0.16)" : "rgba(42,120,214,0.12)",
              borderWidth: 2,
              fill: true,
              tension: 0.3,
              pointRadius: 0,
              // Appears on hover only — a dot on all thirty days is noise.
              pointHoverRadius: 4,
              pointHoverBackgroundColor: accent,
              pointHoverBorderColor: p.surface,
              pointHoverBorderWidth: 2,
            },
          ],
        }}
      />
    </div>
  );
}

/** Magnitude across a handful of named things. Horizontal, because the names
 *  are words ("yetkazib berish") and a vertical axis would either clip them or
 *  turn them on their side. */
export function BreakdownChart({
  labels,
  data,
  money = true,
  height,
}: {
  labels: string[];
  data: number[];
  money?: boolean;
  height?: number;
}) {
  const dark = useDark();
  const p = palette(dark);
  const opts = baseOptions(dark, money) as ChartOptions<"bar">;
  return (
    <div style={{ height: height ?? Math.max(120, labels.length * 42 + 24) }}>
      <Bar
        options={{
          ...opts,
          indexAxis: "y",
          scales: {
            x: {
              beginAtZero: true,
              grid: { color: p.grid },
              border: { display: false },
              ticks: {
                color: p.text,
                precision: money ? undefined : 0,
                callback: (v) => (money ? shortMoney(Number(v)) : v),
              },
            },
            y: { grid: { display: false }, border: { display: false }, ticks: { color: p.ink } },
          },
        }}
        data={{
          labels,
          datasets: [
            {
              label: "",
              data,
              // Colour follows the entity's position in a fixed list, never
              // its rank: re-sorting must not repaint the bars.
              backgroundColor: labels.map((_, i) => p.series[i % p.series.length]),
              borderRadius: 4,
              // A surface-coloured gap so two adjacent fills never touch.
              borderWidth: 2,
              borderColor: p.surface,
              barPercentage: 0.7,
            },
          ],
        }}
      />
    </div>
  );
}

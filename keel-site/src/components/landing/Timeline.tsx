import type { Dict } from "@/lib/i18n/dict";

/** Three steps to being open.
 *
 *  ⚠️ **The objection this answers is time, not price.** The page already
 *  argues about money in four places; what actually stops an owner who is
 *  convinced by the price is the fortnight they imagine losing to a migration
 *  — typing the menu in, retraining the staff, running two systems at once. So
 *  every step carries a duration, and the durations are the point of the
 *  block: without them these are three sentences about a process.
 *
 *  The connecting line is drawn only from `sm:` up. Stacked on a phone it ran
 *  from the middle of one card to the middle of the next, through the text. */
export default function Timeline({ t }: { t: Dict }) {
  return (
    <ol className="grid gap-6 sm:grid-cols-3">
      {t.timeline.steps.map((s, i) => (
        <li key={s.name} className="relative flex flex-col">
          {/* ⚠️ One segment per step rather than one line across the row: the
              badges sit at the left edge of their column, not its centre, so a
              line drawn between two percentages of the row misses them at one
              end and runs past the last one at the other. A segment that
              starts at its own badge and crosses the gap to the next is true
              whatever the columns are doing. */}
          {i < t.timeline.steps.length - 1 && (
            <span
              aria-hidden
              className="pointer-events-none absolute -right-6 left-14 top-[1.375rem] hidden border-t-2 border-dashed border-line-strong sm:block"
            />
          )}
          <span className="icon-badge font-display text-sm font-semibold ring-4 ring-page">
            {String(i + 1).padStart(2, "0")}
          </span>
          <p className="mt-4 font-display text-lg font-semibold text-ink">{s.name}</p>
          <p className="mt-2 text-sm leading-relaxed text-ink-muted">{s.desc}</p>
          {/* ⚠️ Pushed to the bottom of its own column. The three descriptions
              are different lengths, so in the flow the three durations land at
              three different heights — and three numbers that are meant to be
              read against each other stop being a row. */}
          <p className="mt-auto pt-3">
            <span className="inline-flex items-center gap-1.5 rounded-lg bg-emerald-500/10 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-400">
              {s.meta}
            </span>
          </p>
        </li>
      ))}
    </ol>
  );
}

"use client";

// Which course the next dish belongs to.
//
// ⚠️ **A course is a plan, not a state.** It says when the waiter means to send
// this; whether it has gone is the line's own `fired`. Two facts, two places —
// storing "course two is away" on the check would be a second thing to be wrong
// about something the lines already know.
//
// ⚠️ **Words and numbers, not glyphs.** This was a row of "— I II III" with no
// label, beside the photo switch, and it was reported as "buttons next to the
// picture toggle that do nothing": pressing one changed nothing anybody could
// see, because what it changes is the *next* dish. So it now says what it is
// ("Kurs"), counts in digits, and, while a course is chosen, says in a line
// under the bar what will happen to the next dish.

import { useAdminT } from "@/lib/i18n/admin";

/** How many courses a screen offers. Three is what a menu has — starters,
 *  mains, dessert — and a fourth is a banquet, which is planned on paper. */
export const COURSES = [1, 2, 3];

export default function CourseTabs({
  value,
  onPick,
}: {
  /** 0 = send it with everything else. */
  value: number;
  onPick: (course: number) => void;
}) {
  const t = useAdminT();
  return (
    <div className="flex shrink-0 items-center gap-1.5">
      <span className="hidden text-xs font-semibold uppercase tracking-wide text-ink-muted sm:inline">
        {t.till.course}
      </span>
      <div className="till-seg-track" role="group" aria-label={t.till.course}>
        <button
          className={value === 0 ? "till-seg-on" : "till-seg"}
          onClick={() => onPick(0)}
          title={t.till.noCourse}
          aria-pressed={value === 0}
        >
          {t.till.courseAll}
        </button>
        {COURSES.map((c) => (
          <button
            key={c}
            className={`${value === c ? "till-seg-on" : "till-seg"} min-w-10 px-3`}
            onClick={() => onPick(value === c ? 0 : c)}
            aria-label={t.till.courseOf(c)}
            title={t.till.courseHint(c)}
            aria-pressed={value === c}
          >
            {c}
          </button>
        ))}
      </div>
    </div>
  );
}

/** The sentence under the bar while a course is chosen. */
export function CourseNote({ value }: { value: number }) {
  const t = useAdminT();
  if (value <= 0) return null;
  return (
    <p className="shrink-0 border-b border-line bg-[rgb(var(--till-accent-tint))] px-3 py-1.5 text-[13px] font-medium text-[rgb(var(--till-accent-ink))]">
      {t.till.courseHint(value)}
    </p>
  );
}

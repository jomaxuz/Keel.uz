"use client";

// Which course the next dish belongs to.
//
// ⚠️ **A course is a plan, not a state.** It says when the waiter means to send
// this; whether it has gone is the line's own `fired`. Two facts, two places —
// storing "course two is away" on the check would be a second thing to be wrong
// about something the lines already know.
//
// ⚠️ **Off by default and invisible until used.** A counter selling coffee
// never numbers a course, and a control that is on every screen for a feature
// most branches never touch is a control that gets pressed by accident.

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
    <div className="till-seg-track shrink-0" aria-label={t.till.course}>
      <button
        className={value === 0 ? "till-seg-on" : "till-seg"}
        onClick={() => onPick(0)}
        title={t.till.noCourse}
      >
        —
      </button>
      {COURSES.map((c) => (
        <button
          key={c}
          className={`${value === c ? "till-seg-on" : "till-seg"} px-3`}
          onClick={() => onPick(c)}
          aria-label={t.till.courseOf(c)}
        >
          {"I".repeat(c)}
        </button>
      ))}
    </div>
  );
}

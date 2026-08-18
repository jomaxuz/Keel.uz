"use client";

// The bar across the top of both till screens.
//
// ⚠️ **One component, two screens, and that is the point.** A waiter moves
// between the monoblock and the floor tablet during a shift; a header that
// drifts between them costs a look every time. It also carries the four facts
// somebody standing at a till needs and cannot get anywhere else: which machine
// this is, whether a shift is open, whose name every action is being recorded
// under, and the time — the monoblock runs fullscreen with no OS chrome, so the
// clock in the corner of the room is the only other place to find it.

import { useEffect, useState } from "react";
import { LuLockOpen, LuLogOut } from "react-icons/lu";

import LangSwitch from "@/components/site/LangSwitch";
import { formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";

export default function TillChrome({
  title,
  personName,
  roleLabel,
  branchName,
  shiftOpenedAt,
  device,
  onLock,
  children,
}: {
  /** "Keel POS" on the counter, "Keel · Zal" in the room. */
  title: string;
  personName: string;
  /** Cashier or waiter — the permission this session is being used under. */
  roleLabel: string;
  branchName?: string;
  /** When the drawer was opened, if it is open. */
  shiftOpenedAt?: string;
  /** A bound monoblock locks; a staff login signs out. */
  device: boolean;
  onLock: () => void;
  /** Screen-specific controls (the floor's zone filter and its tally). */
  children?: React.ReactNode;
}) {
  const t = useAdminT();

  return (
    <header className="till-chrome flex h-14 shrink-0 items-center gap-3 px-4">
      {/* Whose machine this is. ⚠️ Our mark, in our amber — `brand` is the
          owner's accent and would make the logo a different logo per install. */}
      <div className="flex shrink-0 items-center gap-2.5">
        <span className="flex h-8 w-8 items-center justify-center rounded-[9px] bg-ink text-[15px] font-bold text-keel">
          K
        </span>
        <span className="text-[17px] font-bold tracking-tight">{title}</span>
      </div>

      <Divider />

      {/* ⚠️ The open shift, where it cannot be missed. A cashier who cannot see
          which shift they are selling into finds out at the count — and by then
          the answer is a discrepancy rather than a fact. */}
      {shiftOpenedAt && (
        <span className="hidden shrink-0 items-center gap-2 text-[13px] text-[rgb(var(--till-mid))] md:flex">
          <span
            className="h-2 w-2 rounded-full"
            style={{ background: "rgb(var(--till-ok))" }}
          />
          {t.till.shiftOpen} · {formatTime(shiftOpenedAt)}
        </span>
      )}
      {branchName && (
        <span className="hidden truncate text-[13px] text-ink-muted lg:block">
          {branchName}
        </span>
      )}

      {children}

      <div className="ml-auto flex shrink-0 items-center gap-2.5">
        {/* Who every void, discount and closed check will be recorded against —
            whoever is unlocked, never whoever set the monoblock up. */}
        <span className="flex h-9 w-9 items-center justify-center rounded-full bg-[rgb(var(--till-accent-tint))] text-xs font-bold text-[rgb(var(--till-accent-ink))]">
          {initials(personName)}
        </span>
        <span className="hidden flex-col leading-tight sm:flex">
          <span className="text-[13px] font-semibold">{personName}</span>
          <span className="text-[11px] text-[rgb(var(--till-dim))]">
            {roleLabel}
          </span>
        </span>
        <Clock />
        <Divider />
        {/* ⚠️ No theme toggle: these screens are always light (forcedLight in
            lib/theme.tsx). A control that does nothing is worse than an absent
            one — the cashier presses it, nothing happens, and the next button
            that genuinely fails gets pressed twice too. */}
        <LangSwitch />
        {/* ⚠️ On a bound monoblock this locks rather than logs out — there is no
            account to sign out of, and clearing the device token would mean
            fetching a new link from the panel to sell anything.

            ⚠️ **An open padlock, not the word.** The header is chrome on a
            768px-tall screen and every row it takes is a row the dish grid does
            not get; a padlock is also read faster than a word by somebody
            talking to a guest while reaching for it. Open because that is the
            state it describes — pressing it closes the padlock the lock screen
            then shows. The word stays as the accessible name and the tooltip. */}
        <button
          className="till-btn flex w-11 items-center justify-center px-0"
          aria-label={device ? t.till.lock : t.till.logout}
          title={device ? t.till.lock : t.till.logout}
          onClick={onLock}
        >
          {device ? (
            <LuLockOpen className="h-[1.15rem] w-[1.15rem]" aria-hidden />
          ) : (
            // A different action, so a different icon: signing out of an account
            // is not locking a shared machine.
            <LuLogOut className="h-[1.15rem] w-[1.15rem]" aria-hidden />
          )}
        </button>
      </div>
    </header>
  );
}

function Divider() {
  return (
    <span
      className="hidden h-7 w-px shrink-0 md:block"
      style={{ background: "var(--line)" }}
    />
  );
}

/** The time, because the monoblock runs fullscreen and there is no clock on it.
 *
 *  ⚠️ Rendered only after mount. The server has no idea what time it is on the
 *  counter, and a server-rendered clock is a hydration mismatch that React
 *  reports as an error on the busiest screen in the building. */
function Clock() {
  const [now, setNow] = useState<Date | null>(null);
  useEffect(() => {
    setNow(new Date());
    // Once a second would repaint the header sixty times a minute for a display
    // that only changes once; the offset keeps it roughly on the minute.
    const timer = setInterval(() => setNow(new Date()), 15_000);
    return () => clearInterval(timer);
  }, []);
  if (!now) return null;
  const hh = String(now.getHours()).padStart(2, "0");
  const mm = String(now.getMinutes()).padStart(2, "0");
  // ⚠️ Built by hand rather than with toLocaleTimeString, which follows the
  // device language and prints "2:38 PM" on an English tablet — see lib/format.
  return (
    <span className="till-num hidden text-[15px] text-ink-soft sm:block">
      {hh}:{mm}
    </span>
  );
}

/** "Dilnoza Abdullayeva" → "DA". One word gives one letter. */
function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "—";
  return parts
    .slice(0, 2)
    .map((p) => p[0]!.toUpperCase())
    .join("");
}

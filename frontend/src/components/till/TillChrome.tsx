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
  /** Whether this screen can lock at all — a bound monoblock, or a branch
   *  that has given somebody a PIN.
   *
   *  ⚠️ **Not "is this a device".** A till signed in with a staff login and
   *  PINs set locks exactly like a monoblock: the pad comes back and the next
   *  person names themselves. Keying the button off the device alone put a
   *  sign-out where a padlock belonged — and signing out of the shared account
   *  mid-service is a different, worse thing than locking the screen. */
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
      {/* ⚠️ **Our mark, not a letter in a box** — and in `keel-deep` rather
          than the brand amber: on the till's near-white chrome #F5A524 is a
          pale smear, and this screen is the one the machine sits on all day.
          `brand` is deliberately not used: that is the restaurant's accent,
          chosen per install, and our logo drawn in it would be a different
          logo in every kitchen.

          Inline rather than a file: this header is the first thing on screen
          when a monoblock wakes up, and a logo that arrives on a second
          request is a logo that flickers. */}
      <div className="flex shrink-0 items-center gap-2">
        <svg
          viewBox="0 0 32 32"
          className="h-9 w-9 text-keel-deep"
          fill="none"
          stroke="currentColor"
          strokeWidth={2.6}
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden
        >
          <path d="M5 6c0 9.5 4.4 14.5 11 14.5S27 15.5 27 6" />
          <path d="M16 20.5V29" />
        </svg>
        <span className="text-[19px] font-bold tracking-tight">keel</span>
        {/* The screen's own name stays as the accessible title, not as chrome:
            the cashier knows which machine they are standing at. */}
        <span className="sr-only">{title}</span>
      </div>

      <Divider />

      {/* ⚠️ The open shift, where it cannot be missed. A cashier who cannot see
          which shift they are selling into finds out at the count — and by then
          the answer is a discrepancy rather than a fact. */}
      {/* ⚠️ **The dot and the hour survive every width.** A cashier who cannot
          see which shift they are selling into finds out at the count, and by
          then the answer is a discrepancy rather than a fact — so on a 1024px
          monoblock the sentence is dropped and the fact is not. */}
      {shiftOpenedAt && (
        <span
          className="flex shrink-0 items-center gap-2 whitespace-nowrap text-[13px] text-[rgb(var(--till-mid))]"
          title={`${t.till.shiftOpen} · ${formatTime(shiftOpenedAt)}`}
        >
          <span
            className="h-2 w-2 rounded-full"
            style={{ background: "rgb(var(--till-ok))" }}
          />
          <span className="hidden xl:inline">{t.till.shiftOpen} · </span>
          <span className="till-num">{formatTime(shiftOpenedAt)}</span>
        </span>
      )}
      {branchName && (
        <span className="hidden max-w-[10rem] truncate text-[13px] text-ink-muted 2xl:block">
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
        {/* ⚠️ Bounded, and both lines truncate. A staff name is free text and a
            long one used to push the clock and the lock button off the end of a
            1024px header — the two controls a cashier reaches for most. */}
        <span className="hidden min-w-0 max-w-[9rem] flex-col leading-tight sm:flex">
          <span className="truncate text-[13px] font-semibold">
            {personName}
          </span>
          <span className="truncate text-[11px] text-[rgb(var(--till-dim))]">
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

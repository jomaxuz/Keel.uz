"use client";

// The rail down the left of the till.
//
// ⚠️ **Three destinations, and every one of them is a screen that exists.** The
// design draws five (orders, tables, receipts, couriers, reports) plus a greyed
// "settings"; four of those are not built, and a rail of buttons that answer
// "not yet" is how a room learns to stop pressing things. What is here is what
// the till actually does: the room, the menu, and the drawer.
//
// ⚠️ **Icon over label, both always visible.** Icon-only rails are learnable in
// a week and unusable on the first evening, which is the evening a new waiter
// is standing at it during service.

import type { ReactNode } from "react";
import { LuPower } from "react-icons/lu";

export interface TillNavItem {
  id: string;
  icon: ReactNode;
  label: string;
  /** Something needs attention behind this one — an unfiled receipt, a check
   *  nobody has sent to the kitchen. */
  dot?: boolean;
  disabled?: boolean;
}

export default function TillNav({
  items,
  value,
  onPick,
  onExit,
  exitLabel,
}: {
  items: TillNavItem[];
  value: string;
  onPick: (id: string) => void;
  /** Retire this screen, or absent when whoever is unlocked may not.
   *
   *  ⚠️ **In the rail but never one of the destinations.** It sits after a
   *  divider, pinned to the far end, in the warning colour — because the rail
   *  is a place where every other button changes what you are looking at and
   *  this one takes the machine out of service. A control that looked like the
   *  fourth tab would be pressed like the fourth tab.
   *
   *  ⚠️ Absent rather than disabled for a cashier or a waiter: a greyed-out
   *  control is one somebody keeps pressing and eventually asks a manager to
   *  press for them. The server refuses it too; this only decides whether it
   *  appears. */
  onExit?: () => void;
  exitLabel?: string;
}) {
  return (
    // ⚠️ A row on a narrow screen, a rail on a wide one — never hidden. These
    // screens are built for a 1024×768 monoblock and a 1280×800 tablet, but the
    // same URL gets opened on a phone to check something, and a navigation that
    // disappears below a breakpoint is a screen with no way out of it.
    <nav className="till-rail flex w-full shrink-0 flex-row items-center gap-2 overflow-x-auto border-b px-2 py-2 lg:w-[6.5rem] lg:flex-col lg:border-b-0 lg:border-r lg:px-0 lg:py-3">
      {items.map((it) => {
        const on = it.id === value;
        return (
          <button
            key={it.id}
            onClick={() => onPick(it.id)}
            disabled={it.disabled}
            aria-current={on ? "page" : undefined}
            className={`relative flex w-[4.75rem] shrink-0 flex-col items-center gap-1.5 rounded-[12px] py-3 transition disabled:opacity-35 ${
              on
                ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                : "text-ink-muted hover:bg-ink/[0.04] hover:text-ink-soft"
            }`}
          >
            <span className="text-[22px] leading-none">{it.icon}</span>
            <span className="text-[12px] font-semibold tracking-[0.01em]">
              {it.label}
            </span>
            {it.dot && (
              <span
                className="absolute right-3 top-2.5 h-2 w-2 rounded-full"
                style={{ background: "rgb(var(--till-accent))" }}
              />
            )}
          </button>
        );
      })}

      {onExit && (
        <>
          {/* ⚠️ A divider, not a gap. On a rail of four evenly spaced buttons
              the eye reads spacing as rhythm rather than as meaning; a line is
              the only separation that survives being glanced at. */}
          <span
            aria-hidden
            className="ml-1 h-8 w-px shrink-0 bg-ink/10 lg:ml-0 lg:mt-1 lg:h-px lg:w-8"
          />
          <button
            onClick={onExit}
            aria-label={exitLabel}
            title={exitLabel}
            // ⚠️ `ml-auto` on a phone, `mt-auto` on the monoblock: the rail is
            // a row under one breakpoint and a column over it, and "the far
            // end" is a different axis in each. Without both, this lands in the
            // middle of the row on a tablet — beside the tabs, which is the one
            // place it must not be.
            className="ml-auto flex w-[4.75rem] shrink-0 flex-col items-center gap-1.5 rounded-[12px] py-3 text-[rgb(var(--till-late))] transition hover:bg-ink/[0.04] lg:ml-0 lg:mt-auto"
          >
            <span className="text-[22px] leading-none">
              <LuPower />
            </span>
            <span className="text-[12px] font-semibold tracking-[0.01em]">
              {exitLabel}
            </span>
          </button>
        </>
      )}
    </nav>
  );
}

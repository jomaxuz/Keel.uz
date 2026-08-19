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
}: {
  items: TillNavItem[];
  value: string;
  onPick: (id: string) => void;
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
    </nav>
  );
}

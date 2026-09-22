"use client";

import { createContext, useContext } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import { panelWords, type PanelWords } from "@/lib/panelWords";

// What the business behind this counter sells, on the till.
//
// ⚠️ **The till is the screen furthest from the panel and the one somebody
// stands at all day.** It has no sidebar, no brand switcher and no scope — so
// `usePanelWords` cannot help it — and every word on it was a kitchen's: "Taom
// qidirish" over a shelf of shampoo, "Chek bo'sh — menyudan taom tanlang" in
// front of a queue. The session call the lock screen already makes now carries
// the type (handlers/tillpin.go), and this carries it down.
//
// ⚠️ **One value, not the session object.** The screens under here need a word,
// not a session; passing the whole thing would make every one of them a place
// that could start reading something else from it.
//
// ⚠️ Empty is a restaurant — the same fallback the panel, the site and the
// server all make, so a till talking to an older server says what it said
// yesterday rather than saying nothing.
const TillBusinessContext = createContext("");

export function TillBusinessProvider({
  type,
  children,
}: {
  type: string;
  children: React.ReactNode;
}) {
  return (
    <TillBusinessContext.Provider value={type}>
      {children}
    </TillBusinessContext.Provider>
  );
}

/** The words this counter uses. */
export function useTillWords(): PanelWords {
  const t = useAdminT();
  return panelWords(t, { businessType: useContext(TillBusinessContext) });
}

"use client";

// Everything that turns a page into a till, mounted once.
//
// ⚠️ **This exists because the pieces were listed in three places and one of
// them was a copy.** `kassa/layout.tsx` and `zal/layout.tsx` are Next route
// conventions — they take no props and cannot be imported — so the Windows
// application (backend/desktop) reproduced them by hand, with a comment saying
// a fourth piece would have to be added here too. It was not: `AskProvider`
// landed in both layouts and not in the shell, so every question on the Windows
// till fell through to `window.confirm` — the browser's box, with the
// restaurant's domain over it, on the machine that is sold as a cash register.
// The browser till was fixed and the one on the counter was not, which is the
// worst shape this bug can take: the report comes back as "it is still old".
//
// So the list lives in a component and the three surfaces mount the component.
// A copy cannot fall behind what it no longer contains.
//
// ⚠️ **Order and nesting are part of the contract, not formatting**:
//   - `TillAppliance` before everything: a press has to register on the dialog
//     and the keyboard too, and both render outside `<main>`.
//   - `AskProvider` and the keyboard **inside** `.appliance`, never around it —
//     a dialog outside that class is selectable, zoomable and answers a press
//     where the finger *ended*, which is the surface every earlier local fix
//     kept missing.

import type { ReactNode } from "react";

import CrashReporter from "@/components/CrashReporter";
import AskProvider from "@/components/ui/Ask";
import OnScreenKeyboard from "@/components/till/OnScreenKeyboard";
import TillAppliance from "@/components/till/TillAppliance";
import { StaffProvider } from "@/lib/staff";

export default function TillShell({
  role,
  children,
}: {
  /** What the person at this screen is — "kassir", "ofitsiant". ⚠️ A role,
   *  never a name: it is what makes a crash report reproducible, and the
   *  cashier's identity is not ours to collect from somebody else's staff. */
  role: string;
  children: ReactNode;
}) {
  return (
    <StaffProvider>
      <CrashReporter app="till" role={role} />
      <TillAppliance />
      <div className="appliance">
        <AskProvider look="till">
          <div className="min-h-dvh bg-bg">{children}</div>
          <OnScreenKeyboard />
        </AskProvider>
      </div>
    </StaffProvider>
  );
}

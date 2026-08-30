"use client";

// Starts the crash reporter for one app. Rendered once, from that app's layout.
//
// ⚠️ **A component rather than a call, because every layout that needs it is a
// server component.** Six of the seven apps here are, and a hook cannot run in
// one — so the alternative was making six layouts client components to install
// a listener, which would pull their whole subtree onto the client to do it.

import { useEffect } from "react";
import { installReporter, type ReportApp } from "@/lib/report";

export default function CrashReporter({
  app,
  role,
}: {
  app: ReportApp;
  /** What the person at this screen is, where the app knows it. ⚠️ A role, never
   *  a name: "kassir" is what makes a report reproducible, and the cashier's
   *  identity is not ours to collect from somebody else's staff. */
  role?: string;
}) {
  useEffect(() => {
    installReporter(app, () => ({ role }));
  }, [app, role]);
  return null;
}

"use client";

// Answered by the landing site for any address that matches no route — including
// `/console/…` paths a stale bookmark points at, since the console is part of
// this app.
//
// ⚠️ A client component so the language survives. The provider is mounted by the
// root layout with the language the middleware resolved, and reading headers
// again here would make `/_not-found` dynamic for every visitor to change
// nothing.

import { LostPage } from "@/components/DeadEnd";

export default function NotFound() {
  return <LostPage />;
}

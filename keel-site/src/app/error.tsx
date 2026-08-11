"use client";

// A page threw. On the landing site that is usually the control plane being
// unreachable — which is exactly what /status exists to say, so this page points
// there rather than at the marketing home.

import { useEffect } from "react";
import { BrokenPage } from "@/components/DeadEnd";

export default function SiteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("[keel-site]", error);
  }, [error]);

  return <BrokenPage digest={error.digest} onRetry={reset} />;
}

// The reference customers shown on the landing page.
//
// Fetched on the server during the page render, from the control plane over
// the internal network — the same way the tenant resolver works, and for the
// same reason: the browser never talks to the control API directly.

export interface Partner {
  name: string;
  /** Absolute, on the customer's own domain. Empty when they have no logo —
   *  the strip renders their name as a wordmark instead of dropping them. */
  logoUrl: string;
  url: string;
}

/** Where a *server* render reaches the control plane.
 *
 *  In production Caddy puts keel.uz and the control API on one host, so the
 *  browser uses a relative path — but a server render has no origin to be
 *  relative to. This is the only place that needs the internal address, and it
 *  is read at request time rather than baked in: a value sealed into the build
 *  is the trap `rewrites()` already set once. */
export const CONTROL = process.env.CONTROL_ORIGIN ?? "http://keel-control:9000";

/** The control plane's unauthenticated, server-to-server prefix.
 *
 *  **Not `/api/v1`.** That prefix carries the dashboard session; these two
 *  endpoints live under `/internal`, beside `resolve` and `tls-ask`, because
 *  the caller is this container rather than a browser. Getting it wrong is
 *  silent: both helpers swallow the 404 and return "no partners" and "cannot
 *  reach the control plane" — which is exactly what shipped, and exactly what
 *  a real outage looks like. `handlers/router_test.go` now pins the paths. */
export const INTERNAL = "/internal";

/** One hour of measured uptime. `seen: false` means no sample exists — the
 *  platform was not running, or was not yet measured. Drawn as a gap, never as
 *  a failure: a status page that paints ignorance red is one nobody believes
 *  the second time. */
export interface StatusHour {
  hour: string;
  checks: number;
  ok: number;
  maxMs: number;
  note?: string;
  seen: boolean;
}

export interface StatusDay {
  day: string;
  checks: number;
  ok: number;
}

export interface PlatformStatus {
  /** What the running platform calls itself, from the binary answering the request —
   *  see handlers/version.go for why not a file, a tag or an env var. */
  version?: string;
  /** What the number means: "test" while it still does. */
  stage?: string;
  up: boolean;
  /** Null when nothing has ever been measured. */
  lastCheck: string | null;
  uptime90d: number;
  hours: StatusHour[];
  days: StatusDay[];
  now: string;
}

/** How long a page render will wait on the control plane before giving up.
 *
 *  ⚠️ **A `try/catch` does not protect against a hang, only against an error.**
 *  Both pages below are server-rendered per request, so without a deadline a
 *  slow control plane does not degrade keel.uz — it *stops* it, for every
 *  visitor, with the "we render anyway" comment sitting right there being
 *  wrong. It took down the deploy's own health check first, which is the
 *  cheapest possible place to find out.
 *
 *  Two seconds: this is a call to a neighbouring container on the same docker
 *  network. Anything slower than that is not slow, it is broken, and the
 *  fallback is a better answer than a spinner. */
const TIMEOUT_MS = 2000;

export async function getStatus(): Promise<PlatformStatus | null> {
  try {
    const res = await fetch(`${CONTROL}${INTERNAL}/status`, {
      cache: "no-store",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (!res.ok) return null;
    return (await res.json()) as PlatformStatus;
  } catch {
    // Unreachable, or slower than the deadline — from this page's point of
    // view the same thing, and the same honest answer. The control plane being
    // unreachable *is* the status; returned as null so the page can say so,
    // rather than as an error page that says nothing.
    return null;
  }
}

export async function getPartners(): Promise<Partner[]> {
  try {
    const res = await fetch(`${CONTROL}${INTERNAL}/partners`, {
      // Re-fetched every few minutes rather than on every visit: the control
      // plane already caches this, and the landing page is the most-requested
      // page on the platform.
      next: { revalidate: 300 },
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    if (!res.ok) return [];
    const body = (await res.json()) as { partners?: Partner[] };
    return body.partners ?? [];
  } catch {
    // A marketing page must render when the control plane is having a bad
    // day. An empty strip is invisible; a 500 is the front door.
    return [];
  }
}

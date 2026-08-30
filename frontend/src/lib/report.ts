// What broke, on its way to the people who can fix it.
//
// ⚠️ **The point is that nobody has to be told.** The support queue depends on
// a restaurant noticing, deciding it is worth reporting, and describing it —
// and every one of those steps loses faults: the ones that hit one cashier on
// one tablet, the ones people work around, and the ones nobody can put into
// words. This path has none of them in it, so a bug is usually fixed before the
// owner writes in about it.
//
// ⚠️ **It must never make the situation worse.** Everything here is wrapped,
// queued, fire-and-forget and silent: a reporter that throws inside an error
// handler turns one broken screen into a broken screen with a broken error
// handler, in the code least likely to be exercised. There is no retry loop, no
// user-visible failure, and no console noise of its own.
//
// ⚠️ **It carries no guest data and no credentials.** A message, a stack, a
// route, a role, a version. Never a request body, never headers, never a name
// or a phone number — the field that *could* hold one eventually will.

import { API_URL } from "@/lib/api";
import { apiOverride } from "@/lib/tokenStore";

/** Where this app's backend is.
 *
 * ⚠️ The runtime override comes first, exactly as every other call does: the
 * native apps have no origin to be relative to, and the address is a fact about
 * the account somebody signed into. Getting this wrong would post a Windows
 * till's crashes at whatever host the binary was built against. */
function reportUrl(): string {
  return (apiOverride() || API_URL) + "/report";
}

/** Which program is reporting. Matches the ids the console groups by. */
export type ReportApp =
  "panel" | "site" | "till" | "waiter" | "kitchen" | "courier";

type Report = {
  app: ReportApp;
  message: string;
  stack?: string;
  where?: string;
  context?: string;
  branch?: string;
  role?: string;
  version?: string;
  platform?: string;
  session?: string;
  at: string;
};

/** How many distinct faults one page will send before it stops.
 *
 *  ⚠️ A render loop calls this on every frame. Without a ceiling the first
 *  broken component would post continuously for as long as the tab is open,
 *  which is a denial of service the restaurant runs against itself. */
const MAX_PER_PAGE = 8;

/** The same fault, again, within this window is not news. */
const DEDUPE_MS = 60_000;

let app: ReportApp = "site";
let sent = 0;
const seen = new Map<string, number>();
let queue: Report[] = [];
let timer: ReturnType<typeof setTimeout> | null = null;
let installed = false;

/** Extra facts the app knows about itself, set once it knows them. */
let describe: () => { branch?: string; role?: string } = () => ({});

/** An id for this install, so "one tablet" can be told from "everybody".
 *
 *  ⚠️ Random and stored locally — not a device fingerprint and not a user id.
 *  All the console needs is whether two reports came from one place, and the
 *  platform hashes even this before storing it. */
function sessionId(): string {
  try {
    const k = "keel.report.session";
    let v = localStorage.getItem(k);
    if (!v) {
      v = Math.random().toString(36).slice(2) + Date.now().toString(36);
      localStorage.setItem(k, v);
    }
    return v;
  } catch {
    // Private windows and blocked site data. A report without an install id is
    // still a report; refusing to send one would lose exactly the browsers
    // whose settings are unusual enough to break things.
    return "";
  }
}

/** Turn whatever was thrown into a message and a stack.
 *
 *  ⚠️ Anything can be thrown in JavaScript — a string, a number, `undefined`, a
 *  Response. `err.message` on those is undefined, which used to arrive as the
 *  literal word "undefined" and grouped every unrelated fault into one row. */
function describeError(err: unknown): { message: string; stack?: string } {
  if (err instanceof Error) {
    return { message: err.message || err.name || "Error", stack: err.stack };
  }
  if (typeof err === "string") return { message: err };
  try {
    return { message: JSON.stringify(err) ?? String(err) };
  } catch {
    return { message: String(err) };
  }
}

/** Report one fault. Safe to call from anywhere, including an error boundary. */
export function report(
  err: unknown,
  ctx?: { where?: string; context?: string },
) {
  try {
    if (typeof window === "undefined" || sent >= MAX_PER_PAGE) return;

    const { message, stack } = describeError(err);
    if (!message) return;

    // ⚠️ Deduped in the browser as well as grouped on the server. Grouping
    // makes the *list* readable; this stops the network being used for four
    // hundred identical posts on the way there.
    const key = message + "|" + (ctx?.where ?? "");
    const now = Date.now();
    const last = seen.get(key);
    if (last && now - last < DEDUPE_MS) return;
    seen.set(key, now);
    sent++;

    const extra = describe();
    queue.push({
      app,
      message: message.slice(0, 400),
      stack: stack?.slice(0, 4000),
      where: ctx?.where ?? location.pathname,
      context: ctx?.context,
      branch: extra.branch,
      role: extra.role,
      version: process.env.NEXT_PUBLIC_BUILD ?? "",
      platform: navigator.userAgent.slice(0, 120),
      session: sessionId(),
      at: new Date().toISOString(),
    });

    // Batched, because faults arrive in bursts: one broken render throws in
    // three components before the boundary catches it.
    if (!timer) timer = setTimeout(flush, 2000);
  } catch {
    // The reporter failing is not itself reportable. See the note at the top.
  }
}

function flush() {
  timer = null;
  const batch = queue;
  queue = [];
  if (batch.length === 0) return;

  const body = JSON.stringify({ reports: batch });
  try {
    // ⚠️ `sendBeacon` first, because the commonest moment to have something to
    // report is the moment the page is going away — a crash the guest responds
    // to by closing the tab. A `fetch` started there is cancelled by the
    // browser; a beacon is handed to the browser to deliver afterwards.
    if (navigator.sendBeacon?.(reportUrl(), body)) return;
  } catch {
    // Falls through.
  }
  void fetch(reportUrl(), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
    keepalive: true,
  }).catch(() => {});
}

/** Start listening. Called once per app, from its layout.
 *
 *  ⚠️ **Both handlers, not just onerror.** A rejected promise with no `.catch`
 *  never reaches `window.onerror`, and in this codebase most failures are
 *  awaited API calls — which is to say the majority of what there is to report
 *  arrives only through `unhandledrejection`. */
export function installReporter(
  which: ReportApp,
  facts?: () => { branch?: string; role?: string },
) {
  app = which;
  if (facts) describe = facts;
  if (installed || typeof window === "undefined") return;
  installed = true;

  window.addEventListener("error", (e) => {
    report(e.error ?? e.message, { where: location.pathname });
  });
  window.addEventListener("unhandledrejection", (e) => {
    report(e.reason, {
      where: location.pathname,
      context: "unhandled promise",
    });
  });
  // A page going away takes the queue with it unless it is pushed now.
  window.addEventListener("pagehide", () => {
    if (timer) clearTimeout(timer);
    flush();
  });
}

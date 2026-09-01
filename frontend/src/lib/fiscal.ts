import type { FiscalJob, FiscalReply } from "@/lib/types";

/** The four sentences this module can produce, in the cashier's language.
 *
 *  ⚠️ **Passed in rather than written here.** They are read on a till that is
 *  three-language, and a module outside React cannot ask the dictionary for
 *  itself — so the screen that has `t` hands them over. They also travel to the
 *  server, which stores the text as the reason a filing did not happen. */
export type FiscalWords = {
  badAddress: string;
  blocked: string;
  timedOut: string;
  unreachable: string;
};

/** Making the server's call to the cash register, from the one machine that can.
 *
 *  The registered virtual cash register is a program on a PC inside the
 *  restaurant, on an address like http://192.168.14.65:9090 with no
 *  authentication — it is unauthenticated precisely because it is unroutable
 *  from outside the building, which is also why our server cannot use it. This
 *  tablet is on that network, so it carries the document one hop.
 *
 *  ⚠️ **Nothing here composes, edits or interprets the document.** The job
 *  arrives with its body already serialised and goes out byte for byte; the
 *  reply comes back raw for the server to parse. Every shortcut that felt
 *  tempting — reading the fiscal sign here to show it a moment sooner, retrying
 *  on a "looks like an error" body — moves a piece of tax handling onto the
 *  least trusted machine in the system. */

/** Why the browser refused before a packet was sent.
 *
 *  ⚠️ **Checked in advance rather than caught afterwards**, and this is the
 *  whole reason this function exists. A browser blocking an http:// call from
 *  an https:// page reports it as a bare `TypeError: Failed to fetch` — exactly
 *  what an unplugged cable, a wrong port and a switched-off PC all report. So
 *  the one cause that is a *configuration* rather than a fault would be the one
 *  the cashier could never distinguish, and they would spend the evening
 *  checking a network that was working perfectly.
 *
 *  Two separate browser rules bite here, and both are deliberate browser
 *  behaviour we cannot ask our way out of from JavaScript:
 *
 *   - **Mixed content** — a secure page may not load an insecure subresource.
 *   - **Private Network Access** — a public page may not reach a private
 *     address, even over https, unless the private end opts in with a CORS
 *     header the register does not send.
 *
 *  The workable answers are all deployment ones, so the message names them
 *  instead of asking the user to guess. */
function blockedReason(url: string, w: FiscalWords): string | null {
  if (typeof window === "undefined") return null;
  let target: URL;
  try {
    target = new URL(url);
  } catch {
    return w.badAddress;
  }
  if (
    window.location.protocol === "https:" &&
    target.protocol === "http:" &&
    !isLoopback(target.hostname)
  ) {
    return w.blocked;
  }
  return null;
}

/** Whether the address is this very machine.
 *
 *  ⚠️ **The one http:// address a secure page may still call.** Browsers treat
 *  localhost as a potentially trustworthy origin, so mixed-content blocking
 *  does not apply to it — and this is not a loophole to exploit but the
 *  deployment Multikassa's own documentation assumes: its examples give
 *  `http://localhost:8080` and `http://localhost:12346` as the base address,
 *  because the register expects the till software to run on the register's own
 *  PC.
 *
 *  Getting this wrong is expensive in the quiet direction: refusing the
 *  vendor's recommended setup, with a message telling the restaurant to weaken
 *  a browser setting they never needed to touch. */
function isLoopback(host: string): boolean {
  return (
    host === "localhost" ||
    host === "127.0.0.1" ||
    host === "[::1]" ||
    host === "::1" ||
    host.endsWith(".localhost")
  );
}

/** Run one job and report exactly what happened.
 *
 *  ⚠️ **Never throws.** The caller is the payment flow, and a check whose money
 *  is already in the drawer must not be left in an error boundary because a PC
 *  in the corner did not answer. Every outcome — refusal, timeout, block — comes
 *  back as a reply the server can record, which is what makes the filing
 *  retryable instead of lost.
 *
 *  ⚠️ The reply body is returned **whatever the status**. This register answers
 *  business refusals with HTTP 500 and a readable body ("#2D — the shift is not
 *  open"), so discarding the body on a non-2xx would throw away the only
 *  sentence that tells the cashier what to do. */
export async function runFiscalJob(job: FiscalJob, words: FiscalWords): Promise<FiscalReply> {
  const blocked = blockedReason(job.url, words);
  if (blocked) return { status: 0, body: "", networkError: blocked };

  const abort = new AbortController();
  const timer = setTimeout(() => abort.abort(), job.timeoutMs || 15000);
  try {
    const res = await fetch(job.url, {
      method: job.method || "POST",
      headers: job.headers,
      body: job.method === "GET" ? undefined : job.body,
      signal: abort.signal,
      // ⚠️ No credentials and no cache. The register has no session to carry,
      // and a cached filing response would be a receipt shown for a sale that
      // was never sent.
      credentials: "omit",
      cache: "no-store",
    });
    return { status: res.status, body: await res.text() };
  } catch (e) {
    return { status: 0, body: "", networkError: describeFailure(e, words) };
  } finally {
    clearTimeout(timer);
  }
}

/** Turn a fetch failure into something a cashier can act on.
 *
 *  A timeout and a refusal send somebody to different places — one to wait and
 *  look at the register's screen, the other to check the address — and the
 *  browser reports both as the same opaque TypeError, so the distinction has to
 *  be reconstructed from what we know about the call we made. */
function describeFailure(e: unknown, w: FiscalWords): string {
  if (e instanceof DOMException && e.name === "AbortError") {
    return w.timedOut;
  }
  return w.unreachable;
}

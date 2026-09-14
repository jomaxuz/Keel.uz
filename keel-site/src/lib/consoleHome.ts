import type { Me } from "./api";

// Where an account lands after signing in, and where `/console` sends it.
//
// ⚠️ **Only the owner lands on the overview.** For everybody else it was a page
// of platform numbers that are not theirs to act on — and for a sales account,
// whose permissions hide most of them, a page that said "no data" first thing
// every morning. Each role goes straight to the screen its work is on.
export function consoleHome(me: Pick<Me, "can">): string {
  if (me.can.overview) return "/console";
  if (me.can.tenants) return "/console/tenants";
  if (me.can.support) return "/console/support";
  // Nothing granted is a misconfigured account; the customer list answers it
  // with its own refusal rather than with a redirect loop.
  return "/console/tenants";
}

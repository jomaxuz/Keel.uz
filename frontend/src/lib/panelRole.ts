// Where a panel account lands.
//
// ⚠️ **Its own module because two screens need the same answer**: the login
// form, which navigates after a successful sign-in, and the layout, which
// redirects somebody who arrived at a page that is not theirs. Two copies of
// "where does this role start" drift, and the drift is a role that signs in
// onto a screen it is refused — reported as "my login does not work".

import type { PanelRole } from "./types";

/** The first screen this role can actually work on.
 *
 *  ⚠️ Not `/admin` for the limited roles: the dashboard is the company's
 *  numbers and answers forbidden for both of them, and a panel that opens on an
 *  error reads as a broken account rather than as a boundary. */
export function homeFor(role: PanelRole | string | undefined): string {
  if (role === "operator") return "/admin/orders";
  if (role === "stock") return "/admin/stock";
  return "/admin";
}

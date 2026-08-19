import type { Staff } from "@/lib/types";

/** Whoever is signed into this browser, if anybody.
 *
 *  ⚠️ **A bound monoblock has nobody.** It carries a device token for its
 *  branch and no staff account at all — which is the whole point of binding it
 *  from the panel with a link. Tests default to that, because it is both the
 *  ordinary installation and the one where a screen gated on `staff` silently
 *  shows nothing. */
let staff: Staff | null = null;

export function setSignedInStaff(next: Staff | null) {
  staff = next;
}

export function currentStaff(): Staff | null {
  return staff;
}

/** A staff account with both till permissions, for the older login-based till. */
export function tillStaff(over: Partial<Staff> = {}): Staff {
  return {
    id: "s9",
    name: "Aziz",
    canWaiter: true,
    canCashier: true,
    isActive: true,
    ...over,
  } as Staff;
}

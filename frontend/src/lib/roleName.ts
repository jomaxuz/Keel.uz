// What to print in the corner of a till: the name of the role the person
// holding the screen was given, in the language the screen is in.
//
// ⚠️ **One helper because the till reads the name from two places.** An
// unlocked screen knows the person from the PIN (`TillPerson`), a screen signed
// in with a staff login knows them from `/staff/me` (`Staff`) — and the corner
// has to say the same word either way. Two copies of the fallback chain drift,
// and the drift shows up as a manager reading as a waiter on one of the two.
//
// ⚠️ **A name to print, never a permission.** The spelling of a role grants
// nothing — see models/staffrole.go. `canExit` and the `can*` flags are the
// answers; this is only the label.

import { contentText } from "./i18n/content";
import type { Lang } from "./i18n";
import type { Staff, TillPerson } from "./types";

export function roleLabelOf(
  person: TillPerson | null | undefined,
  staff: Staff | null | undefined,
  lang: Lang,
): string | undefined {
  if (person?.role)
    return contentText(person.role, person.roleRu, person.roleEn, lang);
  if (staff?.roleName)
    return contentText(
      staff.roleName,
      staff.roleNameRu,
      staff.roleNameEn,
      lang,
    );
  // Undefined rather than "": the caller's own fallback ("Ofitsiant", or the
  // cashier/waiter pair on the till) is a real answer for an account with no
  // role at all, which is every install that predates them.
  return undefined;
}

// The zero ObjectID, as the API actually sends it.
//
// Go's `json:"...,omitempty"` does **not** omit an empty primitive.ObjectID:
// omitempty has no effect on arrays, and an ObjectID is a [12]byte. So a field
// that is "not set" arrives as this string — which is perfectly truthy in
// JavaScript. Every `if (order.courierId)` written against such a field is
// therefore always true.
//
// This trap is worth a named helper rather than a comparison sprinkled around,
// because it looks correct at every call site where it is wrong.
const ZERO_ID = "000000000000000000000000";

/** True when the field actually points at something. */
export function hasId(id: string | null | undefined): id is string {
  return !!id && id !== ZERO_ID;
}

/** The id, or "" when it is unset — for values passed on to an API or a form. */
export function realId(id: string | null | undefined): string {
  return hasId(id) ? id : "";
}

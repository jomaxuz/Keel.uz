import { DEFAULT_LANG, getDict, type Lang } from "./i18n";

// UZS is stored and displayed as whole so'm (no tiyin). Group thousands with a
// plain ASCII space.
//
// NOTE: intentionally NOT using Intl.NumberFormat — Node (SSR) and the browser
// can pick different uz-UZ grouping separators (regular vs narrow no-break
// space), which triggers a React hydration mismatch. Manual grouping is
// deterministic across server and client.
export function formatPrice(
  amount: number,
  currency = "UZS",
  lang: Lang = DEFAULT_LANG,
): string {
  const grouped = Math.round(amount)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  // "so'm" / "сум" / "UZS" depending on the active language.
  return currency === "UZS"
    ? `${grouped} ${getDict(lang).som}`
    : `${grouped} ${currency}`;
}

export function weekdayName(day: number, lang: Lang = DEFAULT_LANG): string {
  return getDict(lang).weekdays[day] ?? "";
}

// "998901234567" → "+998 90 123 45 67" (stored form → human form).
export function formatUzPhone(phone: string): string {
  const d = phone.replace(/\D/g, "");
  if (d.length !== 12 || !d.startsWith("998")) return phone;
  return `+${d.slice(0, 3)} ${d.slice(3, 5)} ${d.slice(5, 8)} ${d.slice(8, 10)} ${d.slice(10)}`;
}

// ---- Dates and times ----
//
// Formatted by hand rather than through `toLocaleString`, for two reasons that
// both bit this project:
//
//  1. **24-hour, everywhere.** `Intl` follows the locale — and an `en` locale,
//     or a phone set to US English, renders "1:16 PM". A shift board that
//     reads "11:00 — 23:00" on one device and "11:00 AM — 11:00 PM" on the
//     next is not a shift board.
//  2. **Deterministic across SSR and the browser.** Node and the browser can
//     disagree about a locale's separators, which shows up as a React
//     hydration mismatch — the same reason `formatPrice` above avoids `Intl`.
//
// The format is numeric and language-neutral, so it needs no translation.

function asDate(value: string | number | Date): Date {
  return value instanceof Date ? value : new Date(value);
}

const pad2 = (n: number) => String(n).padStart(2, "0");

/** "14:05" — always 24-hour. */
export function formatTime(value: string | number | Date): string {
  const d = asDate(value);
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

/** "01.08.2026" */
export function formatDate(value: string | number | Date): string {
  const d = asDate(value);
  return `${pad2(d.getDate())}.${pad2(d.getMonth() + 1)}.${d.getFullYear()}`;
}

/** "01.08.2026 14:05" */
export function formatDateTime(value: string | number | Date): string {
  return `${formatDate(value)} ${formatTime(value)}`;
}

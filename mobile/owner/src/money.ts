/** Whole so'm, grouped every three digits.
 *
 *  ⚠️ **Not `toLocaleString`.** Android's ICU data varies by version and vendor,
 *  so the same total is grouped on one phone and not on the next — and two
 *  spellings of one number on two waiters' phones is read as two numbers. This
 *  is also how every other Keel screen prints money, which is the point: a
 *  guest compares the phone against the paper. */
export function money(n: number): string {
  return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, " ");
}

// The published day, spelled in the reader's language.
//
// ⚠️ **Formatted on the server, from the stored day.** A date built in the
// browser is a date in the reader's own timezone — and an article published at
// nine in the evening in Tashkent shows the day before to anybody west of it.
//
// ⚠️ **A hand-written month list, not `Intl`.** Node's ICU data differs by
// build, so the same post is dated one way on the server and another after
// hydration — which React reports as a mismatch and a reader sees as a flicker.

/** ⚠️ **Formatted on the server, from the stored day.** A date built in the
 *  browser is a date in the reader's own timezone — and an article published at
 *  nine in the evening in Tashkent shows the day before to anybody west of it. */
export function dateOf(iso: string | undefined, lang: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const months: Record<string, string[]> = {
    uz: ["yanvar", "fevral", "mart", "aprel", "may", "iyun", "iyul", "avgust", "sentabr", "oktabr", "noyabr", "dekabr"],
    ru: ["января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"],
    en: ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"],
  };
  const m = months[lang] ?? months.uz;
  return lang === "en"
    ? `${m[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
    : `${d.getDate()} ${m[d.getMonth()]} ${d.getFullYear()}`;
}

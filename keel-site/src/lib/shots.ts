import type { Lang } from "./i18n/dict";

/** The landing page's screenshots, in the language the page is being read in.
 *
 *  ⚠️ **These pictures are the page's argument, not its decoration.** The whole
 *  pitch is "this is the real till, the real floor plan, the real kitchen
 *  screen" — and for two thirds of the visitors that claim used to be made in a
 *  language they could not read. A Russian owner looking at an Uzbek till is
 *  being shown a screen that is not theirs, which undercuts the exact thing the
 *  picture is there to say.
 *
 *  ⚠️ **No fallback, deliberately.** A missing frame here should 404 loudly in
 *  the browser's network panel rather than quietly serve the Uzbek one: falling
 *  back restores the original problem and hides it at the same time. The
 *  frames are produced by `scripts/landing-shots.mjs`, which takes all three
 *  in one run — a language cannot go missing on its own.
 */
export function shot(lang: Lang, name: string): string {
  return `/shots/${name}.${lang}.webp`;
}

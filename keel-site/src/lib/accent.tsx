import { Fragment } from "react";

/** Section headings are two-coloured: the half that carries the claim is drawn
 *  in the brand colour, the rest in ink. Which half that is depends on the
 *  sentence, so it cannot be decided in the layout — it is marked in the
 *  dictionary with `*asterisks*`, beside the words themselves.
 *
 *  ⚠️ **The marker has to be stripped wherever the string is not React.** The
 *  same `hero.title` is drawn into the Open Graph image, and there an asterisk
 *  is simply a typo on every share preview the product ever gets. That is why
 *  the stripping is a function next to the renderer rather than a `replace()`
 *  at the one call site that needed it first — the next call site is where it
 *  would be forgotten.
 *
 *  Unmarked strings render exactly as they are, so a translation that has not
 *  been marked yet is a heading in one colour, never a broken one. */
export function accent(text: string) {
  const parts = text.split("*");
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <span key={i} className="text-signal-600 dark:text-signal-400">
        {part}
      </span>
    ) : (
      <Fragment key={i}>{part}</Fragment>
    ),
  );
}

/** The same string for anywhere that takes text rather than nodes: metadata,
 *  the OG image, `alt` and `title` attributes. */
export function plain(text: string) {
  return text.replace(/\*/g, "");
}

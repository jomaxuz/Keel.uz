// One article's body.
//
// ⚠️ **A warning is a colour and a shape, not a bold sentence in a paragraph.**
// Everything in these articles is read in a hurry, standing up, usually while
// something is already going wrong — and the sentences that matter are the ones
// that say what breaks. Given the same weight as the prose around them they are
// read at the same speed as the prose around them, which is to say skipped.

import Link from "next/link";
import type { Block } from "@/lib/help/types";
import type { Lang } from "@/lib/i18n/dict";
import { localePath } from "@/lib/i18n/url";
import { FIGURES, help } from "@/lib/help";
import Figure from "./Figure";
import { Rich } from "./Rich";

export default function Blocks({
  blocks,
  lang,
  figureHint,
  seeAlso,
}: {
  blocks: Block[];
  lang: Lang;
  figureHint: string;
  seeAlso: string;
}) {
  return (
    <div className="space-y-4">
      {blocks.map((b, i) => {
        if ("h" in b) {
          return (
            <h2
              key={i}
              className="h-display pt-4 text-lg sm:text-xl"
              // Anchored so a support reply can link to the paragraph rather
              // than to the article and "scroll down a bit".
              id={slugify(b.h)}
            >
              {b.h}
            </h2>
          );
        }
        if ("p" in b) {
          return (
            <p key={i} className="text-[15px] leading-7 text-ink-soft">
              <Rich text={b.p} />
            </p>
          );
        }
        if ("steps" in b) {
          return (
            <ol key={i} className="space-y-2.5">
              {b.steps.map((s, j) => (
                <li key={j} className="flex gap-3 text-[15px] leading-7 text-ink-soft">
                  <span className="mt-1 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
                    {j + 1}
                  </span>
                  <span>
                    <Rich text={s} />
                  </span>
                </li>
              ))}
            </ol>
          );
        }
        if ("list" in b) {
          return (
            <ul key={i} className="space-y-2">
              {b.list.map((s, j) => (
                <li key={j} className="flex gap-3 text-[15px] leading-7 text-ink-soft">
                  <span className="mt-3 h-1.5 w-1.5 shrink-0 rounded-full bg-signal-500" />
                  <span>
                    <Rich text={s} />
                  </span>
                </li>
              ))}
            </ul>
          );
        }
        if ("warn" in b) {
          return (
            <div
              key={i}
              className="rounded-2xl border border-signal-500/40 bg-signal-500/[0.07] px-4 py-3.5 text-[15px] leading-7 text-ink-soft"
            >
              <span className="mr-1.5 font-semibold text-signal-600 dark:text-signal-400">
                ⚠
              </span>
              <Rich text={b.warn} />
            </div>
          );
        }
        if ("tip" in b) {
          return (
            <div
              key={i}
              className="rounded-2xl border border-line bg-raised px-4 py-3.5 text-[15px] leading-7 text-ink-soft"
            >
              <Rich text={b.tip} />
            </div>
          );
        }
        if ("fig" in b) {
          return (
            <Figure
              key={i}
              name={b.fig}
              spec={FIGURES[b.fig]}
              notes={b.notes}
              hint={figureHint}
            />
          );
        }
        if ("table" in b) {
          return (
            // ⚠️ Its own scroller. A four-column table on a phone otherwise
            // widens the page, and then every paragraph in the article scrolls
            // sideways with it.
            <div key={i} className="-mx-1 overflow-x-auto">
              <table className="w-full min-w-[420px] text-sm">
                <thead>
                  <tr className="border-b border-line text-left text-xs uppercase tracking-wide text-ink-muted">
                    {b.table.head.map((h, j) => (
                      <th key={j} className="px-3 py-2 font-semibold">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {b.table.rows.map((row, j) => (
                    <tr key={j} className="border-b border-line/60 align-top">
                      {row.map((cell, k) => (
                        <td key={k} className="px-3 py-2.5 text-ink-soft">
                          <Rich text={cell} />
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        }
        if ("see" in b) {
          const list = b.see
            .map((slug) => help[lang].articles.find((a) => a.slug === slug))
            .filter((a) => a !== undefined);
          if (list.length === 0) return null;
          return (
            <div key={i} className="rounded-2xl border border-line bg-raised px-4 py-3.5">
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {seeAlso}
              </p>
              <ul className="mt-2 space-y-1">
                {list.map((a) => (
                  <li key={a.slug}>
                    <Link
                      href={localePath(lang, `/help/${a.slug}`)}
                      className="text-[15px] font-medium text-signal-600 underline-offset-4 hover:underline dark:text-signal-400"
                    >
                      {a.title}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          );
        }
        return null;
      })}
    </div>
  );
}

/** A heading turned into an anchor. Latin letters and digits only: an id built
 *  from Cyrillic survives the round trip through a browser but not through
 *  every chat client that will be asked to carry the link. */
function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
}

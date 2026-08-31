// A screenshot with numbered outlines drawn over it.
//
// ⚠️ **The outlines are drawn by the browser, not baked into the picture.**
// One file serves all three languages, the numbers stay crisp at any zoom, and
// the words live in the article where a translator can reach them. A PNG with
// "Bu yerga bosing" burnt into it needs three of every screenshot, kept in step
// by hand — and the first one that slips shows a Russian reader an Uzbek arrow.
//
// ⚠️ **The legend is underneath, not floating.** A tooltip anchored to a box is
// unreadable on a phone, which is where a cook or a counter hand opens this. A
// numbered list under the frame reads the same on every screen, and it is what
// a printed manual has always done.

import Image from "next/image";
import type { Figure as FigureSpec } from "@/lib/help/types";
import type { Lang } from "@/lib/i18n/dict";
import { Rich } from "./Rich";

export default function Figure({
  name,
  spec,
  lang,
  notes,
  hint,
}: {
  name: string;
  spec?: FigureSpec;
  lang: Lang;
  /** Callout key → what it is. Order here is the order of the numbers. */
  notes?: Record<string, string>;
  /** "Rasmni kattalashtirish uchun bosing" — said once per article. */
  hint?: string;
}) {
  // ⚠️ **Falls back to Uzbek rather than to nothing.** A frame captured in one
  // language and not yet in another is a normal state to be in for an
  // afternoon; a blank space where a screenshot should be is not, and the
  // Uzbek panel at least shows the reader the right screen.
  const shot = spec?.[lang] ?? spec?.uz;
  const src = spec?.[lang] ? `/help/${name}.${lang}.webp` : `/help/${name}.uz.webp`;
  const boxes = shot?.notes ?? {};
  // ⚠️ Only the callouts this article asked for, and only those the capture
  // actually found. A number in the legend with no box on the picture is worse
  // than no callout at all: the reader hunts for it.
  const shown = Object.entries(notes ?? {}).filter(([k]) => boxes[k]);

  return (
    <figure className="my-7">
      <div className="overflow-hidden rounded-2xl border border-line bg-raised">
        <a
          href={src}
          target="_blank"
          rel="noreferrer"
          className="relative block"
          // Reserved before the image loads, so an article does not jump under
          // the reader's finger as its screenshots arrive.
          style={{ aspectRatio: `${shot?.w ?? 1440} / ${shot?.h ?? 900}` }}
        >
          <Image
            src={src}
            alt=""
            fill
            sizes="(max-width: 820px) 100vw, 760px"
            className="object-cover"
          />
          {shown.map(([key], i) => {
            const b = boxes[key];
            return (
              <span
                key={key}
                className="absolute rounded-md ring-2 ring-signal-500"
                style={{
                  left: `${b.x * 100}%`,
                  top: `${b.y * 100}%`,
                  width: `${b.w * 100}%`,
                  height: `${b.h * 100}%`,
                  // A wash rather than a fill: the point is to find the button,
                  // not to hide it.
                  background: "rgb(245 165 36 / 0.14)",
                }}
              >
                {/* The badge sits on the outline's top-left corner and is
                    pulled half outside it, so a small target — a tab, a
                    checkbox — is circled rather than covered. */}
                <span className="absolute -left-2.5 -top-2.5 grid h-5 w-5 place-items-center rounded-full bg-signal-500 text-[11px] font-bold text-hull-950 shadow">
                  {i + 1}
                </span>
              </span>
            );
          })}
        </a>
      </div>

      {shown.length > 0 && (
        <ol className="mt-3 space-y-1.5">
          {shown.map(([key, text], i) => (
            <li key={key} className="flex gap-2.5 text-sm text-ink-soft">
              <span className="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full bg-signal-500/15 text-[11px] font-bold text-signal-600 dark:text-signal-400">
                {i + 1}
              </span>
              <span>
                <Rich text={text} />
              </span>
            </li>
          ))}
        </ol>
      )}

      {hint && (
        <figcaption className="mt-2 text-xs text-ink-muted">{hint}</figcaption>
      )}
    </figure>
  );
}

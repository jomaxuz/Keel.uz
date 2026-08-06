// The customers who agreed to be named, drifting past.
//
// A marquee rather than a grid because the list is short and will stay short
// for a while: eight logos in a grid look like eight, and eight logos moving
// look like a platform. When it stops being short, this becomes a grid — the
// component, not the page, is where that decision lives.
//
// Mechanics worth knowing:
//
//   • **The strip is duplicated and the animation moves exactly one copy's
//     width.** That is what makes the loop seamless; any other distance
//     produces a visible jump once per cycle.
//   • **It pauses on hover and honours reduced-motion.** These are links, and
//     a link that walks away from the cursor is a link nobody clicks. Motion
//     that cannot be stopped is also the one accessibility failure a marketing
//     page reliably ships.
//   • **A customer with no logo is rendered as a wordmark**, not dropped —
//     silently punishing the restaurant that never uploaded one is not a
//     reference page, it is a filter nobody asked for.

import type { Partner } from "@/lib/partners";

export default function Partners({ items }: { items: Partner[] }) {
  if (items.length === 0) return null;

  // Short lists would leave a gap before the seam; repeating them fills the
  // strip first, and only then is it doubled for the loop.
  const min = 8;
  const filled: Partner[] = [];
  while (filled.length < min) filled.push(...items);
  const strip = [...filled, ...filled];

  return (
    <div
      className="partners-mask relative overflow-hidden"
      // The whole strip is one hover target: pausing only the logo under the
      // cursor would still move it out from under the cursor.
      style={{ ["--count" as string]: filled.length }}
    >
      <ul className="partners-track flex w-max items-center gap-10">
        {strip.map((p, i) => (
          <li key={`${p.url}-${i}`} className="shrink-0">
            <a
              href={p.url}
              target="_blank"
              rel="noreferrer"
              className="flex h-16 items-center gap-3 rounded-2xl px-4 opacity-70 transition-opacity hover:opacity-100"
              // The duplicated half is decoration; a screen reader that read
              // every name twice would be describing an animation, not a list.
              aria-hidden={i >= filled.length}
              tabIndex={i >= filled.length ? -1 : undefined}
            >
              {p.logoUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={p.logoUrl}
                  alt={p.name}
                  loading="lazy"
                  className="h-10 w-auto max-w-[140px] object-contain"
                />
              ) : (
                <span className="font-display text-lg font-semibold text-ink-soft">
                  {p.name}
                </span>
              )}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

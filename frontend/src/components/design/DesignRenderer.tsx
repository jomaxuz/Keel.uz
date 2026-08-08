// Turning a design document into a page.
//
// The renderer is deliberately dull: it orders bands, applies the grid, and
// hands each block the data the page already loaded. Everything that could go
// wrong was already made impossible upstream — widths are columns, styles are
// enums, and the backend sanitises the document on every read (see
// models/design.go).
//
// ⚠️ **Two rules that are the feature, not details:**
//
//   - **No design means today's page.** The fallback is `DEFAULT_SECTIONS`,
//     which is the order `app/(site)/page.tsx` used to render in — the same
//     components, in the same sequence. Existing customers must see no change
//     on the day this ships, and "renders an empty page when the document is
//     missing" would have been the easiest possible way to break every site at
//     once.
//   - **On a phone every band is full width.** That is the entire responsive
//     story, and it works only because a width is a number of columns rather
//     than a position. A pixel canvas would need a second layout drawn by hand
//     for the Telegram mini app, and it would be drawn badly or not at all.

import type { DesignSection, PageDesign } from "@/lib/types";
import {
  AboutBlock,
  CategoriesBlock,
  CtaBlock,
  GalleryBlock,
  HeroBlock,
  HoursAddressBlock,
  MenuGridBlock,
  PerksBlock,
  type BlockData,
} from "./blocks";

/** The page as it is today, expressed as bands.
 *
 *  ⚠️ Mirrors `models.DefaultSections()` in the backend. The two exist
 *  separately because either side can be asked for the default without the
 *  other — the console needs it to seed a new design, the site needs it when no
 *  document exists — and a fallback that depends on a round trip is a fallback
 *  that fails exactly when things are already failing. */
export const DEFAULT_SECTIONS: DesignSection[] = [
  { type: "hero", variant: "full", span: 12 },
  { type: "perks", variant: "cards", span: 12 },
  { type: "categories", variant: "tiles", span: 12 },
  {
    type: "menu-grid",
    variant: "cards",
    span: 12,
    binding: { popularOnly: true, limit: 8 },
  },
  { type: "hours-address", variant: "map", span: 12 },
];

const BLOCKS = {
  hero: HeroBlock,
  perks: PerksBlock,
  categories: CategoriesBlock,
  "menu-grid": MenuGridBlock,
  "hours-address": HoursAddressBlock,
  about: AboutBlock,
  gallery: GalleryBlock,
  cta: CtaBlock,
} as const;

/** Column spans, written out because Tailwind cannot see a computed class name.
 *
 *  Interpolating `lg:col-span-${n}` would compile to nothing at all — the class
 *  is not in the source for the compiler to find — and the page would silently
 *  lose every width. The map is the fix and it is also the allowlist: a span
 *  outside 1–12 cannot produce a class. */
const SPAN_CLASS: Record<number, string> = {
  1: "lg:col-span-1",
  2: "lg:col-span-2",
  3: "lg:col-span-3",
  4: "lg:col-span-4",
  5: "lg:col-span-5",
  6: "lg:col-span-6",
  7: "lg:col-span-7",
  8: "lg:col-span-8",
  9: "lg:col-span-9",
  10: "lg:col-span-10",
  11: "lg:col-span-11",
  12: "lg:col-span-12",
};

/** Background tones, from the design system's own tokens.
 *
 *  Never a colour: a band painted `#f4f1ea` would stay that colour in dark mode
 *  and would ignore the restaurant's accent. */
const TONE_CLASS: Record<string, string> = {
  "": "",
  surface: "bg-surface",
  raised: "bg-raised",
  charcoal: "bg-charcoal text-white",
  brand: "bg-brand text-white",
};

const PAD_CLASS: Record<string, string> = {
  "": "",
  sm: "py-6",
  md: "py-12",
  lg: "py-20",
};

export default function DesignRenderer({
  design,
  data,
}: {
  design?: PageDesign | null;
  data: BlockData;
}) {
  const sections =
    design && design.status === "published" && design.sections.length > 0
      ? design.sections
      : DEFAULT_SECTIONS;

  return (
    <main>
      {/* One row of twelve. A band with span 12 fills it; two sixes share it.
          Below `lg` the grid is a single column, so everything stacks in the
          order it was drawn — no second layout, nothing to forget. */}
      <div className="grid grid-cols-1 lg:grid-cols-12">
        {sections.map((section, i) => {
          if (section.hidden) return null;
          const Block = BLOCKS[section.type];
          // A type the frontend does not know is a band drawn by a newer
          // console. Skipped rather than crashed: the rest of the page is still
          // the restaurant's site.
          if (!Block) return null;

          const style = section.style ?? {};
          const wrapper = [
            SPAN_CLASS[section.span] ?? SPAN_CLASS[12],
            TONE_CLASS[style.tone ?? ""] ?? "",
            PAD_CLASS[style.padding ?? ""] ?? "",
            style.align === "center" ? "text-center" : "",
            style.rounded ? "overflow-hidden rounded-3xl" : "",
          ]
            .filter(Boolean)
            .join(" ");

          return (
            <div key={`${section.type}-${i}`} className={wrapper}>
              <Block d={data} section={section} />
            </div>
          );
        })}
      </div>
    </main>
  );
}

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
import type { Dict, Lang } from "@/lib/i18n/dictionaries";
import CanvasBlock, { type CanvasWidgets } from "./CanvasBlock";
// ⚠️ How a band looks lives beside how an element looks, and for the same
// reason: the console's live preview redraws both in the browser when a tone or
// a width changes, instead of re-pointing the iframe at the customer's real
// site. One copy, or the copy that is not rendering the page drifts.
import { bandClass, bandWidthVars } from "./canvasStyle";
import {
  AboutSection,
  BannersSection,
  BannerSection,
  CategoriesSection,
  CtaSection,
  GallerySection,
  HeroSection,
  ImageTextSection,
  PerksSection,
  RichTextSection,
} from "./SchemaBlocks";
import PreviewGate from "./PreviewGate";
import DesignPopup from "./DesignPopup";
import {
  AboutBlock,
  CategoriesBlock,
  CtaBlock,
  HoursAddressBlock,
  MenuGridBlock,
  SearchBlock,
  PerksBlock,
  ReviewsBlock,
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
  // ⚠️ Under the hero and above the categories, which is where a promotional strip
  // belongs: the first thing after the site's face, and before the guest starts choosing.
  // It draws nothing when the restaurant has no banners, so every existing site is
  // unchanged until somebody adds one.
  { type: "banners", variant: "carousel", span: 12 },
  // ⚠️ Above the perks and the categories, because it is the shortcut past both.
  // Half the guests who land here already know what they want, and their path
  // was otherwise: scroll the hero, find the categories, guess which one holds
  // lag'mon. It draws nothing on a site with no menu.
  { type: "search", variant: "bar", span: 12 },
  { type: "perks", variant: "cards", span: 12 },
  { type: "categories", variant: "tiles", span: 12 },
  {
    type: "menu-grid",
    variant: "cards",
    span: 12,
    binding: { popularOnly: true, limit: 8 },
  },
  // ⚠️ Below the menu and above the address: social proof belongs after the
  // guest has seen what is on offer and before they decide to come. It draws
  // nothing until the restaurant switches reviews on **and** publishes some, so
  // every existing site is unchanged.
  { type: "reviews", variant: "cards", span: 12 },
  { type: "hours-address", variant: "map", span: 12 },
];

// ⚠️ Sections that read their own settings take precedence over the original
// fixed bands of the same name. That is the migration: `hero` used to render the
// restaurant's name and cover with no way to change either, and it still does when
// no settings are set — the new component falls back to exactly that data. So an
// existing design keeps rendering what it rendered, and a settings panel now has
// something to change.
const BLOCKS = {
  hero: HeroSection,
  "rich-text": RichTextSection,
  "image-text": ImageTextSection,
  banner: BannerSection,
  banners: BannersSection,
  perks: PerksSection,
  categories: CategoriesSection,
  "menu-grid": MenuGridBlock,
  search: SearchBlock,
  "hours-address": HoursAddressBlock,
  reviews: ReviewsBlock,
  about: AboutSection,
  gallery: GallerySection,
  cta: CtaSection,
} as const;





export default function DesignRenderer({
  design,
  data,
  lang,
  t,
  preview,
}: {
  design?: PageDesign | null;
  data: BlockData;
  /** Needed by the freely drawn bands, whose text is typed per language. */
  lang?: Lang;
  t?: Dict;
  /** ⚠️ True only when this render came from a console preview token. It mounts
   *  the geometry bridge, which is the only editor-related code that ever reaches
   *  a restaurant's site — and never on a guest's page. */
  preview?: boolean;
}) {
  const sections =
    design && design.status === "published" && design.sections.length > 0
      ? design.sections
      : DEFAULT_SECTIONS;

  // ⚠️ The popup is pulled out of the flow entirely. It is drawn as a band in the
  // editor — that is how somebody composes one — but a band rendered in place
  // would be a rectangle sitting in the middle of the page instead of a dialog
  // over it.
  const popup = sections.find((s) => s.type === "popup" && !s.hidden);
  const bands = sections.filter((s) => s.type !== "popup");

  return (
    // ⚠️ A backstop for the same failure the canvas clips at source: any band
    // whose contents run wider than the page would otherwise widen the
    // document and give the whole site a horizontal scrollbar. `clip`, never
    // `hidden` — `hidden` would make this a scroll container and break the
    // sticky header above it.
    <main className="overflow-x-clip">
      {/* ⚠️ The designer's own corrections, and the one free-form value in the
          whole document. It is refused outright by the backend if it contains
          anything that could close this element (see sanitizeCSS), and it can only
          ever be written by the console — a tenant owner cannot reach the field. */}
      {preview && <PreviewGate />}
      {design?.customCss && (
        <style dangerouslySetInnerHTML={{ __html: design.customCss }} />
      )}
      {popup && lang && t && (
        <DesignPopup section={popup} lang={lang} closeLabel={t.common.close} />
      )}
      {/* One row of twelve. A band with span 12 fills it; two sixes share it.
          Below `lg` the grid is a single column, so everything stacks in the
          order it was drawn — no second layout, nothing to forget. */}
      <div className="grid grid-cols-1 lg:grid-cols-12">
        {bands.map((section, i) => {
          if (section.hidden) return null;
          // A freely drawn band has no fixed block: it is whatever was composed
          // inside it. Handled before the lookup so `canvas` does not need a
          // no-op entry in BLOCKS.
          if (section.type === "canvas") {
            if (!lang || !t) return null;
            return (
              <div key={`canvas-${i}`} className="lg:col-span-12">
                <CanvasBlock
                  canvas={section.canvas}
                  bandIndex={sections.indexOf(section)}
                  lang={lang}
                  widgets={canvasWidgets(data, section)}
                />
              </div>
            );
          }
          const Block = BLOCKS[section.type as keyof typeof BLOCKS];
          // A type the frontend does not know is a band drawn by a newer
          // console. Skipped rather than crashed: the rest of the page is still
          // the restaurant's site.
          if (!Block) return null;

          const style = section.style ?? {};

          return (
            <div
              key={`${section.type}-${i}`}
              // ⚠️ **Marked like a drawn band, although nothing inside it can be
              // dragged.** The mark is what lets the console's preview answer
              // "which band did I just click on" — without it a hero was inert
              // in a preview whose whole promise is that the page is the
              // navigation — and it is what the browser-side patch finds when a
              // tone or a width changes.
              data-keel-band={sections.indexOf(section)}
              className={bandClass(section)}
              style={bandWidthVars(style.width)}
            >
              <Block d={data} section={section} />
            </div>
          );
        })}
      </div>
    </main>
  );
}

// canvasWidgets hands the functional blocks to a freely drawn band.
//
// ⚠️ The widgets are the existing blocks, unchanged. A canvas positions and sizes
// them; what they render — real dish names, real prices, translations, options,
// a working cart button — stays the code that already does that correctly. An
// editor that made a designer rebuild a menu out of text boxes would produce a
// page that looks right and cannot take an order.
function canvasWidgets(data: BlockData, section: DesignSection): CanvasWidgets {
  const inner = { ...section, span: 12 } as DesignSection;
  return {
    "widget-menu": <MenuGridBlock d={data} section={inner} />,
    "widget-categories": <CategoriesBlock d={data} section={inner} />,
    "widget-hours": <HoursAddressBlock d={data} section={inner} />,
    // The map lives inside the hours band today; a canvas gets the same one
    // rather than a second map component to keep in step with it.
    "widget-map": <HoursAddressBlock d={data} section={{ ...inner, variant: "map" }} />,
    "widget-cart": <CtaBlock d={data} section={{ ...inner, variant: "buttons" }} />,
  };
}

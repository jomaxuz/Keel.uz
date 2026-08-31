// One glyph per section of the manual.
//
// ⚠️ **Objects, not documents** — the same rule the panel's own navigation
// follows. Thirteen cards in a grid, each headed by a clipboard-ish rectangle,
// is thirteen cards nobody can tell apart at a glance, and the icon then costs
// attention without paying any back. A shelf, a printer, a scooter, a pair of
// people: each one is the thing the section is about.
//
// ⚠️ **Hand-drawn inline**, for the reason `Icons.tsx` next door is: this is a
// marketing site whose job is to open fast on Uzbek mobile data, and pulling in
// an icon package to draw thirteen glyphs ships hundreds of kilobytes so one of
// them can be a printer.
//
// ⚠️ Stroked in `currentColor` with no fill, so one set serves the light card,
// the dark theme and the accent tint — and `aria-hidden`, because the section
// title sits right beside it and a screen reader announcing "box, Store" reads
// the same thing twice.

import type { SectionId } from "@/lib/help/types";

const PATHS: Record<SectionId, React.ReactNode> = {
  // A flag: the beginning of something, not a rocket — nothing here is a launch.
  start: (
    <>
      <path d="M5 21V4" />
      <path d="M5 4h11l-2 3.5L16 11H5" />
    </>
  ),
  // A browser window — what the guest opens.
  site: (
    <>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M3 9h18M7 6.5h.01M10 6.5h.01" />
    </>
  ),
  // An open book: the menu is a thing that is read.
  menu: (
    <>
      <path d="M12 6.5C10.5 5 8.5 4.5 4 5v13c4.5-.5 6.5 0 8 1.5 1.5-1.5 3.5-2 8-1.5V5c-4.5-.5-6.5 0-8 1.5Z" />
      <path d="M12 6.5v13" />
    </>
  ),
  // A docket with a torn edge — the order as it reaches the kitchen.
  orders: (
    <>
      <path d="M6 3h12v16.5l-2-1.2-2 1.2-2-1.2-2 1.2-2-1.2-2 1.2Z" />
      <path d="M9.5 8h5M9.5 12h5" />
    </>
  ),
  // A pin with a route into it, rather than a scooter: the section is about
  // where you deliver, and only then about who takes it.
  delivery: (
    <>
      <path d="M17 10c0 4-5 9-5 9s-5-5-5-9a5 5 0 0 1 10 0Z" />
      <circle cx="12" cy="10" r="1.8" />
    </>
  ),
  // A counter screen on a stand.
  till: (
    <>
      <rect x="3" y="4" width="18" height="11" rx="1.8" />
      <path d="M8 20h8M12 15v5" />
    </>
  ),
  printers: (
    <>
      <path d="M7 9V4h10v5" />
      <rect x="3" y="9" width="18" height="7" rx="1.6" />
      <path d="M7 14h10v6H7z" />
    </>
  ),
  // Stacked boxes: a shelf, not a warehouse building.
  stock: (
    <>
      <rect x="3" y="12" width="8" height="8" rx="1" />
      <rect x="13" y="12" width="8" height="8" rx="1" />
      <rect x="8" y="4" width="8" height="8" rx="1" />
    </>
  ),
  team: (
    <>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M3 20c0-3.3 2.7-5.5 6-5.5s6 2.2 6 5.5" />
      <path d="M16 5.5a3.2 3.2 0 0 1 0 6M17.5 14.8c2.1.6 3.5 2.4 3.5 5.2" />
    </>
  ),
  // A card with a person on it — the customer as a record, which is what this
  // section is: the database, not the people in the room.
  customers: (
    <>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <circle cx="9" cy="11" r="2.2" />
      <path d="M5.8 16.2c.5-1.5 1.7-2.3 3.2-2.3s2.7.8 3.2 2.3M15 10h3.5M15 13.5h3.5" />
    </>
  ),
  // A plug: something outside the product, wired into it.
  integrations: (
    <>
      <path d="M9 3v5M15 3v5" />
      <path d="M6 8h12v3a6 6 0 0 1-12 0Z" />
      <path d="M12 17v4" />
    </>
  ),
  reports: (
    <>
      <path d="M4 20V10M10 20V4M16 20v-7M22 20H2" />
    </>
  ),
  settings: (
    <>
      <path d="M4 7h11M19 7h1M4 17h5M13 17h7" />
      <circle cx="17" cy="7" r="2.2" />
      <circle cx="11" cy="17" r="2.2" />
    </>
  ),
};

export default function SectionIcon({
  id,
  className = "h-5 w-5",
}: {
  id: SectionId;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {PATHS[id]}
    </svg>
  );
}

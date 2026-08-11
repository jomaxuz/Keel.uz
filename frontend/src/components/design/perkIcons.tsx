// The perk strip's drawings, in one place.
//
// Three call sites need the same set: the console's perk band, the home page
// strip a restaurant edits from its own panel, and the panel's icon picker. A
// second copy would drift, and the way it would show up is an owner choosing an
// icon that renders as something else on the site — which reads as the site
// being broken rather than as two lists disagreeing.
//
// ⚠️ **Names, never paths.** Neither the console nor the owner writes SVG: the
// stored value is a key into this map, so an unknown one falls back to a star
// instead of putting markup on the page.

export const PERK_PATHS: Record<string, string> = {
  star: "M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 18l-5.8 3 1.1-6.5L2.6 9.8l6.5-.9L12 3Z",
  clock: "M12 7v5l3 2M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z",
  truck: "M3 7h11v9H3zM14 10h4l3 3v3h-7zM7 19a2 2 0 1 0 0-4 2 2 0 0 0 0 4Zm10 0a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
  leaf: "M20 4C10 4 4 9 4 17v3M20 4c0 8-5 12-12 12",
  fire: "M12 22c4 0 7-2.7 7-6.5 0-4.5-4.5-6.5-4-11.5-3 1.5-5 4-5 7 0-1.5-1-2.5-2-3-.7 1.5-3 3.4-3 7.5C5 19.3 8 22 12 22Z",
  check: "m5 13 4 4L19 7",
  heart: "M12 21s-8-4.8-8-10a4.5 4.5 0 0 1 8-2.8A4.5 4.5 0 0 1 20 11c0 5.2-8 10-8 10Z",
  chef: "M7 21h10M6 17h12v-2a6 6 0 0 0-12 0v2Z",
  phone:
    "M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8.1 9.9a16 16 0 0 0 6 6l1.3-1.2a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z",
  pin: "M12 22s7-5.6 7-12a7 7 0 1 0-14 0c0 6.4 7 12 7 12ZM12 11a2 2 0 1 0 0-4 2 2 0 0 0 0 4Z",
  card: "M3 7h18v10H3V7Zm0 4h18M7 15h3",
};

/** The three the built-in copy has always been drawn with, in its order. */
export const DEFAULT_PERK_ICONS = ["truck", "leaf", "card"];

/** The picker's order. Kept explicit so the panel does not depend on object
 *  key order, and so a new drawing can be added without moving the ones an
 *  owner has already learned the position of. */
export const PERK_ICON_NAMES = [
  "truck",
  "leaf",
  "check",
  "clock",
  "star",
  "fire",
  "heart",
  "chef",
  "card",
  "phone",
  "pin",
] as const;

export function PerkIcon({
  name,
  className = "h-7 w-7 text-brand",
}: {
  name: string;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      <path d={PERK_PATHS[name] ?? PERK_PATHS.star} />
    </svg>
  );
}

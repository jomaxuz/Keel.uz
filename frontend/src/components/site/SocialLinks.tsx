// The restaurant's social accounts, as icons.
//
// ⚠️ Inline SVG rather than an icon font or an emoji: the same reason the flags are
// (Windows has no glyphs for these at all), and a font for three shapes is a request on
// every page for three shapes.
//
// Only what is filled in is drawn. A greyed-out Facebook logo on a restaurant with no
// Facebook page says "we are on Facebook and neglecting it", which is worse than saying
// nothing.

const PATHS: Record<string, string> = {
  instagram:
    "M12 2.2c3.2 0 3.6 0 4.9.1 1.2.1 1.8.2 2.2.4.6.2 1 .5 1.4 1 .4.4.7.8 1 1.4.2.4.4 1 .4 2.2.1 1.3.1 1.7.1 4.9s0 3.6-.1 4.9c-.1 1.2-.2 1.8-.4 2.2-.2.6-.5 1-1 1.4-.4.4-.8.7-1.4 1-.4.2-1 .4-2.2.4-1.3.1-1.7.1-4.9.1s-3.6 0-4.9-.1c-1.2-.1-1.8-.2-2.2-.4-.6-.2-1-.5-1.4-1-.4-.4-.7-.8-1-1.4-.2-.4-.4-1-.4-2.2C2.2 15.6 2.2 15.2 2.2 12s0-3.6.1-4.9c.1-1.2.2-1.8.4-2.2.2-.6.5-1 1-1.4.4-.4.8-.7 1.4-1 .4-.2 1-.4 2.2-.4C8.4 2.2 8.8 2.2 12 2.2Zm0 5.1a4.7 4.7 0 1 0 0 9.4 4.7 4.7 0 0 0 0-9.4Zm0 7.7a3 3 0 1 1 0-6 3 3 0 0 1 0 6Zm6-7.9a1.1 1.1 0 1 1-2.2 0 1.1 1.1 0 0 1 2.2 0Z",
  telegram:
    "M21.9 4.3 19 19.2c-.2 1-.8 1.2-1.6.8l-4.4-3.3-2.1 2c-.2.3-.5.4-.8.4l.3-4.4 8.1-7.3c.3-.3 0-.5-.5-.2L8 12.1l-4.3-1.3c-.9-.3-.9-.9.2-1.3l16.5-6.4c.8-.3 1.5.2 1.5 1.2Z",
  facebook:
    "M13.5 21v-8h2.7l.4-3.1h-3.1V7.9c0-.9.3-1.5 1.6-1.5h1.6V3.6c-.3 0-1.3-.1-2.4-.1-2.4 0-4 1.4-4 4.1v2.3H7.6V13h2.3v8h3.6Z",
};

const LABEL: Record<string, string> = {
  instagram: "Instagram",
  telegram: "Telegram",
  facebook: "Facebook",
};

// ⚠️ Each network's own colour, and these are the only hard-coded colours on the site.
//
// Everywhere else a colour is a design token, so a restaurant's accent and dark mode both
// keep working. These cannot be: Telegram's blue and Instagram's gradient are how somebody
// finds the icon without reading it, and re-tinting them to the restaurant's accent turns
// three recognisable marks into three identical shapes. They are also the same in both
// themes for the same reason — a brand mark that changes colour at night is a different
// mark.
const BRAND: Record<string, string> = {
  telegram: "#229ED9",
  facebook: "#1877F2",
  // Instagram is a gradient, not a colour; the middle of it reads correctly at 20px and
  // needs no extra element to draw.
  instagram: "#E1306C",
};

export default function SocialLinks({
  socials,
  className = "",
  size = "md",
}: {
  socials?: { instagram?: string; telegram?: string; facebook?: string };
  className?: string;
  /** `sm` is the footer, where the row sits under a column of links. */
  size?: "sm" | "md";
}) {
  const entries = (["instagram", "telegram", "facebook"] as const)
    .map((key) => ({ key, href: socials?.[key] }))
    .filter((x) => !!x.href);

  if (entries.length === 0) return null;

  return (
    <div className={`flex flex-wrap items-center gap-2.5 ${className}`}>
      {entries.map(({ key, href }) => (
        <a
          key={key}
          href={href}
          target="_blank"
          rel="noreferrer"
          // The name stays in the label rather than beside the icon: three logos read
          // faster than three words, and a screen reader still hears which is which.
          aria-label={LABEL[key]}
          title={LABEL[key]}
          style={{ color: BRAND[key] }}
          className={`flex items-center justify-center rounded-full border border-line bg-surface transition-transform hover:scale-105 ${
            size === "sm" ? "h-9 w-9" : "h-10 w-10"
          }`}
        >
          <svg
            viewBox="0 0 24 24"
            fill="currentColor"
            className={size === "sm" ? "h-4.5 w-4.5" : "h-5 w-5"}
            aria-hidden
          >
            <path d={PATHS[key]} />
          </svg>
        </a>
      ))}
    </div>
  );
}

// The restaurant's social accounts, as icons.
//
// ⚠️ **`react-icons`, imported one icon at a time.** The package is a few thousand icons and
// importing from the top level pulls the index of all of them; `react-icons/si` and
// `react-icons/fa6` are separate entry points, and only the components actually named end up
// in the bundle. That distinction is the whole cost question — it is the difference between
// three shapes and a megabyte.
//
// ⚠️ **Each in its own colour, and these are the only hard-coded colours on the site.**
// Everywhere else a colour is a design token so the restaurant's accent and dark mode both
// keep working. These cannot be: Telegram's blue and Instagram's pink are how somebody finds
// the icon without reading it, and re-tinting them to the accent turns three recognisable
// marks into three identical shapes. They stay the same in both themes for the same reason —
// a brand mark that changes colour at night is a different mark.
//
// Only what is filled in is drawn. A greyed-out Facebook logo on a restaurant with no
// Facebook page says "we are there and neglecting it", which is worse than saying nothing.

import { FaFacebook, FaInstagram, FaTelegram } from "react-icons/fa6";

const NETWORKS = [
  { key: "instagram", label: "Instagram", Icon: FaInstagram, color: "#E1306C" },
  { key: "telegram", label: "Telegram", Icon: FaTelegram, color: "#229ED9" },
  { key: "facebook", label: "Facebook", Icon: FaFacebook, color: "#1877F2" },
] as const;

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
  const entries = NETWORKS.map((n) => ({ ...n, href: socials?.[n.key] })).filter(
    (n) => !!n.href,
  );
  if (entries.length === 0) return null;

  return (
    <div className={`flex flex-wrap items-center gap-2.5 ${className}`}>
      {entries.map(({ key, label, Icon, color, href }) => (
        <a
          key={key}
          href={href}
          target="_blank"
          rel="noreferrer"
          // The name lives in the label rather than beside the icon: three logos read faster
          // than three words, and a screen reader still hears which is which.
          aria-label={label}
          title={label}
          style={{ color }}
          className={`flex items-center justify-center rounded-full border border-line bg-surface transition-transform hover:scale-110 ${
            size === "sm" ? "h-9 w-9" : "h-10 w-10"
          }`}
        >
          <Icon className={size === "sm" ? "h-5 w-5" : "h-[22px] w-[22px]"} />
        </a>
      ))}
    </div>
  );
}

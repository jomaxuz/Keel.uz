// The feature icons.
//
// ⚠️ **Hand-drawn inline, not a library**, for the reason the chart palette and
// the Keel mark are copied rather than imported: this is a marketing page whose
// whole job is to load fast on a phone on Uzbek mobile data, and pulling an
// icon pack in to draw twelve glyphs ships a few hundred kilobytes so that one
// of them can be a shopping bag. Twelve paths cost nothing and cannot break on
// a version bump.
//
// ⚠️ **Every one is stroked in `currentColor` with no fill**, so a single set
// works on the light card, on the dark hull panel and inside the accent tint —
// and so the dark theme needs no second copy. A filled icon here would have to
// be redrawn for each surface, which is how an icon set drifts into three
// slightly different shopping bags.
//
// ⚠️ They are decorative and every one carries `aria-hidden`: the name beside
// the icon is the label, and a screen reader announcing "shopping bag,
// Buyurtma va yetkazish" reads the same thing twice.

type IconProps = { className?: string };

/** The shared frame. 24×24, 1.6 stroke — thin enough to sit beside body text
 *  without shouting, thick enough to survive at 20px on a phone. */
function Svg({
  className = "h-5 w-5",
  children,
}: IconProps & { children: React.ReactNode }) {
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
      {children}
    </svg>
  );
}

export function IconSite(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="2.5" y="4" width="19" height="16" rx="2.5" />
      <path d="M2.5 9h19M6 6.5h.01M8.5 6.5h.01M11 6.5h.01" />
    </Svg>
  );
}

export function IconDelivery(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M2.5 7.5h10v8h-10z" />
      <path d="M12.5 10.5h4l3 3v2h-7z" />
      <circle cx="6.5" cy="18" r="1.8" />
      <circle cx="16.5" cy="18" r="1.8" />
    </Svg>
  );
}

export function IconPanel(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="2.5" y="3.5" width="19" height="17" rx="2.5" />
      <path d="M8.5 3.5v17M12 15v2.5M15 11.5v6M18 8v9.5" />
    </Svg>
  );
}

export function IconTelegram(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M21 4.5 2.8 11.2l5 1.6 1.7 5.2 2.6-2.9 4.6 3.3z" />
      <path d="M7.8 12.8 21 4.5l-9.5 9.6" />
    </Svg>
  );
}

export function IconQr(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="3" y="3" width="7" height="7" rx="1.5" />
      <rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" />
      <path d="M14 14h3v3h-3zM20 14h1M14 20h3M20 17.5v3.5" />
    </Svg>
  );
}

export function IconPayment(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="2.5" y="5" width="19" height="14" rx="2.5" />
      <path d="M2.5 10h19M6 15h3" />
    </Svg>
  );
}

export function IconTill(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="3" y="9" width="18" height="11" rx="2" />
      <path d="M6 9V5.5A1.5 1.5 0 0 1 7.5 4h9A1.5 1.5 0 0 1 18 5.5V9" />
      <path d="M9.5 14h5" />
    </Svg>
  );
}

export function IconKitchen(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M4 10.5h16v6a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2z" />
      <path d="M3 10.5h18" />
      <path d="M8.5 7V4.5M12 7V4M15.5 7V4.5" />
    </Svg>
  );
}

export function IconCrm(p: IconProps) {
  return (
    <Svg {...p}>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M3.5 19.5c0-3 2.5-5 5.5-5s5.5 2 5.5 5" />
      <path d="M16.5 6.5a3 3 0 0 1 0 5.6M18.5 19.5c0-2-.6-3.4-1.6-4.4" />
    </Svg>
  );
}

export function IconReports(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M3.5 20h17" />
      <path d="M6 20V13M10.5 20V8.5M15 20v-4.5M19.5 20V5" />
    </Svg>
  );
}

export function IconCall(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M5 3.5h3l1.5 4-2 1.5a12 12 0 0 0 5.5 5.5l1.5-2 4 1.5v3a2 2 0 0 1-2.2 2C9.4 18.4 5.6 14.6 3 7.7A2 2 0 0 1 5 3.5z" />
    </Svg>
  );
}

export function IconStaff(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="3" y="5.5" width="18" height="15" rx="2.5" />
      <path d="M3 10h18M8 3.5v4M16 3.5v4" />
      <path d="M8 14h2M14 14h2M8 17h2" />
    </Svg>
  );
}

export function IconStock(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M3.5 8 12 3.5 20.5 8v8L12 20.5 3.5 16z" />
      <path d="M3.5 8 12 12.5 20.5 8M12 12.5v8" />
    </Svg>
  );
}

export function IconFiscal(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M6 3.5h12v17l-2-1.4-2 1.4-2-1.4-2 1.4-2-1.4-2 1.4z" />
      <path d="M9 8h6M9 11.5h6M9 15h3" />
    </Svg>
  );
}

export function IconLock(p: IconProps) {
  return (
    <Svg {...p}>
      <rect x="4.5" y="10" width="15" height="10" rx="2.5" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3M12 14v2" />
    </Svg>
  );
}

export function IconOffline(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M5 12.5a7 7 0 0 1 10.5-6M19 11.5a7 7 0 0 1-9.5 7.8" />
      <path d="M3 3l18 18" />
      <circle cx="12" cy="12" r="1.4" />
    </Svg>
  );
}

export function IconPrinter(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M7 8V3.5h10V8" />
      <rect x="3" y="8" width="18" height="8" rx="2" />
      <path d="M7 13h10v7.5H7z" />
    </Svg>
  );
}

export function IconSplit(p: IconProps) {
  return (
    <Svg {...p}>
      <path d="M4 20V9.5a2 2 0 0 1 2-2h4M4 20h6" />
      <path d="M20 20V9.5a2 2 0 0 0-2-2h-4M20 20h-6" />
      <path d="M12 3.5v6M12 6.5 9.5 9M12 6.5 14.5 9" />
    </Svg>
  );
}

export function IconLang(p: IconProps) {
  return (
    <Svg {...p}>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M3.5 12h17" />
      <path d="M12 3.5c2.2 2.4 3.3 5.3 3.3 8.5s-1.1 6.1-3.3 8.5c-2.2-2.4-3.3-5.3-3.3-8.5S9.8 5.9 12 3.5z" />
    </Svg>
  );
}

/** Index for the feature grid: the dictionary carries words, this carries the
 *  glyph, and the two are joined by position.
 *
 *  ⚠️ **By position, deliberately not by a key stored in the dictionary.** The
 *  translations are three files a translator edits; an icon name living in
 *  them is an icon name somebody eventually translates. If the list grows the
 *  extra items fall through to a neutral dot rather than crashing — a missing
 *  glyph must never be a blank marketing page. */
export const FEATURE_ICONS = [
  IconSite,
  IconDelivery,
  IconPanel,
  IconTelegram,
  IconQr,
  IconPayment,
  IconTill,
  IconCrm,
  IconReports,
  IconCall,
  IconStaff,
  IconLang,
];

export const TILL_ICONS = [
  IconTill,
  IconSplit,
  IconFiscal,
  IconPrinter,
  IconLock,
  IconOffline,
  IconStock,
  IconReports,
];

/** The fallback, for a list that outgrew its icons. */
export function IconDot(p: IconProps) {
  return (
    <Svg {...p}>
      <circle cx="12" cy="12" r="7" />
    </Svg>
  );
}

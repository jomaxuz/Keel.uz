// What Keel already talks to.
//
// The first question a restaurant asks is whether its own till is on the list,
// and the second is what happens if it is not. So this section answers both on
// one screen: the systems grouped by the job they do, and — as the last card
// rather than a footnote — an offer to write the one that is missing.
//
// **Icons are drawn here, not fetched.** A restaurant's till vendor has a logo
// we do not have a licence to use, and hotlinking somebody's brand asset is
// both a legal question and a broken image waiting to happen. Each group gets
// one glyph that says what *kind* of thing it is; the names carry the
// identity, which is what a reader is actually scanning for.
//
// Unfinished integrations are listed as "soon" rather than hidden: an honest
// "soon" keeps the conversation going where an absence ends it.

import type { getT } from "@/lib/i18n/server";

type Dict = Awaited<ReturnType<typeof getT>>;

/** One row of the section: a job, and the systems that do it. */
const GROUPS = [
  {
    key: "pos",
    icon: TillIcon,
    items: [
      { name: "iiko", ready: true },
      { name: "Syrve", ready: true },
      { name: "Poster", ready: true },
      { name: "Clopos", ready: true },
      { name: "r_keeper", ready: true },
      { name: "Jowi", ready: false },
      { name: "Paloma", ready: false },
      { name: "AliPOS", ready: false },
    ],
  },
  {
    key: "pay",
    icon: CardIcon,
    items: [
      { name: "Payme", ready: true },
      { name: "Click", ready: true },
      { name: "Uzum", ready: true },
      { name: "ATMOS", ready: true },
    ],
  },
  {
    // The restaurant's own bot, not ours. Listed among the integrations rather
    // than among the features because that is the question being asked here:
    // an owner who already runs a bot wants to know whether Keel uses theirs.
    key: "telegram",
    icon: TelegramIcon,
    items: [
      { name: "Telegram Bot API", ready: true },
      { name: "Mini App", ready: true },
    ],
  },
  {
    key: "sms",
    icon: SmsIcon,
    items: [
      { name: "Eskiz", ready: true },
      { name: "Play Mobile", ready: true },
      { name: "getsms.uz", ready: true },
      { name: "OneSignal", ready: true },
    ],
  },
  {
    key: "map",
    icon: MapIcon,
    items: [
      { name: "2GIS", ready: true },
      { name: "Yandex Maps", ready: true },
      { name: "Google Maps", ready: true },
    ],
  },
  {
    key: "phone",
    icon: PhoneIcon,
    items: [{ name: "onlinePBX", ready: true }],
  },
  {
    key: "delivery",
    icon: ScooterIcon,
    items: [
      { name: "Yandex Delivery", ready: true },
      { name: "Millennium", ready: false },
    ],
  },
] as const;

export default function Integrations({ t }: { t: Dict }) {
  return (
    <div className="space-y-px overflow-hidden rounded-3xl border border-line bg-line">
      {GROUPS.map((g) => {
        const group = t.integrations.groups[g.key];
        const Icon = g.icon;
        return (
          <div key={g.key} className="bg-surface p-6 sm:flex sm:gap-6">
            <div className="sm:w-64 sm:shrink-0">
              <span className="grid h-9 w-9 place-items-center rounded-xl bg-signal-500/15 text-signal-600 dark:text-signal-400">
                <Icon />
              </span>
              <p className="mt-3 font-display text-base font-semibold text-ink">
                {group.title}
              </p>
              <p className="mt-1 text-sm leading-relaxed text-ink-muted">
                {group.desc}
              </p>
            </div>
            <div className="mt-4 flex flex-wrap gap-2 sm:mt-0 sm:flex-1 sm:content-start">
              {g.items.map((it) => (
                <span
                  key={it.name}
                  className={`inline-flex h-fit items-center gap-2 rounded-xl border px-3.5 py-2 text-sm font-semibold ${
                    it.ready
                      ? "border-line-strong bg-surface text-ink"
                      : "border-dashed border-line text-ink-muted"
                  }`}
                >
                  {it.name}
                  {!it.ready && (
                    <span className="text-[11px] font-medium uppercase tracking-wide">
                      {t.integrations.soon}
                    </span>
                  )}
                </span>
              ))}
            </div>
          </div>
        );
      })}

      {/* The answer to "mine is not on the list", given the same weight as the
          list itself — because for the restaurant reading it, it is the more
          important half. */}
      <div className="bg-raised p-6 sm:flex sm:items-center sm:gap-6">
        <div className="sm:w-64 sm:shrink-0">
          <span className="grid h-9 w-9 place-items-center rounded-xl bg-signal-500/15 text-signal-600 dark:text-signal-400">
            <PlugIcon />
          </span>
          <p className="mt-3 font-display text-base font-semibold text-ink">
            {t.integrations.custom.title}
          </p>
        </div>
        <p className="mt-3 text-sm leading-relaxed text-ink-soft sm:mt-0 sm:flex-1">
          {t.integrations.custom.desc}
        </p>
      </div>
    </div>
  );
}

// ---- Icons ----
//
// One stroke weight, one 24-grid, `currentColor` throughout, so a group's glyph
// takes the accent from its container and nothing has to be recoloured by hand
// for dark mode.

function icon(children: React.ReactNode) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {children}
    </svg>
  );
}

function TillIcon() {
  return icon(
    <>
      <rect x="3" y="9" width="18" height="12" rx="2" />
      <path d="M7 9V5a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v4" />
      <path d="M7 14h4" />
    </>,
  );
}

function CardIcon() {
  return icon(
    <>
      <rect x="2.5" y="5" width="19" height="14" rx="2.5" />
      <path d="M2.5 10h19" />
      <path d="M6.5 15h3" />
    </>,
  );
}

function SmsIcon() {
  return icon(
    <>
      <path d="M21 12a8 8 0 0 1-8 8H7l-4 3v-6.5A8 8 0 0 1 11 4h2a8 8 0 0 1 8 8Z" />
      <path d="M9 12h.01M13 12h.01M17 12h.01" />
    </>,
  );
}

function MapIcon() {
  return icon(
    <>
      <path d="M12 21s7-6.3 7-11a7 7 0 1 0-14 0c0 4.7 7 11 7 11Z" />
      <circle cx="12" cy="10" r="2.6" />
    </>,
  );
}

function TelegramIcon() {
  // The paper plane, drawn on the same 24-grid as the rest. Filled rather than
  // stroked, because a one-stroke outline of this shape reads as an arrow.
  return (
    <svg aria-hidden viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor">
      <path d="M21.9 4.3 18.7 19c-.2 1-.9 1.3-1.8.8l-4.9-3.6-2.4 2.3c-.3.3-.5.5-1 .5l.3-5 9.1-8.2c.4-.4-.1-.6-.6-.2L6.2 12.7 1.4 11.2c-1-.3-1-1 .2-1.5l19-7.3c.9-.3 1.6.2 1.3 1.9Z" />
    </svg>
  );
}

function PhoneIcon() {
  return icon(
    <path d="M4 4h4l2 5-2.5 1.5a12 12 0 0 0 6 6L15 14l5 2v4a1 1 0 0 1-1.1 1A17 17 0 0 1 3 5.1 1 1 0 0 1 4 4Z" />,
  );
}

function ScooterIcon() {
  return icon(
    <>
      <circle cx="5.5" cy="17.5" r="2.5" />
      <circle cx="18.5" cy="17.5" r="2.5" />
      <path d="M8 17.5h8" />
      <path d="M18.5 17.5 16 6h-3" />
      <path d="M16 10h-6a4 4 0 0 0-4 4v3.5" />
    </>,
  );
}

function PlugIcon() {
  return icon(
    <>
      <path d="M9 3v6M15 3v6" />
      <path d="M6 9h12v3a6 6 0 0 1-12 0V9Z" />
      <path d="M12 18v3" />
    </>,
  );
}

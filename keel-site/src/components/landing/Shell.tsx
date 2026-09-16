import Link from "next/link";
import { Logo } from "@/components/Logo";
import Reveal from "@/components/landing/Reveal";
import { accent } from "@/lib/accent";
import type { getT } from "@/lib/i18n/server";
import { localePath } from "@/lib/i18n/url";
import type { Lang } from "@/lib/i18n/dict";
import { EMAIL, TELEGRAM } from "@/lib/links";
// ⚠️ The same record the offer and the privacy policy are written from. A
// footer with its own copy of the address is the copy that stops matching the
// documents — and the first reader to notice is the one comparing them.
import { COMPANY } from "@/lib/legal";

// ---- The shell every landing page is built from ----
//
// ⚠️ **Extracted because there is more than one landing page now.** The page
// was one file with thirteen sections, and a visitor arriving on it met the
// deepest technical block second — before knowing why any of it mattered. The
// till, the rival comparison and the integration list moved to /kassa, which
// means the section shell, the hull motif and the footer are shared rather
// than owned by the home page. Nothing about how they render changed.


/** One section shell so spacing, width and heading rhythm are decided once. */
export function Section({
  id,
  eyebrow,
  title,
  lead,
  tone,
  align,
  children,
}: {
  id?: string;
  eyebrow: string;
  title: string;
  lead?: string;
  tone?: "raised";
  align?: "left";
  children: React.ReactNode;
}) {
  // ⚠️ **Centred by default, and that is a rhythm decision rather than a taste
  // one.** Eight sections all opening with a left-aligned eyebrow, heading and
  // lead is the thing that made this page read as a document: nothing announces
  // that a new argument has started, so the eye keeps going at the same speed
  // and stops somewhere in the middle. A centred heading is a full stop.
  // Sections whose body is a two-column layout keep the left edge, because a
  // centred heading over a left-aligned column is neither.
  const centred = align !== "left";
  return (
    <section
      id={id}
      className={`scroll-mt-20 border-t border-line py-20 sm:py-24 ${
        tone === "raised" ? "bg-raised" : ""
      }`}
    >
      <div className="container-page">
        <Reveal className={centred ? "text-center" : ""}>
          <p className="eyebrow">{eyebrow}</p>
          <h2
            className={`h-display mt-3 max-w-2xl text-3xl sm:text-4xl ${
              centred ? "mx-auto" : ""
            }`}
          >
            {accent(title)}
          </h2>
          {lead && (
            <p className={`mt-4 max-w-2xl text-ink-soft ${centred ? "mx-auto" : ""}`}>
              {lead}
            </p>
          )}
        </Reveal>
        <Reveal className="mt-10">{children}</Reveal>
      </div>
    </section>
  );
}

/** The signature motif: the hull curve, oversized and low-contrast. Repeated
 *  in the closing panel so the page opens and closes on the same shape. */
export function HullBackdrop({ subtle = false }: { subtle?: boolean }) {
  return (
    <svg
      aria-hidden
      viewBox="0 0 1200 400"
      preserveAspectRatio="none"
      className={`pointer-events-none absolute inset-x-0 bottom-0 h-40 w-full ${
        subtle ? "opacity-[0.5]" : "opacity-80"
      }`}
    >
      <defs>
        <linearGradient id="hull" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="rgb(245 165 36)" stopOpacity="0" />
          <stop offset="50%" stopColor="rgb(245 165 36)" stopOpacity="0.55" />
          <stop offset="100%" stopColor="rgb(245 165 36)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[0, 26, 52].map((dy, i) => (
        <path
          key={dy}
          d={`M-40 ${120 + dy} C 260 ${360 + dy}, 940 ${360 + dy}, 1240 ${120 + dy}`}
          fill="none"
          stroke="url(#hull)"
          strokeWidth={i === 0 ? 2 : 1}
          opacity={i === 0 ? 1 : 0.45}
        />
      ))}
    </svg>
  );
}


export function TelegramMark() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor" aria-hidden>
      <path d="M21.9 4.3 18.7 19c-.2 1-.9 1.3-1.8.8l-4.9-3.6-2.4 2.3c-.3.3-.5.5-1 .5l.3-5 9.1-8.2c.4-.4-.1-.6-.6-.2L6.2 12.7 1.4 11.2c-1-.3-1-1 .2-1.5l19-7.3c.9-.3 1.6.2 1.3 1.9Z" />
    </svg>
  );
}

export function Check() {
  return (
    <svg viewBox="0 0 24 24" className="mt-0.5 h-4 w-4 shrink-0 text-signal-500" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}

export function Footer({ t, lang }: { t: Awaited<ReturnType<typeof getT>>; lang: Lang }) {
  return (
    <footer className="border-t border-line bg-raised">
      <div className="container-page grid gap-10 py-14 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <Logo />
          <p className="mt-3 max-w-xs text-sm text-ink-muted">{t.footer.tagline}</p>
        </div>
        <div>
          <p className="text-sm font-semibold text-ink">{t.footer.product}</p>
          <ul className="mt-3 space-y-2 text-sm text-ink-muted">
            {/* ⚠️ Rooted at "/", the way the header's links already are. The
                footer is on /status and the legal pages too, and a bare
                "#pricing" there scrolls nowhere and reads as a dead link —
                which is what these five were doing. */}
            <li><a href="/#product" className="hover:text-ink">{t.nav.product}</a></li>
            {/* ⚠️ Real pages now, not anchors. The till and the integration
                list left the home page — a "/#till" here would scroll to the
                top of a page that no longer has that section, which reads as a
                broken link rather than as a moved one. */}
            <li><Link href={localePath(lang, "/kassa")} className="hover:text-ink">{t.nav.till}</Link></li>
            <li><Link href={localePath(lang, "/kassa#integrations")} className="hover:text-ink">{t.nav.integrations}</Link></li>
            <li><a href="/#pricing" className="hover:text-ink">{t.nav.pricing}</a></li>
            <li><a href="/#calc" className="hover:text-ink">{t.nav.calc}</a></li>
            <li><a href="/#faq" className="hover:text-ink">{t.nav.faq}</a></li>
            {/* ⚠️ In the footer as well as the header, and that is not a
                duplicate. The header link is found by somebody browsing; this
                one is found by somebody who has scrolled to the bottom of a
                page because they did not find what they needed above it —
                which is the same person, one minute later and less patient. */}
            <li><Link href={localePath(lang, "/help")} className="hover:text-ink">{t.nav.help}</Link></li>
            {/* ⚠️ A real page, so a real Link with the locale prefix — a bare
                href drops it and sends a Russian visitor to the Uzbek page. */}
            <li><Link href={localePath(lang, "/download")} className="hover:text-ink">{t.download.eyebrow}</Link></li>
          </ul>
        </div>
        <div>
          <p className="text-sm font-semibold text-ink">{t.footer.company}</p>
          <ul className="mt-3 space-y-2 text-sm text-ink-muted">
            <li><a href="/#who" className="hover:text-ink">{t.nav.who}</a></li>
            <li><a href="/#cta" className="hover:text-ink">{t.nav.start}</a></li>
            {/* In the footer rather than the top nav: a status link somebody
                notices before anything is wrong is a link that suggests
                something might be. */}
            <li><Link href={localePath(lang, "/status")} className="hover:text-ink">{t.status.eyebrow}</Link></li>
            {/* ⚠️ In the footer, where every site keeps them — and reachable from every page,
                because an offer somebody has to search for is an offer they can say they never
                saw. */}
            <li><Link href={localePath(lang, "/public-offer")} className="hover:text-ink">{t.legal.offer}</Link></li>
            <li><Link href={localePath(lang, "/privacy-policy")} className="hover:text-ink">{t.legal.privacy}</Link></li>
          </ul>
        </div>
        <div>
          <p className="text-sm font-semibold text-ink">{t.footer.contact}</p>
          <ul className="mt-3 space-y-2 text-sm text-ink-muted">
            {/* ⚠️ **First, and tappable.** A phone number that is only printed
                is a number somebody has to retype while holding the phone they
                would have called from. `tel:` takes no spaces on some Android
                dialers, so the two forms are kept apart in `COMPANY`. */}
            <li><a href={`tel:${COMPANY.phone}`} className="hover:text-ink">{COMPANY.phoneText}</a></li>
            <li><a href={TELEGRAM} className="hover:text-ink">Telegram</a></li>
            <li><a href={`mailto:${EMAIL}`} className="hover:text-ink">{EMAIL}</a></li>
          </ul>
        </div>
      </div>
      <div className="border-t border-line">
        <div className="container-page space-y-2 py-6 text-xs text-ink-muted">
          {/* ⚠️ **Who we legally are, on every page.** It was only inside the
              offer and the privacy policy — two clicks away and inside a wall
              of text. A visitor deciding whether to hand a business over to us
              should not have to open a legal document to find out whether we
              are a company at all.

              ⚠️ It also decides a verification we cannot pass without it: Meta
              compares the legal name, the address and the phone on the site
              against the registration document, and a site that shows none of
              them is refused. Read off `COMPANY`, so the footer and the
              documents cannot drift apart. */}
          <p>
            {lang === "ru"
              ? COMPANY.nameRu
              : lang === "en"
                ? COMPANY.nameEn
                : COMPANY.nameUz}
            {" · "}STIR {COMPANY.inn}
            {" · "}
            {lang === "ru"
              ? COMPANY.addressRu
              : lang === "en"
                ? COMPANY.addressEn
                : COMPANY.addressUz}
            {" · "}
            <a href={`tel:${COMPANY.phone}`} className="hover:text-ink">
              {COMPANY.phoneText}
            </a>
            {" · "}
            <a href={`mailto:${COMPANY.email}`} className="hover:text-ink">
              {COMPANY.email}
            </a>
          </p>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <span>© {new Date().getFullYear()} Keel. {t.footer.rights}.</span>
            <span>keel.uz</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
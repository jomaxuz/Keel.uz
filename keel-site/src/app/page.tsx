import Link from "next/link";
import Header from "@/components/Header";
import Partners from "@/components/Partners";
import { KeelMark } from "@/components/Logo";
import {
  FEATURE_ICONS,
  IconDot,
  IconTill,
  IconOffline,
  IconPanel,
  IconReports,
  IconSite,
  IconTelegram,
  IconDelivery,
} from "@/components/Icons";
import { ChannelVisual } from "@/components/Visual3D";
import Reveal from "@/components/landing/Reveal";
import {
  Check,
  Footer,
  HullBackdrop,
  Section,
  TelegramMark,
} from "@/components/landing/Shell";
import Mockup from "@/components/landing/Mockup";
import { FloatChip } from "@/components/landing/Frame";
import Calculator from "@/components/landing/Calculator";
import Timeline from "@/components/landing/Timeline";
import { accent } from "@/lib/accent";
import { getLang, getT } from "@/lib/i18n/server";
import { localePath } from "@/lib/i18n/url";
import { getPartners } from "@/lib/partners";
import { TELEGRAM } from "@/lib/links";
import { shot } from "@/lib/shots";
import { cleanRefCode, REF_PARAM } from "@/lib/referral";

export default async function Home({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  // Set when this render came from `keel.uz/h/<kod>`, which redirects here.
  //
  // ⚠️ **Read from the address, not from the cookie.** The cookie is what tells
  // *us* the link was used, and it lives for a year — reading it here would put
  // a partner's name on the page of somebody who arrived on their own three
  // weeks later, and then quote a code that has nothing to do with them.
  const referral = cleanRefCode(
    (await searchParams)[REF_PARAM] as string | undefined,
  );
  const t = await getT();
  const lang = await getLang();
  // Read on the server so the strip is in the first paint: a marketing page
  // that pops its social proof in a second late has already been scrolled past.
  const partners = await getPartners();

  return (
    <>
      <Header />
      {/* ⚠️ **Named, above everything.** A leaflet from a register engineer is
          only worth following if the page it opens does not look like an
          advert somebody found. The line says who sent them and, because the
          signup is a Telegram conversation rather than a form, the code they
          should mention when they write. */}
      {referral && (
        <div className="border-b border-line bg-signal-500/[0.08]">
          <div className="container-page py-2.5 text-sm text-ink-soft">
            {t.referral.line}{" "}
            <span className="rounded-md border border-line bg-surface px-1.5 py-0.5 font-semibold text-ink">
              {referral}
            </span>
          </div>
        </div>
      )}

      {/* ---- Hero ---- */}
      <section className="hull-glow relative overflow-hidden">
        <HullBackdrop />
        <div className="container-page relative grid gap-14 pb-16 pt-20 sm:pb-20 sm:pt-28 lg:grid-cols-[1.05fr_.95fr] lg:items-center">
          <div className="animate-rise">
            <p className="eyebrow">{t.hero.eyebrow}</p>
            <h1 className="h-display mt-4 text-[2.6rem] leading-[1.05] sm:text-6xl">
              {accent(t.hero.title)}
            </h1>
            <p className="mt-6 max-w-xl text-lg leading-relaxed text-ink-soft">
              {t.hero.lead}
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <a href={TELEGRAM} className="btn-primary px-6 py-3.5 text-base">
                {t.hero.ctaPrimary}
              </a>
              {/* ⚠️ A page, not an anchor — the till moved to /kassa. The
                  label already said "see the till", which is now literally
                  where it goes. */}
              <Link
                href={localePath(lang, "/kassa")}
                className="btn-ghost px-6 py-3.5 text-base"
              >
                {t.hero.ctaSecondary}
              </Link>
            </div>
            <p className="mt-4 text-sm text-ink-muted">{t.hero.note}</p>

            <dl className="mt-12 grid max-w-lg grid-cols-3 gap-6 border-t border-line pt-8">
              {[
                [t.hero.stat1, t.hero.stat1v],
                [t.hero.stat2, t.hero.stat2v],
                [t.hero.stat3, t.hero.stat3v],
              ].map(([label, value]) => (
                <div key={label}>
                  <dt className="text-xs uppercase tracking-wider text-ink-muted">{label}</dt>
                  <dd className="h-display mt-1 text-xl sm:text-2xl">{value}</dd>
                </div>
              ))}
            </dl>
          </div>

          {/* ⚠️ **The till itself, and it is a photograph now.** The drawing of
              the monoblock stood here first, on the argument that the thing
              being sold is the machine on the counter. It answers "what am I
              buying" — and then the visitor scrolls past nine real screens and
              understands that the one picture placed where it mattered most was
              the one nobody had photographed. The machine is still drawn, in the
              section that talks about machines; the first image on the page is
              now the program running on it.

              ⚠️ Nothing sits behind it, and that was tried with the drawing: a
              shop-and-phone illustration at low opacity left a corner of an
              awning uncovered, which reads as something broken behind the panel
              rather than as texture. A partly occluded object is not decoration.

              ⚠️ `priority` because this is the page's largest paint. */}
          <div className="relative hidden lg:block">
            <Mockup
              src={shot(lang, "till")}
              alt={t.till.shotAlt}
              w={1600}
              h={1000}
              priority
              badge={<IconTill className="h-5 w-5" />}
              chip={
                <FloatChip
                  icon={<IconOffline className="h-4 w-4" />}
                  value={t.till.chipOffline}
                  label={t.till.chipOfflineNote}
                />
              }
              chipSide="left"
              // ⚠️ Wider than its column, and the section clips it. Fitted
              // inside the grid the shot is about 500px and the check on it is
              // a smear — the hero would be showing a screenshot rather than
              // showing a screen. Running it off the right edge buys 200px of
              // legibility and says the thing carries on past the fold, which
              // is true.
              className="animate-drift lg:w-[128%]"
            />
          </div>
        </div>
      </section>
      {/* ---- Why it costs less ---- */}
      {/* The actual pitch, and it is placed high because it is the argument a
          restaurant is weighing while they read anything else on this page.
          The honest caveat is part of it, not a footnote: an aggregator does
          bring customers we do not, and a comparison that hides that gets
          called out by the first owner who has run both. */}
      <Section
        id="compare"
        eyebrow={t.compare.eyebrow}
        title={t.compare.title}
        lead={t.compare.lead}
        tone="raised"
      >
        <div className="overflow-x-auto rounded-3xl border border-line bg-surface">
          <table className="w-full min-w-[640px] text-sm">
            <thead className="border-b border-line text-left text-xs uppercase tracking-wider text-ink-muted">
              <tr>
                <th className="px-5 py-4">{t.compare.thCase}</th>
                <th className="px-5 py-4 text-right">{t.compare.thRevenue}</th>
                <th className="px-5 py-4 text-right">{t.compare.thAgg}</th>
                <th className="px-5 py-4 text-right">{t.compare.thKeel}</th>
                <th className="px-5 py-4 text-right">{t.compare.thKeep}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {t.compare.rows.map((r) => (
                <tr key={r.c}>
                  <td className="whitespace-nowrap px-5 py-4 font-semibold text-ink">
                    {r.c}
                  </td>
                  <td className="whitespace-nowrap px-5 py-4 text-right tabular-nums text-ink-soft">
                    {r.r}
                  </td>
                  <td className="whitespace-nowrap px-5 py-4 text-right tabular-nums text-ink-muted line-through">
                    {r.a}
                  </td>
                  <td className="whitespace-nowrap px-5 py-4 text-right tabular-nums font-semibold text-ink">
                    {r.k}
                  </td>
                  {/* The number the whole page exists to put in front of
                      somebody. Given the accent, and nothing else on the row
                      competes with it. */}
                  <td className="whitespace-nowrap px-5 py-4 text-right">
                    <span className="h-display text-lg text-signal-600 dark:text-signal-400">
                      {r.s}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {/* ⚠️ Beside the honest caveat rather than above the table: the table
            is the argument and nothing should compete with it. The drawing says
            the same thing the paragraph does — a shop and a customer's phone
            with a line between them, and no aggregator in the middle — which is
            the one place on this page where an illustration is the argument
            rather than an ornament. */}
        <div className="mt-5 flex flex-wrap items-center gap-8">
          <p className="max-w-3xl flex-1 text-sm leading-relaxed text-ink-muted">
            {t.compare.honest}
          </p>
          <ChannelVisual decorative className="hidden h-32 shrink-0 lg:block" />
        </div>
      </Section>
      {/* ---- Who it's for ---- */}
      <Section id="who" eyebrow={t.who.eyebrow} title={t.who.title} lead={t.who.lead}>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {t.who.items.map((it) => (
            <div
              key={it.name}
              className="card transition hover:-translate-y-0.5 hover:border-line-strong"
            >
              <p className="font-display text-lg font-semibold text-ink">{it.name}</p>
              <p className="mt-2 text-sm leading-relaxed text-ink-muted">{it.desc}</p>
            </div>
          ))}
        </div>
      </Section>
      {/* ---- Features ---- */}
      <Section
        id="product"
        eyebrow={t.features.eyebrow}
        title={t.features.title}
        lead={t.features.lead}
        tone="raised"
      >
        <div className="grid gap-px overflow-hidden rounded-3xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
          {t.features.items.map((it, i) => {
            // ⚠️ Joined to the glyph **by position**, not by a key in the
            // dictionary: those three files are edited by translators, and an
            // icon name living among the words is an icon name somebody
            // eventually translates. A list that outgrows its icons falls
            // through to a neutral dot rather than crashing the page.
            const Icon = FEATURE_ICONS[i] ?? IconDot;
            return (
            <div key={it.name} className="bg-surface p-6">
              <span className="icon-badge">
                <Icon className="h-[1.35rem] w-[1.35rem]" />
              </span>
              <p className="mt-4 font-display text-base font-semibold text-ink">{it.name}</p>
              <p className="mt-2 text-sm leading-relaxed text-ink-muted">{it.desc}</p>
            </div>
            );
          })}
        </div>

        {/* ⚠️ **The panel, full width, directly under the list of what it
            does.** The grid above is nine claims in nine boxes; this is the one
            place a visitor sees what any of them looks like on a Tuesday
            evening. A feature list that never shows the product is a brochure.
            */}
        <div className="mt-16 grid items-center gap-10 lg:grid-cols-[1.35fr_1fr]">
          <Mockup
            src={shot(lang, "orders")}
            alt={t.shots.ordersAlt}
            w={1500}
            h={938}
            badge={<IconPanel className="h-5 w-5" />}
            chip={
              <FloatChip
                icon={<IconTelegram className="h-4 w-4" />}
                value={t.till.chipSound}
                label={t.till.chipSoundNote}
              />
            }
          />
          <div>
            <h3 className="h-display text-2xl">{t.shots.panelTitle}</h3>
            <p className="mt-3 leading-relaxed text-ink-muted">{t.shots.panelLead}</p>
          </div>
        </div>

        {/* The three doors the same menu opens onto. Shown together on purpose:
            the claim is that they are one thing, and three separate cards
            further apart would say the opposite. */}
        <div className="mt-16 text-center">
          <h3 className="h-display text-2xl sm:text-3xl">{accent(t.shots.channelsTitle)}</h3>
          <p className="mx-auto mt-3 max-w-2xl text-ink-soft">{t.shots.channelsLead}</p>
        </div>
        {/* ⚠️ **The site is not one of three equal columns.** Three across, the
            browser shot came out 349px wide and the menu inside it was a smear;
            the phones were fine, because a phone screenshot is narrow to begin
            with. So the desktop screen takes the wide half and the two phones
            share the other — which is also the true proportion of the thing:
            one site, and two ways to carry it. */}
        <div className="mt-10 grid items-center gap-10 lg:grid-cols-[1.5fr_1fr]">
          <div>
            <Mockup
              src={shot(lang, "site")}
              alt={t.shots.siteAlt}
              w={1500}
              h={938}
              kind="browser"
              badge={<IconSite className="h-5 w-5" />}
            />
            <p className="mt-5 text-center font-display text-base font-semibold text-ink">
              {t.shots.siteLabel}
            </p>
          </div>
          <div className="grid grid-cols-2 gap-6">
            {[
              {
                src: shot(lang, "miniapp"),
                alt: t.shots.miniAppAlt,
                label: t.shots.miniAppLabel,
                icon: <IconTelegram className="h-4 w-4" />,
              },
              {
                src: shot(lang, "courier"),
                alt: t.shots.courierAlt,
                label: t.shots.courierLabel,
                icon: <IconDelivery className="h-4 w-4" />,
              },
            ].map((c) => (
              <div key={c.label}>
                <Mockup
                  src={c.src}
                  alt={c.alt}
                  w={560}
                  h={694}
                  kind="phone"
                  badge={c.icon}
                />
                <p className="mt-4 text-center font-display text-sm font-semibold text-ink">
                  {c.label}
                </p>
              </div>
            ))}
          </div>
        </div>
      </Section>
      {/* ---- Pricing ---- */}
      <Section
        id="pricing"
        align="left"
        eyebrow={t.pricing.eyebrow}
        title={t.pricing.title}
        lead={t.pricing.lead}
      >
        <div className="grid gap-6 lg:grid-cols-[1.1fr_.9fr]">
          <div className="relative overflow-hidden rounded-3xl border border-line-strong bg-surface p-8 sm:p-10">
            <div className="pointer-events-none absolute -right-16 -top-16 h-52 w-52 rounded-full bg-signal-500/10 blur-2xl" />
            <div className="relative flex flex-wrap items-end gap-3">
              <span className="h-display text-6xl sm:text-7xl">800</span>
              <span className="pb-3 text-lg font-semibold text-ink-soft">
                {t.pricing.unit}
              </span>
              <span className="pb-3.5 text-sm text-ink-muted">/ {t.pricing.perOrder}</span>
            </div>

            {/* The ladder, in the price block rather than a footnote.
                
                "The more you sell, the less each order costs" is the half of
                the pricing a growing restaurant actually cares about, and it
                is the answer to the arithmetic they will do anyway: at four
                hundred orders a day a flat rate is a large monthly line item,
                and a large line item gets negotiated. Better to have already
                answered it on the page. */}
            <div className="relative mt-6 rounded-2xl border border-line bg-raised p-4">
              <p className="text-sm font-semibold text-ink">{t.pricing.tiersTitle}</p>
              <ul className="mt-3 space-y-1.5">
                {t.pricing.tiers.map((row) => (
                  <li
                    key={row.range}
                    className="flex items-baseline justify-between gap-4 text-sm"
                  >
                    <span className="text-ink-soft">{row.range}</span>
                    <span className="tabular-nums font-semibold text-ink">
                      {row.price}
                    </span>
                  </li>
                ))}
              </ul>
              <p className="mt-3 text-xs text-ink-muted">{t.pricing.tiersNote}</p>
            </div>

            <p className="mt-6 text-sm font-semibold uppercase tracking-wider text-ink-muted">
              {t.pricing.includedTitle}
            </p>
            <ul className="mt-4 grid gap-3 sm:grid-cols-2">
              {t.pricing.included.map((x) => (
                <li key={x} className="flex items-start gap-2.5 text-sm text-ink-soft">
                  <Check />
                  <span>{x}</span>
                </li>
              ))}
            </ul>
            <a href={TELEGRAM} className="btn-primary mt-8 px-6 py-3.5 text-base">
              {t.pricing.cta}
            </a>
            <p className="mt-3 text-xs text-ink-muted">{t.pricing.trial}</p>
          </div>

          <div className="grid gap-6">
            {/* ⚠️ **The dashboard, beside the numbers it explains, and now the
                real one.** It used to open the page; the counter took that slot,
                which is right — but this is the one place a visitor sees what a
                month of per-order billing actually looks like in the panel, and
                a drawn card of invented figures was making that point with a
                picture of nothing. */}
            <div className="hidden lg:block">
              <Mockup
                src={shot(lang, "dashboard")}
                alt={t.shots.dashboardAlt}
                w={1500}
                h={938}
                badge={<IconReports className="h-5 w-5" />}
              />
            </div>

            {/* Free menu loading.

                First of the three cards, and deliberately the loudest: the
                objection that actually stops a small owner is not the monthly
                rate, it is the evening they imagine spending typing eighty
                dishes into a screen. The price they are comparing is the one
                they can compute; this is the cost they cannot, so it is the one
                worth answering with a number. */}
            <div className="card border-signal-500/40">
              <div className="flex items-start justify-between gap-4">
                <p className="h-display text-xl">{t.pricing.setupTitle}</p>
                <span className="shrink-0 rounded-lg bg-signal-500/15 px-2.5 py-1 text-xs font-semibold text-signal-600 dark:text-signal-400">
                  {t.pricing.setupBadge}
                </span>
              </div>
              <p className="mt-3 text-sm leading-relaxed text-ink-muted">
                {t.pricing.setupDesc}
              </p>
            </div>

            {/* Chains. Separate card rather than a fourth row in the ladder:
                the ladder answers "what do I pay", this answers "is it worth
                talking to you at all" — and a chain reads the second question
                first. */}
            <div className="card">
              <div className="flex items-start justify-between gap-4">
                <p className="h-display text-xl">{t.pricing.chainsTitle}</p>
                <span className="shrink-0 rounded-lg border border-line-strong px-2.5 py-1 text-xs font-semibold text-ink-soft">
                  {t.pricing.chainsBadge}
                </span>
              </div>
              <p className="mt-3 text-sm leading-relaxed text-ink-muted">
                {t.pricing.chainsDesc}
              </p>
              <ul className="mt-4 grid gap-2.5">
                {t.pricing.chainsPoints.map((x) => (
                  <li key={x} className="flex items-start gap-2.5 text-sm text-ink-soft">
                    <Check />
                    <span>{x}</span>
                  </li>
                ))}
              </ul>
            </div>

            <div className="card flex flex-col justify-between">
              <div>
                <p className="eyebrow">{t.pricing.addonTitle}</p>
                <p className="h-display mt-3 text-xl">{t.pricing.addonName}</p>
                <p className="mt-3 text-sm leading-relaxed text-ink-muted">
                  {t.pricing.addonDesc}
                </p>
              </div>
              {/* Shown rather than described: the customer should see exactly
                  what sits in their footer before deciding to pay to remove it. */}
              <div className="mt-6 rounded-xl border border-dashed border-line-strong bg-page px-4 py-3">
                <span className="inline-flex items-center gap-1.5 text-xs font-medium text-ink-muted">
                  <KeelMark className="h-3.5 w-3.5" />
                  Powered by Keel
                </span>
              </div>
            </div>
          </div>
        </div>
      </Section>
      {/* ---- What it comes to ----

          Directly under the two price sections rather than at the end of the
          page: it is the only block that adds the counter's monthly rate to the
          per-order ladder, and a visitor who has just read both is holding both
          halves of the sum. Further down it would be answering a question they
          have already answered wrongly. */}
      <Section
        id="calc"
        eyebrow={t.calc.eyebrow}
        title={t.calc.title}
        lead={t.calc.lead}
        tone="raised"
      >
        <Calculator />
      </Section>

      {/* ---- The counter lives on its own page now ----

          ⚠️ **A link, where three hundred lines of section used to be.** The
          till was the second thing on this page and the deepest thing on it,
          and a visitor met it before they knew why any of it mattered. It is
          not weaker for moving — it is the part somebody reads once they have
          decided to look, and this line is how they get there. */}
      <Section
        eyebrow={t.till.eyebrow}
        title={t.till.title}
        lead={t.till.lead}
        tone="raised"
      >
        <div className="flex flex-wrap justify-center gap-3">
          <Link href={localePath(lang, "/kassa")} className="btn-primary px-6 py-3.5 text-base">
            {t.nav.till}
          </Link>
          <Link
            href={localePath(lang, "/kassa#integrations")}
            className="btn-ghost px-6 py-3.5 text-base"
          >
            {t.nav.integrations}
          </Link>
        </div>
      </Section>

      {/* ---- Partners ---- */}
      {/* Rendered only when somebody has actually agreed to be named. An
          empty "our customers" strip says more than no strip at all, and what
          it says is that there are none. */}
      {partners.length > 0 && (
        <Section
          eyebrow={t.partners.eyebrow}
          title={t.partners.title}
          lead={t.partners.lead}
          tone="raised"
        >
          <Partners items={partners} />
        </Section>
      )}

      {/* ---- Three steps ---- */}
      <Section
        id="start"
        eyebrow={t.timeline.eyebrow}
        title={t.timeline.title}
        lead={t.timeline.lead}
      >
        <Timeline t={t} />
      </Section>
      {/* ---- FAQ ---- */}
      <Section id="faq" eyebrow={t.faq.eyebrow} title={t.faq.title}>
        <div className="mx-auto max-w-3xl divide-y divide-line rounded-2xl border border-line bg-surface">
          {t.faq.items.map((it) => (
            <details key={it.q} className="group px-6 py-5">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-4 font-display text-base font-semibold text-ink [&::-webkit-details-marker]:hidden">
                {it.q}
                <span className="grid h-6 w-6 shrink-0 place-items-center rounded-lg border border-line text-ink-muted transition group-open:rotate-45">
                  <svg viewBox="0 0 24 24" className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round">
                    <path d="M12 5v14M5 12h14" />
                  </svg>
                </span>
              </summary>
              <p className="mt-3 text-sm leading-relaxed text-ink-muted">{it.a}</p>
            </details>
          ))}
        </div>
      </Section>
      {/* ---- CTA ---- */}
      <section id="cta" className="container-page pb-24">
        <div className="hull-glow relative overflow-hidden rounded-3xl border border-line-strong bg-surface px-6 py-14 text-center sm:px-12 sm:py-20">
          <HullBackdrop subtle />
          <div className="relative">
            <h2 className="h-display mx-auto max-w-2xl text-3xl sm:text-4xl">{t.cta.title}</h2>
            <p className="mx-auto mt-4 max-w-xl text-ink-soft">{t.cta.lead}</p>
            <div className="mt-8 flex justify-center">
              <a href={TELEGRAM} className="btn-primary gap-2.5 px-7 py-3.5 text-base">
                <TelegramMark />
                {t.cta.button}
              </a>
            </div>
          </div>
        </div>
      </section>

      <Footer t={t} lang={lang} />
    </>
  );
}

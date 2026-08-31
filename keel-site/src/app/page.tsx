import Link from "next/link";
import Header from "@/components/Header";
import Integrations from "@/components/Integrations";
import Partners from "@/components/Partners";
import { KeelMark, Logo } from "@/components/Logo";
import {
  FEATURE_ICONS,
  TILL_ICONS,
  IconDot,
  IconKitchen,
  IconTill,
  IconOffline,
  IconPanel,
  IconPayment,
  IconReports,
  IconSite,
  IconStaff,
  IconStock,
  IconTelegram,
  IconDelivery,
} from "@/components/Icons";
import {
  ChannelVisual,
  LadderVisual,
  PriceVisual,
} from "@/components/Visual3D";
import Reveal from "@/components/landing/Reveal";
import Mockup from "@/components/landing/Mockup";
import { FloatChip } from "@/components/landing/Frame";
import Calculator from "@/components/landing/Calculator";
import Timeline from "@/components/landing/Timeline";
import { accent } from "@/lib/accent";
import { getLang, getT } from "@/lib/i18n/server";
import { localePath } from "@/lib/i18n/url";
import type { Lang } from "@/lib/i18n/dict";
import { getPartners } from "@/lib/partners";
import { EMAIL, TELEGRAM } from "@/lib/links";

export default async function Home() {
  const t = await getT();
  const lang = await getLang();
  // Read on the server so the strip is in the first paint: a marketing page
  // that pops its social proof in a second late has already been scrolled past.
  const partners = await getPartners();

  return (
    <>
      <Header />

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
              <a href="#till" className="btn-ghost px-6 py-3.5 text-base">
                {t.hero.ctaSecondary}
              </a>
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
              src="/shots/till.webp"
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

      {/* ---- The till ----

          Placed **between the product grid and the integrations**, and that is
          a sales decision rather than a layout one. The feature grid above ends
          on "your own channel"; the integrations section below answers "does it
          work with the till I already run". This is the section that changes
          the second question into "or replace it", and it has to be read before
          the list of other people's systems rather than after it.

          It is also the one part of the page with its own price table, because
          the counter is the one part of the product that is not billed per
          order — and a visitor who read the per-order ladder above and then
          meets a monthly figure with no explanation assumes the worst. */}
      <Section
        id="till"
        align="left"
        eyebrow={t.till.eyebrow}
        title={t.till.title}
        lead={t.till.lead}
      >
        {/* The object being sold, drawn rather than described. A till is a
            physical thing on a counter and a paragraph about one is much harder
            to picture than the thing itself. */}
        {/* ⚠️ The mockup takes the wider track now. At `.9fr` the payment
            window came out 482px and the figures in it were unreadable — the
            column beside it is a list of eight short lines and did not need the
            room. */}
        <div className="grid gap-10 lg:grid-cols-[1.05fr_.95fr] lg:items-center">
          {/* ⚠️ **The payment window, not a drawing of the machine.** The
              monoblock stood here — the hardware argument, "this is a physical
              thing you put in your shop", which no screenshot makes. It lost
              anyway: beside a list headed "what is in the till", a drawing
              answers a question nobody in that column is asking. This is the
              twelve seconds the whole section is about — cash, card, transfer
              or debt, the discount, and the change. */}
          <Mockup
            src="/shots/pay.webp"
            alt={t.till.payAlt}
            w={1400}
            h={1086}
            badge={<IconPayment className="h-5 w-5" />}
          />

          <div>
            <p className="text-sm font-semibold uppercase tracking-wider text-ink-muted">
              {t.till.featuresTitle}
            </p>
            <div className="mt-5 grid gap-x-6 gap-y-5 sm:grid-cols-2">
              {t.till.features.map((it, i) => {
                const Icon = TILL_ICONS[i] ?? IconDot;
                return (
                  <div key={it.name} className="flex gap-3.5">
                    <span className="icon-badge mt-0.5">
                      <Icon className="h-[1.35rem] w-[1.35rem]" />
                    </span>
                    <div className="min-w-0">
                      <p className="font-display text-[15px] font-semibold text-ink">
                        {it.name}
                      </p>
                      <p className="mt-1 text-sm leading-relaxed text-ink-muted">
                        {it.desc}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>

        {/* ⚠️ **The picture is gone from here because it is the hero now.**
            This block held the till screen; putting the same screenshot at the
            top of the page and again two screens down is the page showing one
            thing twice and calling it two. The sentence it was captioned with
            is worth keeping — it is the line that tells a cashier what they are
            looking at — so it stays as a lead-in to the two screens that are
            not the hero. */}
        <p className="mt-14 max-w-2xl text-ink-soft">{t.till.shotLead}</p>

        {/* ⚠️ **The other two screens, because a till is three screens.** The
            monoblock above is what a visitor pictures when they hear "kassa";
            the floor tablet and the pass screen are the two they do not, and
            they are the two that decide whether this can replace the system a
            restaurant already runs. Drawn rather than listed: "zal xaritasi" is
            a phrase, a room with one table lit up is the thing itself.

            ⚠️ **Their own copy, not the feature grid's.** These first reused
            two items from the list above, which put the same sentence on screen
            twice within a scroll — and the second copy reads as a page that has
            lost track of what it already said. */}
        <p className="mt-14 text-sm font-semibold uppercase tracking-wider text-ink-muted">
          {t.till.screensTitle}
        </p>
        {/* ⚠️ **In a frame, not beside the text.** These two were a drawing at
            the left edge of a card with a paragraph next to it, and at that
            size the drawing was decoration — it said "there is an illustration
            here", not "this is a screen". Inside a device frame the same SVG
            is read as something running on a tablet in a dining room, which is
            the claim the card is making. The frames come from the page's own
            tokens (`components/landing/Frame.tsx`) so they belong in both
            themes; the picture inside them is what the screenshots replace. */}
        {/* ⚠️ **Rows, not a two-column grid of cards.** Side by side, the floor
            plan came out about 490px wide and its table totals were unreadable
            — a picture that says "a screenshot exists" rather than showing
            anything. Given a row each, the mockup is 640px and the sides
            alternate, so two screens do not read as one repeated card. */}
        <div className="mt-5 grid gap-10">
          {[
            {
              shot: "/shots/floor.webp",
              alt: t.till.floorAlt,
              w: 1400,
              h: 973,
              it: t.till.screens[0],
              kind: "tablet" as const,
              B: <IconStaff className="h-5 w-5" />,
            },
            {
              shot: "/shots/kds.webp",
              alt: t.till.kitchenAlt,
              w: 1500,
              h: 938,
              it: t.till.screens[1],
              kind: "screen" as const,
              B: <IconKitchen className="h-5 w-5" />,
            },
          ].map(({ shot, alt, w, h, it, kind, B }, i) => (
            <div
              key={it.name}
              // ⚠️ **The mirrored row needs its tracks mirrored too.** Moving
              // the mockup to the right with `order-2` puts it in the *second*
              // track, and with the ratio left alone that is the narrow one —
              // so the alternating row silently halved its own picture, and
              // the kitchen display's nine tickets became unreadable at 455px.
              // Reordering and re-sizing are two different things.
              className={`grid items-center gap-8 ${
                i % 2 === 1
                  ? "lg:grid-cols-[1fr_1.35fr] lg:[&>*:first-child]:order-2"
                  : "lg:grid-cols-[1.35fr_1fr]"
              }`}
            >
              {/* The badge names the room the screen is in, and follows the
                  mockup to the outer corner: pinned left on both rows it sat on
                  the mirrored screenshot's own heading. */}
              <Mockup
                src={shot}
                alt={alt}
                w={w}
                h={h}
                kind={kind}
                badge={B}
                side={i % 2 === 1 ? "right" : "left"}
              />
              <div>
                <p className="font-display text-xl font-semibold text-ink">
                  {it.name}
                </p>
                <p className="mt-3 leading-relaxed text-ink-muted">{it.desc}</p>
              </div>
            </div>
          ))}
        </div>

        {/* ---- The plans ---- */}
        <div className="mt-16">
          <div className="flex flex-wrap items-end justify-between gap-6">
            <div className="max-w-2xl">
              <h3 className="h-display text-2xl sm:text-3xl">{t.till.plansTitle}</h3>
              <p className="mt-3 text-ink-soft">{t.till.plansLead}</p>
            </div>
            {/* The ladder as an object. ⚠️ The bars rise while the copy says the
                per-register price falls — they are the plan sizes, not the
                prices, and the caption underneath says so. A picture that
                contradicted the sentence beside it would be worse than none. */}
            <LadderVisual decorative className="hidden h-28 shrink-0 sm:block" />
          </div>

          <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {t.till.plans.map((p) => (
              <div
                key={p.name}
                className={`relative flex flex-col rounded-3xl border bg-surface p-6 ${
                  p.featured
                    ? "border-signal-500/50 shadow-xl shadow-signal-500/10"
                    : "border-line"
                }`}
              >
                <p className="font-display text-lg font-semibold text-ink">{p.name}</p>
                {/* ⚠️ The unit on its own line, always. Left to wrap it sits
                    beside "450 000" and under "1 250 000", so four cards in a
                    row have their prices at three different heights — which
                    reads as three different kinds of number rather than one
                    ladder. */}
                <div className="mt-4">
                  <p className="h-display text-3xl leading-none">{p.price}</p>
                  <p className="mt-1.5 text-sm text-ink-muted">
                    {t.pricing.unit} / {t.till.thPrice.toLowerCase()}
                  </p>
                </div>
                <p className="mt-3 inline-flex w-fit items-center gap-1.5 rounded-lg bg-signal-500/12 px-2.5 py-1 text-xs font-semibold text-signal-600 dark:text-signal-400">
                  {p.registers}
                </p>
                <p className="mt-4 text-sm leading-relaxed text-ink-muted">
                  {p.includes}
                </p>
              </div>
            ))}
          </div>

          {/* The two things the table cannot say without a footnote, given
              their own cards because both of them decide a sale: the small café
              that wants its food cost, and the chain doing arithmetic on
              branches. */}
          <div className="mt-6 grid gap-4 lg:grid-cols-2">
            <div className="card flex items-start gap-4 border-signal-500/40">
              <span className="icon-badge">
                <IconStock className="h-[1.35rem] w-[1.35rem]" />
              </span>
              <div>
                <p className="font-display text-lg font-semibold text-ink">
                  {t.till.addonTitle}
                </p>
                <p className="mt-2 text-sm leading-relaxed text-ink-muted">
                  {t.till.addonDesc}
                </p>
              </div>
            </div>
            <div className="card flex items-center gap-5">
              <div className="min-w-0">
                <p className="font-display text-lg font-semibold text-ink">
                  {t.till.chainTitle}
                </p>
                <p className="mt-2 text-sm leading-relaxed text-ink-muted">
                  {t.till.chainDesc}
                </p>
              </div>
              <PriceVisual decorative className="hidden h-28 shrink-0 sm:block" />
            </div>
          </div>

          {/* ⚠️ **The stockroom, shown rather than claimed.** "Ombor va
              tannarx" is the one line in the plans table an owner does not
              believe until they see it: every till says it does inventory, and
              most of them mean a text field. A screen with real balances and
              real values on it is the answer. */}
          <div className="mt-6 grid items-center gap-10 lg:grid-cols-[1fr_1.35fr]">
            <div className="lg:order-2">
              <Mockup
                src="/shots/stock.webp"
                alt={t.shots.stockAlt}
                w={1500}
                h={938}
                side="right"
                badge={<IconStock className="h-5 w-5" />}
                chip={
                  <FloatChip
                    icon={<IconReports className="h-4 w-4" />}
                    value={t.till.chipCost}
                    label={t.till.chipCostNote}
                  />
                }
                chipSide="left"
              />
            </div>
            <div className="lg:order-1">
              <h3 className="h-display text-2xl">{t.shots.stockTitle}</h3>
              <p className="mt-3 leading-relaxed text-ink-muted">{t.shots.stockLead}</p>
            </div>
          </div>

          {/* ---- What we never restrict ----

              ⚠️ On the marketing page on purpose, not buried in a contract.
              Every item here is something a competitor does sell separately,
              and a restaurant comparing plans has no way to know that until the
              month they need it. Saying it out loud is worth more than the
              revenue it gives up — and it is the part of the pricing we would
              least like to be caught quietly changing. */}
          <div className="mt-6 overflow-hidden rounded-3xl border border-line bg-hull-900 p-8 text-white/90 sm:p-10">
            <p className="font-display text-xl font-semibold text-white">
              {t.till.neverTitle}
            </p>
            <p className="mt-2 max-w-2xl text-sm leading-relaxed text-white/70">
              {t.till.neverLead}
            </p>
            <ul className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {t.till.never.map((x) => (
                <li key={x} className="flex items-start gap-2.5 text-sm">
                  <svg
                    viewBox="0 0 24 24"
                    className="mt-0.5 h-4 w-4 shrink-0 text-signal-400"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth={2.5}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden
                  >
                    <path d="M20 6 9 17l-5-5" />
                  </svg>
                  <span className="text-white/85">{x}</span>
                </li>
              ))}
            </ul>
            <p className="mt-6 max-w-3xl text-sm leading-relaxed text-white/60">
              {t.till.limitsNote}
            </p>
          </div>
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
            src="/shots/orders.webp"
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
              src="/shots/site.webp"
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
                src: "/shots/miniapp.webp",
                alt: t.shots.miniAppAlt,
                label: t.shots.miniAppLabel,
                icon: <IconTelegram className="h-4 w-4" />,
              },
              {
                src: "/shots/courier.webp",
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

      {/* ---- Rivals ----

          ⚠️ Named competitors with their own published numbers, and the honest
          note that one of them is cheaper than us. The owner is going to open
          both tabs anyway; a page that makes them do it themselves loses the
          one moment where the comparison can be framed — and a table that
          quietly omitted the cheap flat-rate option would be discredited by the
          first owner who found it, along with everything else on this page. */}
      <Section
        id="rivals"
        eyebrow={t.rivals.eyebrow}
        title={t.rivals.title}
        lead={t.rivals.lead}
        tone="raised"
      >
        <div className="overflow-x-auto rounded-2xl border border-line bg-surface">
          <table className="w-full min-w-[42rem] text-left text-sm">
            <thead className="border-b border-line text-xs uppercase tracking-wider text-ink-muted">
              <tr>
                <th className="px-5 py-4 font-medium">{t.rivals.thOrders}</th>
                <th className="px-5 py-4 text-right font-medium text-ink">{t.rivals.thKeel}</th>
                <th className="px-5 py-4 text-right font-medium">{t.rivals.thPerOrder}</th>
                <th className="px-5 py-4 text-right font-medium">{t.rivals.thSubscription}</th>
                <th className="px-5 py-4 text-right font-medium">{t.rivals.thDiff}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {t.rivals.rows.map((row) => (
                <tr key={row.c}>
                  <td className="px-5 py-4">
                    <span className="font-semibold text-ink">{row.c}</span>{" "}
                    <span className="text-xs text-ink-muted">{row.perDay}</span>
                  </td>
                  {/* Ours is the only column with weight on it. Four columns of
                      equal-looking numbers is a table nobody reads to the end. */}
                  <td className="px-5 py-4 text-right font-display font-semibold tabular-nums text-ink">
                    {row.keel}
                  </td>
                  <td className="px-5 py-4 text-right tabular-nums text-ink-muted">{row.perOrder}</td>
                  <td className="px-5 py-4 text-right tabular-nums text-ink-muted">{row.subscription}</td>
                  <td className="px-5 py-4 text-right">
                    <span className="rounded-lg bg-signal-500/15 px-2 py-1 text-xs font-semibold tabular-nums text-signal-600 dark:text-signal-400">
                      {row.diff}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <ul className="mt-6 grid gap-2.5 sm:grid-cols-2">
          {t.rivals.notes.map((x) => (
            <li key={x} className="flex items-start gap-2.5 text-sm text-ink-soft">
              <Check />
              <span>{x}</span>
            </li>
          ))}
        </ul>
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
                src="/shots/dashboard.webp"
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

      {/* ---- Integrations ---- */}
      {/* Was a bare row of till names. Widened to every system Keel talks to,
          because "does it work with what I already run" is the question that
          decides the sale — and answered with the offer to write the missing
          one, which is the more important half for whoever is not on the
          list. */}
      <Section
        id="integrations"
        eyebrow={t.integrations.eyebrow}
        title={t.integrations.title}
        lead={t.integrations.lead}
      >
        <Integrations t={t} />
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

/** One section shell so spacing, width and heading rhythm are decided once. */
function Section({
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
function HullBackdrop({ subtle = false }: { subtle?: boolean }) {
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


function TelegramMark() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" fill="currentColor" aria-hidden>
      <path d="M21.9 4.3 18.7 19c-.2 1-.9 1.3-1.8.8l-4.9-3.6-2.4 2.3c-.3.3-.5.5-1 .5l.3-5 9.1-8.2c.4-.4-.1-.6-.6-.2L6.2 12.7 1.4 11.2c-1-.3-1-1 .2-1.5l19-7.3c.9-.3 1.6.2 1.3 1.9Z" />
    </svg>
  );
}

function Check() {
  return (
    <svg viewBox="0 0 24 24" className="mt-0.5 h-4 w-4 shrink-0 text-signal-500" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}

function Footer({ t, lang }: { t: Awaited<ReturnType<typeof getT>>; lang: Lang }) {
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
            <li><a href="/#till" className="hover:text-ink">{t.nav.till}</a></li>
            <li><a href="/#integrations" className="hover:text-ink">{t.nav.integrations}</a></li>
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
            <li><a href={TELEGRAM} className="hover:text-ink">Telegram</a></li>
            <li><a href={`mailto:${EMAIL}`} className="hover:text-ink">{EMAIL}</a></li>
          </ul>
        </div>
      </div>
      <div className="border-t border-line">
        <div className="container-page flex flex-col gap-2 py-6 text-xs text-ink-muted sm:flex-row sm:items-center sm:justify-between">
          <span>© {new Date().getFullYear()} Keel. {t.footer.rights}.</span>
          <span>keel.uz</span>
        </div>
      </div>
    </footer>
  );
}

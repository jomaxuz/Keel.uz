import type { Metadata } from "next";
import Header from "@/components/Header";
import Integrations from "@/components/Integrations";
import {
  TILL_ICONS,
  IconDot,
  IconKitchen,
  IconOffline,
  IconPayment,
  IconReports,
  IconStaff,
  IconStock,
  IconTill,
} from "@/components/Icons";
import { LadderVisual, PriceVisual } from "@/components/Visual3D";
import Mockup from "@/components/landing/Mockup";
import { FloatChip } from "@/components/landing/Frame";
import { Check, Footer, Section } from "@/components/landing/Shell";
import { getLang, getT } from "@/lib/i18n/server";
import { shot } from "@/lib/shots";

// ---- The counter, on its own page ----
//
// ⚠️ **This was the second section of the home page, and it was the biggest.**
// Three hundred lines of till detail, a rival comparison and an integration
// list, met by somebody who had read one screen and did not yet know why any of
// it mattered. The complaint that moved it was the plain one: people arrived
// and could not tell what the page was about.
//
// It is not cut, because it is not weak — it is the deepest thing we can say
// about the part of the product we mainly sell. It is here instead, one click
// away, for the visitor who has already decided they are interested. The home
// page keeps the argument; this page keeps the proof.

export const metadata: Metadata = {
  // ⚠️ Just the page's own name: the layout appends " | Keel" through its
  // title template, and spelling the brand here too gave "Kassa — Keel | Keel".
  title: "Kassa",
  description:
    "Keel kassasi: zal, oshxona ekrani, oflayn rejim va fiskal chek. Mavjud tizimingiz bilan ishlaydi yoki uni almashtiradi.",
};

export default async function KassaPage() {
  const t = await getT();
  const lang = await getLang();

  return (
    <>
      <Header />
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
            src={shot(lang, "pay")}
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
              shot: shot(lang, "floor"),
              alt: t.till.floorAlt,
              w: 1400,
              h: 973,
              it: t.till.screens[0],
              kind: "tablet" as const,
              B: <IconStaff className="h-5 w-5" />,
            },
            {
              shot: shot(lang, "kds"),
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

          {/* ---- The shop ladder ----

              ⚠️ **A second table on the same page, not a link to another
              one.** A grocery comparing tills is reading REGOS at 149 000 and
              BILLZ at 299 000; a page headlined 450 000 is a page it has
              already left, and "shops are cheaper — ask us" is a sentence
              nobody stays for. The rungs it will actually be sold are the rungs
              it sees, at the moment it is deciding. */}
          <div className="mt-16">
            <h3 className="h-display text-2xl">{t.till.shopTitle}</h3>
            <p className="mt-3 max-w-3xl text-base leading-relaxed text-ink-muted">
              {t.till.shopLead}
            </p>
            <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {t.till.shopPlans.map((p) => (
                <div
                  key={p.name}
                  className={`relative flex flex-col rounded-3xl border bg-surface p-6 ${
                    p.featured
                      ? "border-signal-500/50 shadow-xl shadow-signal-500/10"
                      : "border-line"
                  }`}
                >
                  <p className="font-display text-lg font-semibold text-ink">
                    {p.name}
                  </p>
                  {/* The unit on its own line, for the reason the table above
                      keeps it there: four prices at three heights read as three
                      kinds of number rather than one ladder. */}
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
            <p className="mt-4 text-sm leading-relaxed text-ink-muted">
              {t.till.shopNote}
            </p>
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
                src={shot(lang, "stock")}
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
      <Footer t={t} lang={lang} />
    </>
  );
}

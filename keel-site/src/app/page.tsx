import Link from "next/link";
import Header from "@/components/Header";
import Integrations from "@/components/Integrations";
import Partners from "@/components/Partners";
import { KeelMark, Logo } from "@/components/Logo";
import { getT } from "@/lib/i18n/server";
import { getPartners } from "@/lib/partners";
import { EMAIL, TELEGRAM } from "@/lib/links";

export default async function Home() {
  const t = await getT();
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
              {t.hero.title}
            </h1>
            <p className="mt-6 max-w-xl text-lg leading-relaxed text-ink-soft">
              {t.hero.lead}
            </p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <a href={TELEGRAM} className="btn-primary px-6 py-3.5 text-base">
                {t.hero.ctaPrimary}
              </a>
              <a href="#compare" className="btn-ghost px-6 py-3.5 text-base">
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

          <div className="relative hidden lg:block">
            <HeroCard t={t} />
          </div>
        </div>
      </section>

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
        <p className="mt-5 max-w-3xl text-sm leading-relaxed text-ink-muted">
          {t.compare.honest}
        </p>
      </Section>

      {/* ---- Features ---- */}
      <Section
        id="product"
        eyebrow={t.features.eyebrow}
        title={t.features.title}
        tone="raised"
      >
        <div className="grid gap-px overflow-hidden rounded-3xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
          {t.features.items.map((it, i) => (
            <div key={it.name} className="bg-surface p-6">
              <span className="grid h-9 w-9 place-items-center rounded-xl bg-signal-500/15 font-display text-sm font-semibold text-signal-600 dark:text-signal-400">
                {i + 1}
              </span>
              <p className="mt-4 font-display text-base font-semibold text-ink">{it.name}</p>
              <p className="mt-2 text-sm leading-relaxed text-ink-muted">{it.desc}</p>
            </div>
          ))}
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

      {/* ---- Pricing ---- */}
      <Section
        id="pricing"
        eyebrow={t.pricing.eyebrow}
        title={t.pricing.title}
        lead={t.pricing.lead}
        tone="raised"
      >
        <div className="grid gap-6 lg:grid-cols-[1.1fr_.9fr]">
          <div className="relative overflow-hidden rounded-3xl border border-line-strong bg-surface p-8 sm:p-10">
            <div className="pointer-events-none absolute -right-16 -top-16 h-52 w-52 rounded-full bg-signal-500/10 blur-2xl" />
            <div className="relative flex flex-wrap items-end gap-3">
              <span className="h-display text-6xl sm:text-7xl">1 000</span>
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

      <Footer t={t} />
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
  children,
}: {
  id?: string;
  eyebrow: string;
  title: string;
  lead?: string;
  tone?: "raised";
  children: React.ReactNode;
}) {
  return (
    <section
      id={id}
      className={`scroll-mt-20 border-t border-line py-20 sm:py-24 ${
        tone === "raised" ? "bg-raised" : ""
      }`}
    >
      <div className="container-page">
        <p className="eyebrow">{eyebrow}</p>
        <h2 className="h-display mt-3 max-w-2xl text-3xl sm:text-4xl">{title}</h2>
        {lead && <p className="mt-4 max-w-2xl text-ink-soft">{lead}</p>}
        <div className="mt-10">{children}</div>
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

/** A small mock of what the customer actually gets, rather than a stock photo:
 *  the thing being sold is a screen, so the hero shows a screen. */
function HeroCard({ t }: { t: Awaited<ReturnType<typeof getT>> }) {
  return (
    <div className="animate-drift rounded-3xl border border-line-strong bg-surface p-5 shadow-2xl shadow-hull-950/10">
      <div className="flex items-center justify-between border-b border-line pb-4">
        <Logo />
        <span className="rounded-lg bg-signal-500/15 px-2.5 py-1 text-xs font-semibold text-signal-600 dark:text-signal-400">
          {t.dash.active}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-3 py-5">
        {[
          [t.dash.monthOrders, "1 284"],
          [t.dash.active, "37"],
          [t.dash.monthBillable, "1 284 000"],
        ].map(([k, v]) => (
          <div key={k} className="rounded-xl bg-raised p-3">
            <p className="truncate text-[11px] text-ink-muted">{k}</p>
            <p className="h-display mt-1 text-lg">{v}</p>
          </div>
        ))}
      </div>
      {/* A plain bar chart in divs: no chart library on a landing page. */}
      <div className="flex h-24 items-end gap-1.5">
        {[38, 52, 44, 61, 55, 72, 66, 84, 70, 91, 78, 96].map((h, i) => (
          <div
            key={i}
            style={{ height: `${h}%` }}
            className="flex-1 rounded-t-md bg-gradient-to-t from-signal-500/25 to-signal-500"
          />
        ))}
      </div>
      <p className="mt-3 text-xs text-ink-muted">{t.dash.last30}</p>
    </div>
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

function Footer({ t }: { t: Awaited<ReturnType<typeof getT>> }) {
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
            <li><a href="#product" className="hover:text-ink">{t.nav.product}</a></li>
            <li><a href="#integrations" className="hover:text-ink">{t.nav.integrations}</a></li>
            <li><a href="#pricing" className="hover:text-ink">{t.nav.pricing}</a></li>
            <li><a href="#faq" className="hover:text-ink">{t.nav.faq}</a></li>
          </ul>
        </div>
        <div>
          <p className="text-sm font-semibold text-ink">{t.footer.company}</p>
          <ul className="mt-3 space-y-2 text-sm text-ink-muted">
            <li><a href="#who" className="hover:text-ink">{t.nav.who}</a></li>
            <li><a href="#cta" className="hover:text-ink">{t.nav.start}</a></li>
            {/* In the footer rather than the top nav: a status link somebody
                notices before anything is wrong is a link that suggests
                something might be. */}
            <li><Link href="/status" className="hover:text-ink">{t.status.eyebrow}</Link></li>
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

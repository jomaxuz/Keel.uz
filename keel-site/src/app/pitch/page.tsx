import type { Metadata } from "next";
import Header from "@/components/Header";
import { KeelMark } from "@/components/Logo";
import Reveal from "@/components/landing/Reveal";
import { Footer, HullBackdrop, Section, TelegramMark } from "@/components/landing/Shell";
import PitchVideo from "@/components/pitch/PitchVideo";
import {
  AiBlock,
  Architecture,
  DevStages,
  Flow,
  Integrations,
  LiveLinks,
  Modules,
  Pains,
  Principles,
  ProblemVisual,
  RequirementsNav,
  Roadmap,
  Shot,
  Stages,
} from "@/components/pitch/Pitch";
import { accent } from "@/lib/accent";
import { dicts } from "@/lib/i18n/dict";
import { COMPANY } from "@/lib/legal";
import { EMAIL, TELEGRAM } from "@/lib/links";
import { PITCH, PITCH_CONFIG } from "@/lib/pitch";
import { shot } from "@/lib/shots";

// keel.uz/pitch — the Pitch Day 3.0 page.
//
// ⚠️ **One story, top to bottom, in the order the jury checks it**: problem,
// how it works, the product, proof it is live, team, why us, roadmap, how it is
// built, integrations, the demo video, the live link. The checklist under the
// hero maps each organiser requirement to its section, so nothing has to be
// hunted for.
//
// ⚠️ **Uzbek only, on purpose.** The page is for one event held in Uzbek; the
// words live in `lib/pitch.ts` so a translation is data, not a second page.
// `/ru/pitch` renders the same Uzbek page rather than 404ing, and canonical
// points at the one address that is submitted.
//
// ⚠️ **Not in the sitemap.** It is a page for a jury, not one somebody
// searches for; keeping it out leaves the main site's crawl budget where it is.

export const metadata: Metadata = {
  // The layout's template appends " | Keel"; this title is complete on its own.
  title: { absolute: PITCH.meta.title },
  description: PITCH.meta.description,
  alternates: { canonical: "https://keel.uz/pitch" },
  openGraph: {
    title: PITCH.meta.title,
    description: PITCH.meta.description,
    url: "https://keel.uz/pitch",
    siteName: "Keel",
    locale: "uz_UZ",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: PITCH.meta.title,
    description: PITCH.meta.description,
  },
};

export default function PitchPage() {
  // ⚠️ The footer in Uzbek too: the page is one language, and a Russian footer
  // under it (from a visitor's cookie) reads as a page assembled from parts.
  const t = dicts.uz;
  const P = PITCH;
  const video = PITCH_CONFIG.video;
  const founder = PITCH_CONFIG.founder;
  const founderLinks = [
    { label: "Telegram", href: founder.links.telegram },
    { label: "LinkedIn", href: founder.links.linkedin },
    { label: "GitHub", href: founder.links.github },
  ].filter((l) => l.href);

  return (
    <>
      <Header />
      <main id="main" lang="uz">
        {/* ── 00 · Hero ── */}
        <section className="relative overflow-hidden pb-16 pt-10 sm:pt-16">
          <HullBackdrop subtle />
          <div className="container-page relative">
            <div className="grid items-center gap-12 lg:grid-cols-[1fr_1.1fr]">
              <div className="animate-rise motion-reduce:animate-none">
                <span className="inline-flex items-center gap-2 rounded-full border border-signal-500/40 bg-signal-500/10 px-3 py-1 text-xs font-semibold text-signal-600 dark:text-signal-400">
                  <KeelMark className="h-4 w-4" />
                  {P.hero.badge}
                </span>
                <h1 className="h-display mt-5 text-4xl leading-[1.08] sm:text-5xl lg:text-[3.4rem]">
                  {accent(P.hero.title)}
                </h1>
                <p className="mt-5 max-w-xl text-lg text-ink-soft">{P.hero.lead}</p>
                <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                  <a href="#demo" className="btn-primary whitespace-nowrap px-6 py-3.5 text-base">
                    <svg viewBox="0 0 24 24" className="h-4 w-4" fill="currentColor" aria-hidden>
                      <path d="M8 5.5v13l10.5-6.5L8 5.5Z" />
                    </svg>
                    {P.hero.ctaVideo}
                  </a>
                  <a href="#live" className="btn-ghost whitespace-nowrap px-6 py-3.5 text-base">
                    {P.hero.ctaProduct} <span aria-hidden>→</span>
                  </a>
                </div>
              </div>
              <div className="relative">
                <Shot name="till" alt={P.hero.shotAlt} kind="screen" priority />
                <div className="absolute -bottom-8 -left-4 hidden w-[26%] sm:block">
                  <Shot name="miniapp" alt="Telegram mini app: restoran menyusi" kind="phone" />
                </div>
              </div>
            </div>
            <RequirementsNav />
          </div>
        </section>

        {/* ── 01 · Problem → Solution ── */}
        <Section id="problem" eyebrow={P.problem.eyebrow} title={P.problem.title} lead={P.problem.lead}>
          <ProblemVisual />
          <Pains />
          <div className="mt-12 rounded-3xl border border-signal-500/40 bg-signal-500/5 p-6 text-center sm:p-10">
            <h3 className="h-display text-2xl sm:text-3xl">{accent(P.problem.solutionTitle)}</h3>
            <p className="mx-auto mt-3 max-w-2xl text-ink-soft">{P.problem.solutionLead}</p>
          </div>
        </Section>

        {/* ── 02 · How it works ── */}
        <Section id="how" tone="raised" eyebrow={P.flow.eyebrow} title={P.flow.title} lead={P.flow.lead}>
          <Flow />
        </Section>

        {/* ── 03 · Modules ── */}
        <Section id="product" eyebrow={P.modules.eyebrow} title={P.modules.title} lead={P.modules.lead}>
          <Modules />
        </Section>

        {/* ── 04 · Working product ── */}
        <Section id="working" tone="raised" eyebrow={P.live.eyebrow} title={P.live.title} lead={P.live.lead}>
          <div className="flex flex-col items-center gap-8">
            <Stages items={P.live.stages} />
            <div className="w-full">
              <LiveLinks />
              <p className="mt-4 text-center text-sm text-ink-muted">{P.live.note}</p>
            </div>
          </div>
        </Section>

        {/* ── 05 · Team ── */}
        <Section id="team" align="left" eyebrow={P.team.eyebrow} title={P.team.title} lead={P.team.lead}>
          <div className="grid gap-6 lg:grid-cols-[360px_1fr]">
            <div className="card">
              <div className="flex items-center gap-4">
                <span className="grid h-16 w-16 shrink-0 place-items-center rounded-2xl bg-hull-900 font-display text-xl font-semibold text-signal-400 dark:bg-hull-800" aria-hidden>
                  YN
                </span>
                <div>
                  <h3 className="font-display text-xl font-semibold text-ink">{founder.name}</h3>
                  <p className="text-sm text-ink-soft">{founder.role}</p>
                </div>
              </div>
              <p className="mt-4 text-sm text-ink-muted">
                {COMPANY.nameUz} · STIR {COMPANY.inn}
              </p>
              <div className="mt-4 border-t border-line pt-4">
                <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">{P.team.linksTitle}</p>
                <ul className="mt-2 flex flex-wrap gap-2 text-sm">
                  {founderLinks.map((l) => (
                    <li key={l.label}>
                      <a href={l.href} className="btn-ghost px-3 py-1.5 text-xs" target="_blank" rel="noreferrer">
                        {l.label}
                      </a>
                    </li>
                  ))}
                  <li>
                    <a href={TELEGRAM} className="btn-ghost px-3 py-1.5 text-xs" target="_blank" rel="noreferrer">
                      KEEL Telegram
                    </a>
                  </li>
                  <li>
                    <a href={`mailto:${EMAIL}`} className="btn-ghost px-3 py-1.5 text-xs">
                      {EMAIL}
                    </a>
                  </li>
                  <li>
                    <a href="/developers" className="btn-ghost px-3 py-1.5 text-xs">
                      API hujjati
                    </a>
                  </li>
                </ul>
              </div>
            </div>
            <div className="card">
              <p className="eyebrow">Mas'uliyat va ko'nikmalar</p>
              <ul className="mt-3 grid gap-2 sm:grid-cols-2">
                {P.team.responsibilities.map((r) => (
                  <li key={r} className="flex gap-2 text-sm text-ink-soft">
                    <span aria-hidden className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-signal-500" />
                    {r}
                  </li>
                ))}
              </ul>
              <p className="eyebrow mt-6">Texnologiyalar</p>
              <ul className="mt-3 flex flex-wrap gap-2">
                {P.team.stack.map((s) => (
                  <li key={s} className="rounded-lg border border-line-strong bg-raised px-2.5 py-1 font-mono text-xs text-ink">
                    {s}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </Section>

        {/* ── 06 · Why us ── */}
        <Section id="why-us" tone="raised" eyebrow={P.whyUs.eyebrow} title={P.whyUs.title} lead={P.whyUs.lead}>
          <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {P.whyUs.points.map((p, i) => (
              <li key={p.title} className="card">
                <span className="font-display text-sm font-bold text-signal-600 dark:text-signal-400">
                  {String(i + 1).padStart(2, "0")}
                </span>
                <h3 className="mt-1 font-semibold text-ink">{p.title}</h3>
                <p className="mt-1.5 text-sm text-ink-soft">{p.desc}</p>
              </li>
            ))}
          </ul>
        </Section>

        {/* ── 07 · Roadmap ── */}
        <Section id="roadmap" eyebrow={P.roadmap.eyebrow} title={P.roadmap.title} lead={P.roadmap.lead}>
          <Roadmap />
        </Section>

        {/* ── 08 · Implementation ── */}
        <Section id="tech" tone="raised" eyebrow={P.tech.eyebrow} title={P.tech.title} lead={P.tech.lead}>
          <div className="space-y-14">
            <div>
              <h3 className="mb-5 font-display text-xl font-semibold text-ink">{P.tech.archTitle}</h3>
              <Architecture />
            </div>
            <div>
              <h3 className="mb-5 font-display text-xl font-semibold text-ink">{P.tech.stagesTitle}</h3>
              <DevStages />
            </div>
            <AiBlock />
            <div>
              <h3 className="mb-5 font-display text-xl font-semibold text-ink">{P.tech.principlesTitle}</h3>
              <Principles />
            </div>
          </div>
        </Section>

        {/* ── 09 · Integrations ── */}
        <Section id="integrations" eyebrow={P.integrations.eyebrow} title={P.integrations.title} lead={P.integrations.lead}>
          <Integrations />
        </Section>

        {/* ── 10 · Demo video ── */}
        <Section id="demo" tone="raised" eyebrow={P.demo.eyebrow} title={P.demo.title}>
          <div className="grid gap-8 lg:grid-cols-[1.5fr_1fr]">
            <PitchVideo
              youtubeId={video.youtubeId}
              mp4={video.mp4}
              poster={video.poster || shot("uz", "till")}
              duration={video.duration}
              labels={{
                play: P.demo.play,
                placeholderTitle: P.demo.placeholderTitle,
                placeholderLead: P.demo.placeholderLead,
                title: P.meta.title,
              }}
            />
            <div id="demo-about" className="scroll-mt-24">
              <h3 className="font-display text-xl font-semibold text-ink">{P.demo.aboutTitle}</h3>
              <p className="mt-2 text-sm text-ink-soft">{P.demo.aboutLead}</p>
              <ol className="mt-4 space-y-2">
                {P.demo.aboutSteps.map((s, i) => (
                  <li key={s} className="flex gap-3 text-sm text-ink-soft">
                    <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
                      {i + 1}
                    </span>
                    {s}
                  </li>
                ))}
              </ol>
            </div>
          </div>
        </Section>

        {/* ── 11 · Live product ── */}
        <section id="live" className="scroll-mt-20 border-t border-line py-20 sm:py-24">
          <div className="container-page">
            <Reveal className="mx-auto max-w-3xl text-center">
              <p className="eyebrow">{P.liveCta.eyebrow}</p>
              <h2 className="h-display mt-3 text-3xl sm:text-4xl">{P.liveCta.title}</h2>
              <p className="mx-auto mt-4 max-w-xl text-ink-soft">{P.liveCta.lead}</p>
              <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
                <a href="/" className="btn-primary px-7 py-4 text-base">
                  {P.liveCta.button}
                </a>
                <a href={TELEGRAM} target="_blank" rel="noreferrer" className="btn-ghost px-6 py-4 text-base">
                  <TelegramMark /> {P.liveCta.telegram}
                </a>
              </div>
            </Reveal>
          </div>
        </section>

        {/* ── Finale ── */}
        <section className="relative overflow-hidden bg-hull-950 py-20 text-white sm:py-28">
          <HullBackdrop />
          <div className="container-page relative text-center">
            <KeelMark className="mx-auto h-12 w-12 text-signal-500" />
            <p className="mx-auto mt-6 max-w-4xl whitespace-pre-line font-display text-3xl font-semibold leading-tight tracking-tight sm:text-5xl">
              {P.finale.title.split("*").map((part, i) =>
                i % 2 === 1 ? (
                  <span key={i} className="text-signal-400">
                    {part}
                  </span>
                ) : (
                  <span key={i}>{part}</span>
                ),
              )}
            </p>
            <p className="mt-5 text-lg text-white/75">{P.finale.lead}</p>
            <p className="mt-8 text-sm text-white/60">
              <a href="/" className="hover:text-white">keel.uz</a>
              {" · "}
              <a href={`tel:${COMPANY.phone}`} className="hover:text-white">{COMPANY.phoneText}</a>
              {" · "}
              <a href={`mailto:${EMAIL}`} className="hover:text-white">{EMAIL}</a>
            </p>
          </div>
        </section>
      </main>
      <Footer t={t} lang="uz" />
    </>
  );
}

import { KeelMark } from "@/components/Logo";
import Mockup from "@/components/landing/Mockup";
import { Check } from "@/components/landing/Shell";
import { accent } from "@/lib/accent";
import { PITCH } from "@/lib/pitch";
import { shot } from "@/lib/shots";

// The pieces of keel.uz/pitch. Server components: nothing here needs the
// browser, so the page ships as HTML and the only script is the video button
// and the scroll reveal the rest of the site already uses.
//
// ⚠️ **Diagrams are HTML and CSS, not pictures.** They have to reflow from a
// 1440px projector down to a 390px phone; an image of a diagram either shrinks
// its labels to nothing on the phone or wastes the projector.

const SIZES: Record<string, { w: number; h: number }> = {
  till: { w: 1600, h: 1000 },
  floor: { w: 1400, h: 973 },
  pay: { w: 1400, h: 1086 },
  kds: { w: 1500, h: 938 },
  stock: { w: 1500, h: 938 },
  dashboard: { w: 1500, h: 938 },
  orders: { w: 1500, h: 938 },
  site: { w: 1500, h: 938 },
  miniapp: { w: 560, h: 694 },
  courier: { w: 560, h: 694 },
};

/** A real screenshot (public/shots, Uzbek frame) at its own size. */
export function Shot({
  name,
  alt,
  kind = "browser",
  priority = false,
  className = "",
}: {
  name: string;
  alt: string;
  kind?: "browser" | "phone" | "tablet" | "screen";
  priority?: boolean;
  className?: string;
}) {
  const s = SIZES[name] ?? { w: 1500, h: 938 };
  return (
    <Mockup
      src={shot("uz", name)}
      alt={alt}
      w={s.w}
      h={s.h}
      kind={kind}
      priority={priority}
      className={className}
    />
  );
}

// ───────────────────────────── Requirements checklist

/** The organisers' list, each item a link to its section. ⚠️ The point is the
 *  jury's time: they arrive with eight boxes to tick and should be able to tick
 *  them from the top of the page. */
export function RequirementsNav() {
  return (
    <nav aria-labelledby="req-title" className="card mt-12 p-5 sm:p-6">
      <p id="req-title" className="eyebrow">
        {PITCH.requirementsTitle}
      </p>
      <ol className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {PITCH.requirements.map((r, i) => (
          <li key={r.id}>
            <a
              href={`#${r.id}`}
              className="group flex items-center gap-3 rounded-xl border border-line px-3 py-2.5 text-sm text-ink-soft transition hover:border-signal-500/60 hover:text-ink focus:outline-none focus-visible:ring-2 focus-visible:ring-signal-500"
            >
              <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
                {i + 1}
              </span>
              <span className="flex-1">{r.label}</span>
              <span aria-hidden className="text-ink-muted transition group-hover:translate-x-0.5 group-hover:text-signal-600 motion-reduce:transition-none">
                →
              </span>
            </a>
          </li>
        ))}
      </ol>
    </nav>
  );
}

// ───────────────────────────── Problem → Solution

// Eight systems around the business, at even angles. Percent of the box.
const NODE_POS = PITCH.problem.nodes.map((_, i) => {
  const a = (i / PITCH.problem.nodes.length) * Math.PI * 2 - Math.PI / 2;
  return { x: 50 + Math.cos(a) * 37, y: 50 + Math.sin(a) * 38 };
});
// A few cross-wires between neighbours-but-one: the tangle is the point.
const TANGLE: [number, number][] = [
  [0, 3], [1, 5], [2, 6], [3, 7], [4, 1], [5, 0], [6, 2], [7, 4],
];

export function ProblemVisual() {
  const p = PITCH.problem;
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      {/* Today: everything wired to everything, through the owner. */}
      <figure className="card relative p-0">
        <figcaption className="flex items-center justify-between border-b border-line px-5 py-3 text-sm">
          <span className="font-semibold text-ink">Bugun</span>
          <span className="text-ink-muted">8 ta alohida tizim</span>
        </figcaption>
        <div className="relative mx-auto aspect-[5/4] w-full max-w-lg">
          <svg
            aria-hidden
            viewBox="0 0 100 100"
            preserveAspectRatio="none"
            className="absolute inset-0 h-full w-full"
          >
            {TANGLE.map(([a, b]) => (
              <line
                key={`${a}-${b}`}
                x1={NODE_POS[a].x}
                y1={NODE_POS[a].y}
                x2={NODE_POS[b].x}
                y2={NODE_POS[b].y}
                className="stroke-ink-muted/30"
                strokeWidth={0.35}
                strokeDasharray="1.2 1.2"
                vectorEffect="non-scaling-stroke"
              />
            ))}
            {NODE_POS.map((n, i) => (
              <line
                key={i}
                x1={50}
                y1={50}
                x2={n.x}
                y2={n.y}
                className="stroke-red-500/45"
                strokeWidth={1}
                strokeDasharray="3 3"
                vectorEffect="non-scaling-stroke"
              />
            ))}
          </svg>
          <span className="absolute left-1/2 top-1/2 grid h-16 w-16 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full border border-red-500/40 bg-red-500/10 text-center text-[11px] font-semibold leading-tight text-red-700 dark:text-red-300 sm:h-20 sm:w-20 sm:text-xs">
            Restoran
            <br />
            egasi
          </span>
          {p.nodes.map((n, i) => (
            <span
              key={n}
              className="absolute -translate-x-1/2 -translate-y-1/2 whitespace-nowrap rounded-lg border border-line-strong bg-surface px-2 py-1 text-[11px] font-medium text-ink-soft shadow-sm sm:px-2.5 sm:text-xs"
              style={{ left: `${NODE_POS[i].x}%`, top: `${NODE_POS[i].y}%` }}
            >
              {n}
            </span>
          ))}
        </div>
      </figure>

      {/* With KEEL: one hub, every module on the same data. */}
      <figure className="card relative flex flex-col p-0">
        <figcaption className="flex items-center justify-between border-b border-line px-5 py-3 text-sm">
          <span className="font-semibold text-ink">KEEL bilan</span>
          <span className="text-ink-muted">1 ta platforma</span>
        </figcaption>
        <div className="flex flex-1 flex-col items-center justify-center px-5 py-8">
          <span className="flex items-center gap-2 rounded-2xl bg-hull-900 px-5 py-3 text-white shadow-lg dark:bg-hull-800">
            <KeelMark className="h-7 w-7 text-signal-500" />
            <span className="font-display text-xl font-semibold tracking-tight">keel</span>
          </span>
          <span aria-hidden className="h-6 w-px bg-signal-500" />
          <div className="relative w-full max-w-md rounded-2xl border border-signal-500/50 p-3">
            <ul className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {p.hub.map((m) => (
                <li
                  key={m}
                  className="rounded-lg bg-signal-500/10 px-2 py-2 text-center text-[11px] font-semibold leading-tight text-ink sm:text-xs"
                >
                  {m}
                </li>
              ))}
            </ul>
            <p className="mt-3 text-center text-xs text-ink-muted">
              Bitta menyu · bitta mijozlar bazasi · bitta hisobot
            </p>
          </div>
        </div>
      </figure>
    </div>
  );
}

export function Pains() {
  return (
    <ul className="mt-5 grid gap-4 md:grid-cols-3">
      {PITCH.problem.pains.map((x) => (
        <li key={x.title} className="card">
          <p className="font-semibold text-ink">{x.title}</p>
          <p className="mt-1.5 text-sm text-ink-soft">{x.desc}</p>
        </li>
      ))}
    </ul>
  );
}

// ───────────────────────────── How it works

export function Flow() {
  return (
    <ol className="grid gap-x-6 gap-y-10 md:grid-cols-2 lg:grid-cols-3">
      {PITCH.flow.steps.map((s, i) => (
        <li key={s.n} className="flex flex-col">
          <div className="flex items-center gap-3">
            <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-signal-500 font-display text-sm font-bold text-hull-950">
              {s.n}
            </span>
            <h3 className="font-display text-lg font-semibold text-ink">{s.title}</h3>
            {i < PITCH.flow.steps.length - 1 && (
              <span aria-hidden className="ml-auto hidden h-px flex-1 bg-gradient-to-r from-signal-500/60 to-transparent lg:block" />
            )}
          </div>
          <p className="mt-2 min-h-[3.75rem] text-sm text-ink-soft">{s.desc}</p>
          <Shot name={s.shot} alt={s.alt} kind={s.shot === "pay" ? "screen" : "browser"} className="mt-4" />
        </li>
      ))}
    </ol>
  );
}

// ───────────────────────────── Modules

export function Modules() {
  const m = PITCH.modules;
  return (
    <>
      {/* Six tracks on lg: three groups of two, then two of three — five cards
          with no hole in the second row. */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-6">
        {m.groups.map((g, i) => (
          <section
            key={g.name}
            className={`card ${i < 3 ? "lg:col-span-2" : "lg:col-span-3"} ${i === 4 ? "sm:col-span-2 lg:col-span-3" : ""}`}
            aria-label={g.name}
          >
            <h3 className="eyebrow">{g.name}</h3>
            <ul className="mt-3 space-y-3">
              {g.items.map((it) => (
                <li key={it.name}>
                  <p className="font-semibold text-ink">{it.name}</p>
                  <p className="text-sm text-ink-soft">{it.desc}</p>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
      <div className="mt-10 grid grid-cols-2 items-end gap-5 lg:grid-cols-[1.5fr_1.5fr_.75fr_.75fr]">
        {m.gallery.map((g) => (
          <figure key={g.shot} className={g.kind === "browser" ? "col-span-2 sm:col-span-1" : ""}>
            <Shot name={g.shot} alt={g.alt} kind={g.kind as "browser" | "phone"} />
            <figcaption className="mt-2 text-center text-xs text-ink-muted">{g.label}</figcaption>
          </figure>
        ))}
      </div>
    </>
  );
}

// ───────────────────────────── Working product

export function Stages({ items }: { items: readonly string[] }) {
  return (
    <ol className="flex flex-wrap items-center gap-2" aria-label="Bosib o'tilgan bosqichlar">
      {items.map((s, i) => (
        <li key={s} className="flex items-center gap-2">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/40 bg-emerald-500/10 px-3 py-1.5 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={3} aria-hidden>
              <path d="M20 6 9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            {s}
          </span>
          {i < items.length - 1 && <span aria-hidden className="text-ink-muted">→</span>}
        </li>
      ))}
    </ol>
  );
}

export function LiveLinks() {
  return (
    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {PITCH.live.links.map((l) => (
        <li key={l.href}>
          <a
            href={l.href}
            className="group flex h-full items-center justify-between gap-4 rounded-2xl border border-line bg-surface px-5 py-4 transition hover:border-signal-500/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-signal-500"
          >
            <span>
              <span className="block font-semibold text-ink">{l.label}</span>
              <span className="block text-sm text-ink-muted">{l.desc}</span>
            </span>
            <span aria-hidden className="text-lg text-signal-600 transition group-hover:translate-x-1 motion-reduce:transition-none dark:text-signal-400">
              →
            </span>
          </a>
        </li>
      ))}
    </ul>
  );
}

// ───────────────────────────── Roadmap

const STATE_STYLE = {
  done: {
    dot: "bg-signal-500 text-hull-950 border-signal-500",
    tag: "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  },
  current: {
    dot: "bg-surface text-signal-600 border-signal-500 ring-4 ring-signal-500/25",
    tag: "bg-signal-500 text-hull-950",
  },
  next: {
    dot: "bg-surface text-ink-muted border-line-strong border-dashed",
    tag: "bg-raised text-ink-muted border border-line",
  },
} as const;

export function Roadmap() {
  const r = PITCH.roadmap;
  return (
    <ol className="relative grid gap-6 lg:grid-cols-7 lg:gap-3">
      {/* The rail: vertical on a phone, horizontal from lg. */}
      <span aria-hidden className="absolute bottom-2 left-[15px] top-2 w-px bg-line-strong lg:hidden" />
      <span aria-hidden className="absolute left-4 right-4 top-[15px] hidden h-px bg-line-strong lg:block" />
      {r.stages.map((s, i) => {
        const st = STATE_STYLE[s.state as keyof typeof STATE_STYLE];
        return (
          <li key={s.name} className="relative flex gap-4 lg:flex-col lg:gap-3">
            <span
              className={`relative z-10 grid h-8 w-8 shrink-0 place-items-center rounded-full border-2 text-xs font-bold ${st.dot}`}
              aria-hidden
            >
              {s.state === "done" ? "✓" : i + 1}
            </span>
            <div>
              <span className={`inline-block rounded-full px-2 py-0.5 text-[11px] font-bold uppercase tracking-wide ${st.tag}`}>
                {r.stateLabel[s.state as keyof typeof r.stateLabel]}
              </span>
              <h3 className="mt-1.5 font-display font-semibold text-ink">{s.name}</h3>
              <p className="mt-1 text-sm text-ink-soft">{s.desc}</p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

// ───────────────────────────── Architecture

export function Architecture() {
  const t = PITCH.tech;
  const [clients, edge, server, data, control] = t.layers;
  const layer = (l: (typeof t.layers)[number], tone = "") => (
    <div className={`rounded-2xl border p-4 ${tone || "border-line bg-surface"}`}>
      <p className="eyebrow">{l.name}</p>
      <ul className="mt-2 flex flex-wrap gap-1.5">
        {l.items.map((it) => (
          <li key={it} className="rounded-lg border border-line bg-raised px-2.5 py-1 text-xs text-ink-soft">
            {it}
          </li>
        ))}
      </ul>
    </div>
  );
  const arrow = (
    <span aria-hidden className="mx-auto block h-5 w-px bg-signal-500">
      <span className="sr-only">↓</span>
    </span>
  );
  return (
    <figure aria-label={t.archTitle} className="grid gap-5 lg:grid-cols-[1fr_320px]">
      <div>
        {layer(clients)}
        {arrow}
        {layer(edge)}
        {arrow}
        {layer(server, "border-signal-500/60 bg-signal-500/5")}
        {arrow}
        {layer(data)}
      </div>
      <div className="flex flex-col gap-5">
        {layer(control)}
        <div className="rounded-2xl border border-dashed border-line-strong p-4">
          <p className="eyebrow">Tashqi xizmatlar</p>
          <p className="mt-2 text-sm text-ink-soft">
            To'lov, fiskal kassa, tashqi POS, yetkazish, SMS va telefoniya, Telegram, EDI va 1C — restoran serveridan ulanadi.
          </p>
          <a href="#integrations" className="mt-2 inline-block text-sm font-semibold text-signal-600 hover:underline dark:text-signal-400">
            Ro'yxat →
          </a>
        </div>
      </div>
    </figure>
  );
}

export function DevStages() {
  const t = PITCH.tech;
  return (
    <ol className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {t.stages.map((s, i) => {
        const st = STATE_STYLE[s.state as keyof typeof STATE_STYLE];
        return (
          <li key={s.name} className="card flex gap-3 p-4">
            <span className={`grid h-7 w-7 shrink-0 place-items-center rounded-full border-2 text-xs font-bold ${st.dot}`} aria-hidden>
              {s.state === "done" ? "✓" : i + 1}
            </span>
            <div>
              <p className="font-semibold text-ink">
                {s.name}{" "}
                <span className={`ml-1 inline-block rounded-full px-2 py-0.5 align-middle text-[10px] font-bold uppercase tracking-wide ${st.tag}`}>
                  {PITCH.roadmap.stateLabel[s.state as keyof typeof PITCH.roadmap.stateLabel]}
                </span>
              </p>
              <p className="mt-1 text-sm text-ink-soft">{s.desc}</p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

export function AiBlock() {
  const t = PITCH.tech;
  return (
    <div className="grid gap-5 lg:grid-cols-[1.3fr_1fr]">
      <div className="card">
        <h3 className="font-display text-lg font-semibold text-ink">{t.aiTitle}</h3>
        <p className="mt-1 text-sm text-ink-soft">{t.aiLead}</p>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2">
          {t.aiNow.map((a) => (
            <li key={a.name} className="flex gap-2.5">
              <Check />
              <span>
                <span className="block text-sm font-semibold text-ink">{a.name}</span>
                <span className="block text-sm text-ink-soft">{a.desc}</span>
              </span>
            </li>
          ))}
        </ul>
      </div>
      <div className="flex flex-col gap-5">
        <div className="card">
          <p className="eyebrow">Ishlab chiqishda</p>
          <p className="mt-2 text-sm text-ink-soft">{t.aiDev}</p>
        </div>
        <div className="card border-dashed">
          <p className="eyebrow !text-ink-muted">Rejada</p>
          <p className="mt-2 text-sm text-ink-soft">{t.aiPlanned}</p>
        </div>
      </div>
    </div>
  );
}

export function Principles() {
  return (
    <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
      {PITCH.tech.principles.map((p) => (
        <li key={p.name} className="rounded-2xl border border-line bg-surface p-4">
          <p className="font-semibold text-ink">{p.name}</p>
          <p className="mt-1 text-sm text-ink-soft">{p.desc}</p>
        </li>
      ))}
    </ul>
  );
}

// ───────────────────────────── Integrations

const INT_TONE = {
  built: "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  testing: "border-signal-500/50 bg-signal-500/10 text-signal-600 dark:text-signal-400",
  planned: "border-line-strong bg-raised text-ink-muted",
} as const;

export function Integrations() {
  return (
    <div className="grid gap-5 lg:grid-cols-3">
      {PITCH.integrations.groups.map((g) => (
        <section key={g.state} className="card" aria-label={g.label}>
          <span className={`inline-block rounded-full border px-3 py-1 text-xs font-bold ${INT_TONE[g.state as keyof typeof INT_TONE]}`}>
            {g.label}
          </span>
          <ul className="mt-4 space-y-3">
            {g.items.map((it) => (
              <li key={it.name}>
                <p className="text-sm font-semibold text-ink">{it.name}</p>
                <p className="text-sm text-ink-soft">{it.list}</p>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

/** The headline with its accent half, for blocks that are not `Section`s. */
export function Headline({ text, className = "" }: { text: string; className?: string }) {
  return <h2 className={`h-display text-3xl sm:text-4xl ${className}`}>{accent(text)}</h2>;
}

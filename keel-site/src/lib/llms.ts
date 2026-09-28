// keel.uz for language models: `/llms.txt` and `/llms-full.txt`.
//
// ⚠️ **Built from the same sources the pages are, on every request.** The index
// and the full text are not files anybody edits: the help articles come from
// `lib/help` (the same blocks the knowledge base renders), the product pages
// from the dictionary the landing is drawn from, and the blog from the control
// plane. A hand-written llms.txt is correct on the day it is written and wrong
// after the next article — which is the sitemap's lesson, and the reason
// `sitemap.ts` is generated too.
//
// ⚠️ **One `URL:` line under every page heading in the full file, and the
// control plane depends on it.** The console's watcher (control/handlers/
// llms.go) splits this file on those lines, hashes each page, and pings the
// search engines for exactly the pages whose text changed. Renaming the marker
// here silently turns every change into "the whole file changed".
//
// ⚠️ **Plain markdown, no HTML, no pictures.** A model reads words; a
// screenshot's numbered callouts are kept as a list, because the sentence
// beside callout 2 is the useful half of the figure.

import type { Lang } from "@/lib/i18n/dict";
import { dicts } from "@/lib/i18n/dict";
import { ALL_LANGS, ORIGIN, localeUrl } from "@/lib/i18n/url";
import { help } from "@/lib/help";
import type { Article, Block } from "@/lib/help/types";
import { developers } from "@/lib/developers";

/** A published blog post, as much of it as the files need. */
export type LlmsPost = {
  slug: string;
  title: string;
  excerpt: string;
  body?: string;
  publishedAt?: string;
};

/** The marker the control plane splits the full file on. See the note above. */
export const URL_MARKER = "URL: ";

const words: Record<
  Lang,
  {
    summary: string;
    about: string;
    pages: string;
    help: string;
    blog: string;
    docs: string;
    languages: string;
    optional: string;
    full: string;
    landing: string;
    kassa: string;
    download: string;
    developers: string;
    offer: string;
    privacy: string;
    who: string;
    features: string;
    pricing: string;
    tills: string;
    shops: string;
    faq: string;
    see: string;
    published: string;
    perMonth: string;
    screen: string;
  }
> = {
  uz: {
    summary:
      "Keel — O'zbekistondagi restoran, kafe va do'konlar uchun kassa, zal, oshxona ekrani, ombor, o'z sayti, yetkazib berish va Telegram bot bitta tizimda.",
    about:
      "Bu fayl sun'iy intellekt yordamchilari uchun: keel.uz sahifalarining qisqa ro'yxati. Har sahifaning to'liq matni /llms-full.txt da. Narxlar so'mda (UZS).",
    pages: "Asosiy sahifalar",
    help: "Qo'llanma",
    blog: "Blog",
    docs: "Hujjatlar",
    languages: "Boshqa tillar",
    optional: "Optional",
    full: "Barcha sahifalarning to'liq matni",
    landing: "Bosh sahifa",
    kassa: "Kassa",
    download: "Kassa dasturini yuklab olish",
    developers: "Ochiq API (dasturchilar uchun)",
    offer: "Ommaviy oferta",
    privacy: "Maxfiylik siyosati",
    who: "Kimlar uchun",
    features: "Imkoniyatlar",
    pricing: "Onlayn buyurtmalar narxi",
    tills: "Kassa tariflari (restoran)",
    shops: "Kassa tariflari (do'kon)",
    faq: "Ko'p so'raladigan savollar",
    see: "Shuningdek",
    published: "Chop etilgan",
    perMonth: "so'm / oy",
    screen: "Ekranda",
  },
  ru: {
    summary:
      "Keel — касса, зал, кухонный экран, склад, собственный сайт, доставка и Telegram-бот в одной системе для ресторанов, кафе и магазинов Узбекистана.",
    about:
      "Этот файл — для ИИ-ассистентов: краткий список страниц keel.uz. Полный текст каждой страницы — в /ru/llms-full.txt. Цены в сумах (UZS).",
    pages: "Основные страницы",
    help: "Руководство",
    blog: "Блог",
    docs: "Документы",
    languages: "Другие языки",
    optional: "Optional",
    full: "Полный текст всех страниц",
    landing: "Главная",
    kassa: "Касса",
    download: "Скачать программу кассы",
    developers: "Открытый API (для разработчиков)",
    offer: "Публичная оферта",
    privacy: "Политика конфиденциальности",
    who: "Для кого",
    features: "Возможности",
    pricing: "Стоимость онлайн-заказов",
    tills: "Тарифы кассы (ресторан)",
    shops: "Тарифы кассы (магазин)",
    faq: "Частые вопросы",
    see: "См. также",
    published: "Опубликовано",
    perMonth: "сум / мес",
    screen: "На экране",
  },
  en: {
    summary:
      "Keel is one system for restaurants, cafés and shops in Uzbekistan: point of sale, floor plan, kitchen screen, stock, the business's own website, delivery and a Telegram bot.",
    about:
      "This file is for AI assistants: a short index of keel.uz. The full text of every page is in /en/llms-full.txt. Prices are in Uzbek som (UZS).",
    pages: "Main pages",
    help: "Help centre",
    blog: "Blog",
    docs: "Legal",
    languages: "Other languages",
    optional: "Optional",
    full: "Full text of every page",
    landing: "Home",
    kassa: "Point of sale",
    download: "Download the till app",
    developers: "Open API (for developers)",
    offer: "Public offer",
    privacy: "Privacy policy",
    who: "Who it is for",
    features: "Features",
    pricing: "Online order pricing",
    tills: "Till plans (restaurants)",
    shops: "Till plans (shops)",
    faq: "Frequently asked questions",
    see: "See also",
    published: "Published",
    perMonth: "som / month",
    screen: "On screen",
  },
};

// ---- Small helpers ----

/** The dictionary's emphasis marks (`*word*`) are for the page's colour, not
 *  for a reader. */
function plain(s: string): string {
  return s.replace(/\*/g, "").trim();
}

/** A help article's inline marks as markdown: `*bold*` becomes `**bold**`,
 *  `` `button` `` already is markdown. */
function inline(s: string): string {
  return s.replace(/\*([^*\n]+)\*/g, "**$1**");
}

function cell(s: string): string {
  return inline(s).replace(/\|/g, "\\|").replace(/\n/g, " ");
}

/** Blog text points at its own pictures with site-relative paths; a model
 *  reading the file elsewhere needs the host. */
function absolute(s: string): string {
  return s.replace(/\]\(\//g, `](${ORIGIN}/`);
}

function link(title: string, url: string, note?: string): string {
  const n = note ? plain(note) : "";
  return `- [${plain(title)}](${url})${n ? `: ${n}` : ""}`;
}

const langName: Record<Lang, string> = { uz: "O'zbekcha", ru: "Русский", en: "English" };

// ---- Help articles ----

function blockMd(b: Block, lang: Lang, titleOf: (slug: string) => string): string {
  const w = words[lang];
  if ("p" in b) return inline(b.p);
  if ("h" in b) return `### ${plain(b.h)}`;
  if ("steps" in b) return b.steps.map((s, i) => `${i + 1}. ${inline(s)}`).join("\n");
  if ("list" in b) return b.list.map((s) => `- ${inline(s)}`).join("\n");
  if ("warn" in b) return `> ⚠️ ${inline(b.warn)}`;
  if ("tip" in b) return `> 💡 ${inline(b.tip)}`;
  if ("fig" in b) {
    const notes = Object.values(b.notes ?? {});
    if (!notes.length) return "";
    return `${w.screen}:\n${notes.map((n) => `- ${inline(n)}`).join("\n")}`;
  }
  if ("table" in b) {
    const { head, rows } = b.table;
    return [
      `| ${head.map(cell).join(" | ")} |`,
      `| ${head.map(() => "---").join(" | ")} |`,
      ...rows.map((r) => `| ${r.map(cell).join(" | ")} |`),
    ].join("\n");
  }
  if ("see" in b) {
    return `${w.see}: ${b.see
      .map((s) => `[${titleOf(s)}](${localeUrl(lang, `/help/${s}`)})`)
      .join(", ")}`;
  }
  return "";
}

function articleMd(a: Article, lang: Lang, titleOf: (slug: string) => string): string {
  const body = a.body
    .map((b) => blockMd(b, lang, titleOf))
    .filter(Boolean)
    .join("\n\n");
  return `${inline(a.lead)}\n\n${body}`;
}

// ---- The two files ----

function pageList(lang: Lang) {
  const w = words[lang];
  const t = dicts[lang];
  return [
    { path: "/", title: w.landing, note: t.hero.lead },
    { path: "/kassa", title: w.kassa, note: t.till.lead },
    { path: "/download", title: w.download, note: t.download.lead },
    { path: "/developers", title: w.developers, note: developers[lang].description },
    { path: "/help", title: w.help, note: help[lang].ui.lead },
    { path: "/blog", title: w.blog, note: t.blog.lead },
  ];
}

/** `/llms.txt` — the index, per the llmstxt.org shape: a title, a one-line
 *  summary, a paragraph, then sections of links. */
export function buildLlmsIndex(lang: Lang, posts: LlmsPost[]): string {
  const w = words[lang];
  const h = help[lang];
  const out: string[] = [];

  out.push("# Keel", "", `> ${w.summary}`, "", w.about, "");

  out.push(`## ${w.pages}`, "");
  for (const p of pageList(lang)) out.push(link(p.title, localeUrl(lang, p.path), p.note));
  out.push(link(w.full, localeUrl(lang, "/llms-full.txt")), "");

  // The knowledge base, section by section — the order the site shows it in.
  for (const s of h.sections) {
    const list = h.articles.filter((a) => a.section === s.id);
    if (!list.length) continue;
    out.push(`## ${w.help}: ${plain(s.title)}`, "");
    for (const a of list) out.push(link(a.title, localeUrl(lang, `/help/${a.slug}`), a.lead));
    out.push("");
  }

  if (posts.length) {
    out.push(`## ${w.blog}`, "");
    for (const p of posts) out.push(link(p.title, localeUrl(lang, `/blog/${p.slug}`), p.excerpt));
    out.push("");
  }

  out.push(`## ${w.docs}`, "");
  out.push(link(w.offer, localeUrl(lang, "/public-offer")));
  out.push(link(w.privacy, localeUrl(lang, "/privacy-policy")), "");

  // ⚠️ "Optional" is the spec's own word for what a short context may skip,
  // so it stays in English in all three files.
  out.push(`## ${w.optional}`, "");
  for (const l of ALL_LANGS) {
    if (l === lang) continue;
    out.push(link(`${w.languages}: ${langName[l]}`, localeUrl(l, "/llms.txt")));
  }
  out.push("");
  return out.join("\n");
}

/** One page of the full file. The `URL:` line is what the control plane keys
 *  its change detection on — see the note at the top. */
function section(title: string, url: string, body: string): string {
  return `## ${plain(title)}\n\n${URL_MARKER}${url}\n\n${body.trim()}\n`;
}

function landingMd(lang: Lang): string {
  const t = dicts[lang];
  const w = words[lang];
  const parts: string[] = [];
  parts.push(`${plain(t.hero.title)}.`, "", plain(t.hero.lead), "", plain(t.hero.note));
  parts.push("", `### ${w.who}`, "");
  for (const i of t.who.items) parts.push(`- **${i.name}** — ${i.desc}`);
  parts.push("", `### ${w.features}`, "", plain(t.features.lead), "");
  for (const i of t.features.items) parts.push(`- **${i.name}** — ${i.desc}`);
  parts.push("", `### ${w.pricing}`, "", plain(t.pricing.lead), "");
  for (const i of t.pricing.tiers) parts.push(`- ${i.range}: ${i.price}`);
  parts.push("", plain(t.pricing.tiersNote), "");
  for (const i of t.pricing.included) parts.push(`- ${i}`);
  parts.push("", `**${plain(t.pricing.setupTitle)}** — ${plain(t.pricing.setupDesc)}`);
  parts.push("", `**${plain(t.pricing.chainsTitle)}** — ${plain(t.pricing.chainsDesc)}`);
  parts.push("", `### ${w.faq}`, "");
  for (const f of t.faq.items) parts.push(`**${plain(f.q)}**`, "", plain(f.a), "");
  return parts.join("\n");
}

function kassaMd(lang: Lang): string {
  const t = dicts[lang].till;
  const w = words[lang];
  const parts: string[] = [`${plain(t.title)}.`, "", plain(t.lead), ""];
  for (const s of t.screens) parts.push(`- **${s.name}** — ${s.desc}`);
  parts.push("", `### ${plain(t.featuresTitle)}`, "");
  for (const f of t.features) parts.push(`- **${f.name}** — ${f.desc}`);
  parts.push("", `### ${w.tills}`, "", plain(t.plansLead), "");
  for (const p of t.plans) {
    parts.push(`- **${p.name}** — ${p.price} ${w.perMonth}, ${p.registers}: ${p.includes}`);
  }
  parts.push("", `### ${w.shops}`, "", plain(t.shopLead), "");
  for (const p of t.shopPlans) {
    parts.push(`- **${p.name}** — ${p.price} ${w.perMonth}, ${p.registers}: ${p.includes}`);
  }
  return parts.join("\n");
}

function downloadMd(lang: Lang): string {
  const d = dicts[lang].download;
  return [
    plain(d.lead),
    "",
    `### ${plain(d.stepsTitle)}`,
    "",
    `1. ${d.step1}`,
    `2. ${d.step2}`,
    `3. ${d.step3}`,
    "",
    `### ${plain(d.reqTitle)}`,
    "",
    `- ${d.req1}`,
    `- ${d.req2}`,
    `- ${d.req3}`,
    "",
    `${plain(d.noteTitle)}: ${d.note}`,
  ].join("\n");
}

function developersMd(lang: Lang): string {
  const doc = developers[lang];
  const parts: string[] = [plain(doc.lead)];
  for (const s of doc.sections) {
    parts.push("", `### ${plain(s.title)}`);
    for (const b of s.blocks) {
      parts.push("");
      if (b.kind === "p" || b.kind === "note") parts.push(b.kind === "note" ? `> ${b.text}` : b.text);
      else if (b.kind === "list") parts.push(b.items.map((i) => `- ${i}`).join("\n"));
      else if (b.kind === "code") parts.push("```" + "\n" + b.code.trim() + "\n" + "```");
      else if (b.kind === "table") {
        parts.push(
          [
            `| ${b.head.map(cell).join(" | ")} |`,
            `| ${b.head.map(() => "---").join(" | ")} |`,
            ...b.rows.map((r) => `| ${r.map(cell).join(" | ")} |`),
          ].join("\n"),
        );
      }
    }
  }
  return parts.join("\n");
}

/** `/llms-full.txt` — every page's words, one `##` section per page. */
export function buildLlmsFull(lang: Lang, posts: LlmsPost[]): string {
  const w = words[lang];
  const h = help[lang];
  const titles = new Map(h.articles.map((a) => [a.slug, plain(a.title)]));
  const titleOf = (slug: string) => titles.get(slug) ?? slug;
  const out: string[] = [`# Keel — ${w.full}`, "", `> ${w.summary}`, ""];

  out.push(section(w.landing, localeUrl(lang, "/"), landingMd(lang)));
  out.push(section(w.kassa, localeUrl(lang, "/kassa"), kassaMd(lang)));
  out.push(section(w.download, localeUrl(lang, "/download"), downloadMd(lang)));
  out.push(section(w.developers, localeUrl(lang, "/developers"), developersMd(lang)));

  for (const s of h.sections) {
    for (const a of h.articles.filter((x) => x.section === s.id)) {
      out.push(
        section(
          `${w.help} › ${plain(s.title)} › ${a.title}`,
          localeUrl(lang, `/help/${a.slug}`),
          articleMd(a, lang, titleOf),
        ),
      );
    }
  }

  for (const p of posts) {
    const date = p.publishedAt ? `${w.published}: ${p.publishedAt.slice(0, 10)}\n\n` : "";
    out.push(
      section(
        `${w.blog} › ${p.title}`,
        localeUrl(lang, `/blog/${p.slug}`),
        `${date}${p.excerpt ? `${p.excerpt}\n\n` : ""}${absolute(p.body ?? "")}`,
      ),
    );
  }
  return out.join("\n");
}

/** How the two routes answer: markdown, readable in a browser, short cache. */
export function markdownResponse(text: string): Response {
  return new Response(text, {
    headers: {
      // ⚠️ `text/plain` rather than `text/markdown`: a browser offers to
      // download `text/markdown` instead of showing it, and the person most
      // likely to open this file by hand is checking what it says.
      "content-type": "text/plain; charset=utf-8",
      // Five minutes at the edge: the blog behind it is cached for one, and a
      // crawler hammering this route must not become a control-plane load.
      "cache-control": "public, max-age=300",
    },
  });
}

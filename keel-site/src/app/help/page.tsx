// keel.uz/help — the manual.
//
// ⚠️ **On the marketing site rather than inside the panel, and that is a
// deliberate second home.** The panel already carries short answers for the
// screen somebody is standing on (`lib/help/articles.ts` there). This is the
// other half: the long form, findable from a search engine, readable before
// anybody has an account, and linkable from a support reply to a person who
// cannot log in — which is a large share of the calls that reach us.
//
// ⚠️ **Server-rendered, in three languages, each with its own address.** A
// knowledge base that only exists after JavaScript runs is one no crawler ever
// reads, and a restaurant owner searching "техкарта как составить" is exactly
// the reader this page is worth writing for.

import type { Metadata } from "next";
import Link from "next/link";
import Header from "@/components/Header";
import SearchBox, { type SearchItem } from "@/components/help/SearchBox";
import SectionIcon from "@/components/help/SectionIcon";
import { HelpIndexJsonLd } from "@/components/help/HelpJsonLd";
import { help } from "@/lib/help";
import { getLang, getPath } from "@/lib/i18n/server";
import { alternatesFor, localePath } from "@/lib/i18n/url";
import { TELEGRAM } from "@/lib/links";

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  const path = await getPath();
  const t = help[lang].ui;
  return {
    title: `${t.title} · Keel`,
    description: t.lead,
    alternates: alternatesFor(lang, path),
  };
}

export default async function HelpIndex() {
  const lang = await getLang();
  const { ui, sections, articles } = help[lang];

  const items: SearchItem[] = articles.map((a) => ({
    slug: a.slug,
    title: a.title,
    lead: a.lead,
    section: a.section,
    hay: [a.title, a.lead, ...(a.keys ?? [])].join(" ").toLowerCase(),
  }));

  return (
    <>
      <HelpIndexJsonLd
        lang={lang}
        title={ui.title}
        lead={ui.lead}
        sections={sections}
      />
      <Header />
      <main className="container-page py-14 sm:py-20">
        <p className="eyebrow">Keel</p>
        <h1 className="h-display mt-2 text-3xl sm:text-4xl">{ui.title}</h1>
        <p className="mt-3 max-w-2xl text-lg text-ink-soft">{ui.lead}</p>

        <div className="mt-7 max-w-xl">
          <SearchBox
            items={items}
            lang={lang}
            placeholder={ui.searchPlaceholder}
            empty={ui.searchEmpty}
            count={ui.searchCount}
          />
        </div>

        <div className="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {sections.map((s) => {
            const list = articles.filter((a) => a.section === s.id);
            if (list.length === 0) return null;
            return (
              <section key={s.id} className="card">
                {/* The icon and the title on one line: an icon stacked above a
                    heading pushes every card taller by a row, and thirteen
                    cards is where that stops being free. */}
                <div className="flex items-center gap-2.5">
                  <span className="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-signal-500/10 text-signal-600 dark:text-signal-400">
                    <SectionIcon id={s.id} />
                  </span>
                  <h2 className="h-display text-lg">{s.title}</h2>
                </div>
                <p className="mt-2 text-sm text-ink-muted">{s.lead}</p>
                <ul className="mt-4 space-y-1.5">
                  {list.map((a) => (
                    <li key={a.slug}>
                      <Link
                        href={localePath(lang, `/help/${a.slug}`)}
                        className="block text-[15px] text-ink-soft underline-offset-4 transition hover:text-ink hover:underline"
                      >
                        {a.title}
                      </Link>
                    </li>
                  ))}
                </ul>
              </section>
            );
          })}
        </div>

        {/* ⚠️ At the bottom of the index and at the bottom of every article.
            The manual answers the questions somebody thought to ask; the way
            out has to be on the page they are on when they run out. */}
        <div className="card mt-12 max-w-2xl">
          <h2 className="h-display text-lg">{ui.askUs}</h2>
          <p className="mt-1 text-ink-soft">{ui.askUsLead}</p>
          <a href={TELEGRAM} className="btn-primary mt-4">
            Telegram
          </a>
        </div>
      </main>
    </>
  );
}

// keel.uz/download — where a restaurant gets the till program.
//
// ⚠️ **One file for every restaurant, and that is the whole point of this
// page.** The alternative that suggests itself — a build per customer, with the
// server address compiled in — turns every new restaurant into a release, and
// the customer whose build was skipped finds out when their till cannot sell.
// The program asks which restaurant it belongs to on first launch instead, so
// the same download works everywhere and this page can simply exist.

import type { Metadata } from "next";
import Header from "@/components/Header";
import { getT } from "@/lib/i18n/server";
import { VERSION } from "@/lib/version";

// ⚠️ **Not NEXT_PUBLIC_, and this page is a server component so it does not
// need to be.** A NEXT_PUBLIC_ value is sealed into the bundle at build time,
// which means setting it in docker-compose.saas.yml would do nothing at all —
// the same trap GOOGLE_SITE_VERIFICATION and META_PIXEL_ID are commented
// against two services above. Read at render time, so the release URL is a
// deploy setting rather than a rebuild.
const DOWNLOAD_URL = process.env.TILL_DOWNLOAD_URL ?? "";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: `${t.download.title} · Keel`, description: t.download.lead };
}

export default async function DownloadPage() {
  const t = await getT();

  return (
    <>
      <Header />
      <main className="container-page py-16 sm:py-24">
        <p className="eyebrow">{t.download.eyebrow}</p>
        <h1 className="section-title mt-2">{t.download.title}</h1>
        <p className="mt-4 max-w-2xl text-lg text-ink-soft">{t.download.lead}</p>

        {/* ⚠️ The button is only a button when there is something behind it.
            A download link that 404s costs more trust than an honest "not
            yet" — the person clicking it is deciding whether to run their
            restaurant on this. */}
        <div className="mt-8">
          {DOWNLOAD_URL ? (
            <>
              <a href={DOWNLOAD_URL} className="btn btn-primary text-base">
                {t.download.button}
                {/* ⚠️ The product's one version, not a separate number for the
                    download: four artefacts that version independently give a
                    support call four things to establish before it starts. */}
                <span className="ml-2 opacity-70">{VERSION}</span>
              </a>
              <p className="mt-3 text-sm text-ink-muted">{t.download.perBranch}</p>
            </>
          ) : (
            <div className="card max-w-xl p-5">
              <p className="font-medium">{t.download.soonTitle}</p>
              <p className="mt-1 text-ink-soft">{t.download.soon}</p>
            </div>
          )}
        </div>

        <section className="mt-16 grid gap-10 sm:grid-cols-2">
          <div>
            <h2 className="text-lg font-semibold">{t.download.stepsTitle}</h2>
            <ol className="mt-4 space-y-4">
              {[t.download.step1, t.download.step2, t.download.step3].map((step, i) => (
                <li key={i} className="flex gap-3">
                  <span
                    aria-hidden
                    className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-ink text-xs font-semibold text-cream"
                  >
                    {i + 1}
                  </span>
                  <span className="text-ink-soft">{step}</span>
                </li>
              ))}
            </ol>
          </div>

          <div>
            <h2 className="text-lg font-semibold">{t.download.reqTitle}</h2>
            <ul className="mt-4 space-y-2 text-ink-soft">
              {[t.download.req1, t.download.req2, t.download.req3].map((req) => (
                <li key={req} className="flex gap-2">
                  <span aria-hidden className="text-ink-muted">
                    ·
                  </span>
                  {req}
                </li>
              ))}
            </ul>

            <div className="card mt-8 p-5">
              <p className="font-medium">{t.download.noteTitle}</p>
              <p className="mt-1 text-sm text-ink-soft">{t.download.note}</p>
            </div>
          </div>
        </section>
      </main>
    </>
  );
}

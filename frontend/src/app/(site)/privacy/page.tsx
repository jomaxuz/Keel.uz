import { api } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { getTranslations } from "@/lib/i18n/server";
import { privacyDoc } from "@/lib/privacy";
import type { RestaurantResponse } from "@/lib/types";

export async function generateMetadata() {
  const { lang } = await getTranslations();
  // The title comes from the document itself, so the tab and the heading cannot disagree.
  return { title: privacyDoc(lang, "", []).title };
}

export default async function PrivacyPage() {
  const { lang, t } = await getTranslations();

  let data: RestaurantResponse | null = null;
  try {
    data = await api.getRestaurant(await getSiteScope());
  } catch {
    // Backend unreachable. ⚠️ The page still renders: this is the document a guest opens
    // *because* they are worried about their data, and a blank page at that moment is the
    // worst possible answer. Only the name and the phone number are missing, and the
    // fallbacks say so honestly.
  }
  const rest = data?.restaurant;
  const who = rest?.name || t.common.restaurant;

  // How to reach the restaurant about their data — the one thing that must be per-restaurant,
  // because the request goes to them and not to us.
  const contact = [rest?.phones?.[0], rest?.address?.text].filter(Boolean) as string[];
  const doc = privacyDoc(lang, who, contact);

  return (
    <main className="container-page py-12 sm:py-16">
      {/* Reading width, not page width: a legal document at full desktop width is a document
          nobody finishes, and this one is meant to be read rather than displayed. */}
      <div className="mx-auto max-w-3xl">
        <h1 className="font-display text-3xl font-bold text-ink sm:text-4xl">{doc.title}</h1>
        {/* ⚠️ The date stays at the top. For a document somebody may have relied on months
            ago, which version they read is part of what they were told. */}
        <p className="mt-2 text-sm text-ink-muted">{doc.updated}</p>
        <p className="mt-6 text-base leading-relaxed text-ink-soft">{doc.intro}</p>

        <div className="mt-10 space-y-9">
          {doc.sections.map((s) => (
            <section key={s.title}>
              <h2 className="font-display text-xl font-bold text-ink">{s.title}</h2>
              <div className="mt-3 space-y-2.5 text-sm leading-relaxed text-ink-soft">
                {s.body.map((line, i) =>
                  // Lines written as "— …" are list items and are rendered as such: a run of
                  // eight dashes inside one paragraph is where a reader gives up.
                  line.startsWith("— ") ? (
                    <p key={i} className="pl-4 -indent-4">
                      {line}
                    </p>
                  ) : (
                    <p key={i}>{line}</p>
                  ),
                )}
              </div>
            </section>
          ))}
        </div>
      </div>
    </main>
  );
}

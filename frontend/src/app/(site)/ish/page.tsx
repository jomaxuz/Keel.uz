// "We are hiring": the vacancies, and a form that asks for two things.
//
// ⚠️ On the site rather than only in a Telegram channel, because the people who see it are
// the guests. A waiter is hired from the same street the customers walk down, and a
// restaurant with this page fills a shift faster than one posting where nobody in that
// street is looking.

import { api } from "@/lib/api";
import { getTranslations } from "@/lib/i18n/server";
import JobsView from "@/components/site/JobsView";
import type { Vacancy } from "@/lib/types";

export async function generateMetadata() {
  const { t } = await getTranslations();
  return { title: t.jobs.title };
}

export default async function JobsPage() {
  const { t } = await getTranslations();

  let vacancies: Vacancy[] = [];
  try {
    vacancies = await api.vacancies();
  } catch {
    // Backend unreachable: the page says there is nothing open rather than failing the
    // route — which is also the honest answer to somebody who came looking for a job.
  }

  return (
    <main>
      <section className="border-b border-line bg-surface">
        <div className="container-page py-12 sm:py-16">
          <p className="eyebrow">{t.jobs.eyebrow}</p>
          <h1 className="mt-2 section-title">{t.jobs.title}</h1>
          <p className="mt-3 max-w-xl text-ink-muted">{t.jobs.subtitle}</p>
        </div>
      </section>

      <JobsView vacancies={vacancies} />
    </main>
  );
}

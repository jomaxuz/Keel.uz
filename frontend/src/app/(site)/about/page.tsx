import { api, imageUrl } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { weekdayName } from "@/lib/format";
import CallLink from "@/components/site/CallLink";
import ContactFeedback from "@/components/site/ContactFeedback";
import LiveMap from "@/components/map/LiveMap";
import RouteButtons from "@/components/map/RouteButtons";
import SocialLinks from "@/components/site/SocialLinks";
import { getTranslations } from "@/lib/i18n/server";
import { localized } from "@/lib/i18n/site-content";
import type { RestaurantResponse } from "@/lib/types";

export async function generateMetadata() {
  const { t } = await getTranslations();
  return { title: t.nav.about };
}

export default async function AboutPage() {
  const { lang, t } = await getTranslations();

  let data: RestaurantResponse | null = null;
  try {
    data = await api.getRestaurant(await getSiteScope());
  } catch {
    // backend unreachable
  }
  const rest = data?.restaurant;

  const hours = [...(rest?.workingHours ?? [])].sort(
    (a, b) => ((a.day + 6) % 7) - ((b.day + 6) % 7),
  );

  const cover = imageUrl(rest?.coverUrl, 1200);

  return (
    <main>
      <section className="relative overflow-hidden bg-charcoal text-white">
        {cover && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={cover}
            alt=""
            className="absolute inset-0 h-full w-full object-cover opacity-30"
          />
        )}
        <div className="absolute inset-0 bg-gradient-to-br from-charcoal via-charcoal/85 to-charcoal/40" />
        <div className="container-page relative py-16 sm:py-20">
          <p className="text-xs font-bold uppercase tracking-[0.18em] text-brand-light">
            {t.about.eyebrow}
          </p>
          <h1 className="mt-3 break-words font-display text-3xl font-bold tracking-tight sm:text-4xl lg:text-5xl">
            {localized(rest?.content?.aboutTitle, lang) ||
              rest?.name ||
              t.common.restaurant}
          </h1>
          {rest?.description && (
            <p className="mt-4 max-w-2xl leading-relaxed text-white/70">
              {rest.description}
            </p>
          )}
        </div>
      </section>

      {/* The restaurant's own story, written in the admin panel. */}
      {localized(rest?.content?.aboutText, lang) && (
        <section className="container-page pt-12">
          <div className="card whitespace-pre-line p-6 leading-relaxed text-ink-soft sm:p-8">
            {localized(rest?.content?.aboutText, lang)}
          </div>
        </section>
      )}

      <div className="container-page grid grid-cols-1 gap-6 py-12 md:grid-cols-2">
        <div className="card p-6 sm:p-8">
          <h2 className="font-display text-lg font-bold">{t.about.contact}</h2>
          <ul className="mt-3 space-y-2 text-sm text-ink-muted">
            {rest?.phones?.map((p) => (
              <li key={p}>
                <CallLink phone={p} className="hover:text-brand">
                  {p}
                </CallLink>
              </li>
            ))}
            {rest?.address?.text && <li>{rest.address.text}</li>}
          </ul>

          {/* Icons rather than three words: three logos read faster, and only what is
              filled in is drawn — a greyed-out logo says "we are on Facebook and
              neglecting it". */}
          <SocialLinks socials={rest?.socials} className="mt-5" />
        </div>

        {hours.length > 0 && (
          <div className="card p-6 sm:p-8">
            <h2 className="font-display text-lg font-bold">{t.about.hours}</h2>
            <div className="mt-3 divide-y divide-line">
              {hours.map((h) => (
                <div
                  key={h.day}
                  className="flex items-center justify-between py-2.5 text-sm"
                >
                  <span className="font-medium">{weekdayName(h.day, lang)}</span>
                  <span className="text-ink-muted">
                    {h.isClosed ? t.common.closed : `${h.open} – ${h.close}`}
                  </span>
                </div>
              ))}
            </div>
            {/* Xarita M4 (checkout) bosqichida qo'shiladi */}
          </div>
        )}
      </div>

      {/* ⚠️ The map is the answer to the question this page is opened with — "where is
          it" — and an address in text is that answer only for somebody who already
          knows the city. Drawn from the branch's own point, so a two-branch company
          shows the one the guest is looking at. */}
      {rest?.address?.lat && rest.address.lng ? (
        <section className="container-page pb-12">
          <div className="card overflow-hidden p-0">
            <LiveMap
              points={[
                {
                  id: "restaurant",
                  lat: rest.address.lat,
                  lng: rest.address.lng,
                  label: rest.name,
                  kind: "restaurant",
                },
              ]}
              fallbackCenter={{ lat: rest.address.lat, lng: rest.address.lng }}
              className="h-72 w-full sm:h-96"
            />
          </div>
          {/* The buttons a phone already has an app for. Cheaper than any map we could
              draw, and it is what somebody standing outside actually wants. */}
          <div className="mt-4">
            <RouteButtons
              target={{
                lat: rest.address.lat,
                lng: rest.address.lng,
                label: rest.address.text || rest.name,
              }}
            />
          </div>
        </section>
      ) : null}

      <section className="container-page pb-16">
        <ContactFeedback />
      </section>
    </main>
  );
}

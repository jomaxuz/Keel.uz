// Schema.org data, for the search result rather than the page.
//
// This is the part of SEO that actually changes what a person sees: a plain
// blue link, or a card with the opening hours, the phone number, the rating
// and a map pin. Google and Yandex both read it, and neither will infer any of
// it from the markup — an address in a `<p>` is a paragraph.
//
// Two rules kept it honest:
//
//   • **Only what the restaurant actually filled in.** An empty field is left
//     out rather than sent as "", because a schema full of blanks is how a
//     site gets its rich results turned off entirely.
//   • **`LocalBusiness`, narrowed to `Restaurant` only when it is one.** Keel
//     sells to pharmacies and flower shops too, and telling Google a florist
//     serves cuisine is the kind of wrong that gets structured data ignored.

import type { Restaurant } from "@/lib/types";

/** Maps the free-text business kind onto a schema.org type.
 *
 *  Unknown kinds fall back to LocalBusiness, which is always true and always
 *  safe. Guessing a narrower type would be worse than saying less. */
function schemaType(kind?: string): string {
  const k = (kind ?? "").toLowerCase();
  if (/restoran|restaurant|kafe|cafe|choyxona|osh/.test(k)) return "Restaurant";
  if (/dorixona|apteka|pharmac/.test(k)) return "Pharmacy";
  if (/nonvoy|bakery|qandolat/.test(k)) return "Bakery";
  if (/gul|flower|floris/.test(k)) return "Florist";
  return "LocalBusiness";
}

/** Weekday numbers as the app stores them (0 = Sunday) → schema.org names. */
const DAYS = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];

export default function StructuredData({
  restaurant,
  origin,
  kind,
}: {
  restaurant: Restaurant | null;
  origin: string;
  kind?: string;
}) {
  if (!restaurant) return null;

  const hours = (restaurant.workingHours ?? [])
    .filter((h) => !h.isClosed && h.open && h.close)
    .map((h) => ({
      "@type": "OpeningHoursSpecification",
      dayOfWeek: DAYS[h.day] ?? undefined,
      opens: h.open,
      closes: h.close,
    }));

  const data: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": schemaType(kind),
    name: restaurant.name,
    url: origin,
  };
  if (restaurant.description) data.description = restaurant.description;
  if (restaurant.logoUrl) data.image = restaurant.logoUrl;
  if (restaurant.phones?.length) data.telephone = restaurant.phones[0];
  if (restaurant.address?.text) {
    data.address = {
      "@type": "PostalAddress",
      streetAddress: restaurant.address.text,
      addressCountry: "UZ",
    };
  }
  // Coordinates make the map pin in a search result. Only sent when the
  // restaurant actually placed one: 0,0 is in the Atlantic.
  if (restaurant.address?.lat && restaurant.address?.lng) {
    data.geo = {
      "@type": "GeoCoordinates",
      latitude: restaurant.address.lat,
      longitude: restaurant.address.lng,
    };
  }
  if (hours.length) data.openingHoursSpecification = hours;
  const socials = [
    restaurant.socials?.instagram,
    restaurant.socials?.telegram,
    restaurant.socials?.facebook,
  ].filter(Boolean);
  if (socials.length) data.sameAs = socials;
  if (restaurant.delivery?.enabled) {
    data.hasDeliveryMethod = "https://schema.org/OnSitePickup";
    data.servesCuisine = undefined;
  }
  // The menu, as a URL rather than inline: a full menu graph is large, changes
  // daily, and Google reads the linked page anyway.
  if (schemaType(kind) === "Restaurant") data.hasMenu = `${origin}/menu`;

  return (
    <script
      type="application/ld+json"
      // Nothing here comes from a visitor — every field is the restaurant's
      // own profile — and JSON.stringify escapes what little could bite.
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}

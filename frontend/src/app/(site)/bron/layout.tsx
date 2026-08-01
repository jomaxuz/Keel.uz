import type { ReactNode } from "react";
import { getTranslations } from "@/lib/i18n/server";

// The booking form is a client page, so its title lives here. It is a personal
// flow, not a landing page — keep it out of search results.
export async function generateMetadata() {
  const { t } = await getTranslations();
  return { title: t.booking.title, robots: { index: false, follow: false } };
}

export default function BookingLayout({ children }: { children: ReactNode }) {
  return children;
}

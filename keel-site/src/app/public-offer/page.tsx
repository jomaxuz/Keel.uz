import type { Metadata } from "next";
import LegalPage from "@/components/LegalPage";
import { getLang } from "@/lib/i18n/server";
import { PUBLIC_OFFER } from "@/lib/legal";

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  return { title: PUBLIC_OFFER[lang]?.title ?? PUBLIC_OFFER.uz.title };
}

export default async function PublicOfferPage() {
  // ⚠️ The language comes from the same place every other page reads it — the URL first, then
  // the cookie. A legal document that renders in a language the visitor did not choose is a
  // document they have not agreed to.
  const lang = await getLang();
  return <LegalPage doc={PUBLIC_OFFER[lang] ?? PUBLIC_OFFER.uz} />;
}

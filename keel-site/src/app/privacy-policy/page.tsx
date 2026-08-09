import type { Metadata } from "next";
import LegalPage from "@/components/LegalPage";
import { getLang } from "@/lib/i18n/server";
import { PRIVACY_POLICY } from "@/lib/legal";

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  return { title: PRIVACY_POLICY[lang]?.title ?? PRIVACY_POLICY.uz.title };
}

export default async function PrivacyPolicyPage() {
  const lang = await getLang();
  return <LegalPage doc={PRIVACY_POLICY[lang] ?? PRIVACY_POLICY.uz} />;
}

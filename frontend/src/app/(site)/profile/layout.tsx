import type { Metadata } from "next";
import { getTranslations } from "@/lib/i18n/server";

// The page itself is a client component, so its title lives here.
export async function generateMetadata(): Promise<Metadata> {
  const { t } = await getTranslations();
  return { title: t.profile.ordersTitle, description: t.login.requiredText, robots: { index: false, follow: true } };
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

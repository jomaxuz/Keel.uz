"use client";

import { createContext, useContext } from "react";

import { useI18n } from "@/lib/i18n/client";
import { siteWords, type SiteWords } from "@/lib/siteWords";

// What this site sells, on the client.
//
// ⚠️ **The pages that most need it are the ones furthest from the server.** The
// basket and the checkout are client components — they live off the cart, not
// off a fetch — and they are where a shop's guest reads "Taomlar (3)" over a
// pair of trainers and an input that suggests they ask for less onion. The
// layout already knows the business type; this carries it down rather than
// making two screens fetch the profile again for a word.
//
// ⚠️ Empty is a restaurant, the same fallback every other reading makes: a site
// rendered by an older build says what it said yesterday.
const BusinessContext = createContext("");

export function BusinessProvider({
  type,
  children,
}: {
  type: string;
  children: React.ReactNode;
}) {
  return <BusinessContext.Provider value={type}>{children}</BusinessContext.Provider>;
}

export function useBusinessType(): string {
  return useContext(BusinessContext);
}

/** The words this business uses, for a client component. */
export function useSiteWords(): SiteWords {
  const { t } = useI18n();
  return siteWords(t, useContext(BusinessContext));
}

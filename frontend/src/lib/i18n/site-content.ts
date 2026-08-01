// Editable site copy (About page, footer, hero tagline) written by the
// restaurant in the admin panel. Same rule as the menu: Uzbek is the base and
// an empty translation falls back to it, so a restaurant that only fills in
// Uzbek still gets a coherent site in every language.

import type { Lang } from "./index";
import type { LocalizedText } from "../types";

export function localized(
  text: LocalizedText | undefined | null,
  lang: Lang,
  fallback = "",
): string {
  if (!text) return fallback;
  const base = text.uz?.trim() ?? "";
  const picked =
    lang === "ru" ? text.ru?.trim() : lang === "en" ? text.en?.trim() : base;
  return picked || base || fallback;
}

export const EMPTY_LOCALIZED: LocalizedText = { uz: "", ru: "", en: "" };

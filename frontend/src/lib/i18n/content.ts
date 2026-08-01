// Picks the right language for *database content* (category and dish names,
// descriptions). Unlike UI strings, this text is entered by the restaurant in
// the admin panel: Uzbek is the base and RU/EN are optional, so an empty
// translation always falls back to the base text.

import type { Lang } from "./index";

type Translatable = {
  name: string;
  nameRu?: string;
  nameEn?: string;
};

type Describable = {
  description?: string;
  descriptionRu?: string;
  descriptionEn?: string;
};

function pick(base: string, ru?: string, en?: string, lang: Lang = "uz"): string {
  if (lang === "ru") return ru?.trim() || base;
  if (lang === "en") return en?.trim() || base;
  return base;
}

export function contentName(entity: Translatable, lang: Lang): string {
  return pick(entity.name, entity.nameRu, entity.nameEn, lang);
}

export function contentDescription(entity: Describable, lang: Lang): string {
  return pick(
    entity.description ?? "",
    entity.descriptionRu,
    entity.descriptionEn,
    lang,
  );
}

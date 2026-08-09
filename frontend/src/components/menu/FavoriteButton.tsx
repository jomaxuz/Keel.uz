"use client";

// The heart on a dish card.
//
// ⚠️ **Over the photograph, top right, and always drawn** — including for a guest who is
// not signed in, where it sends them to the login instead of failing. A control that
// appears only once somebody has an account is a control nobody discovers, and the reason
// to have an account is exactly this kind of thing.
//
// ⚠️ It stops the click from reaching the card's link. The whole card is a link to the
// dish, so without that the heart both hearts and navigates — and the guest ends up on a
// page they did not ask for, unsure whether the tap worked.

import { useRouter } from "next/navigation";
import { useFavorites } from "@/lib/favorites";
import { useI18n } from "@/lib/i18n/client";
import { localePath } from "@/lib/i18n";

export default function FavoriteButton({ id }: { id: string }) {
  const { ids, enabled, toggle } = useFavorites();
  const { t, lang } = useI18n();
  const router = useRouter();
  const on = ids.has(id);

  return (
    <button
      type="button"
      aria-label={on ? t.contact.unlike : t.contact.like}
      title={on ? t.contact.unlike : t.contact.like}
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (!enabled) {
          router.push(localePath(lang, "/login?next=/menu&reason=favorite"));
          return;
        }
        void toggle(id);
      }}
      // A backdrop rather than a bare icon: the photograph underneath is any colour,
      // and an outline heart on a pale dish is invisible.
      className="absolute right-2.5 top-2.5 z-10 flex h-9 w-9 items-center justify-center rounded-full bg-surface/85 text-ink-soft shadow-card backdrop-blur transition-colors hover:text-brand"
    >
      <svg
        viewBox="0 0 24 24"
        className={`h-5 w-5 ${on ? "text-brand" : ""}`}
        fill={on ? "currentColor" : "none"}
        stroke="currentColor"
        strokeWidth="1.8"
        aria-hidden
      >
        <path d="M12 21s-8-4.8-8-10a4.5 4.5 0 0 1 8-2.8A4.5 4.5 0 0 1 20 11c0 5.2-8 10-8 10Z" />
      </svg>
    </button>
  );
}

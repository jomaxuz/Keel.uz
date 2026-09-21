"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { imageUrl } from "@/lib/api";
import { useCart } from "@/lib/cart";
import { useUser } from "@/lib/user";
import { formatPrice } from "@/lib/format";
import { useI18n } from "@/lib/i18n/client";
import { useSiteWords } from "@/lib/business";
import Recommendations from "@/components/menu/Recommendations";
import { contentName } from "@/lib/i18n/content";

export default function CartPage() {
  const {
    lines,
    subtotal,
    count,
    setQty,
    setComment,
    remove,
    clear,
    removedCount,
    dismissRemoved,
  } = useCart();
  const { user, loading } = useUser();
  const { lang, t } = useI18n();
  // "Mahsulotlar" in a shop, "Taomlar" in a restaurant — see lib/siteWords.ts.
  const w = useSiteWords();
  const router = useRouter();

  // Orders require an account: unauthenticated customers are sent to the login
  // page and bounced straight back to checkout afterwards.
  function goToCheckout() {
    if (loading) return;
    if (!user) {
      router.push("/login?next=/checkout&reason=order");
      return;
    }
    router.push("/checkout");
  }

  if (lines.length === 0) {
    return (
      <main className="container-page py-24 text-center">
        {removedCount > 0 && (
          <p className="mx-auto mb-6 max-w-md rounded-2xl border border-amber-300/60 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-200">
            {t.cart.removedNotice(removedCount)}
          </p>
        )}
        <h1 className="font-display text-2xl font-bold">{t.cart.emptyTitle}</h1>
        <p className="mt-3 text-ink-muted">{w.basketEmpty}</p>
        <Link href={w.href} className="btn-primary mt-6 px-6 py-3">
          {t.common.goToMenu}
        </Link>
      </main>
    );
  }

  return (
    <main className="container-page py-10">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="font-display text-3xl font-bold tracking-tight">{t.cart.title}</h1>
        <button
          type="button"
          onClick={clear}
          className="text-sm text-ink-muted hover:text-brand"
        >
          {t.cart.clear}
        </button>
      </div>

      {removedCount > 0 && (
        <div className="mb-5 flex items-start justify-between gap-4 rounded-2xl border border-amber-300/60 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-200">
          <p>{t.cart.removedNotice(removedCount)}</p>
          <button
            type="button"
            onClick={dismissRemoved}
            className="shrink-0 font-semibold hover:underline"
          >
            ✕
          </button>
        </div>
      )}

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
        {/* Lines */}
        <ul className="divide-y divide-line rounded-3xl border border-line bg-surface shadow-card">
          {lines.map((line) => {
            const img = imageUrl(line.imageUrl, 300);
            return (
              <li key={line.lineId} className="p-4">
                {/* On a phone the stepper and the line total drop to a second
                    row: side by side with the thumbnail and the name they add
                    up to more than a 360px screen. */}
                <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
                <div className="h-16 w-16 shrink-0 overflow-hidden rounded-lg bg-ink/5">
                  {img ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={img}
                      alt={contentName(line, lang)}
                      className="h-full w-full object-cover"
                    />
                  ) : null}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{contentName(line, lang)}</p>
                  {line.options?.length > 0 && (
                    <p className="truncate text-xs text-ink-muted">
                      {line.options
                        .map((o) => contentName(o.choice, lang))
                        .join(" · ")}
                    </p>
                  )}
                  {/* A combo's courses, so the basket says what is being paid
                      for rather than just the set's name. */}
                  {line.comboContents?.length ? (
                    <p className="truncate text-xs text-ink-muted">
                      {line.comboContents
                        .map((c) => (c.qty > 1 ? `${c.name} × ${c.qty}` : c.name))
                        .join(" · ")}
                    </p>
                  ) : null}
                  <p className="text-sm text-ink-muted">
                    {formatPrice(line.price, undefined, lang)}
                  </p>
                </div>

                <button
                  type="button"
                  onClick={() => remove(line.lineId)}
                  className="order-3 shrink-0 px-1 text-ink-muted/70 hover:text-brand sm:order-last"
                  aria-label={t.cart.remove}
                >
                  ✕
                </button>

                <div className="order-4 flex w-full items-center justify-between gap-4 sm:order-3 sm:w-auto">
                  <div className="flex items-center rounded-xl border border-line-strong">
                    <button
                      type="button"
                      onClick={() => setQty(line.lineId, line.qty - 1)}
                      className="px-3.5 py-2 text-ink-muted hover:text-brand"
                      aria-label={t.item.decrease}
                    >
                      −
                    </button>
                    <span className="w-8 text-center text-sm font-medium">
                      {line.qty}
                    </span>
                    <button
                      type="button"
                      onClick={() => setQty(line.lineId, line.qty + 1)}
                      className="px-3.5 py-2 text-ink-muted hover:text-brand"
                      aria-label={t.item.increase}
                    >
                      +
                    </button>
                  </div>

                  <div className="font-semibold sm:w-28 sm:text-right">
                    {formatPrice(line.price * line.qty, undefined, lang)}
                  </div>
                </div>
                </div>

                {/* A note for this dish alone — the kitchen sees it on the
                    order ("piyozsiz", "achchiq qilmang"). */}
                <label className="mt-3 block">
                  <span className="sr-only">{t.cart.itemComment}</span>
                  <input
                    className="input w-full text-sm"
                    maxLength={200}
                    placeholder={w.comment}
                    value={line.comment ?? ""}
                    onChange={(e) => setComment(line.lineId, e.target.value)}
                  />
                </label>
              </li>
            );
          })}
        </ul>

        {/* Summary */}
        <aside className="h-fit rounded-3xl border border-line bg-surface shadow-card p-6">
          <h2 className="font-display text-lg font-bold">{t.cart.summary}</h2>
          <div className="mt-4 flex justify-between text-sm">
            <span className="text-ink-muted">{w.basket(count)}</span>
            <span className="font-medium">{formatPrice(subtotal, undefined, lang)}</span>
          </div>
          <div className="mt-1 flex justify-between gap-4 text-sm">
            <span className="shrink-0 text-ink-muted">{t.cart.delivery}</span>
            <span className="text-right text-ink-muted/70">{t.cart.atCheckout}</span>
          </div>
          <div className="mt-4 flex justify-between border-t border-line pt-4 text-base font-bold">
            <span>{t.cart.total}</span>
            <span>{formatPrice(subtotal, undefined, lang)}</span>
          </div>

          <button
            type="button"
            onClick={goToCheckout}
            disabled={loading}
            className="btn-primary mt-6 w-full px-6 py-3"
          >
            {t.cart.checkout}
          </button>
          {!loading && !user && (
            <p className="mt-2 text-center text-xs text-ink-muted">
              {t.login.authRequired}
            </p>
          )}
          <Link
            href={w.href}
            className="mt-3 block text-center text-sm text-ink-muted hover:text-brand"
          >
            {t.cart.continue}
          </Link>
        </aside>
      </div>

      {/* ⚠️ Below the basket and its checkout button, never above. This is the
          page where the guest has already decided; a suggestion that pushes the
          button they came for off the screen costs an order to win an upsell.
          Seeded from the whole basket, so the answer is "what goes with this
          meal" rather than three unrelated answers about three dishes. */}
      <Recommendations
        itemIds={lines.map((l) => l.menuItemId)}
        currency="UZS"
        title={t.recommend.cartTitle}
      />
    </main>
  );
}

"use client";

// Telling search engines the site exists.
//
// ⚠️ **The honest part of this screen is the part that says what cannot be
// done.** The obvious feature is one button that submits every page to Google.
// There is no public way to do that: Google's Indexing API is documented for
// job postings and live streams only, and the sitemap ping endpoint was
// withdrawn in 2023. A button that quietly did nothing would be worse than no
// button — somebody would press it and stop asking why nothing was indexed.
//
// So the screen splits into what is pushed and what is pulled. IndexNow pushes,
// and covers Yandex — which in Uzbekistan is not the consolation prize. Google
// pulls, from a sitemap it re-reads on its own once, and the one manual step is
// spelled out here rather than assumed.

import { useCallback, useEffect, useState } from "react";
import { seoPing, seoStatus, type SeoPingResult, type SeoStatus } from "@/lib/api";

export default function SeoPage() {
  const [status, setStatus] = useState<SeoStatus | null>(null);
  const [result, setResult] = useState<SeoPingResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState("");

  const load = useCallback(() => {
    seoStatus()
      .then(setStatus)
      .catch((e) => setError(String(e.message ?? e)));
  }, []);
  useEffect(load, [load]);

  async function push() {
    setBusy(true);
    setError("");
    setResult(null);
    try {
      setResult(await seoPing());
      load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    } finally {
      setBusy(false);
    }
  }

  async function copy(what: string) {
    try {
      await navigator.clipboard.writeText(what);
      setCopied(what);
      window.setTimeout(() => setCopied(""), 1800);
    } catch {
      setError("Nusxa olinmadi — havolani qo'lda belgilang.");
    }
  }

  const last = status?.last;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="h-display text-2xl">Qidiruv tizimlari</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-muted">
          Sayt va qo'llanmaning uch tildagi barcha sahifalari{" "}
          <strong className="text-ink">sitemap</strong> da. Yandex va Bingga
          ularni bir tugma bilan yuborish mumkin; Google esa sitemapni o'zi
          o'qiydi — pastda nima uchun ekani yozilgan.
        </p>
      </div>

      {error && (
        <p className="rounded-xl bg-signal-500/10 px-4 py-2 text-sm text-signal-600 dark:text-signal-400">
          {error}
        </p>
      )}

      {/* ---- What we have ---- */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            Sitemapdagi manzillar
          </p>
          <p className="h-display mt-1 text-2xl">{status?.urls ?? "…"}</p>
          <p className="mt-1 text-xs text-ink-muted">uch til bilan birga</p>
        </div>
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            IndexNow kaliti
          </p>
          <p className="h-display mt-1 text-2xl">
            {status ? (status.hasKey ? "bor" : "yo'q") : "…"}
          </p>
          <p className="mt-1 text-xs text-ink-muted">
            {status?.hasKey
              ? "yuborish mumkin"
              : "INDEXNOW_KEY muhit o'zgaruvchisi kerak"}
          </p>
        </div>
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            Oxirgi yuborish
          </p>
          <p className="h-display mt-1 text-2xl">
            {last ? new Date(last.lastPingAt).toLocaleDateString() : "—"}
          </p>
          <p className="mt-1 text-xs text-ink-muted">
            {last ? `${last.lastCount} ta manzil · javob ${last.lastStatus}` : "hali yuborilmagan"}
          </p>
        </div>
      </div>

      {/* ---- Push: Yandex and Bing ---- */}
      <div className="card">
        <h2 className="h-display text-lg">Yandex va Bingga yuborish</h2>
        <p className="mt-1 max-w-3xl text-sm text-ink-muted">
          IndexNow — sahifani <strong className="text-ink">itarib</strong>{" "}
          yuborish protokoli: ro'yxat yuboriladi va qidiruv tizimi keyingi
          skanerini kutmasdan bir necha daqiqada o'qiydi. Yandex, Bing, Seznam
          va Naver qabul qiladi. O'zbekistonda Yandexning ulushi katta, ya'ni bu
          yerda bu tugma haqiqiy foyda beradi.
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button
            className="btn-primary"
            disabled={busy || !status?.hasKey || !status?.urls}
            onClick={push}
          >
            {busy ? "Yuborilmoqda…" : "Hamma sahifani yuborish"}
          </button>
          {!status?.hasKey && (
            <span className="text-sm text-ink-muted">
              Avval <code className="text-ink">INDEXNOW_KEY</code> ni qo'ying —
              kalit <code className="text-ink">{status?.keyLocation}</code> da
              ochilishi kerak.
            </span>
          )}
        </div>

        {result && (
          <div className="mt-4 rounded-2xl border border-line bg-raised px-4 py-3 text-sm">
            <p className="font-medium text-ink">
              {result.ok
                ? `${result.urls} ta manzil qabul qilindi`
                : `Rad etildi (${result.status})`}
            </p>
            {/* ⚠️ The engine's own status, not a green tick. 403 means the key
                file is unreadable, 422 means a URL is not on this host — the
                fixes are different, and "yuborilmadi" names neither. */}
            {!result.ok && (
              <p className="mt-1 text-ink-muted">
                {result.status === 403
                  ? "Kalit fayli o'qilmadi — INDEXNOW_KEY va /indexnow.txt bir xilmi?"
                  : result.status === 422
                    ? "Manzillardan biri boshqa domenga tegishli."
                    : result.message || "Sabab qaytmadi."}
              </p>
            )}
          </div>
        )}
      </div>

      {/* ---- Pull: Google ---- */}
      <div className="card">
        <h2 className="h-display text-lg">Google</h2>
        {/* ⚠️ This paragraph is the feature. It is what stops somebody looking
            for a button that does not exist and concluding the site is
            broken. */}
        <p className="mt-1 max-w-3xl text-sm text-ink-muted">
          Googlega sahifani <strong className="text-ink">itarib</strong> bo'lmaydi:
          uning Indexing API si faqat vakansiya va jonli efir uchun, sitemap
          «ping» manzili esa 2023 yilda yopilgan. Google uchun ishlaydigan yagona
          yo'l — sitemap, va uni Search Consolega{" "}
          <strong className="text-ink">bir marta</strong> qo'shish kifoya:
          keyin Google uni o'zi qayta-qayta o'qiydi.
        </p>
        <ol className="mt-4 space-y-2.5 text-sm text-ink-soft">
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              1
            </span>
            <span>
              Search Consolega kiring va domenni qo'shing:{" "}
              <a
                href="https://search.google.com/search-console"
                target="_blank"
                rel="noreferrer"
                className="text-signal-600 underline-offset-4 hover:underline dark:text-signal-400"
              >
                search.google.com/search-console
              </a>
            </span>
          </li>
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              2
            </span>
            <span>
              Domen tasdiqlangach, <strong className="text-ink">Sitemaps</strong>{" "}
              bo'limiga shu manzilni qo'shing:
            </span>
          </li>
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              3
            </span>
            <span>
              Yandex.Webmasterga ham xuddi shu sitemapni qo'shing — IndexNow
              tezlikni beradi, sitemap esa to'liqlikni.
            </span>
          </li>
        </ol>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <code className="rounded-xl border border-line bg-raised px-3 py-2 text-sm text-ink">
            {status?.sitemap ?? "…"}
          </code>
          <button
            className="btn-ghost text-sm"
            disabled={!status?.sitemap}
            onClick={() => copy(status!.sitemap)}
          >
            {copied === status?.sitemap ? "Nusxa olindi" : "Nusxa olish"}
          </button>
          <a
            href={status?.sitemap}
            target="_blank"
            rel="noreferrer"
            className="btn-ghost text-sm"
          >
            Ochish
          </a>
        </div>
      </div>

      {/* ---- What is already done ---- */}
      <div className="card">
        <h2 className="h-display text-lg">Sahifalarda nima bor</h2>
        <ul className="mt-3 space-y-2 text-sm text-ink-soft">
          <li>
            • <strong className="text-ink">Uch til, uch manzil</strong> — har
            sahifa <code>hreflang</code> bilan bog'langan, ya'ni Google ularni
            takroriy sahifa deb hisoblamaydi va har birini o'z tilidagi
            qidiruvda ko'rsatadi.
          </li>
          <li>
            • <strong className="text-ink">Qo'llanmaning har maqolasi</strong> —{" "}
            <code>TechArticle</code> va <code>BreadcrumbList</code>. Natijada
            havola o'rniga «Keel › Qo'llanma › Ombor» ko'rinadi.
          </li>
          <li>
            • <strong className="text-ink">Bosh sahifa</strong> —{" "}
            <code>Organization</code>, <code>SoftwareApplication</code> (narx
            bilan) va <code>FAQPage</code>. Oxirgisi natijada qo'shimcha joy
            beradi.
          </li>
          <li>
            • <strong className="text-ink">robots.txt</strong> — konsol va
            status yopilgan, sitemap ko'rsatilgan (Yandex uni aynan shu yerdan
            oladi).
          </li>
        </ul>
      </div>
    </div>
  );
}

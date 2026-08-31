"use client";

// The A5 sheet somebody leaves on a counter.
//
// ⚠️ **This exists because the owner is usually not there.** Walking into a
// restaurant at two in the afternoon finds a cashier, and the person who
// decides is at home or in a supplier's warehouse. Without something to leave
// behind, that visit produced nothing; with it, the visit produced a QR code
// sitting next to the till until somebody is annoyed enough by the notebook to
// scan it.
//
// ⚠️ **The pitch is the till, not the website.** A place with no system does
// not lie awake wanting a website — it wants to know what today's takings were.
// Selling the site first puts us in the crowded half of the market, against
// people who already have a sales team; selling the till puts us in front of
// somebody whose alternative is a paper notebook.
//
// ⚠️ **Printed, so nothing here may depend on the screen.** No dark theme, no
// hover, no shadow: paper is white and a shadow is a grey smear on it.

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import Qr from "@/components/console/Qr";
import { ORIGIN } from "@/lib/i18n/url";
import { cleanRefCode } from "@/lib/referral";
import { EMAIL, TELEGRAM } from "@/lib/links";

/** Two languages, because the sheet is read by one person and never
 *  translated afterwards. Russian is not a nicety here: a good share of the
 *  places worth leaving this in are run in it. */
const COPY = {
  uz: {
    eyebrow: "Restoran, kafe va do'konlar uchun",
    title: "Kassangiz yo'qmi?",
    lead: "Daftar va kalkulyator o'rniga — bitta dastur. Kassa, zal, oshxona ekrani, ombor va tannarx. Sayt, yetkazish va Telegram bot ham shu menyudan ishlaydi.",
    points: [
      "Kunlik tushum va har bir chek — ekranda, kun oxirida emas",
      "Ombor va texkarta: bir porsiya qanchaga tushishini bilib turasiz",
      "Ofitsiant, oshpaz va kassirning o'z ekrani",
      "Sayt va Telegram bot — agregatorga komissiya bermay",
    ],
    priceLabel: "Oyiga",
    price: "450 000 so'mdan",
    priceNote: "iiko'dan 33–63% arzon · 14 kun bepul · menyuni biz kiritamiz",
    scan: "Skanerlang",
    scanNote: "yoki shu manzilga kiring",
    contact: "Savollar",
    langLabel: "O'zbekcha",
  },
  ru: {
    eyebrow: "Для ресторанов, кафе и магазинов",
    title: "Нет кассовой программы?",
    lead: "Вместо тетради и калькулятора — одна программа. Касса, зал, кухонный экран, склад и себестоимость. Сайт, доставка и Telegram-бот работают из того же меню.",
    points: [
      "Выручка за день и каждый чек — на экране, а не в конце месяца",
      "Склад и техкарты: вы знаете, во сколько обходится порция",
      "У официанта, повара и кассира — свой экран",
      "Сайт и Telegram-бот — без комиссии агрегатору",
    ],
    priceLabel: "В месяц",
    price: "от 450 000 сум",
    priceNote: "на 33–63% дешевле iiko · 14 дней бесплатно · меню заведём мы",
    scan: "Отсканируйте",
    scanNote: "или откройте этот адрес",
    contact: "Вопросы",
    langLabel: "Русский",
  },
} as const;

export default function LeafletPage() {
  const params = useSearchParams();
  const [lang, setLang] = useState<"uz" | "ru">("uz");
  const [code, setCode] = useState(cleanRefCode(params.get("ref") ?? ""));
  const t = COPY[lang];

  // ⚠️ The partner's own address when there is one, so a sheet left by the
  // register engineer is attributable when somebody scans it three weeks later.
  const url = code ? `${ORIGIN}/h/${code}` : ORIGIN;
  const shown = url.replace(/^https?:\/\//, "");
  const handle = TELEGRAM.replace(/^https?:\/\/t\.me\//, "@");

  return (
    <div className="space-y-6">
      {/* ---- Controls. Not printed. ---- */}
      <div className="no-print space-y-4">
        <div>
          <h1 className="h-display text-2xl">Varaqa</h1>
          <p className="mt-1 max-w-2xl text-sm text-ink-muted">
            Peshtaxtaga qoldirish uchun A5 varaqa. Ega joyda bo'lmasa ham
            tashrif behuda ketmaydi. Hamkor kodi kiritilsa, QR o'sha hamkorning
            havolasini ochadi.
          </p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <label className="block text-sm">
            <span className="text-ink-muted">Til</span>
            <select
              className="select mt-1 w-40"
              value={lang}
              onChange={(e) => setLang(e.target.value as "uz" | "ru")}
            >
              <option value="uz">{COPY.uz.langLabel}</option>
              <option value="ru">{COPY.ru.langLabel}</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Hamkor kodi (ixtiyoriy)</span>
            <input
              className="input mt-1 w-56"
              placeholder="fiskal"
              value={code}
              onChange={(e) => setCode(cleanRefCode(e.target.value))}
            />
          </label>
          <button className="btn-primary" onClick={() => window.print()}>
            Chop etish
          </button>
        </div>
        <p className="text-xs text-ink-muted">
          Chop etishda qog'oz o'lchamini <strong>A5</strong> qilib qo'ying va
          &laquo;fon rasmlari&raquo; (background graphics) ni yoqing — aks holda
          rangli maydonlar oq chiqadi.
        </p>
      </div>

      {/* ---- The sheet ---- */}
      <div className="sheet">
        <div className="sheet-inner">
          <div className="sheet-top">
            <div className="brand">
              <span className="mark" aria-hidden />
              <span className="wordmark">keel</span>
            </div>
            <p className="eyebrow-print">{t.eyebrow}</p>
            <h2 className="headline">{t.title}</h2>
            <p className="lead">{t.lead}</p>

            {/* ⚠️ Inside the top block, not between it and the price. As a
                third block the sheet's `space-between` pushed a hand's width of
                nothing into the middle of the page — which on paper reads as a
                template that failed to fill rather than as breathing room. */}
            <ul className="points">
              {t.points.map((p) => (
                <li key={p}>
                  <span className="tick" aria-hidden>
                    ✓
                  </span>
                  <span>{p}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="sheet-bottom">
            <div className="price">
              <p className="price-label">{t.priceLabel}</p>
              <p className="price-value">{t.price}</p>
              <p className="price-note">{t.priceNote}</p>
              <p className="contact">
                {t.contact}: {handle} · {EMAIL}
              </p>
            </div>
            <div className="qr">
              <Qr value={url} size={128} />
              <p className="qr-label">{t.scan}</p>
              <p className="qr-url">{shown}</p>
              <p className="qr-note">{t.scanNote}</p>
            </div>
          </div>
        </div>
      </div>

      {/* ⚠️ Plain CSS, and every colour written out rather than taken from the
          theme tokens. The tokens change with the reader's dark mode; paper
          does not, and a sheet that prints in dark theme is a black rectangle
          and an empty cartridge. */}
      <style jsx global>{`
        .sheet {
          width: 148mm;
          height: 210mm;
          margin: 0 auto;
          background: #fdfcf9;
          color: #0a1a28;
          border: 1px solid #e2ded6;
          box-sizing: border-box;
        }
        .sheet-inner {
          height: 100%;
          padding: 14mm 12mm;
          display: flex;
          flex-direction: column;
          justify-content: space-between;
          box-sizing: border-box;
          font-family: var(--font-sans), system-ui, sans-serif;
        }
        .brand {
          display: flex;
          align-items: center;
          gap: 6px;
        }
        .mark {
          width: 14px;
          height: 14px;
          border-radius: 50%;
          background: #f5a524;
          display: inline-block;
        }
        .wordmark {
          font-weight: 700;
          font-size: 18px;
          letter-spacing: -0.02em;
        }
        .eyebrow-print {
          margin: 10mm 0 0;
          font-size: 10px;
          font-weight: 700;
          letter-spacing: 0.18em;
          text-transform: uppercase;
          color: #d2870f;
        }
        .headline {
          margin: 3mm 0 0;
          font-size: 30px;
          line-height: 1.1;
          font-weight: 700;
          letter-spacing: -0.02em;
        }
        .lead {
          margin: 4mm 0 0;
          font-size: 12.5px;
          line-height: 1.5;
          color: #374858;
        }
        .points {
          list-style: none;
          margin: 7mm 0 0;
          padding: 0;
          display: grid;
          gap: 3mm;
        }
        .points li {
          display: flex;
          gap: 7px;
          font-size: 12.5px;
          line-height: 1.4;
        }
        .tick {
          color: #d2870f;
          font-weight: 700;
        }
        .sheet-bottom {
          display: flex;
          align-items: flex-end;
          justify-content: space-between;
          gap: 8mm;
          border-top: 1px solid #e2ded6;
          padding-top: 5mm;
        }
        .price-label {
          margin: 0;
          font-size: 10px;
          letter-spacing: 0.16em;
          text-transform: uppercase;
          color: #6e7e8c;
        }
        .price-value {
          margin: 1mm 0 0;
          font-size: 22px;
          font-weight: 700;
          letter-spacing: -0.01em;
        }
        .price-note {
          margin: 2mm 0 0;
          font-size: 11px;
          line-height: 1.45;
          color: #374858;
          max-width: 62mm;
        }
        .contact {
          margin: 4mm 0 0;
          font-size: 11px;
          color: #374858;
        }
        .qr {
          text-align: center;
        }
        .qr-label {
          margin: 2mm 0 0;
          font-size: 11px;
          font-weight: 700;
        }
        .qr-url {
          margin: 0.5mm 0 0;
          font-size: 11px;
          font-weight: 600;
          color: #0a1a28;
        }
        .qr-note {
          margin: 0.5mm 0 0;
          font-size: 9.5px;
          color: #6e7e8c;
        }

        @media print {
          /* ⚠️ Declared here as well as told to the operator above: a browser
             that is not given a page size prints A5 content onto A4 and leaves
             a third of the sheet blank, which reads as a broken template. */
          @page {
            size: A5;
            margin: 0;
          }
          body {
            background: #fff;
          }
          /* ⚠️ The cookie notice too. It is position:fixed, so it does not
             scroll off the page — it prints, over the price.
             (No backticks in this block: it is a template literal, and one
             would end the string in the middle of a stylesheet.) */
          .no-print,
          header,
          footer,
          .fixed {
            display: none !important;
          }
          .sheet {
            border: none;
            margin: 0;
          }
        }
      `}</style>
    </div>
  );
}

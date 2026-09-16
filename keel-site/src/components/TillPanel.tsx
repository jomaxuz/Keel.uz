"use client";

// Selling a restaurant its counter.
//
// ⚠️ **This screen is the only place the till is switched on**, and it sits in
// the console rather than in the restaurant's own settings for the same reason
// the watermark toggle does: an owner given this control would move themselves
// to the top rung. It is behind "provision", so an agent chasing a sale cannot
// change what a customer is billed.
//
// The shape of the panel is the pricing policy:
//
//   • the price is computed and shown **before** it is saved, because the
//     person agreeing it on the phone should not meet the number on an invoice
//     a month later;
//   • Enterprise has no automatic price, so it demands one — a plan invoiced at
//     zero looks like a working system until somebody reconciles a quarter;
//   • switching off keeps the rung, the add-ons and the date, so a customer
//     paused for a month comes back exactly as they were;
//   • the paid-until date is a date and not a tick, because the screens
//     standing in the restaurant count down to it.

import { useCallback, useEffect, useState } from "react";
import {
  tillSubscription,
  setTillSubscription,
  type TillSubscription,
} from "@/lib/api";

/** Add-ons that can be bought on a rung that does not include them. Mirrors
 *  billing.AddonPrice — the price itself comes from the server, so this list
 *  only decides what is offered. */
const ADDONS = [
  { id: "stock", label: "Ombor va tannarx", price: 290_000 },
  // ⚠️ **Without this row the assistant could not be sold at all.** It is
  // included at Pro and Enterprise, and every rung below had no way to be
  // given it — the entitlement check looked for an add-on the console could
  // not grant. Mirrors billing.AddonAI / AIMonthly.
  { id: "ai", label: "AI yordamchisi", price: 250_000 },
  // ⚠️ **In no plan, unlike the assistant above.** The briefing costs us one
  // model call a morning; this runs a restaurant's Meta campaigns — a model
  // call per question and an API quota measured per ad account. Mirrors
  // billing.ModAds / AdsMonthly.
  { id: "ads", label: "Reklama (AI targetolog)", price: 1_250_000 },
];

// One bought increment of the AI assistant's daily allowance.
//
// ⚠️ **A daily cap sold monthly, and both numbers belong on this screen.** The
// cap is daily because that is what stops one stuck browser tab spending a
// month's allowance in an afternoon — and the allowance is real money we pay
// out. It is priced monthly because that is how a restaurant reads a bill.
// Mirrors billing.AIExtraBlock / AIExtraMonthly, which is where the server
// enforces it.
const AI_BLOCK = 10;
const AI_BLOCK_PRICE = 100_000;

// One television in the dining room, per month. Mirrors billing.TVScreenMonthly.
//
// ⚠️ **Priced per screen, unlike everything else on this panel.** The rest of
// the till is priced by the room; a screen is the one thing a restaurant buys
// more of inside the same room — by the counter, by the door, over the till.
const TV_SCREEN_PRICE = 50_000;

const PLAN_LABEL: Record<string, string> = {
  start: "Start",
  standard: "Standard",
  pro: "Pro",
  enterprise: "Enterprise",
};

// ⚠️ Analysis and the CRM are absent, and deliberately: they are in the price
// for every customer. Selling them would mean a restaurant that buys a till
// loses screens it already had — see billing/plans.go.
const MODULE_LABEL: Record<string, string> = {
  stock: "Ombor va tannarx",
  multibranch: "Ko'p filial / brend",
  posint: "Tashqi kassa (iiko, Poster…)",
  franchise: "Franshiza boshqaruvi",
  // ⚠️ Never in a plan's own list — it is granted by the screen count below —
  // but named here because the restaurant's mirrored `modules` carries it, and
  // a bare "tv" on a screen is a word nobody in a sales call can explain.
  tv: "TV ekranlar",
  // Named here for the same reason `tv` is: the restaurant's mirrored `modules`
  // carries it, and a bare "ads" on a screen is a word nobody in a sales call
  // can explain.
  ads: "Reklama (AI targetolog)",
};

function money(n: number) {
  return n.toLocaleString("ru-RU").replace(/,/g, " ");
}

export default function TillPanel({ tenantId }: { tenantId: string }) {
  const [sub, setSub] = useState<TillSubscription | null>(null);
  const [plan, setPlan] = useState("start");
  const [addons, setAddons] = useState<string[]>([]);
  // Blocks of ten daily assistant requests, sold on top of the plan.
  const [aiExtra, setAiExtra] = useState(0);
  // How many televisions this restaurant pays for. ⚠️ The number is the whole
  // entitlement — it grants the module and prices it.
  const [tvScreens, setTvScreens] = useState(0);
  const [branches, setBranches] = useState(1);
  const [override, setOverride] = useState(0);
  const [paidUntil, setPaidUntil] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const s = await tillSubscription(tenantId);
      setSub(s);
      setPlan(s.plan || "start");
      setAddons(s.addons ?? []);
      setAiExtra(s.aiExtra ?? 0);
      setTvScreens(s.tvScreens ?? 0);
      setBranches(s.branches || 1);
      setOverride(s.priceOverride || 0);
      // ⚠️ Sliced from a date the server already rendered as a day, never from
      // a timestamp: a `time.Time` marshals as UTC, so cutting the first ten
      // characters of one in Tashkent returns the previous day.
      setPaidUntil(s.paidUntil ? s.paidUntil.slice(0, 10) : "");
      setNote(s.note ?? "");
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  const chosen = sub?.plans.find((p) => p.id === plan);

  // The same arithmetic the server bills from, drawn here so the operator sees
  // the number before saving. ⚠️ It is a preview, not the source: the invoice
  // is issued from the server's own copy, so a stale bundle cannot agree a
  // price the customer is not charged.
  const preview = (() => {
    if (!chosen) return 0;
    if (override > 0) return override;
    if (chosen.individual) return 0;
    let total = 0;
    for (let n = 1; n <= Math.max(1, branches); n++) {
      const off = n <= 1 ? 0 : n < 5 ? 30 : 40;
      total += Math.round((chosen.monthly * (100 - off)) / 100);
    }
    for (const a of addons) {
      if (chosen.modules.includes(a)) continue;
      total += ADDONS.find((x) => x.id === a)?.price ?? 0;
    }
    // ⚠️ In the same total as everything else. A line item priced on a
    // different screen is a line item that goes missing from an invoice.
    total += aiExtra * AI_BLOCK_PRICE;
    total += tvScreens * TV_SCREEN_PRICE;
    return total;
  })();

  async function save(enabled: boolean) {
    setBusy(true);
    setError("");
    try {
      await setTillSubscription(
        tenantId,
        enabled
          ? {
              enabled,
              plan,
              addons,
              aiExtra,
              tvScreens,
              branches,
              priceOverride: override,
              paidUntil: paidUntil ? new Date(paidUntil).toISOString() : null,
              note,
            }
          : { enabled: false, note },
      );
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!sub) return null;

  return (
    <section className="card">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm font-semibold text-ink">Kassa (POS) obunasi</p>
        {sub.enabled ? (
          <span className="rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
            {PLAN_LABEL[sub.plan ?? ""] ?? sub.plan} · {money(sub.monthly)}{" "}
            so'm/oy
          </span>
        ) : (
          <span className="rounded-full bg-ink/10 px-2.5 py-0.5 text-xs font-semibold text-ink-muted">
            ulanmagan
          </span>
        )}
      </div>
      <p className="mt-1 text-xs text-ink-muted">
        Kassani faqat shu yerdan ulaymiz. Hisob oyiga bir marta, onlayn
        buyurtmalar va watermark bilan bitta hisob-fakturada chiqadi.
      </p>

      <div className="mt-4 rounded-2xl border border-line bg-raised p-4">
        <label className="text-sm font-medium">Tarif</label>
        <div className="mt-2 grid gap-2 sm:grid-cols-2">
          {sub.plans.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => setPlan(p.id)}
              className={`rounded-xl border p-3 text-left text-sm transition ${
                plan === p.id
                  ? "border-brand bg-brand/5"
                  : "border-line hover:border-line-strong"
              }`}
            >
              <span className="flex items-baseline justify-between gap-2">
                <span className="font-semibold">
                  {PLAN_LABEL[p.id] ?? p.id}
                </span>
                <span className="text-xs text-ink-muted">
                  {p.individual ? "kelishiladi" : `${money(p.monthly)} so'm`}
                </span>
              </span>
              <span className="mt-1 block text-xs text-ink-muted">
                {p.registers === 0
                  ? "cheksiz kassa"
                  : `${p.registers} kassagacha`}
              </span>
              {p.modules.length > 0 && (
                <span className="mt-1 block text-[11px] leading-snug text-ink-muted">
                  {p.modules.map((m) => MODULE_LABEL[m] ?? m).join(" · ")}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* ⚠️ Offered on every rung, and skipped in the price when the rung
            already includes it. Stock is buyable on Start on purpose: the jump
            from 450 000 to Pro is the cliff that sends a café that wants its
            food cost to a competitor. */}
        <div className="mt-4">
          <span className="text-sm font-medium">Qo&apos;shimcha modullar</span>
          <div className="mt-2 flex flex-wrap gap-2">
            {ADDONS.map((a) => {
              const included = chosen?.modules.includes(a.id);
              const on = included || addons.includes(a.id);
              return (
                <button
                  key={a.id}
                  type="button"
                  disabled={included}
                  onClick={() =>
                    setAddons((prev) =>
                      prev.includes(a.id)
                        ? prev.filter((x) => x !== a.id)
                        : [...prev, a.id],
                    )
                  }
                  className={`rounded-full border px-3 py-1.5 text-xs transition ${
                    on
                      ? "border-brand bg-brand/10 font-semibold"
                      : "border-line text-ink-muted hover:border-line-strong"
                  } ${included ? "opacity-70" : ""}`}
                >
                  {a.label}
                  <span className="ml-1.5 text-ink-muted">
                    {included ? "tarifda bor" : `+${money(a.price)}`}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* ⚠️ **Only once the assistant is actually granted.** A box for buying
            more of something a restaurant does not have is a box that gets
            filled in and does nothing — and then the console shows a charge
            nobody can explain. */}
        {(chosen?.modules.includes("ai") || addons.includes("ai")) && (
          <label className="mt-4 block text-sm">
            <span className="mb-1 block text-xs text-ink-muted">
              AI: qo&apos;shimcha kunlik limit
            </span>
            <div className="flex flex-wrap items-center gap-2">
              <input
                type="number"
                min={0}
                max={50}
                className="input w-24"
                value={aiExtra}
                onChange={(e) =>
                  setAiExtra(
                    Math.max(0, Math.min(50, Number(e.target.value) || 0)),
                  )
                }
              />
              <span className="text-xs text-ink-muted">
                × {AI_BLOCK} so&apos;rov/kun · {money(AI_BLOCK_PRICE)}/oy
              </span>
            </div>
            <span className="mt-1 block text-xs text-ink-muted">
              Tarif bergan limit ustiga qo&apos;shiladi. 0 &mdash; faqat tarif
              limiti.
            </span>
          </label>
        )}

        {/* ⚠️ **A count, not a switch.** The televisions are sold one at a
            time, so the number is the whole answer: 0 means the section is not
            in their plan, 3 means three screens may be paired and three are
            billed. A toggle beside a number is two facts that can disagree —
            and one of those disagreements is a dining room that went dark. */}
        <label className="mt-4 block text-sm">
          <span className="mb-1 block text-xs text-ink-muted">
            TV ekranlar (zaldagi televizorlar)
          </span>
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="number"
              min={0}
              max={30}
              className="input w-24"
              value={tvScreens}
              onChange={(e) =>
                setTvScreens(
                  Math.max(0, Math.min(30, Number(e.target.value) || 0)),
                )
              }
            />
            <span className="text-xs text-ink-muted">
              × {money(TV_SCREEN_PRICE)} so&apos;m/oy
              {tvScreens > 0
                ? ` · jami ${money(tvScreens * TV_SCREEN_PRICE)}`
                : ""}
            </span>
          </div>
          <span className="mt-1 block text-xs text-ink-muted">
            0 &mdash; bo&apos;lim ochilmaydi. Ekran shu songacha ulanadi;
            allaqachon ulangan ekran sonni kamaytirsangiz ham o&apos;chmaydi.
          </span>
        </label>

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <label className="text-sm">
            <span className="mb-1 block text-xs text-ink-muted">Filiallar</span>
            <input
              type="number"
              min={1}
              className="input"
              value={branches}
              onChange={(e) => setBranches(Number(e.target.value))}
            />
          </label>
          <label className="text-sm">
            <span className="mb-1 block text-xs text-ink-muted">
              Kelishilgan narx {chosen?.individual ? "(shart)" : "(ixtiyoriy)"}
            </span>
            <input
              type="number"
              min={0}
              className="input"
              value={override || ""}
              placeholder="—"
              onChange={(e) => setOverride(Number(e.target.value))}
            />
          </label>
          <label className="text-sm">
            <span className="mb-1 block text-xs text-ink-muted">
              To&apos;langan muddat
            </span>
            <input
              type="date"
              className="input"
              value={paidUntil}
              onChange={(e) => setPaidUntil(e.target.value)}
            />
          </label>
        </div>
        {/* ⚠️ Named as what it drives, not as a formality: this date is what the
            monoblock and the floor tablet count down to, and the restaurant
            sees a red corner a week before it. Empty means no countdown. */}
        <p className="mt-1 text-[11px] text-ink-muted">
          Shu sanadan 7, 3 va 1 kun oldin kassa va zal ekranlarida qizil
          ogohlantirish chiqadi. Bo&apos;sh qoldirilsa — ogohlantirish
          yo&apos;q.
        </p>

        <label className="mt-3 block text-sm">
          <span className="mb-1 block text-xs text-ink-muted">Izoh</span>
          <input
            className="input"
            value={note}
            placeholder="kim bilan, qachon kelishildi"
            onChange={(e) => setNote(e.target.value)}
          />
        </label>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm">
            <span className="text-ink-muted">Oyiga: </span>
            {chosen?.individual && override <= 0 ? (
              <span className="font-semibold text-amber-600 dark:text-amber-400">
                kelishilgan summani yozing
              </span>
            ) : (
              <span className="font-semibold">{money(preview)} so&apos;m</span>
            )}
          </p>
          <div className="flex gap-2">
            {sub.enabled && (
              <button
                type="button"
                disabled={busy}
                onClick={() => save(false)}
                className="btn-ghost px-3 py-2 text-xs disabled:opacity-40"
              >
                Kassani o&apos;chirish
              </button>
            )}
            <button
              type="button"
              // Guarded here as well as on the server, so the button explains
              // itself rather than answering with a 400.
              disabled={
                busy || (!!chosen?.individual && override <= 0) || branches < 1
              }
              onClick={() => save(true)}
              className="btn-primary px-4 py-2 text-sm disabled:opacity-40"
            >
              {sub.enabled ? "Saqlash" : "Kassani ulash"}
            </button>
          </div>
        </div>
      </div>

      {sub.enabled && sub.updatedBy && (
        <p className="mt-3 text-xs text-ink-muted">
          Oxirgi o&apos;zgarish: {sub.updatedBy}
          {sub.updatedAt ? ` · ${sub.updatedAt.slice(0, 10)}` : ""}
        </p>
      )}
      {error && (
        <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>
      )}
    </section>
  );
}

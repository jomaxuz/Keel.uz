"use client";

// The upgrade card, and the small button beside a control that is not included.
//
// ⚠️ **It says what is missing, what it costs, and what to do — in that order.**
// A card that only says "not available on your plan" leaves an owner with one
// action, which is to ring somebody and ask a question we could have answered
// on the screen. The price is already on our public pricing page, so putting it
// here costs nothing and removes the whole first half of that call.
//
// ⚠️ **No self-serve upgrade button, and that is deliberate for now.** The plan
// is switched on from the console by a person, because it changes what the
// customer is billed and the money is still collected by hand. A button that
// looked like it would upgrade the account and instead sent a message would be
// the worst of both — so this one says plainly that it is a phone call, and
// carries the number.

import Link from "next/link";

import { TELEGRAM } from "@/lib/links";
import { useSubscription } from "@/lib/subscription";

export const MODULE_LABEL: Record<string, string> = {
  stock: "Ombor va tannarx",
  multibranch: "Ko'p filial va brend",
  posint: "Tashqi kassa integratsiyasi",
  franchise: "Franshiza boshqaruvi",
  // ⚠️ Named here or the upgrade card says "ads" to somebody who has never seen
  // the word — the card exists to explain what is missing, and a module id is
  // not an explanation.
  ads: "Reklama (AI targetolog)",
};

export const PLAN_LABEL: Record<string, string> = {
  start: "Start",
  standard: "Standard",
  pro: "Pro",
  enterprise: "Enterprise",
};

export function money(n: number) {
  return n.toLocaleString("ru-RU").replace(/,/g, " ");
}

/** The full card, drawn in place of a screen the customer has not bought. */
export function UpgradeCard({ module }: { module: string }) {
  const { planFor } = useSubscription();
  const plan = planFor(module);
  const name = MODULE_LABEL[module] ?? module;

  return (
    <div className="card mx-auto max-w-xl p-6 text-center">
      <p className="text-base font-semibold text-ink">{name}</p>
      <p className="mt-2 text-sm text-ink-soft">
        Bu bo&apos;lim hozirgi tarifingizga kirmaydi.
      </p>
      {plan && (
        <p className="mt-4 rounded-2xl border border-line bg-raised px-4 py-3 text-sm">
          <span className="font-semibold">
            {PLAN_LABEL[plan.id] ?? plan.id}
          </span>{" "}
          tarifidan boshlab mavjud
          {!plan.individual && (
            <>
              {" · "}
              <span className="font-semibold">
                {money(plan.monthly)} so&apos;m/oy
              </span>
            </>
          )}
        </p>
      )}
      {/* ⚠️ **Telegram, not a phone call.** The person reading this is standing
          in a restaurant with a phone in their hand — often after closing — and
          a `tel:` link needs somebody free to answer at that moment and leaves
          nothing to come back to. A message sent at eleven at night is still
          waiting for us in the morning, with the question already written down. */}
      <a
        href={TELEGRAM}
        target="_blank"
        rel="noopener noreferrer"
        className="btn-primary mt-4 inline-flex px-5 py-2.5 text-sm"
      >
        Telegramda yozing — tarifni ko&apos;taramiz
      </a>
      <p className="mt-3 text-xs text-ink-muted">
        Yoki hozirgi tarifingizni{" "}
        <Link href="/admin/account" className="underline">
          Hisob
        </Link>{" "}
        bo&apos;limida ko&apos;ring.
      </p>
    </div>
  );
}

/** The inline nudge, for a single control inside a screen that otherwise works.
 *
 *  ⚠️ Renders nothing when the module is available, so a call site can drop it
 *  beside a button unconditionally — a conditional at every call site is a
 *  conditional somebody eventually gets backwards. */
export function UpgradeChip({ module }: { module: string }) {
  const { has, planFor } = useSubscription();
  if (has(module)) return null;
  const plan = planFor(module);
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-300 bg-amber-50 px-2.5 py-1 text-[11px] font-semibold text-amber-800">
      {PLAN_LABEL[plan?.id ?? ""] ?? "Yuqori"} tarifda
    </span>
  );
}

/** Wraps a screen: the screen when it is bought, the card when it is not. */
export default function UpgradeGate({
  module,
  children,
}: {
  module: string;
  children: React.ReactNode;
}) {
  const { has } = useSubscription();
  if (!module || has(module)) return <>{children}</>;
  return <UpgradeCard module={module} />;
}

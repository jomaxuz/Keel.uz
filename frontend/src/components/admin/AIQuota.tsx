"use client";

// What is left of today's assistant, and what to do when it runs out.
//
// ⚠️ **On the account page rather than beside the briefing.** It answers a
// question about the bill, not about the restaurant — and a briefing card that
// spent one of its four lines saying "7 of 20 used" would be a line not spent
// on the restaurant.
//
// ⚠️ **A daily number sold monthly, and the two units are worth explaining on
// the screen.** The cap is daily because that is what stops a stuck browser tab
// spending a month's allowance in an afternoon; it is sold monthly because that
// is how a restaurant thinks about a bill. Somebody reading "10 per day, 100 000
// so'm per month" without that sentence reasonably wonders which one is wrong.

import { useEffect, useState } from "react";
import { LuSparkles } from "react-icons/lu";

import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { AIQuota as Quota } from "@/lib/types";

export default function AIQuota() {
  const t = useAdminT();
  const [q, setQ] = useState<Quota | null>(null);

  useEffect(() => {
    api
      .adminAIQuota()
      .then(setQ)
      .catch(() => setQ(null));
  }, []);

  // ⚠️ A platform with no assistant configured draws nothing at all — not a
  // card saying "off". A restaurant that was never sold this should not be
  // shown a bar at zero and left wondering what it is.
  if (!q || !q.on || !q.entitled) return null;

  const limit = Math.max(1, q.limit ?? 1);
  const used = Math.min(q.used ?? 0, limit);
  const pct = Math.round((used / limit) * 100);
  const out = used >= limit;

  return (
    <div className="card space-y-3 p-4">
      <div className="flex items-center gap-2">
        <LuSparkles className="text-brand" aria-hidden />
        <h2 className="font-semibold">{t.aiQuota.title}</h2>
      </div>

      <div>
        <div className="flex items-baseline justify-between gap-3 text-sm">
          <span className="text-ink-soft">{t.aiQuota.today}</span>
          <span className="tabular-nums font-semibold">
            {used} / {limit}
          </span>
        </div>
        {/* ⚠️ The bar turns before it is full, not at 100%. A restaurant that
            first learns it is close to the limit by hitting it has learned it
            too late to do anything that day. */}
        <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-ink/10">
          <div
            className={`h-full rounded-full transition-all ${
              out
                ? "bg-danger"
                : pct >= 80
                  ? "bg-amber-500"
                  : "bg-brand"
            }`}
            style={{ width: `${Math.max(pct, 2)}%` }}
          />
        </div>
        {/* Where the number comes from, so it can be accounted for rather than
            trusted: the plan's own allowance plus whatever was bought. */}
        {(q.extraBlocks ?? 0) > 0 && (
          <p className="mt-1 text-xs text-ink-muted">
            {t.aiQuota.madeOf(q.planLimit ?? 0, (q.extraBlocks ?? 0) * (q.blockSize ?? 10))}
          </p>
        )}
      </div>

      {/* ⚠️ **The way out, said where somebody meets the wall.** A limit with no
          route past it is a limit people work around by giving up on the
          feature — and the one moment they will read this is the moment they
          have just been refused. */}
      <div
        className={`rounded-xl border p-3 text-sm ${
          out
            ? "border-amber-400/60 bg-amber-50 dark:bg-amber-950/30"
            : "border-line bg-surface-soft"
        }`}
      >
        <p>{out ? t.aiQuota.spent : t.aiQuota.more}</p>
        <p className="mt-1 font-medium">
          {t.aiQuota.price(q.blockSize ?? 10, formatPrice(q.blockPrice ?? 0))}
        </p>
        <a
          href={`https://t.me/${(q.contact ?? "@keeluz").replace(/^@/, "")}`}
          target="_blank"
          rel="noreferrer"
          className="mt-2 inline-block font-semibold text-brand hover:underline"
        >
          {t.aiQuota.write(q.contact ?? "@keeluz")} →
        </a>
      </div>

      <p className="text-xs text-ink-muted">{t.aiQuota.note}</p>
    </div>
  );
}

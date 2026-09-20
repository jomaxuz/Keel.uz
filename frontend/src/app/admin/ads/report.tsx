"use client";

// What the advertising cost, and what came back.
//
// ⚠️ **Two systems on one screen, and the screen never subtracts one from the
// other.** The spend is Meta's, in the currency Meta bills the account in; the
// orders and their value are Meta's attribution of the events this server
// reported. A single "profit from advertising" figure would be made of two
// currencies and somebody else's attribution model — and it is exactly the
// figure an owner would act on and we could not defend.
//
// ⚠️ **Without a pixel there is nothing to attribute**, and the screen says so
// instead of drawing a row of zeroes. A zero that means "nobody ordered" and a
// zero that means "we are not measuring" look identical, and only one of them
// is a reason to stop the campaign.

import type { AdminDict } from "@/lib/i18n/admin";
import type { AdsReport } from "@/lib/types";

function som(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(Math.round(n));
}

export default function AdsReportCard({
  t,
  report,
}: {
  t: AdminDict;
  report: AdsReport | null;
}) {
  if (!report || report.days.length === 0) return null;
  const max = Math.max(...report.days.map((d) => d.spend), 0.0001);

  return (
    <div className="space-y-3">
      <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-4">
        <div>
          <dt className="text-xs text-ink-muted">{t.ads.report.spend}</dt>
          <dd className="font-medium">
            {report.total.spend.toFixed(2)} {report.currency ?? ""}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-ink-muted">{t.ads.report.clicks}</dt>
          <dd className="font-medium">{report.total.clicks}</dd>
        </div>
        <div>
          <dt className="text-xs text-ink-muted">{t.ads.report.orders}</dt>
          <dd className="font-medium">
            {report.pixel ? report.total.purchases : "—"}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-ink-muted">{t.ads.report.revenue}</dt>
          <dd className="font-medium">
            {report.pixel ? `${som(report.total.revenue)} UZS` : "—"}
          </dd>
        </div>
      </dl>

      {!report.pixel && (
        <p className="max-w-2xl text-xs text-ink-muted">
          {t.ads.report.noPixel}
        </p>
      )}

      {/* Spend per day. ⚠️ Width from the figure itself, like the plan's bars:
          the chart is the number, not an impression of it. */}
      <div className="space-y-1">
        {report.days.map((d) => (
          <div key={d.day} className="flex items-center gap-2 text-xs">
            <span className="w-20 shrink-0 text-ink-muted">{d.day}</span>
            <div className="h-2 flex-1 overflow-hidden rounded-full bg-ink/10">
              <div
                className="h-full rounded-full bg-brand"
                style={{ width: `${Math.min(100, (d.spend / max) * 100)}%` }}
              />
            </div>
            <span className="w-24 shrink-0 text-right">
              {d.spend.toFixed(2)} {report.currency ?? ""}
            </span>
            <span className="w-10 shrink-0 text-right text-ink-muted">
              {report.pixel ? d.purchases : ""}
            </span>
          </div>
        ))}
      </div>

      <p className="max-w-2xl text-xs text-ink-muted">
        {t.ads.report.attribution}
        {report.lastSyncAt
          ? ` · ${t.ads.report.asOf}: ${new Date(
              report.lastSyncAt,
            ).toLocaleString()}`
          : ""}
      </p>
    </div>
  );
}

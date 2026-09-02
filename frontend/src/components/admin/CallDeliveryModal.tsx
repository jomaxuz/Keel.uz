"use client";

// Hands one order to an outside delivery service.
//
// Two routes, both first-class. Over the web (kind "link"/"phone") the panel
// prepares everything the service asks for: their form opens with the two
// addresses and the phone already in the URL, every field is copyable on its
// own for forms that take no parameters, and whoever was called is recorded on
// the order. Over the API (kind "api") the server files the request itself.

import { useEffect, useMemo, useState } from "react";
import { api, ApiError } from "@/lib/api";
import Modal from "@/components/admin/Modal";
import QrCode from "@/components/admin/QrCode";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import { formatPrice } from "@/lib/format";
import { formatDateTime } from "@/lib/orderFlow";
import {
  copyRows,
  missingPlaceholders,
  orderSummaryText,
  providerUrl,
} from "@/lib/providerLink";
import type { DeliveryProvider, Order, Restaurant } from "@/lib/types";

export default function CallDeliveryModal({
  order,
  onClose,
  onDone,
}: {
  order: Order;
  onClose: () => void;
  onDone: () => void;
}) {
  const t = useAdminT();
  const { ask } = useAsk();
  const [providers, setProviders] = useState<DeliveryProvider[]>([]);
  // The pickup side of every web form: where the courier collects the food.
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<string>("");
  const [tracking, setTracking] = useState(
    order.externalDelivery?.trackingId ?? "",
  );
  const [cost, setCost] = useState(
    order.externalDelivery?.cost ? String(order.externalDelivery.cost) : "",
  );
  const [note, setNote] = useState(order.externalDelivery?.note ?? "");
  const [copied, setCopied] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Live state of an API-filed request (claim id, status, courier, price).
  const [ext, setExt] = useState(order.externalDelivery ?? null);

  useEffect(() => {
    Promise.all([
      api.adminProviders().catch(() => [] as DeliveryProvider[]),
      api
        .getRestaurant()
        .then((r) => r.restaurant)
        .catch(() => null),
    ])
      .then(([list, shop]) => {
        const active = list.filter((p) => p.isActive);
        setProviders(active);
        setRestaurant(shop);
        setSelected(order.externalDelivery?.providerId ?? active[0]?.id ?? "");
      })
      .finally(() => setLoading(false));
  }, [order.externalDelivery?.providerId]);

  const provider = providers.find((p) => p.id === selected);
  const isWeb = provider ? provider.kind !== "api" : true;

  const summary = useMemo(
    () =>
      orderSummaryText(order, restaurant, {
        order: t.settings.orderWord,
        from: t.settings.callFrom,
        customer: t.receipt.customer,
        phone: t.couriers.phone,
        address: t.receipt.address,
        comment: t.receipt.comment,
        items: t.receipt.items,
        total: t.receipt.total,
        payment: t.receipt.payment,
        cash: t.settings.cashWord,
        online: t.settings.onlineWord,
      }),
    [order, restaurant, t],
  );

  const rows = useMemo(
    () =>
      copyRows(order, restaurant, {
        pickupAddress: t.settings.callFrom,
        pickupPhone: t.settings.callFromPhone,
        address: t.receipt.address,
        comment: t.receipt.comment,
        coords: t.settings.callCoords,
        customer: t.receipt.customer,
        phone: t.couriers.phone,
        items: t.receipt.items,
        cashToCollect: t.settings.callCashToCollect,
      }),
    [order, restaurant, t],
  );

  const template =
    provider?.kind === "link" && provider.url ? provider.url : "";
  const link = template ? providerUrl(template, order, restaurant) : "";
  const missing = template
    ? missingPlaceholders(template, order, restaurant)
    : [];

  async function copy(text: string, key: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(key);
      setTimeout(() => setCopied((c) => (c === key ? null : c)), 2000);
    } catch {
      /* clipboard blocked — the text is on screen anyway */
    }
  }

  // Providers with an API: the server files the request itself.
  async function callApi() {
    setBusy(true);
    setError(null);
    try {
      const res = await api.callProviderApi(order.id, selected);
      setExt(res.externalDelivery);
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function refreshApi() {
    setBusy(true);
    setError(null);
    try {
      const res = await api.syncProviderApi(order.id);
      setExt(res.externalDelivery);
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function cancelApi() {
    if (!(await ask({ title: t.settings.apiConfirmCancel, danger: true })))
      return;
    setBusy(true);
    setError(null);
    try {
      await api.cancelProviderApi(order.id);
      setExt(null);
      onDone();
      onClose();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  // Records who is carrying the order. `close` is false when the record is
  // written alongside opening the provider's site — the operator still needs
  // this window to copy the remaining fields into their form.
  async function mark({ clear = false, close = true } = {}) {
    setBusy(true);
    setError(null);
    try {
      await api.callProvider(order.id, {
        providerId: clear ? "" : selected,
        trackingId: clear ? "" : tracking,
        note: clear ? "" : note,
        cost: clear ? 0 : Number(cost) || 0,
      });
      if (!clear) {
        setExt({
          providerId: selected,
          providerName: provider?.name ?? "",
          trackingId: tracking,
          note,
          cost: Number(cost) || 0,
          calledAt: ext?.calledAt ?? new Date().toISOString(),
        });
      } else {
        setExt(null);
      }
      onDone();
      if (close) onClose();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const calledWithThis = !!ext?.calledAt && ext.providerId === selected;

  return (
    <Modal onClose={onClose}>
      <h3 className="font-display text-lg font-bold">{t.settings.callTitle}</h3>
      <p className="mt-1 text-sm text-ink-muted">
        #{order.number} · {order.customer.name} · {formatPrice(order.total)}
      </p>

      {loading ? (
        <p className="mt-4 text-sm text-ink-muted/70">{t.common.loading}</p>
      ) : providers.length === 0 ? (
        <p className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {t.settings.callNoProviders}
        </p>
      ) : (
        <>
          <div className="mt-4 flex flex-wrap gap-2">
            {providers.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setSelected(p.id)}
                className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                  selected === p.id
                    ? "border-brand bg-brand-tint/50 text-brand-dark"
                    : "border-line-strong text-ink-soft hover:border-brand"
                }`}
              >
                {p.name}
                {p.kind === "api" && (
                  <span className="ml-1 text-[10px] font-bold uppercase opacity-60">
                    api
                  </span>
                )}
              </button>
            ))}
          </div>

          {provider?.note && (
            <p className="mt-2 text-xs text-ink-muted">{provider.note}</p>
          )}

          {calledWithThis && (
            <p className="mt-3 rounded-2xl bg-emerald-50 px-4 py-2.5 text-sm text-emerald-800 dark:bg-emerald-500/10 dark:text-emerald-300">
              {t.settings.calledAtLine(
                ext!.providerName || provider?.name || "—",
                formatDateTime(ext!.calledAt),
              )}
            </p>
          )}

          {/* An API provider needs no copy-paste: the request is filed for us. */}
          {provider?.kind === "api" && (
            <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
              {ext?.claimId ? (
                <div className="space-y-1 text-sm">
                  <p>
                    <span className="text-ink-muted">
                      {t.settings.apiClaim}:{" "}
                    </span>
                    <span className="font-mono text-xs">{ext.claimId}</span>
                  </p>
                  <p>
                    <span className="text-ink-muted">
                      {t.settings.apiStatus}:{" "}
                    </span>
                    <span className="font-semibold">{ext.status}</span>
                  </p>
                  {ext.price && (
                    <p>
                      <span className="text-ink-muted">
                        {t.settings.apiPrice}:{" "}
                      </span>
                      {ext.price}
                    </p>
                  )}
                  {ext.courierName && (
                    <p>
                      <span className="text-ink-muted">
                        {t.settings.apiCourier}:{" "}
                      </span>
                      {ext.courierName}
                      {ext.courierPhone && (
                        <>
                          {" · "}
                          <a
                            href={`tel:${ext.courierPhone}`}
                            className="text-brand hover:underline"
                          >
                            {ext.courierPhone}
                          </a>
                        </>
                      )}
                    </p>
                  )}
                  <div className="mt-3 flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={busy}
                      onClick={refreshApi}
                      className="btn-ghost px-3 py-1.5 text-sm"
                    >
                      {t.settings.apiRefresh}
                    </button>
                    {ext.trackUrl && (
                      <a
                        href={ext.trackUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="btn-ghost px-3 py-1.5 text-sm"
                      >
                        {t.settings.apiTrack}
                      </a>
                    )}
                    <button
                      type="button"
                      disabled={busy}
                      onClick={cancelApi}
                      className="px-3 py-1.5 text-sm text-ink-muted hover:text-red-600"
                    >
                      {t.settings.apiCancel}
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  disabled={busy}
                  onClick={callApi}
                  className="btn-primary w-full py-2.5 text-sm"
                >
                  {busy ? t.settings.apiCalling : t.settings.apiCall}
                </button>
              )}
            </div>
          )}

          {/* ---- The web route: open their form, or fill it field by field ---- */}
          {isWeb && (
            <>
              {/* Step 1 — go to the service. */}
              <p className="mt-5 text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {t.settings.callStepOpen}
              </p>

              {provider?.kind === "phone" ? (
                provider.phone ? (
                  <a
                    href={`tel:${provider.phone.replace(/\s/g, "")}`}
                    className="btn-primary mt-2 inline-flex px-4 py-2 text-sm"
                    onClick={() => {
                      if (!calledWithThis) void mark({ close: false });
                    }}
                  >
                    {t.settings.callDial} · {provider.phone}
                  </a>
                ) : (
                  <p className="mt-2 text-sm text-amber-700 dark:text-amber-300">
                    {t.settings.callNoPhone}
                  </p>
                )
              ) : link ? (
                <div className="mt-2 flex flex-wrap items-start gap-4">
                  <div className="min-w-0 flex-1">
                    <a
                      href={link}
                      target="_blank"
                      rel="noreferrer"
                      className="btn-primary inline-flex px-4 py-2 text-sm"
                      // Opening their form is the moment the order leaves us,
                      // so the record is written right here — one click.
                      onClick={() => {
                        if (!calledWithThis) void mark({ close: false });
                      }}
                    >
                      {t.settings.callOpenAndMark}
                    </a>
                    <details className="mt-2">
                      <summary className="cursor-pointer text-xs text-ink-muted hover:text-ink">
                        {t.settings.callUrlPreview}
                      </summary>
                      <p className="mt-1 break-all font-mono text-[11px] text-ink-muted">
                        {link}
                      </p>
                    </details>
                    {missing.length > 0 && (
                      <p className="mt-2 rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
                        {t.settings.callMissing(missing.join(" "))}
                      </p>
                    )}
                  </div>

                  {/* App links (Yandex Go's delivery flow, for one) only work
                      on a phone — scanning carries them there with every
                      prefilled coordinate intact. */}
                  <div className="shrink-0 text-center">
                    <QrCode value={link} size={132} />
                    <p className="mt-1 max-w-[132px] text-[11px] leading-snug text-ink-muted">
                      {t.settings.callQrHint}
                    </p>
                  </div>
                </div>
              ) : (
                <p className="mt-2 text-sm text-amber-700 dark:text-amber-300">
                  {t.settings.callNoUrl}
                </p>
              )}

              {/* Step 2 — the fields their form asks for, one tap each. */}
              <div className="mt-5 flex items-center justify-between gap-3">
                <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                  {t.settings.callStepFields}
                </p>
                <button
                  type="button"
                  onClick={() => copy(summary, "all")}
                  className="btn-ghost px-3 py-1 text-xs"
                >
                  {copied === "all"
                    ? t.settings.callCopied
                    : t.settings.callCopy}
                </button>
              </div>
              <ul className="mt-2 divide-y divide-line rounded-2xl border border-line">
                {rows.map((row) => (
                  <li
                    key={row.key}
                    className="flex items-center justify-between gap-3 px-3 py-2"
                  >
                    <div className="min-w-0">
                      <p className="text-[11px] uppercase tracking-wide text-ink-muted">
                        {row.label}
                      </p>
                      <p className="truncate text-sm" title={row.value}>
                        {row.key === "cash"
                          ? formatPrice(Number(row.value))
                          : row.value}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => copy(row.value, row.key)}
                      className="shrink-0 rounded-full border border-line-strong px-2.5 py-1 text-xs font-medium text-ink-soft transition-colors hover:border-brand hover:text-brand"
                    >
                      {copied === row.key ? "✓" : t.settings.callCopyField}
                    </button>
                  </li>
                ))}
              </ul>

              {/* Step 3 — what the service told us back. */}
              <p className="mt-5 text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {t.settings.callStepRecord}
              </p>
              <div className="mt-2 grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="font-medium">{t.settings.trackingId}</span>
                  <input
                    className="mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
                    value={tracking}
                    onChange={(e) => setTracking(e.target.value)}
                  />
                </label>
                <label className="block text-sm">
                  <span className="font-medium">{t.settings.callCost}</span>
                  <input
                    className="mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
                    inputMode="numeric"
                    value={cost}
                    onChange={(e) =>
                      setCost(e.target.value.replace(/[^\d]/g, ""))
                    }
                  />
                </label>
              </div>
              <label className="mt-3 block text-sm">
                <span className="font-medium">{t.settings.providerNote}</span>
                <input
                  className="mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </label>
            </>
          )}
        </>
      )}

      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

      <div className="mt-6 flex flex-wrap justify-end gap-3">
        {order.externalDelivery && (
          <button
            type="button"
            disabled={busy}
            onClick={() => mark({ clear: true })}
            className="btn-ghost px-4 py-2 text-sm"
          >
            {t.settings.callClear}
          </button>
        )}
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2">
          {t.common.cancel}
        </button>
        {isWeb && (
          <button
            type="button"
            disabled={busy || !selected}
            onClick={() => mark()}
            className="btn-primary px-5 py-2"
          >
            {busy
              ? "..."
              : calledWithThis
                ? t.common.save
                : t.settings.callMark}
          </button>
        )}
      </div>
    </Modal>
  );
}

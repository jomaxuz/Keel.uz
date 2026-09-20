"use client";

// Outside delivery services (Yandex Delivery, taxi firms, door-to-door
// couriers) for restaurants that do not employ couriers. Each entry is either
// a link — opened with the order details substituted in — or a dispatcher's
// phone number.

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import { ListScroll } from "@/components/admin/PagedList";
import { PLACEHOLDERS } from "@/lib/providerLink";
import { useAdminScope } from "@/lib/adminScope";
import { shipsByPost } from "@/lib/types";
import type { DeliveryProvider } from "@/lib/types";

// Ordering a Yandex courier without a business account happens in the Yandex Go
// app — in Uzbekistan delivery.yandex.uz serves businesses only (that route is
// the "api" kind below), and the taxi web page has no delivery section at all.
// This is the link Yandex's own delivery site hands to phones: it resolves to
// `yandextaxi://route?...` with both points and the delivery tariff filled in,
// so the app opens on Dostavka rather than on a taxi ride. `adj_t` is Adjust's
// tracker token and is required — without it the link 404s.
//
// The tariff is what decides *which* section opens, and the classes differ by
// country. `express` — a car courier — is the one Yandex Go actually sells in
// Uzbekistan: it was checked on a phone in the city and opens straight on
// Dostavka with both addresses in place. Russia's `express_d2d` and `cargo`
// silently fall back to a taxi ride here, so they are not offered.
const YANDEX_GO_LINK =
  "https://yandex.go.link/route" +
  "?tariffClass=express" +
  "&adj_t=pucm71r&adj_campaign=web_ru_delivery&trap_mode=true" +
  "&start-lat={pickupLat}&start-lon={pickupLng}" +
  "&end-lat={lat}&end-lon={lng}";

// Starting points an owner can add with one tap, then edit. The web ones need
// no business account — that is why they come first.
const SAMPLES: {
  name: string;
  kind: "link" | "phone" | "api";
  url?: string;
  phone?: string;
  apiProvider?: string;
  /** Offered only to a business whose goods leave in a parcel. */
  needsPost?: boolean;
  // ⚠️ The note is a dictionary key, not a sentence. It is saved into the
  // provider record when the owner taps the sample, so an Uzbek default would
  // be written into a Russian panel's data and stay there.
  note?: "providerSampleNote";
}[] = [
  {
    name: "Yandex Go — Dostavka",
    kind: "link",
    url: YANDEX_GO_LINK,
    note: "providerSampleNote",
  },
  { name: "Millennium taxi", kind: "phone", phone: "+998712000000" },
  { name: "Yandex Delivery API", kind: "api", apiProvider: "yandex" },
  // ⚠️ **Only where the parcel actually travels by post** — an online store, a
  // clothes shop, a cosmetics shop (`shipsByPost`). A butcher and a florist
  // sell what they bought exactly as a boutique does, and a parcel of mince or
  // of tulips is a parcel nobody collects: offering them a carrier is offering
  // a setting that can only ever produce one.
  { name: "BTS Express", kind: "api", apiProvider: "bts", needsPost: true },
];

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function ProvidersEditor() {
  const t = useAdminT();
  const { ask } = useAsk();
  const scope = useAdminScope();
  const [items, setItems] = useState<DeliveryProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    api
      .adminProviders()
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  async function save(p: DeliveryProvider) {
    setError(null);
    try {
      await api.updateProvider(p.id, p);
      patch(p.id, { apiToken: "" });
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function add(seed?: (typeof SAMPLES)[number]) {
    setError(null);
    try {
      await api.createProvider({
        name: seed?.name ?? "",
        kind: seed?.kind ?? "link",
        url: seed?.url ?? "",
        phone: seed?.phone ?? "",
        note: seed?.note ? t.settings[seed.note] : "",
        apiProvider: seed?.apiProvider ?? "",
        isActive: true,
        sortOrder: items.length,
      });
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function remove(p: DeliveryProvider) {
    if (
      !(await ask({
        title: t.settings.providerConfirmDelete(p.name || "—"),
        danger: true,
      }))
    )
      return;
    try {
      await api.deleteProvider(p.id);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    }
  }

  // Local edit without a round-trip per keystroke; saved on blur.
  const patch = (id: string, p: Partial<DeliveryProvider>) =>
    setItems((prev) => prev.map((x) => (x.id === id ? { ...x, ...p } : x)));

  return (
    <div>
      <p className="text-sm text-ink-muted">{t.settings.providersHint}</p>

      {loading ? (
        <p className="mt-4 text-sm text-ink-muted/70">{t.common.loading}</p>
      ) : (
        <ListScroll className="mt-4 space-y-3 pr-1" max="max-h-[70vh]">
          {items.length === 0 && (
            <p className="rounded-2xl border border-dashed border-line-strong p-4 text-center text-sm text-ink-muted/70">
              {t.settings.providerEmpty}
            </p>
          )}

          {items.map((p) => (
            <div key={p.id} className="rounded-2xl border border-line p-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="font-medium">{t.settings.providerName}</span>
                  <input
                    className={inputCls}
                    value={p.name}
                    onChange={(e) => patch(p.id, { name: e.target.value })}
                    onBlur={() => save(p)}
                  />
                </label>
                <div className="text-sm">
                  <span className="font-medium">{t.settings.providerKind}</span>
                  <div className="mt-1 flex gap-2">
                    {(
                      [
                        ["link", t.settings.providerLink],
                        ["phone", t.settings.providerPhone],
                        ["api", t.settings.providerApi],
                      ] as const
                    ).map(([kind, label]) => (
                      <button
                        key={kind}
                        type="button"
                        onClick={() => {
                          patch(p.id, { kind });
                          save({ ...p, kind });
                        }}
                        className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors ${
                          p.kind === kind
                            ? "border-brand bg-brand-tint/50 text-brand-dark"
                            : "border-line-strong text-ink-soft hover:border-brand"
                        }`}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {p.kind === "api" ? (
                <div className="mt-3 space-y-3">
                  <p className="rounded-xl bg-ink/[0.03] px-3 py-2 text-xs text-ink-muted">
                    {t.settings.providerApiHint}
                  </p>
                  {/* ⚠️ **Which carrier, asked out loud.** It used to be set
                      only by tapping a sample, so a provider added with the
                      plain button was silently Yandex — and the way that shows
                      up is a BTS token posted at Yandex's host, which answers
                      with somebody else's error message.

                      ⚠️ An empty value stays Yandex, which is every provider
                      record written before there was a second carrier (the
                      server reads it the same way). */}
                  <div className="text-sm">
                    <span className="font-medium">
                      {t.settings.providerApiWhich}
                    </span>
                    <div className="mt-1 flex flex-wrap gap-2">
                      {(
                        [
                          ["yandex", "Yandex Delivery"],
                          ["bts", "BTS Express"],
                        ] as const
                      ).map(([id, label]) => (
                        <button
                          key={id}
                          type="button"
                          onClick={() => {
                            patch(p.id, { apiProvider: id });
                            save({ ...p, apiProvider: id });
                          }}
                          className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors ${
                            (p.apiProvider || "yandex") === id
                              ? "border-brand bg-brand-tint/50 text-brand-dark"
                              : "border-line-strong text-ink-soft hover:border-brand"
                          }`}
                        >
                          {label}
                        </button>
                      ))}
                    </div>
                  </div>
                  <label className="block text-sm">
                    <span className="font-medium">
                      {t.settings.providerApiToken}
                    </span>
                    <input
                      className={`${inputCls} font-mono text-xs`}
                      type="password"
                      value={p.apiToken ?? ""}
                      placeholder={p.hasToken ? "••••••••••••" : "y0_AgAAAA..."}
                      onChange={(e) =>
                        patch(p.id, { apiToken: e.target.value })
                      }
                      onBlur={() => save(p)}
                    />
                    {p.hasToken && (
                      <span className="mt-1 block text-xs text-emerald-600">
                        {t.settings.providerApiTokenSet}
                      </span>
                    )}
                  </label>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="block text-sm">
                      <span className="font-medium">
                        {t.settings.providerApiTariff}
                      </span>
                      <input
                        className={inputCls}
                        value={p.apiTariff ?? ""}
                        placeholder="express"
                        onChange={(e) =>
                          patch(p.id, { apiTariff: e.target.value })
                        }
                        onBlur={() => save(p)}
                      />
                    </label>
                    <label className="block text-sm">
                      <span className="font-medium">
                        {t.settings.providerApiBase}
                      </span>
                      <input
                        className={`${inputCls} font-mono text-xs`}
                        value={p.apiBaseUrl ?? ""}
                        placeholder={
                          p.apiProvider === "bts"
                            ? "https://…"
                            : "https://b2b.taxi.yandex.net"
                        }
                        onChange={(e) =>
                          patch(p.id, { apiBaseUrl: e.target.value })
                        }
                        onBlur={() => save(p)}
                      />
                      {/* ⚠️ **Said on the form, not only refused at the first
                          order.** Yandex publishes its host so the field is
                          optional there; BTS publishes none, and an empty one
                          is the most silent mistake on this screen — the form
                          looks filled in, the save succeeds, and the first
                          parcel goes nowhere. */}
                      {p.apiProvider === "bts" && (
                        <span className="mt-1 block text-xs text-amber-600">
                          {t.settings.providerApiBaseBts}
                        </span>
                      )}
                    </label>
                  </div>
                </div>
              ) : p.kind === "phone" ? (
                <label className="mt-3 block text-sm">
                  <span className="font-medium">
                    {t.settings.providerPhoneNum}
                  </span>
                  <input
                    className={inputCls}
                    value={p.phone}
                    placeholder="+998712000000"
                    onChange={(e) => patch(p.id, { phone: e.target.value })}
                    onBlur={() => save(p)}
                  />
                </label>
              ) : (
                <div className="mt-3 text-sm">
                  <label className="block">
                    <span className="font-medium">
                      {t.settings.providerUrl}
                    </span>
                    <input
                      className={`${inputCls} font-mono text-xs`}
                      value={p.url}
                      placeholder="https://example.com/order?address={address}&phone={phone}"
                      onChange={(e) => patch(p.id, { url: e.target.value })}
                      onBlur={() => save(p)}
                    />
                  </label>
                  <p className="mt-1 text-xs text-ink-muted">
                    {t.settings.providerUrlHint}
                  </p>
                  {/* Tapping a placeholder appends it — no need to type the
                      braces or remember the exact spelling. */}
                  <div className="mt-1 flex flex-wrap gap-1">
                    {PLACEHOLDERS.map((ph) => (
                      <button
                        key={ph}
                        type="button"
                        onClick={() => {
                          const next = { ...p, url: p.url + ph };
                          patch(p.id, { url: next.url });
                          save(next);
                        }}
                        className="rounded-full border border-line px-2 py-0.5 font-mono text-[11px] text-ink-muted transition-colors hover:border-brand hover:text-brand"
                      >
                        {ph}
                      </button>
                    ))}
                  </div>
                  <p className="mt-1 text-xs text-ink-muted/80">
                    {t.settings.providerUrlEmptyHint}
                  </p>
                </div>
              )}

              <label className="mt-3 block text-sm">
                <span className="font-medium">{t.settings.providerNote}</span>
                <input
                  className={inputCls}
                  value={p.note}
                  onChange={(e) => patch(p.id, { note: e.target.value })}
                  onBlur={() => save(p)}
                />
              </label>

              <div className="mt-3 flex flex-wrap items-center gap-4 text-sm">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={p.isActive}
                    onChange={(e) => {
                      patch(p.id, { isActive: e.target.checked });
                      save({ ...p, isActive: e.target.checked });
                    }}
                  />
                  {t.settings.providerActive}
                </label>
                <button
                  type="button"
                  onClick={() => remove(p)}
                  className="ml-auto text-sm text-ink-muted hover:text-red-600"
                >
                  {t.common.delete}
                </button>
              </div>
            </div>
          ))}
        </ListScroll>
      )}

      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => add()}
          className="btn-ghost px-3 py-1.5 text-sm"
        >
          {t.settings.providerAdd}
        </button>
        <span className="text-xs text-ink-muted">
          {t.settings.providerSamples}
        </span>
        {SAMPLES.filter(
          (s) => !s.needsPost || shipsByPost(scope.brand),
        ).map((s) => (
          <button
            key={s.name}
            type="button"
            onClick={() => add(s)}
            className="rounded-full border border-line-strong px-3 py-1 text-xs font-medium text-ink-soft transition-colors hover:border-brand hover:text-brand"
          >
            {s.name}
          </button>
        ))}
      </div>
    </div>
  );
}

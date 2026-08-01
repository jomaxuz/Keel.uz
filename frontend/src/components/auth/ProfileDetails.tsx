"use client";

// "My details" card on the profile page: name, verified phone number and saved
// delivery addresses. The phone can only change through a fresh SMS code, so
// the number on file is always one the customer controls.

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useUser } from "@/lib/user";
import { useI18n } from "@/lib/i18n/client";
import { formatUzPhone } from "@/lib/format";
import AddressPicker from "@/components/map/AddressPicker";
import type { LatLng } from "@/components/map/AddressMap";
import type { Restaurant, UserAddress } from "@/lib/types";

// Tashkent, until the restaurant profile is loaded.
const DEFAULT_CENTER: LatLng = { lat: 41.311081, lng: 69.279737 };

export default function ProfileDetails() {
  const { user, update } = useUser();
  const { t } = useI18n();
  // Loaded for the map: where to centre it and which delivery zones to draw.
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setRestaurant(r.restaurant))
      .catch(() => setRestaurant(null));
  }, []);

  const [editing, setEditing] = useState(false);
  const [firstName, setFirstName] = useState(user?.firstName ?? "");
  const [addresses, setAddresses] = useState<UserAddress[]>(
    user?.addresses ?? [],
  );
  const [openAddress, setOpenAddress] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  // Phone change flow
  const [phoneStep, setPhoneStep] = useState<"idle" | "phone" | "code">("idle");
  const [newPhone, setNewPhone] = useState("");
  const [code, setCode] = useState("");
  const [demoCode, setDemoCode] = useState<string | null>(null);

  if (!user) return null;

  function startEdit() {
    setFirstName(user!.firstName ?? "");
    setAddresses(user!.addresses ?? []);
    setError(null);
    setEditing(true);
  }

  async function save() {
    setBusy(true);
    setError(null);
    try {
      const updated = await api.updateMe({
        firstName,
        addresses: addresses.filter((a) => a.text.trim() !== ""),
      });
      update(updated);
      setEditing(false);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.login.error);
    } finally {
      setBusy(false);
    }
  }

  async function requestPhoneCode() {
    setBusy(true);
    setError(null);
    try {
      const res = await api.changePhoneRequest(newPhone);
      setNewPhone(res.phone);
      setDemoCode(res.code ?? null);
      setPhoneStep("code");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.login.error);
    } finally {
      setBusy(false);
    }
  }

  async function verifyPhone() {
    setBusy(true);
    setError(null);
    try {
      const updated = await api.changePhoneVerify(newPhone, code);
      update(updated);
      setPhoneStep("idle");
      setCode("");
      setDemoCode(null);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.login.error);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="card mt-8 p-6">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-lg font-bold">{t.profile.infoTitle}</h2>
        <div className="flex items-center gap-3">
          {saved && (
            <span className="text-xs font-semibold text-emerald-600">
              {t.profile.saved}
            </span>
          )}
          {!editing && (
            <button
              type="button"
              onClick={startEdit}
              className="text-sm font-semibold text-brand hover:underline"
            >
              {t.profile.edit}
            </button>
          )}
        </div>
      </div>

      {/* ---- read mode ---- */}
      {!editing && (
        <dl className="mt-4 space-y-3 text-sm">
          <div className="flex justify-between gap-4">
            <dt className="text-ink-muted">{t.profile.name}</dt>
            <dd className="text-right font-medium">
              {[user.firstName, user.lastName].filter(Boolean).join(" ") || "—"}
            </dd>
          </div>
          <div className="flex justify-between gap-4">
            <dt className="text-ink-muted">{t.profile.phone}</dt>
            <dd className="text-right font-medium">
              {user.phone ? formatUzPhone(user.phone) : t.profile.noPhone}
            </dd>
          </div>
          <div>
            <dt className="text-ink-muted">{t.profile.addresses}</dt>
            <dd className="mt-2 space-y-2">
              {(user.addresses ?? []).length === 0 ? (
                <p className="text-ink-muted/70">{t.profile.noAddresses}</p>
              ) : (
                user.addresses!.map((a, i) => (
                  <div
                    key={`${a.text}-${i}`}
                    className="rounded-xl bg-ink/[0.03] px-3 py-2"
                  >
                    {a.label && (
                      <span className="mr-2 font-semibold">{a.label}:</span>
                    )}
                    {a.text}
                  </div>
                ))
              )}
            </dd>
          </div>
        </dl>
      )}

      {/* ---- edit mode ---- */}
      {editing && (
        <div className="mt-4 space-y-4">
          <label className="block text-sm">
            <span className="font-medium">{t.profile.name}</span>
            <input
              className="input mt-1"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
            />
          </label>

          <div className="text-sm">
            <span className="font-medium">{t.profile.addresses}</span>
            <div className="mt-2 space-y-3">
              {addresses.map((a, i) => (
                <div
                  key={i}
                  className="rounded-2xl border border-line bg-ink/[0.02] p-3"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <input
                      className="input max-w-[180px]"
                      placeholder={t.profile.addressLabel}
                      value={a.label}
                      onChange={(e) =>
                        setAddresses(
                          addresses.map((x, j) =>
                            j === i ? { ...x, label: e.target.value } : x,
                          ),
                        )
                      }
                    />
                    <span className="flex-1 truncate text-ink-muted">
                      {a.text || t.profile.addressText}
                    </span>
                    <button
                      type="button"
                      onClick={() =>
                        setOpenAddress(openAddress === i ? null : i)
                      }
                      className="btn-ghost px-3 py-2 text-xs"
                    >
                      {openAddress === i ? t.profile.hideMap : t.profile.onMap}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setAddresses(addresses.filter((_, j) => j !== i));
                        setOpenAddress(null);
                      }}
                      className="btn-ghost px-3 py-2 text-xs"
                    >
                      {t.profile.removeAddress}
                    </button>
                  </div>

                  {openAddress === i && (
                    <div className="mt-3">
                      <AddressPicker
                        value={{ text: a.text, lat: a.lat, lng: a.lng }}
                        onChange={(next) =>
                          setAddresses(
                            addresses.map((x, j) =>
                              j === i
                                ? {
                                    ...x,
                                    text: next.text,
                                    lat: next.lat,
                                    lng: next.lng,
                                  }
                                : x,
                            ),
                          )
                        }
                        center={
                          a.lat && a.lng
                            ? { lat: a.lat, lng: a.lng }
                            : restaurant?.address?.lat
                              ? {
                                  lat: restaurant.address.lat,
                                  lng: restaurant.address.lng,
                                }
                              : DEFAULT_CENTER
                        }
                        mapClassName="h-56 w-full"
                        zones={restaurant?.delivery.zones}
                        currency={restaurant?.currency}
                      />
                      <input
                        className="input mt-3"
                        placeholder={t.checkout.commentPh}
                        value={a.comment}
                        onChange={(e) =>
                          setAddresses(
                            addresses.map((x, j) =>
                              j === i ? { ...x, comment: e.target.value } : x,
                            ),
                          )
                        }
                      />
                    </div>
                  )}
                </div>
              ))}
              <button
                type="button"
                onClick={() => {
                  setAddresses([
                    ...addresses,
                    { label: "", text: "", lat: 0, lng: 0, comment: "" },
                  ]);
                  setOpenAddress(addresses.length);
                }}
                className="text-sm font-semibold text-brand hover:underline"
              >
                {t.profile.addAddress}
              </button>
            </div>
          </div>

          {error && <p className="text-sm text-brand">{error}</p>}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={save}
              disabled={busy}
              className="btn-primary px-5 py-2.5"
            >
              {busy ? t.profile.saving : t.profile.save}
            </button>
            <button
              type="button"
              onClick={() => setEditing(false)}
              className="btn-ghost px-5 py-2.5"
            >
              {t.profile.cancel}
            </button>
          </div>
        </div>
      )}

      {/* ---- phone change (always available, verified by SMS) ---- */}
      <div className="mt-6 border-t border-line pt-4">
        {phoneStep === "idle" ? (
          <button
            type="button"
            onClick={() => {
              setPhoneStep("phone");
              setError(null);
            }}
            className="text-sm font-semibold text-brand hover:underline"
          >
            {t.profile.changePhoneTitle}
          </button>
        ) : (
          <div className="space-y-3">
            <p className="text-sm font-semibold">{t.profile.changePhoneTitle}</p>
            {phoneStep === "phone" ? (
              <div className="flex flex-wrap gap-2">
                <input
                  className="input max-w-xs"
                  type="tel"
                  value={newPhone}
                  onChange={(e) => setNewPhone(e.target.value)}
                  placeholder="+998 90 123 45 67"
                />
                <button
                  type="button"
                  onClick={requestPhoneCode}
                  disabled={busy}
                  className="btn-primary px-5 py-2.5"
                >
                  {busy ? t.login.sending : t.login.sendCode}
                </button>
              </div>
            ) : (
              <div className="space-y-2">
                <div className="flex flex-wrap gap-2">
                  <input
                    className="input max-w-[160px] text-center font-bold tracking-[0.3em]"
                    inputMode="numeric"
                    maxLength={6}
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  />
                  <button
                    type="button"
                    onClick={verifyPhone}
                    disabled={busy || code.length < 4}
                    className="btn-primary px-5 py-2.5"
                  >
                    {busy ? t.login.verifying : t.login.verify}
                  </button>
                </div>
                {demoCode && (
                  <p className="rounded-xl bg-brand-tint px-3 py-2 text-xs font-semibold text-brand-dark">
                    {t.login.demoNotice(demoCode)}
                  </p>
                )}
              </div>
            )}
            {error && <p className="text-sm text-brand">{error}</p>}
            <button
              type="button"
              onClick={() => {
                setPhoneStep("idle");
                setCode("");
                setDemoCode(null);
                setError(null);
              }}
              className="text-xs text-ink-muted hover:text-brand"
            >
              {t.profile.cancel}
            </button>
          </div>
        )}
      </div>
    </section>
  );
}

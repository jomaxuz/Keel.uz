"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { formatPrice, weekdayName } from "@/lib/format";
import ImageUpload from "@/components/admin/ImageUpload";
import AddressMap, { type LatLng } from "@/components/map/AddressMap";
import AddressAutocomplete from "@/components/map/AddressAutocomplete";
import { reverseGeocode } from "@/lib/geocode";
import DeliveryZonesEditor from "@/components/admin/DeliveryZonesEditor";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import BannersEditor from "@/components/admin/BannersEditor";
import DesignEditor from "@/components/admin/DesignEditor";
import ProvidersEditor from "@/components/admin/ProvidersEditor";
import PaymentsEditor from "@/components/admin/PaymentsEditor";
import SmsEditor from "@/components/admin/SmsEditor";
import TelegramEditor from "@/components/admin/TelegramEditor";
import DataExport from "@/components/admin/DataExport";
import POSEditor from "@/components/admin/POSEditor";
import PBXEditor from "@/components/admin/PBXEditor";
import FloorPlanEditor from "@/components/admin/FloorPlanEditor";
import BranchesEditor from "@/components/admin/BranchesEditor";
import PerksEditor from "@/components/admin/PerksEditor";
import LocalizedField from "@/components/admin/LocalizedField";
import { EMPTY_LOCALIZED } from "@/lib/i18n/site-content";
import { MAP_PROVIDERS, normalizeProvider } from "@/lib/map";
import { forgetMapConfig } from "@/lib/map/config";
import { EMPTY_THEME } from "@/lib/theme-css";
import type {
  BookingSettings,
  LoyaltySettings,
  SiteContent,
  SiteTheme,
} from "@/lib/types";
import Link from "next/link";
import type {
  PreorderSettings,
  Restaurant,
  ReviewSettings,
  WorkingHour,
} from "@/lib/types";

// Display order: Monday-first (backend day: 0=Sunday).
const DAY_ORDER = [1, 2, 3, 4, 5, 6, 0];

function ensureHours(hours: WorkingHour[]): WorkingHour[] {
  return DAY_ORDER.map((day) => {
    const found = hours.find((h) => h.day === day);
    return (
      found ?? { day, open: "09:00", close: "22:00", isClosed: false }
    );
  });
}

const DEFAULT_CENTER: LatLng = { lat: 41.311081, lng: 69.279737 };

// Which fields of this form belong to the branch rather than the company.
// A branch cooks for itself: its address, its phones, its hours, its delivery
// area and its dining room. Everything else — the name, the logo, the socials,
// the site copy and the design — is the company's face and is shared.
const BRANCH_FIELDS = [
  "address",
  "phones",
  "workingHours",
  "delivery",
  "booking",
  "preorder",
] as const;

// And which belong to the brand: the face the guest sees. A company running a
// restaurant and a samsa chain needs two names, two logos and two colour
// schemes — the socials, the currency and the customer base stay shared.
const BRAND_FIELDS = [
  "name",
  "description",
  "logoUrl",
  "coverUrl",
  "content",
  "theme",
] as const;

export default function AdminSettingsPage() {
  const [rest, setRest] = useState<Restaurant | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [designLocked, setDesignLocked] = useState(false);
  const t = useAdminT();
  const [saved, setSaved] = useState(false);
  const scope = useAdminScope();
  // Which branch's own settings this form is editing. With one branch it is
  // simply that one; with several the panel's switcher decides.
  const editedBranch =
    scope.branch ?? (scope.brandBranches.length === 1 ? scope.brandBranches[0] : null);

  useEffect(() => {
    // The company document as stored (raw=1), then the branch's own fields laid
    // over it — so the form edits one coherent picture and `save` splits it
    // back apart.
    api
      .getRestaurant({ raw: true })
      .then((r) => {
        // A console-drawn layout is live, so this page must not offer the theme
        // knobs that would undo it.
        setDesignLocked(!!r.designLocked);
        const base = r.restaurant;
        const brand = scope.brand;
        const merged: Restaurant = {
          ...base,
          // The brand's own face, falling back to the company's where the brand
          // has never been given one.
          ...(brand
            ? {
                name: brand.name || base.name,
                description: brand.description || base.description,
                logoUrl: brand.logoUrl || base.logoUrl,
                coverUrl: brand.coverUrl || base.coverUrl,
                content: brand.content ?? base.content,
                theme: brand.theme ?? base.theme,
              }
            : {}),
          ...(editedBranch
            ? {
                address: editedBranch.address,
                phones: editedBranch.phones ?? [],
                workingHours: editedBranch.workingHours ?? [],
                delivery: editedBranch.delivery,
                booking: editedBranch.booking,
                preorder: editedBranch.preorder,
              }
            : {}),
        };
        setRest({ ...merged, workingHours: ensureHours(merged.workingHours) });
      })
      .catch(() => setRest(null))
      .finally(() => setLoading(false));
  }, [editedBranch]);

  function patch(p: Partial<Restaurant>) {
    setRest((r) => (r ? { ...r, ...p } : r));
  }

  async function save() {
    if (!rest) return;
    setSaving(true);
    setSaved(false);
    try {
      if (editedBranch) {
        // Branch-owned fields go to the branch; the company keeps the rest.
        // Sending them to both would leave two copies that quietly disagree.
        await api.updateBranch(editedBranch.id, {
          ...editedBranch,
          address: rest.address,
          phones: rest.phones,
          workingHours: rest.workingHours,
          delivery: rest.delivery,
          booking: rest.booking,
          preorder: rest.preorder,
        });
      }
      if (scope.isOwner && scope.brand) {
        await api.updateBrand(scope.brand.id, {
          ...scope.brand,
          name: rest.name,
          description: rest.description,
          logoUrl: rest.logoUrl,
          coverUrl: rest.coverUrl,
          content: rest.content,
          theme: rest.theme,
        });
      }
      if (scope.isOwner) {
        const company = { ...rest };
        if (editedBranch) {
          for (const key of BRANCH_FIELDS) {
            delete (company as Record<string, unknown>)[key];
          }
        }
        if (scope.brand) {
          for (const key of BRAND_FIELDS) {
            delete (company as Record<string, unknown>)[key];
          }
        }
        await api.updateRestaurant(company as Restaurant);
      }
      // The form already shows the truth (it is what we just saved); re-reading
      // the company document would put its stale copies back on screen.
      scope.reload();
      // ⚠️ The map config is cached for the whole page load, so without this an
      // owner who has just switched provider or pasted a key would keep seeing
      // the old map — including the zone editor right below this form, which is
      // the one place the change is meant to be checked.
      forgetMapConfig();
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch {
      alert(t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>;
  }
  if (!rest) {
    return (
      <p className="py-10 text-center text-ink-muted/70">
        Restoran ma'lumotini yuklab bo'lmadi.
      </p>
    );
  }
  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  // Address, hours, delivery and the dining room belong to **one** branch. With
  // several branches and none selected they have nowhere to be saved, so those
  // sections say so. Everything else on this page is the company's or the
  // brand's and stays editable regardless — the design and the site copy do not
  // become unreachable just because the company opened a second branch.
  const needsBranch = !editedBranch && scope.brandBranches.length > 1;
  // No picker here: the lens lives in one place — the sidebar switcher, under
  // the language and theme controls — and every screen reads through it. A
  // second copy of the same control on one screen only teaches that the panel
  // has two of them.
  const branchGate = needsBranch ? (
    <p className="rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
      {t.settings.pickBranchFirst}
    </p>
  ) : null;
  // Absent on a branch that has never had this section opened, which reads as
  // off. The defaults here are the server's (see preorderSettings) — an owner
  // switching the feature on should find sensible numbers already in the
  // fields, not four zeros that would refuse every order.
  const preorder: PreorderSettings = rest.preorder ?? {
    enabled: false,
    leadMinutes: 60,
    minMinutes: 60,
    maxDays: 7,
    slotMinutes: 30,
  };
  // Absent on a document written before reviews existed, which reads as off:
  // nothing a guest wrote privately starts appearing because a field was added.
  const reviews: ReviewSettings = rest.reviews ?? {
    enabled: false,
    showAverage: true,
  };
  // Mirrors quoteDelivery: an empty mode means "zones when some are drawn".
  const deliveryMode: "radius" | "zones" =
    rest.delivery.mode ??
    ((rest.delivery.zones ?? []).some((z) => (z.polygon?.length ?? 0) >= 3)
      ? "zones"
      : "radius");

  // What this branch's delivery settings actually mean, checked against the
  // rules the server applies (quoteDeliveryRules + deliveryBranch).
  //
  // ⚠️ Each of these is a setting that is **silently** wrong. Nothing here
  // fails to save, nothing errors, and the form looks complete — the mistake
  // only shows up as a branch that never receives an order, or as one that
  // quietly takes the orders every other branch was supposed to get. That is
  // exactly how a live install ended up sending every Yangiyo'l delivery to
  // Chilonzor: two branches had delivery switched off, and no screen said so.
  const deliveryWarnings: string[] = [];
  if (rest.delivery.enabled) {
    // The origin every distance is measured from. At (0,0) — a branch whose
    // address was typed but never pinned — every distance is thousands of
    // kilometres, so the branch either covers nothing or covers everything.
    if (!rest.address?.lat || !rest.address?.lng) {
      deliveryWarnings.push(t.settings.warnNoPin);
    }
    if (deliveryMode === "radius") {
      // ⚠️ 0 is "no limit", not "does not deliver". With several branches this
      // is the setting that hurts: an unlimited branch is a candidate for
      // every address in the country.
      if (!(rest.delivery.maxKm > 0)) {
        deliveryWarnings.push(
          scope.brandBranches.length > 1
            ? t.settings.warnNoMaxKmMulti
            : t.settings.warnNoMaxKm,
        );
      }
      if (rest.delivery.baseFee === 0 && rest.delivery.perKm === 0) {
        deliveryWarnings.push(t.settings.warnFreeDelivery);
      }
    }
  } else if (scope.brandBranches.length > 1) {
    // Off, with siblings that are on: the branch is invisible to delivery and
    // its share goes to whichever branch is nearest and switched on.
    deliveryWarnings.push(t.settings.warnDeliveryOff);
  }


  const content: SiteContent = {
    aboutTitle: rest.content?.aboutTitle ?? EMPTY_LOCALIZED,
    aboutText: rest.content?.aboutText ?? EMPTY_LOCALIZED,
    footerNote: rest.content?.footerNote ?? EMPTY_LOCALIZED,
    tagline: rest.content?.tagline ?? EMPTY_LOCALIZED,
    // Empty means the built-in strip, never "wanted blank" — the hide flag is
    // the separate answer, and its zero value is today's page.
    perks: rest.content?.perks ?? [],
    hidePerks: rest.content?.hidePerks ?? false,
  };
  const theme: SiteTheme = { ...EMPTY_THEME, ...(rest.theme ?? {}) };
  const booking: BookingSettings = {
    enabled: false,
    width: 1000,
    height: 700,
    slotMinutes: 90,
    maxDaysAhead: 30,
    minNoticeMinutes: 30,
    maxGuests: 20,
    shapes: [],
    tables: [],
    note: "",
    ...(rest.booking ?? {}),
  };
  // An unconfigured install reads as "off", and a missing ceiling as half the
  // order — never as "points may pay for everything".
  const loyalty: LoyaltySettings = {
    enabled: false,
    earnPercent: 5,
    minOrderToEarn: 0,
    maxRedeemPercent: 50,
    welcomePoints: 0,
    ...(rest.loyalty ?? {}),
  };

  const patchContent = (p: Partial<SiteContent>) =>
    patch({ content: { ...content, ...p } });

  const center: LatLng = rest.address.lat
    ? { lat: rest.address.lat, lng: rest.address.lng }
    : DEFAULT_CENTER;

  // Dropping a pin shows the point at once; the text follows once geocoded —
  // same two-step AddressPicker does at checkout. Best-effort: a failed lookup
  // leaves the owner typing the address, not without a marker.
  async function pickAddressOnMap(p: LatLng) {
    patch({ address: { ...rest!.address, lat: p.lat, lng: p.lng } });
    try {
      const text = await reverseGeocode(p.lat, p.lng);
      if (text) {
        setRest((r) =>
          r ? { ...r, address: { ...r.address, text, lat: p.lat, lng: p.lng } } : r,
        );
      }
    } catch {
      /* reverse geocode is best-effort */
    }
  }

  return (
    <div className="max-w-3xl">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t.settings.title}</h1>
        <div className="flex items-center gap-3">
          {saved && <span className="text-sm text-emerald-600">{t.settings.saved}</span>}
          <button
            type="button"
            onClick={save}
            disabled={saving}
            className="btn-primary px-5 py-2 disabled:opacity-60"
          >
            {saving ? t.common.saving : t.common.save}
          </button>
        </div>
      </div>

      {/* Profile */}
      {/* Company and brand identity: the name, the logo, the socials. A branch
          manager runs a kitchen, so these are the owner's. Their branch's own
          address, hours, delivery and dining room follow below. */}
      {scope.isOwner && (
      <Section title={t.settings.profile}>
        <label className="block text-sm">
          <span className="font-medium">{t.settings.restaurantName}</span>
          <input
            className={inputCls}
            value={rest.name}
            onChange={(e) => patch({ name: e.target.value })}
          />
        </label>
        <label className="mt-4 block text-sm">
          <span className="font-medium">{t.settings.description}</span>
          <textarea
            className={inputCls}
            rows={3}
            value={rest.description}
            onChange={(e) => patch({ description: e.target.value })}
          />
        </label>

        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <div>
            <span className="text-sm font-medium">{t.settings.logo}</span>
            <div className="mt-1">
              <ImageUpload
                value={rest.logoUrl}
                onChange={(url) => patch({ logoUrl: url })}
              />
            </div>
          </div>
          <div>
            <span className="text-sm font-medium">{t.settings.coverLabel}</span>
            <div className="mt-1">
              <ImageUpload
                value={rest.coverUrl}
                onChange={(url) => patch({ coverUrl: url })}
              />
            </div>
          </div>
        </div>

        <label className="mt-4 block text-sm">
          <span className="font-medium">{t.settings.phones}</span>
          <input
            className={inputCls}
            value={rest.phones.join(", ")}
            onChange={(e) =>
              patch({
                phones: e.target.value
                  .split(",")
                  .map((p) => p.trim())
                  .filter(Boolean),
              })
            }
          />
        </label>

        <div className="mt-4 grid gap-4 sm:grid-cols-3">
          <label className="block text-sm">
            <span className="font-medium">Instagram</span>
            <input
              className={inputCls}
              value={rest.socials.instagram}
              onChange={(e) =>
                patch({ socials: { ...rest.socials, instagram: e.target.value } })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium">Telegram</span>
            <input
              className={inputCls}
              value={rest.socials.telegram}
              onChange={(e) =>
                patch({ socials: { ...rest.socials, telegram: e.target.value } })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium">Facebook</span>
            <input
              className={inputCls}
              value={rest.socials.facebook}
              onChange={(e) =>
                patch({ socials: { ...rest.socials, facebook: e.target.value } })
              }
            />
          </label>
        </div>
      </Section>
      )}

      {/* Address — the branch's own */}
      <Section title={t.settings.address} blockedBy={branchGate}>

        <label className="block text-sm">
          <span className="font-medium">{t.settings.addressText}</span>
          {/* Same input the guest gets at checkout: typing suggests real
              addresses, and picking one moves the marker below. The map alone
              answers "where", but nobody knows their own restaurant by
              coordinates — the text is what ends up on the contact page. */}
          <AddressAutocomplete
            className={inputCls}
            value={rest.address.text}
            onTextChange={(text) => patch({ address: { ...rest.address, text } })}
            onSelect={(p) =>
              patch({ address: { text: p.text, lat: p.lat, lng: p.lng } })
            }
          />
        </label>
        <p className="mt-4 mb-2 text-sm font-medium">
          {t.settings.pickOnMap}
        </p>
        <AddressMap
          value={rest.address.lat ? { lat: rest.address.lat, lng: rest.address.lng } : null}
          onChange={(p) => pickAddressOnMap(p)}
          center={center}
          className="h-64 w-full"
        />
        <p className="mt-2 text-xs text-ink-muted">
          {rest.address.lat
            ? t.settings.coordinates(
                rest.address.lat.toFixed(5),
                rest.address.lng.toFixed(5),
              )
            : t.settings.noCoordinates}
        </p>
      </Section>

      {/* Working hours */}
      <Section title={t.settings.hours} blockedBy={branchGate}>

        <div className="space-y-2">
          {rest.workingHours.map((h, idx) => (
            <div key={h.day} className="flex items-center gap-3 text-sm">
              <span className="w-24 font-medium">{weekdayName(h.day)}</span>
              <input
                type="time"
                disabled={h.isClosed}
                value={h.open}
                onChange={(e) => {
                  const wh = [...rest.workingHours];
                  wh[idx] = { ...h, open: e.target.value };
                  patch({ workingHours: wh });
                }}
                className="rounded-xl border border-line-strong px-2 py-1 disabled:opacity-40"
              />
              <span className="text-ink-muted/70">–</span>
              <input
                type="time"
                disabled={h.isClosed}
                value={h.close}
                onChange={(e) => {
                  const wh = [...rest.workingHours];
                  wh[idx] = { ...h, close: e.target.value };
                  patch({ workingHours: wh });
                }}
                className="rounded-xl border border-line-strong px-2 py-1 disabled:opacity-40"
              />
              <label className="ml-2 flex items-center gap-1.5">
                <input
                  type="checkbox"
                  checked={h.isClosed}
                  onChange={(e) => {
                    const wh = [...rest.workingHours];
                    wh[idx] = { ...h, isClosed: e.target.checked };
                    patch({ workingHours: wh });
                  }}
                />
                <span className="text-ink-muted">{t.settings.closed}</span>
              </label>
            </div>
          ))}
        </div>
      </Section>

      {/* Guests' words on the site. Company-level, so no branch gate: which
          reviews appear is chosen on the feedback screen, and this is only the
          switch that opens the section. */}
      <Section title={t.settings.reviewsTitle}>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={reviews.enabled}
            onChange={(e) =>
              patch({ reviews: { ...reviews, enabled: e.target.checked } })
            }
          />
          <span className="font-medium">{t.settings.reviewsEnabled}</span>
        </label>
        {reviews.enabled && (
          <label className="mt-3 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={reviews.showAverage}
              onChange={(e) =>
                patch({ reviews: { ...reviews, showAverage: e.target.checked } })
              }
            />
            <span className="font-medium">{t.settings.reviewsAverage}</span>
          </label>
        )}
        {/* ⚠️ The important sentence, and it is a warning rather than a hint:
            an owner who expects this switch to publish everything will
            otherwise read an empty section as a bug and go looking in the
            wrong place. */}
        <p className="mt-3 text-xs text-ink-muted">{t.settings.reviewsHint}</p>
        {reviews.enabled && (
          <Link
            href="/admin/feedback"
            className="btn-ghost mt-3 inline-flex px-4 py-2 text-sm"
          >
            {t.settings.reviewsPick}
          </Link>
        )}
      </Section>

      {/* Ordering for later. Its own section rather than a corner of the
          delivery one: it applies to pickup as much as to delivery, and the
          one number in it — how much warning the kitchen gets — is the whole
          feature. Branch-owned, because the kitchen that cooks it is the only
          one that knows what warning it needs. */}
      <Section title={t.settings.preorderTitle} blockedBy={branchGate}>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={preorder.enabled}
            onChange={(e) =>
              patch({ preorder: { ...preorder, enabled: e.target.checked } })
            }
          />
          <span className="font-medium">{t.settings.preorderEnabled}</span>
        </label>

        {preorder.enabled && (
          <>
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <NumField
                label={t.settings.preorderLead}
                value={preorder.leadMinutes}
                onChange={(v) =>
                  patch({ preorder: { ...preorder, leadMinutes: v } })
                }
              />
              <NumField
                label={t.settings.preorderMin}
                value={preorder.minMinutes}
                onChange={(v) =>
                  patch({ preorder: { ...preorder, minMinutes: v } })
                }
              />
              <NumField
                label={t.settings.preorderDays}
                value={preorder.maxDays}
                onChange={(v) =>
                  patch({ preorder: { ...preorder, maxDays: v } })
                }
              />
              <NumField
                label={t.settings.preorderSlot}
                value={preorder.slotMinutes}
                onChange={(v) =>
                  patch({ preorder: { ...preorder, slotMinutes: v } })
                }
              />
            </div>
            {/* The lead hint first and on its own: it is the field an owner
                will get wrong, and getting it wrong is invisible until food
                comes out at the wrong hour. */}
            <p className="mt-3 text-xs text-ink-muted">
              {t.settings.preorderLeadHint}
            </p>
            <p className="mt-2 text-xs text-ink-muted">
              {t.settings.preorderMinHint}
            </p>
            <p className="mt-2 text-xs text-ink-muted">
              {t.settings.preorderSlotHint}
            </p>
          </>
        )}
      </Section>

      {/* Delivery — one section, one decision: how is the fee computed? */}
      <Section title={t.settings.delivery} blockedBy={branchGate}>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={rest.delivery.enabled}
            onChange={(e) =>
              patch({ delivery: { ...rest.delivery, enabled: e.target.checked } })
            }
          />
          <span className="font-medium">{t.settings.deliveryEnabled}</span>
        </label>

        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <NumField
            label={t.settings.minOrder}
            value={rest.delivery.minOrder}
            onChange={(v) =>
              patch({ delivery: { ...rest.delivery, minOrder: v } })
            }
          />
          <NumField
            label={t.settings.arrivalRadius}
            value={rest.delivery.arrivalRadiusM ?? 0}
            onChange={(v) =>
              patch({ delivery: { ...rest.delivery, arrivalRadiusM: v } })
            }
          />
          <NumField
            label={t.settings.freeFrom}
            value={rest.delivery.freeDeliveryFrom ?? 0}
            onChange={(v) =>
              patch({
                delivery: {
                  ...rest.delivery,
                  freeDeliveryFrom: v > 0 ? v : null,
                },
              })
            }
          />
        </div>
        <p className="mt-2 text-xs text-ink-muted">
          {t.settings.arrivalRadiusHint}
        </p>

        {/* ⚠️ What this branch actually covers, said out loud.
            Every warning below describes a setting that is silently wrong: the
            form looks filled in, nothing errors, and the mistake surfaces
            somewhere else entirely — as a branch that never gets an order, or
            as one that quietly takes everybody else's. */}
        {deliveryWarnings.length > 0 && (
          <div className="mt-4 space-y-2">
            {deliveryWarnings.map((w) => (
              <p
                key={w}
                className="rounded-xl bg-amber-50 px-4 py-3 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300"
              >
                {w}
              </p>
            ))}
          </div>
        )}

        {/* Narx qanday hisoblanadi — ikkitadan biri ishlaydi. */}
        <div className="mt-6 border-t border-line pt-5">
          <p className="text-sm font-semibold">{t.settings.howPriced}</p>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            {(
              [
                [
                  "radius",
                  t.settings.modeRadius,
                  t.settings.modeRadiusHint,
                ],
                [
                  "zones",
                  t.settings.modeZones,
                  t.settings.modeZonesHint,
                ],
              ] as const
            ).map(([mode, title, hint]) => (
              <button
                key={mode}
                type="button"
                onClick={() => patch({ delivery: { ...rest.delivery, mode } })}
                className={`rounded-2xl border px-4 py-3 text-left transition-colors ${
                  deliveryMode === mode
                    ? "border-brand bg-brand-tint/40"
                    : "border-line-strong hover:border-brand/60"
                }`}
              >
                <span className="block text-sm font-semibold">{title}</span>
                <span className="mt-1 block text-xs text-ink-muted">{hint}</span>
              </button>
            ))}
          </div>

          {deliveryMode === "radius" ? (
            <div className="mt-5 grid gap-4 sm:grid-cols-3">
              <NumField
                label={t.settings.baseFee}
                value={rest.delivery.baseFee}
                onChange={(v) =>
                  patch({ delivery: { ...rest.delivery, baseFee: v } })
                }
              />
              <NumField
                label={t.settings.perKm}
                value={rest.delivery.perKm}
                onChange={(v) =>
                  patch({ delivery: { ...rest.delivery, perKm: v } })
                }
              />
              <NumField
                label={t.settings.maxKm}
                value={rest.delivery.maxKm}
                onChange={(v) =>
                  patch({ delivery: { ...rest.delivery, maxKm: v } })
                }
              />
              <p className="text-xs text-ink-muted sm:col-span-3">
                {t.settings.radiusExample(
                  formatPrice(rest.delivery.baseFee),
                  formatPrice(rest.delivery.perKm),
                  formatPrice(
                    rest.delivery.baseFee + rest.delivery.perKm * 4,
                  ),
                )}
                {(rest.delivery.zones?.length ?? 0) > 0 &&
                  t.settings.zonesKept}
              </p>
            </div>
          ) : (
            <div className="mt-5">
              <DeliveryZonesEditor
                zones={rest.delivery.zones ?? []}
                onChange={(zones) =>
                  patch({ delivery: { ...rest.delivery, zones } })
                }
                center={center}
              />
            </div>
          )}
        </div>
      </Section>

      {/* Online payment: Payme, Click, Uzum. Owner-only, like everything else
          that belongs to the company rather than to a branch — a merchant key
          is not a branch manager's to hold. */}
      {scope.isOwner && (
        <Section title={t.payments.title}>
          <PaymentsEditor />
        </Section>
      )}

      {/* The SMS gateway login codes go out through. Owner-only and
          company-level: one contract, one bill, and the sender name is the
          restaurant's own. Placed right after payment because both are
          "credentials somebody emailed the owner once". */}
      {scope.isOwner && (
        <Section title={t.sms.title}>
          <SmsEditor />
        </Section>
      )}

      {/* The restaurant's own Telegram bot: the mini app runs the same site, and
          inside Telegram a guest is signed in without an SMS at all. Next to the
          SMS gateway because they are the same kind of thing — a credential
          somebody was emailed once — and because one replaces the other's cost. */}
      {scope.isOwner && (
        <Section title={t.telegram.title}>
          <TelegramEditor />
        </Section>
      )}

      {/* The phone system. Company-level and owner-only, like the payment
          keys — the restaurant has one number that rings. */}
      {scope.isOwner && (
        <Section title={t.pbx.title}>
          <PBXEditor />
        </Section>
      )}

      {/* The till the restaurant already runs. Per branch, so it sits under
          the branch lens rather than with the company profile. */}
      <Section title={t.pos.navTitle}>
        <POSEditor />
      </Section>

      {/* Outside delivery services */}
      <Section title={t.settings.providersTitle}>
        <ProvidersEditor />
      </Section>

      {/* Brands and branches: the shape of the business */}
      {scope.isOwner && (
        <Section title={t.scope.brandsTitle}>
          <BranchesEditor />
        </Section>
      )}

      {/* Table booking: the room is drawn here, guests book on it */}
      <Section title={t.booking.settingsTitle} blockedBy={branchGate}>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={booking.enabled}
            onChange={(e) =>
              patch({ booking: { ...booking, enabled: e.target.checked } })
            }
          />
          <span className="font-medium">{t.booking.enabled}</span>
        </label>
        <p className="mt-1 text-xs text-ink-muted">{t.booking.enabledHint}</p>

        {/* ⚠️ Worded as "guests choose their table" rather than as "hide the
            plan", and ticked by default, so the sentence the owner reads
            describes the site they already have. A "hide" switch would make the
            common case the one that has to be turned off. The stored field is
            the inverse for the same reason in reverse: its zero value has to be
            today's behaviour for every restaurant that never opens this page. */}
        <label className="mt-3 flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={!booking.hidePlan}
            onChange={(e) =>
              patch({ booking: { ...booking, hidePlan: !e.target.checked } })
            }
          />
          <span className="font-medium">{t.booking.showPlan}</span>
        </label>
        <p className="mt-1 text-xs text-ink-muted">{t.booking.showPlanHint}</p>

        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <label className="block text-sm">
            <span className="font-medium">{t.booking.slotMinutes}</span>
            <input
              type="number"
              min={15}
              step={15}
              className="input mt-1"
              value={booking.slotMinutes}
              onChange={(e) =>
                patch({
                  booking: {
                    ...booking,
                    slotMinutes: Number(e.target.value) || 90,
                  },
                })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium">{t.booking.maxDaysAhead}</span>
            <input
              type="number"
              min={1}
              className="input mt-1"
              value={booking.maxDaysAhead}
              onChange={(e) =>
                patch({
                  booking: {
                    ...booking,
                    maxDaysAhead: Number(e.target.value) || 30,
                  },
                })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium">{t.booking.minNotice}</span>
            <input
              type="number"
              min={0}
              className="input mt-1"
              value={booking.minNoticeMinutes}
              onChange={(e) =>
                patch({
                  booking: {
                    ...booking,
                    minNoticeMinutes: Number(e.target.value) || 0,
                  },
                })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="font-medium">{t.booking.maxGuests}</span>
            <input
              type="number"
              min={1}
              className="input mt-1"
              value={booking.maxGuests}
              onChange={(e) =>
                patch({
                  booking: {
                    ...booking,
                    maxGuests: Number(e.target.value) || 20,
                  },
                })
              }
            />
          </label>
        </div>
        <p className="mt-2 text-xs text-ink-muted">{t.booking.slotHint}</p>

        <h3 className="mt-6 text-sm font-semibold">{t.booking.planTitle}</h3>
        <div className="mt-2">
          <FloorPlanEditor
            value={booking}
            onChange={(next) => patch({ booking: next })}
          />
        </div>
      </Section>

      {/* Editable site copy */}
      {scope.isOwner && (
      <Section title={t.settings.contentTitle}>
        <p className="text-sm text-ink-muted">{t.settings.contentHint}</p>
        <div className="mt-4 space-y-5">
          <LocalizedField
            label={t.settings.tagline}
            value={content.tagline}
            onChange={(v) => patchContent({ tagline: v })}
          />
          <LocalizedField
            label={t.settings.aboutTitle}
            value={content.aboutTitle}
            onChange={(v) => patchContent({ aboutTitle: v })}
          />
          <LocalizedField
            label={t.settings.aboutText}
            value={content.aboutText}
            onChange={(v) => patchContent({ aboutText: v })}
            multiline
          />
          <LocalizedField
            label={t.settings.footerNote}
            value={content.footerNote}
            onChange={(v) => patchContent({ footerNote: v })}
            multiline
          />
          <PerksEditor
            cards={content.perks ?? []}
            hidden={content.hidePerks ?? false}
            onChange={(next) => patchContent(next)}
          />
        </div>
      </Section>
      )}

      {/* Cashback. Company-wide, like the customers it belongs to. */}
      {scope.isOwner && (
        <Section title={t.settings.loyaltyTitle}>
          <p className="text-sm text-ink-muted">{t.settings.loyaltyHint}</p>
          <label className="mt-4 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={loyalty.enabled}
              onChange={(e) =>
                patch({ loyalty: { ...loyalty, enabled: e.target.checked } })
              }
            />
            <span className="font-medium">{t.settings.loyaltyEnabled}</span>
          </label>
          {loyalty.enabled && (
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <label className="block text-sm">
                <span className="font-medium">{t.settings.loyaltyEarn}</span>
                <input
                  type="number"
                  min={0}
                  max={100}
                  className={inputCls}
                  value={loyalty.earnPercent}
                  onChange={(e) =>
                    patch({
                      loyalty: {
                        ...loyalty,
                        earnPercent: Number(e.target.value) || 0,
                      },
                    })
                  }
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.settings.loyaltyRedeem}</span>
                <input
                  type="number"
                  min={1}
                  max={100}
                  className={inputCls}
                  value={loyalty.maxRedeemPercent}
                  onChange={(e) =>
                    patch({
                      loyalty: {
                        ...loyalty,
                        maxRedeemPercent: Number(e.target.value) || 0,
                      },
                    })
                  }
                />
                <span className="mt-1 block text-xs text-ink-muted">
                  {t.settings.loyaltyRedeemHint}
                </span>
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.settings.loyaltyMinOrder}</span>
                <input
                  type="number"
                  min={0}
                  className={inputCls}
                  value={loyalty.minOrderToEarn}
                  onChange={(e) =>
                    patch({
                      loyalty: {
                        ...loyalty,
                        minOrderToEarn: Number(e.target.value) || 0,
                      },
                    })
                  }
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.settings.loyaltyWelcome}</span>
                <input
                  type="number"
                  min={0}
                  className={inputCls}
                  value={loyalty.welcomePoints}
                  onChange={(e) =>
                    patch({
                      loyalty: {
                        ...loyalty,
                        welcomePoints: Number(e.target.value) || 0,
                      },
                    })
                  }
                />
                <span className="mt-1 block text-xs text-ink-muted">
                  {t.settings.loyaltyWelcomeHint}
                </span>
              </label>
            </div>
          )}
        </Section>
      )}

      {/* Which map the site draws with, and the one thing that protects the key.

          ⚠️ **A field per provider, and only the chosen one is shown.** An owner
          who tries Yandex and goes back must not end up with one provider
          holding the other's key — that fails as a blank map with a console
          error nobody in a restaurant reads. The other keys stay stored, so
          switching back needs no retyping. */}
      {scope.isOwner && (
        <Section title={t.settings.mapTitle}>
          <p className="text-sm text-ink-muted">{t.settings.mapIntro}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {MAP_PROVIDERS.map((id) => {
              const on = normalizeProvider(rest.mapProvider) === id;
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => patch({ mapProvider: id })}
                  aria-pressed={on}
                  className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                    on
                      ? "border-brand bg-brand/10 text-brand"
                      : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
                  }`}
                >
                  {t.settings.mapProviderName[id]}
                </button>
              );
            })}
          </div>
          <p className="mt-2 text-xs text-ink-muted">
            {t.settings.mapProviderNote[normalizeProvider(rest.mapProvider)]}
          </p>

          {(() => {
            const provider = normalizeProvider(rest.mapProvider);
            const field =
              provider === "yandex"
                ? ("mapYandexKey" as const)
                : provider === "google"
                  ? ("mapGoogleKey" as const)
                  : ("mapApiKey" as const);
            return (
              <label className="mt-4 block text-sm font-medium">
                {t.settings.mapKeyLabel(t.settings.mapProviderName[provider])}
                <input
                  className="input mt-1"
                  value={(rest[field] as string | undefined) ?? ""}
                  onChange={(e) => patch({ [field]: e.target.value.trim() })}
                  placeholder={t.settings.mapKeyPlaceholder[provider]}
                />
                <span className="mt-1 block text-xs text-ink-muted">
                  {t.settings.mapKeyWhere[provider]}
                </span>
              </label>
            );
          })()}

          {/* Said in the settings page rather than in a document nobody opens:
              an unrestricted key is genuinely unprotected, and the owner is
              the only person who can restrict it. */}
          <p className="mt-3 rounded-xl border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-ink-soft">
            {t.settings.mapKeyWarn}
          </p>
        </Section>
      )}

      {/* Bringing your own domain. */}
      {scope.isOwner && <DomainGuide />}

      {/* Proving the site is yours, to the two search engines that matter here.
          Sits directly under the domain guide because the order is real: a
          token issued for the free subdomain does not verify the owner's own
          domain, so this is the step *after* the domain is connected. */}
      {scope.isOwner && (
        <Section title={t.settings.seoTitle}>
          <p className="text-sm text-ink-soft">{t.settings.seoIntro}</p>
          <label className="mt-4 block text-sm font-medium">
            {t.settings.seoGoogleLabel}
            <input
              className="input mt-1"
              value={rest.seo?.google ?? ""}
              onChange={(e) =>
                patch({ seo: { ...(rest.seo ?? {}), google: e.target.value } })
              }
              placeholder="google-site-verification=..."
            />
          </label>
          <label className="mt-3 block text-sm font-medium">
            {t.settings.seoYandexLabel}
            <input
              className="input mt-1"
              value={rest.seo?.yandex ?? ""}
              onChange={(e) =>
                patch({ seo: { ...(rest.seo ?? {}), yandex: e.target.value } })
              }
              placeholder="a1b2c3d4e5f60000"
            />
          </label>
          {/* Not trimmed on the way in: both consoles show the owner a whole
              meta tag, and a field that silently mangles what was pasted is
              worse than one that accepts it. The tag is unwrapped at render. */}
          <p className="mt-1 text-xs text-ink-muted">{t.settings.seoHint}</p>
          <p className="mt-3 rounded-xl border border-line px-3 py-2 text-xs text-ink-soft">
            {t.settings.seoWarn}
          </p>
        </Section>
      )}

      {/* Taking the business away.
          Renders nothing at all unless the platform has opened a window — see
          components/admin/DataExport.tsx. Placed last: it is rare, it is the
          most dangerous file this panel can produce, and it should never be
          what a hand lands on while scrolling. */}
      {scope.isOwner && <DataExport />}

      {/* Look and feel.
          ⚠️ Locked — and **shown** locked — once Keel has drawn a layout for this
          restaurant. Hiding the block would read as "it disappeared" and produce
          the same phone call, only harder to answer; the accent colour and the
          radius are exactly what would break a paid design. Same shape as
          `hideWatermark`: what the business model rests on does not sit behind
          the customer's own switch. */}
      {/* The strip the restaurant runs itself. Above the design section because it is the
          one an owner actually opens: a promotion changes weekly, the layout does not. */}
      <Section title={t.banners.title}>
        <BannersEditor />
      </Section>

      {scope.isOwner && (
        <Section title={t.settings.designTitle}>
          {designLocked ? (
            <p className="rounded-xl border border-line bg-surface px-3 py-2 text-sm text-ink-soft">
              {t.settings.designLocked}
            </p>
          ) : (
            <DesignEditor
              theme={theme}
              onChange={(next) => patch({ theme: next })}
            />
          )}
        </Section>
      )}
    </div>
  );
}

/** How an owner brings their own domain, with the one check that matters.
 *
 *  The DNS step is where this goes wrong, and it goes wrong silently: the
 *  record is saved at the registrar, nothing visibly happens, and the owner
 *  cannot tell "not propagated yet" from "typed it wrong".
 *
 *  The last step used to be "tell us and we will connect it" — an owner
 *  waiting overnight for a change they had already made. Now the DNS record
 *  *is* the authorisation: pointing a domain at this server is something only
 *  the registrar account holder can do, which is exactly the claim being made,
 *  so "Connect" does the whole thing. */
function DomainGuide() {
  const t = useAdminT();
  const [domain, setDomain] = useState("");
  const [busy, setBusy] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [note, setNote] = useState("");
  const [connected, setConnected] = useState<string[] | null>(null);
  const [res, setRes] = useState<{
    found: string[];
    expected: string[];
    ok: boolean;
  } | null>(null);

  async function check() {
    if (!domain.trim()) return;
    setBusy(true);
    setRes(null);
    setNote("");
    try {
      setRes(await api.adminDomainCheck(domain));
    } catch {
      setRes({ found: [], expected: [], ok: false });
    } finally {
      setBusy(false);
    }
  }

  // The domain is passed in rather than read from state: disconnecting acts on
  // a row in the list, and setState is not applied by the time the handler
  // runs — it would have sent whatever was last typed into the input.
  async function connect(remove: boolean, which?: string) {
    const target = (which ?? domain).trim();
    if (!target) return;
    if (remove && !confirm(t.settings.domainDisconnectConfirm(target))) return;
    setConnecting(true);
    setNote("");
    try {
      const out = await api.adminDomainConnect(target, remove);
      if (out.domains) setConnected(out.domains);
      if (out.ok) {
        setNote(
          remove
            ? t.settings.domainDisconnected(target)
            : t.settings.domainConnected(target),
        );
      } else {
        // Shown as a sentence, never a status code: the reader is a restaurant
        // owner deciding what to change at their registrar.
        setNote(out.unsupported ? t.settings.domainStandalone : (out.error ?? ""));
        if (out.found) {
          setRes({ found: out.found, expected: out.expected ?? [], ok: false });
        }
      }
    } catch (e) {
      setNote(e instanceof Error ? e.message : String(e));
    } finally {
      setConnecting(false);
    }
  }

  return (
    <Section title={t.settings.domainTitle}>
      <p className="text-sm text-ink-soft">{t.settings.domainIntro}</p>

      <ol className="mt-3 space-y-3 text-sm text-ink-soft">
        <li>
          <span className="font-medium text-ink">1.</span> {t.settings.domainStep1}
          <pre className="mt-2 overflow-x-auto rounded-xl border border-line bg-raised px-3 py-2 text-xs">
{`A     @      ${res?.expected?.[0] ?? "…"}
A     www    ${res?.expected?.[0] ?? "…"}`}
          </pre>
          {!res && (
            <span className="text-xs text-ink-muted">
              ({t.settings.domainCheck.toLowerCase()})
            </span>
          )}
        </li>
        <li>
          <span className="font-medium text-ink">2.</span> {t.settings.domainStep2}
        </li>
        <li>
          <span className="font-medium text-ink">3.</span> {t.settings.domainStep3}
        </li>
      </ol>

      <div className="mt-4 flex flex-wrap items-end gap-3">
        <label className="flex-1 text-sm font-medium">
          {t.settings.domainField}
          <input
            className="input mt-1"
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
            placeholder="osh.uz"
          />
        </label>
        <button type="button" className="btn" onClick={check} disabled={busy}>
          {busy ? t.settings.domainChecking : t.settings.domainCheck}
        </button>
        {/* Offered only once DNS actually resolves here. A "Connect" button
            that is always live invites an owner to press it, be refused, and
            conclude the feature is broken rather than that their record has
            not propagated. */}
        {res?.ok && (
          <button
            type="button"
            className="btn-primary"
            onClick={() => connect(false)}
            disabled={connecting}
          >
            {connecting ? t.settings.domainConnecting : t.settings.domainConnect}
          </button>
        )}
      </div>

      {res && (
        <p
          className={`mt-3 rounded-xl px-3 py-2 text-sm ${
            res.ok
              ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
              : "bg-amber-500/10 text-ink-soft border border-amber-500/40"
          }`}
        >
          {res.ok
            ? t.settings.domainOk
            : res.found.length === 0
              ? t.settings.domainNone
              : `${t.settings.domainBad} ${res.found.join(", ")}`}
        </p>
      )}

      {note && (
        <p className="mt-3 rounded-xl bg-ink/5 px-3 py-2 text-sm text-ink-soft">
          {note}
        </p>
      )}

      {connected && connected.length > 0 && (
        <div className="mt-4">
          <p className="text-sm font-medium">{t.settings.domainConnected0}</p>
          <ul className="mt-2 space-y-1 text-sm">
            {connected.map((d, i) => (
              <li key={d} className="flex items-center gap-2">
                <code className="rounded-lg bg-raised px-2 py-1 text-xs">{d}</code>
                {i === 0 ? (
                  <span className="text-xs text-ink-muted">
                    {t.settings.domainPrimaryNote}
                  </span>
                ) : (
                  <button
                    type="button"
                    className="chip"
                    disabled={connecting}
                    onClick={() => connect(true, d)}
                  >
                    {t.settings.domainDisconnect}
                  </button>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Said plainly, because "why can I not just switch it on myself" is the
          next question and the answer is not laziness. */}
      <p className="mt-3 text-xs text-ink-muted">{t.settings.domainWhyManual}</p>
    </Section>
  );
}

function Section({
  title,
  children,
  /** Shown instead of the fields when the section has nowhere to save to. */
  blockedBy,
}: {
  title: string;
  children: React.ReactNode;
  blockedBy?: React.ReactNode;
}) {
  return (
    <section className="mt-6 rounded-3xl border border-line bg-surface shadow-card p-6">
      <h2 className="mb-4 text-lg font-bold">{title}</h2>
      {/* Fields the owner can type into but that would be silently discarded
          are worse than no fields: say why instead of pretending. */}
      {blockedBy ?? children}
    </section>
  );
}

function NumField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="block text-sm">
      <span className="font-medium">{label}</span>
      <input
        type="number"
        className="mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand"
        value={value}
        onChange={(e) => onChange(Number(e.target.value) || 0)}
      />
    </label>
  );
}


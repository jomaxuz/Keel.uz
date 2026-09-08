"use client";

// Brands and branches, managed by the owner without calling anyone.
//
// The template is sold per company, so adding the ninth samsa point has to be a
// button here — not a deployment. A brand is a menu with a face; a branch is a
// place that cooks it, with its own address, hours, delivery area and couriers.
//
// Deleting is deliberately careful: the last one of either cannot go, a brand
// with branches cannot go, and a branch that has ever taken an order is closed
// rather than deleted so its receipts stay answerable.

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import AddressPicker from "@/components/map/AddressPicker";
import KioskSettings from "@/components/admin/KioskSettings";
import TillDeviceSettings from "@/components/admin/TillDeviceSettings";
import type { Branch, Brand } from "@/lib/types";

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

export default function BranchesEditor() {
  const t = useAdminT();
  const { ask } = useAsk();
  const { brands, branches, reload } = useAdminScope();
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);
  const [draft, setDraft] = useState<Record<string, Partial<Branch>>>({});

  async function addBrand() {
    setError(null);
    try {
      await api.createBrand({
        name: t.scope.addBrand.replace("+ ", ""),
        isActive: true,
        features: {
          delivery: true,
          pickup: true,
          dineIn: false,
          booking: false,
        },
      });
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function saveBrand(brand: Brand, patch: Partial<Brand>) {
    setError(null);
    try {
      await api.updateBrand(brand.id, { ...brand, ...patch });
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function removeBrand(brand: Brand) {
    if (
      !(await ask({
        title: t.scope.confirmDeleteBrand(brand.name),
        danger: true,
      }))
    )
      return;
    setError(null);
    try {
      await api.deleteBrand(brand.id);
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    }
  }

  async function addBranch(brandId: string) {
    setError(null);
    try {
      const created = await api.createBranch({
        brandId,
        name: t.scope.branchName,
        isActive: true,
        prepMinutes: 30,
        // Wider than the door: phone GPS is accurate to 10–30 m at best.
        staffRadiusM: 50,
        phones: [],
        workingHours: [],
      });
      reload();
      setOpenId(created.id);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function saveBranch(branch: Branch) {
    setError(null);
    setNotice(null);
    try {
      await api.updateBranch(branch.id, { ...branch, ...draft[branch.id] });
      setDraft((d) => ({ ...d, [branch.id]: {} }));
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    }
  }

  async function removeBranch(branch: Branch) {
    if (
      !(await ask({
        title: t.scope.confirmDeleteBranch(branch.name),
        danger: true,
      }))
    )
      return;
    setError(null);
    try {
      const res = await api.deleteBranch(branch.id);
      setNotice(
        res.deactivated ? t.scope.branchDeactivated : t.scope.branchDeleted,
      );
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    }
  }

  const patch = (id: string, p: Partial<Branch>) =>
    setDraft((d) => ({ ...d, [id]: { ...d[id], ...p } }));

  return (
    <div>
      <p className="text-sm text-ink-muted">{t.scope.brandsHint}</p>
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      {notice && (
        <p className="mt-3 rounded-2xl bg-amber-50 px-4 py-2.5 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
          {notice}
        </p>
      )}

      <div className="mt-4 space-y-5">
        {brands.map((brand) => {
          const own = branches.filter((b) => b.brandId === brand.id);
          return (
            <section
              key={brand.id}
              className="rounded-2xl border border-line p-4"
            >
              <div className="flex flex-wrap items-end gap-3">
                <label className="block text-sm">
                  <span className="font-medium">{t.scope.brandName}</span>
                  <input
                    className={inputCls}
                    defaultValue={brand.name}
                    onBlur={(e) =>
                      e.target.value !== brand.name &&
                      saveBrand(brand, { name: e.target.value })
                    }
                  />
                </label>
                <label className="flex items-center gap-2 pb-2 text-sm">
                  <input
                    type="checkbox"
                    checked={brand.isActive}
                    onChange={(e) =>
                      saveBrand(brand, { isActive: e.target.checked })
                    }
                  />
                  <span>{t.scope.brandActive}</span>
                </label>
                <span className="pb-2 text-xs text-ink-muted">
                  {t.scope.branchCount(own.length)}
                </span>
                <button
                  type="button"
                  onClick={() => removeBrand(brand)}
                  className="ml-auto pb-2 text-xs text-ink-muted hover:text-red-600"
                >
                  {t.common.delete}
                </button>
              </div>

              {/* Which services this brand offers at all — a samsa point has no
                  tables to book. */}
              <div className="mt-3 flex flex-wrap gap-4 text-sm">
                {(
                  [
                    ["delivery", t.scope.featureDelivery],
                    ["pickup", t.scope.featurePickup],
                    ["dineIn", t.scope.featureDineIn],
                    ["booking", t.scope.featureBooking],
                  ] as const
                ).map(([key, label]) => (
                  <label key={key} className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={brand.features?.[key] ?? false}
                      onChange={(e) =>
                        saveBrand(brand, {
                          features: {
                            ...brand.features,
                            [key]: e.target.checked,
                          },
                        })
                      }
                    />
                    <span>{label}</span>
                  </label>
                ))}
              </div>

              <h4 className="mt-5 text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {t.scope.branchesTitle}
              </h4>
              <p className="mt-1 text-xs text-ink-muted">
                {t.scope.branchesHint}
              </p>

              <div className="mt-3 space-y-2">
                {own.map((branch) => {
                  const open = openId === branch.id;
                  const d = { ...branch, ...draft[branch.id] } as Branch;
                  return (
                    <div
                      key={branch.id}
                      className="rounded-xl border border-line bg-ink/[0.02]"
                    >
                      <button
                        type="button"
                        onClick={() => setOpenId(open ? null : branch.id)}
                        className="flex w-full flex-wrap items-center gap-3 p-3 text-left"
                      >
                        <span className="font-medium">{branch.name}</span>
                        {!branch.isActive && (
                          <span className="badge bg-ink/10 text-ink-muted">
                            {t.scope.inactive}
                          </span>
                        )}
                        <span className="truncate text-xs text-ink-muted">
                          {branch.address?.text}
                        </span>
                        <span className="ml-auto text-xs text-ink-muted">
                          {open ? "▲" : "▼"}
                        </span>
                      </button>

                      {open && (
                        <div className="border-t border-line p-3">
                          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.scope.branchName}
                              </span>
                              <input
                                className={inputCls}
                                value={d.name}
                                onChange={(e) =>
                                  patch(branch.id, { name: e.target.value })
                                }
                              />
                            </label>
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.scope.branchPhones}
                              </span>
                              <input
                                className={inputCls}
                                value={(d.phones ?? []).join(", ")}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    phones: e.target.value
                                      .split(",")
                                      .map((p) => p.trim())
                                      .filter(Boolean),
                                  })
                                }
                              />
                            </label>
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.scope.branchCode}
                              </span>
                              <input
                                className={inputCls}
                                value={d.code ?? ""}
                                maxLength={6}
                                placeholder={t.scope.branchCodePh}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    // Uppercase letters and digits only: it is
                                    // read aloud over the phone.
                                    code: e.target.value
                                      .toUpperCase()
                                      .replace(/[^A-Z0-9]/g, ""),
                                  })
                                }
                              />
                              <span className="mt-1 block text-xs text-ink-muted">
                                {t.scope.branchCodeHint}
                              </span>
                            </label>
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.scope.branchPrep}
                              </span>
                              <input
                                type="number"
                                min={0}
                                className={inputCls}
                                value={d.prepMinutes ?? 30}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    prepMinutes: Number(e.target.value) || 0,
                                  })
                                }
                              />
                            </label>
                            {/* Attendance geofence: how close staff must be to
                                this address before the clock-in button works.
                                Lives on the branch because the answer is about
                                this building, not about the company. */}
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.staff.radiusTitle}
                              </span>
                              <input
                                type="number"
                                min={0}
                                className={inputCls}
                                value={d.staffRadiusM ?? 50}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    staffRadiusM: Math.max(
                                      0,
                                      Number(e.target.value) || 0,
                                    ),
                                  })
                                }
                              />
                              <span className="mt-1 block text-xs text-ink-muted">
                                {t.staff.radiusHint}
                              </span>
                              {(d.staffRadiusM ?? 50) > 0 &&
                                (d.staffRadiusM ?? 50) < 20 && (
                                  <span className="mt-1 block rounded-xl bg-amber-50 px-2 py-1.5 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
                                    {t.staff.radiusWarning}
                                  </span>
                                )}
                            </label>
                            {/* ⚠️ Service charge, on the branch rather than the
                                company: a chain's restaurant with waiters
                                charges for it and its counter outlet in a
                                shopping centre does not, and one number for
                                both would put a service charge on a takeaway
                                coffee. It is only ever added to a **table's**
                                bill, which the till decides — the setting
                                cannot know the difference. */}
                            <label className="block text-sm">
                              <span className="font-medium">
                                {t.staff.serviceTitle}
                              </span>
                              <input
                                type="number"
                                min={0}
                                max={100}
                                className={inputCls}
                                value={d.service?.percent ?? 0}
                                onChange={(e) => {
                                  const percent = Math.min(
                                    100,
                                    Math.max(0, Number(e.target.value) || 0),
                                  );
                                  // ⚠️ Zero is off, and off is zero: two ways
                                  // to say "no service charge" is a setting
                                  // that looks on and charges nothing, which
                                  // is a support call nobody can diagnose from
                                  // the screen.
                                  patch(branch.id, {
                                    service: { enabled: percent > 0, percent },
                                  });
                                }}
                              />
                              <span className="mt-1 block text-xs text-ink-muted">
                                {t.staff.serviceHint}
                              </span>
                            </label>
                            <label className="flex items-center gap-2 pt-6 text-sm">
                              <input
                                type="checkbox"
                                checked={d.isActive}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    isActive: e.target.checked,
                                  })
                                }
                              />
                              <span>{t.scope.branchActive}</span>
                            </label>
                          </div>

                          {/* ⚠️ **Beside the attendance settings, because it
                              is one**: it decides whether a PIN opens the till
                              for somebody who has not clocked in. Off by
                              default — a restaurant that does not run
                              attendance would meet it as every PIN being
                              refused, with a queue at the counter and nothing
                              on that screen the cashier can act on. */}
                          <label className="mt-4 flex items-start gap-2 text-sm">
                            <input
                              type="checkbox"
                              className="mt-0.5"
                              checked={d.requireShift ?? false}
                              onChange={(e) =>
                                patch(branch.id, {
                                  requireShift: e.target.checked,
                                })
                              }
                            />
                            <span>
                              {t.staff.requireShift}
                              <span className="mt-0.5 block text-xs text-ink-muted">
                                {t.staff.requireShiftHint}
                              </span>
                            </span>
                          </label>

                          {/* ---- Where this branch's store requests are
                              answered ----
                              ⚠️ **Only when there is somewhere else to point
                              at.** A single-branch restaurant has exactly one
                              answer, and a select with one option is a question
                              that teaches people the screen asks things it
                              already knows. */}
                          {branches.length > 1 && (
                            <label className="mt-4 block text-sm">
                              <span className="font-medium">
                                {t.staff.supplyBranch}
                              </span>
                              <select
                                className={inputCls}
                                value={d.supplyBranchId ?? ""}
                                onChange={(e) =>
                                  patch(branch.id, {
                                    supplyBranchId: e.target.value,
                                  })
                                }
                              >
                                <option value="">
                                  {t.staff.supplyBranchOwn}
                                </option>
                                {/* ⚠️ Itself is not on the list. A branch
                                    supplying itself would build a slip whose
                                    two ends are the same shelf — the same kilo
                                    subtracted and added, and nobody able to
                                    sign for it. */}
                                {branches
                                  .filter((b) => b.id !== branch.id)
                                  .map((b) => (
                                    <option key={b.id} value={b.id}>
                                      {b.name}
                                    </option>
                                  ))}
                              </select>
                              <span className="mt-1 block text-xs text-ink-muted">
                                {t.staff.supplyBranchHint}
                              </span>
                            </label>
                          )}

                          <KioskSettings
                            branch={branch}
                            requireCode={d.requireKioskCode ?? false}
                            onToggle={(v) =>
                              patch(branch.id, { requireKioskCode: v })
                            }
                          />

                          {/* Beside the kiosk link because it is the same job:
                              turning a screen in this branch into one of ours,
                              once, without anybody having to sign in on it. */}
                          <TillDeviceSettings branch={branch} />

                          {/* The point a courier drives from and a guest walks
                              to; also the centre of this branch's zones. */}
                          <div className="mt-4">
                            <AddressPicker
                              value={{
                                text: d.address?.text ?? "",
                                lat: d.address?.lat ?? 0,
                                lng: d.address?.lng ?? 0,
                              }}
                              onChange={(next) =>
                                patch(branch.id, {
                                  address: {
                                    text: next.text,
                                    lat: next.lat,
                                    lng: next.lng,
                                  },
                                })
                              }
                              center={{
                                lat: d.address?.lat || 41.311081,
                                lng: d.address?.lng || 69.279737,
                              }}
                              mapClassName="h-56 w-full"
                            />
                          </div>

                          <div className="mt-4 flex flex-wrap gap-2">
                            <button
                              type="button"
                              onClick={() => saveBranch(branch)}
                              className="btn-primary px-4 py-2 text-sm"
                            >
                              {t.common.save}
                            </button>
                            <button
                              type="button"
                              onClick={() => removeBranch(branch)}
                              className="ml-auto px-3 py-2 text-xs text-ink-muted hover:text-red-600"
                            >
                              {t.common.delete}
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}

                <button
                  type="button"
                  onClick={() => addBranch(brand.id)}
                  className="btn-ghost px-3 py-1.5 text-sm"
                >
                  {t.scope.addBranch}
                </button>
              </div>
            </section>
          );
        })}
      </div>

      <button
        type="button"
        onClick={addBrand}
        className="btn-ghost mt-4 px-4 py-2 text-sm"
      >
        {t.scope.addBrand}
      </button>
    </div>
  );
}

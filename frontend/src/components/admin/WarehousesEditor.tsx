"use client";

// The stores stock is kept in — the bar, the kitchen, the cellar.
//
// ⚠️ **A restaurant does not have one store.** Until this existed every count
// was one list covering all of them, so a shortfall behind the bar cancelled
// against a surplus in the kitchen: arithmetically fine, and the one number an
// owner cannot act on. They are counted by different people on different
// evenings, and the count has to be able to say which room somebody walked into.
//
// ⚠️ **A restaurant with one store never has to come here.** No warehouses
// means every ingredient is in the undivided store and every screen behaves as
// it did before — the same rule as zones, which are not drawn for a single
// room, and as the brand switcher, which is not drawn for a single brand.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { Warehouse } from "@/lib/types";

export default function WarehousesEditor({
  onChange,
}: {
  /** The ingredient form's picker reads the same list, so it has to be told. */
  onChange?: () => void;
}) {
  const t = useAdminT();
  const [rows, setRows] = useState<Warehouse[]>([]);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api
      .adminWarehouses()
      .then((d) => setRows(d.warehouses))
      .catch(() => setRows([]));
  }, []);
  useEffect(load, [load]);

  async function add() {
    const n = name.trim();
    if (!n) return;
    setBusy(true);
    setError("");
    try {
      await api.adminSaveWarehouse({ name: n, sort: rows.length });
      setName("");
      load();
      onChange?.();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function rename(row: Warehouse, next: string) {
    if (!next.trim() || next === row.name) return;
    try {
      await api.adminSaveWarehouse({ ...row, name: next.trim() });
      load();
      onChange?.();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    }
  }

  async function remove(row: Warehouse) {
    setError("");
    try {
      await api.adminDeleteWarehouse(row.id);
      load();
      onChange?.();
    } catch (e) {
      // ⚠️ The refusal names how many ingredients are still filed there. A
      // store deleted out from under them would leave them pointing at nothing
      // — which is not the same as the undivided store, and every count of the
      // undivided store would silently start including the bar's vodka.
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    }
  }

  return (
    <div className="card space-y-3 p-3">
      <div>
        <p className="text-sm font-semibold">{t.warehouses.title}</p>
        <p className="mt-1 text-xs leading-relaxed text-ink-muted">
          {t.warehouses.hint}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {rows.length > 0 && (
        <ul className="space-y-2">
          {rows.map((wh) => (
            <li key={wh.id} className="flex flex-wrap items-center gap-2">
              <input
                className="input h-9 w-56"
                defaultValue={wh.name}
                onBlur={(e) => void rename(wh, e.target.value)}
              />
              <button
                type="button"
                onClick={() => void remove(wh)}
                className="rounded-lg border border-line px-2 py-1 text-xs text-danger"
              >
                {t.common.delete}
              </button>
            </li>
          ))}
        </ul>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <input
          className="input h-9 w-56"
          placeholder={t.warehouses.namePlaceholder}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button
          type="button"
          className="btn"
          disabled={busy || !name.trim()}
          onClick={() => void add()}
        >
          + {t.warehouses.add}
        </button>
      </div>
    </div>
  );
}

/** The picker used wherever something has to name a store.
 *
 *  ⚠️ **Not drawn at all when there are no warehouses**, which is the ordinary
 *  restaurant: a select with one option called "the store" is a control that
 *  teaches people this screen has settings they have to think about. */
export function WarehousePicker({
  warehouses,
  value,
  onChange,
  label,
  className = "w-44",
}: {
  warehouses: Warehouse[];
  value: string;
  onChange: (id: string) => void;
  label?: string;
  className?: string;
}) {
  const t = useAdminT();
  if (warehouses.length === 0) return null;
  return (
    <label className="block text-sm">
      <span className="text-xs text-ink-muted">
        {label ?? t.warehouses.one}
      </span>
      <select
        className={`input mt-1 ${className}`}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {/* ⚠️ The unfiled store keeps a name rather than an empty option. Every
            ingredient that existed before warehouses lives here, and a blank
            row reads as "not chosen yet" — which would have somebody filing
            forty things that were never wrong. */}
        <option value="">{t.warehouses.unfiled}</option>
        {warehouses.map((wh) => (
          <option key={wh.id} value={wh.id}>
            {wh.name}
          </option>
        ))}
      </select>
    </label>
  );
}

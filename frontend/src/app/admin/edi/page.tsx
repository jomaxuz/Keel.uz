"use client";

// The post a company already receives, and what it becomes.
//
// ⚠️ **The screen exists so that a delivery is not typed twice.** A supplier's
// electronic invoice is in Didox on the morning the crates arrive: twenty
// lines, quantities, prices, VAT. Until now a storekeeper read that in one tab
// and typed it into ours in another — which is the step where the shelf and the
// invoice quietly stop agreeing.
//
// ⚠️ **Nothing on this page signs, accepts or rejects.** A signature is made
// with an E-IMZO key on the accountant's own computer; we hold no key and must
// not. A button here that flipped a status would leave the document unsigned at
// the operator and absent from the tax return, while this screen showed it
// done. So the page says so in as many words and links to didox.uz.
//
// ⚠️ **The line matching is confirmed, never assumed.** The server proposes an
// ingredient per row by exact name; a person confirms. A false match here is a
// delivery of beef landing on the shelf of butter, silently, in the one part of
// the product where a wrong number stays invisible until a stocktake.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import type { EdiDocument, EdiSettings, Ingredient } from "@/lib/types";

/** What the operator calls each kind of document. ⚠️ Their codes, not ours —
 *  see models/accounting.go for why they are never renamed. */
const DOC_TYPES: Record<string, string> = {
  "001": "Schyot-faktura",
  "002": "Schyot-faktura",
  "008": "Schyot-faktura (FARM)",
  "021": "Schyot-faktura (qaytarish)",
  "023": "Gibrid schyot-faktura",
  "041": "TTN",
  "005": "Akt",
  "006": "Ishonchnoma",
  "062": "Ishonchnoma",
  "007": "Shartnoma",
  "000": "Hujjat",
};

export default function EdiPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [settings, setSettings] = useState<EdiSettings | null>(null);
  const [docs, setDocs] = useState<EdiDocument[]>([]);
  const [direction, setDirection] = useState<"in" | "out">("in");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [note, setNote] = useState("");
  const [openDoc, setOpenDoc] = useState<EdiDocument | null>(null);
  const [guess, setGuess] = useState<Record<string, string>>({});
  const [picked, setPicked] = useState<Record<number, string>>({});
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [showSettings, setShowSettings] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([api.adminEdiSettings(), api.adminEdiDocuments({ direction })])
      .then(([s, d]) => {
        setSettings(s);
        setDocs(d.documents);
        setError("");
        // The settings open by themselves while there is nothing to connect
        // to: an empty list under a switched-off integration reads as "broken".
        setShowSettings((was) => was || !s.enabled);
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [direction, t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function sync() {
    setBusy(true);
    setNote("");
    try {
      const r = await api.adminEdiSync(30);
      setNote(t.edi.synced(r.incoming, r.outgoing));
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function open(doc: EdiDocument) {
    setBusy(true);
    setError("");
    try {
      const [full, ings] = await Promise.all([
        api.adminEdiDocument(doc.id),
        // The stock list, for the dropdown beside every line.
        ingredients.length
          ? Promise.resolve({ ingredients })
          : api.adminIngredients(),
      ]);
      setIngredients(ings.ingredients);
      setOpenDoc(full.document);
      setGuess(full.guess ?? {});
      const initial: Record<number, string> = {};
      for (const line of full.document.lines ?? []) {
        const g = (full.guess ?? {})[String(line.no)];
        if (g) initial[line.no] = g;
      }
      setPicked(initial);
      if (full.linesError) setError(full.linesError);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function importDoc() {
    if (!openDoc) return;
    setBusy(true);
    try {
      const lines = (openDoc.lines ?? [])
        .filter((l) => picked[l.no])
        .map((l) => ({
          no: l.no,
          ingredientId: picked[l.no],
          qty: l.qty,
          price: l.price,
        }));
      const r = await api.adminEdiImport(openDoc.id, {
        lines,
        createSupplier: true,
      });
      setNote(t.edi.imported(r.lines, formatPrice(r.total)));
      setOpenDoc(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const matched = openDoc
    ? (openDoc.lines ?? []).filter((l) => picked[l.no]).length
    : 0;

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.edi.title}</h1>
      <p className="mt-2 max-w-3xl text-sm text-ink-muted">{t.edi.intro}</p>

      {/* ⚠️ **The sentence about signing sits above everything**, because it is
          the first question an accountant asks and the one a screen like this
          is most likely to answer dishonestly by omission. */}
      <p className="mt-3 max-w-3xl rounded-2xl border border-line bg-surface px-4 py-3 text-sm text-ink-muted">
        {t.edi.signHint}{" "}
        <a
          className="font-semibold text-brand underline"
          href="https://didox.uz"
          target="_blank"
          rel="noreferrer noopener"
        >
          didox.uz
        </a>
      </p>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setDirection("in")}
          className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
            direction === "in" ? "border-brand bg-brand/10" : "border-line"
          }`}
        >
          {t.edi.incoming}
        </button>
        <button
          type="button"
          onClick={() => setDirection("out")}
          className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
            direction === "out" ? "border-brand bg-brand/10" : "border-line"
          }`}
        >
          {t.edi.outgoing}
        </button>
        <button
          type="button"
          onClick={sync}
          disabled={busy || !settings?.enabled}
          className="rounded-lg bg-brand px-3 py-1.5 text-sm font-semibold text-white disabled:opacity-40"
        >
          {busy ? t.common.saving : t.edi.sync}
        </button>
        <button
          type="button"
          onClick={() => setShowSettings((v) => !v)}
          className="rounded-lg border border-line px-3 py-1.5 text-sm font-semibold"
        >
          {t.edi.settings}
        </button>
        {settings?.lastSyncAt && (
          <span className="text-xs text-ink-muted">
            {t.edi.lastSync(formatDate(settings.lastSyncAt))}
          </span>
        )}
      </div>

      {note && <p className="mt-4 text-sm text-emerald-600">{note}</p>}
      {error && <p className="mt-4 text-sm text-danger">{error}</p>}
      {/* The operator's own words, kept: "422 User not registered" sends an
          owner to their accountant, "ulanmadi" sends them to us. */}
      {settings?.lastError && (
        <p className="mt-2 text-sm text-amber-700">{settings.lastError}</p>
      )}

      {showSettings && (
        <EdiSettingsForm
          settings={settings}
          onSaved={(s) => {
            setSettings(s);
            setNote(t.common.save);
            load();
          }}
        />
      )}

      {loading && (
        <p className="mt-6 text-sm text-ink-muted">{t.common.loading}</p>
      )}

      {!loading && docs.length === 0 && (
        <p className="mt-6 text-sm text-ink-muted">{t.edi.empty}</p>
      )}

      <div className="mt-6 overflow-x-auto">
        {docs.length > 0 && (
          <table className="w-full min-w-[46rem] border-collapse text-sm">
            <thead>
              <tr className="text-left text-xs uppercase text-ink-muted/70">
                <th className="py-2 pr-3">{t.edi.docNo}</th>
                <th className="py-2 pr-3">{t.edi.docDate}</th>
                <th className="py-2 pr-3">{t.edi.partner}</th>
                <th className="py-2 pr-3 text-right">{t.edi.total}</th>
                <th className="py-2 pr-3">{t.edi.status}</th>
                <th className="py-2" />
              </tr>
            </thead>
            <tbody>
              {docs.map((d) => (
                <tr key={d.id} className="border-t border-line">
                  <td className="py-2 pr-3">
                    <div className="font-semibold">{d.number || "—"}</div>
                    <div className="text-xs text-ink-muted">
                      {DOC_TYPES[d.type] ?? d.type}
                    </div>
                  </td>
                  <td className="py-2 pr-3 tabular-nums">{d.date}</td>
                  <td className="py-2 pr-3">
                    <div>{d.partnerName || "—"}</div>
                    <div className="text-xs text-ink-muted tabular-nums">
                      {d.partnerTin}
                    </div>
                  </td>
                  <td className="py-2 pr-3 text-right tabular-nums">
                    {formatPrice(d.totalWithVat || d.total)}
                  </td>
                  <td className="py-2 pr-3">
                    <span className="text-xs">
                      {t.edi.statusName(d.status)}
                    </span>
                  </td>
                  <td className="py-2 text-right">
                    {d.purchaseId ? (
                      <span className="text-xs text-emerald-600">
                        {t.edi.alreadyImported}
                      </span>
                    ) : (
                      <button
                        type="button"
                        onClick={() => open(d)}
                        disabled={busy}
                        className="rounded-lg border border-line px-3 py-1 text-xs font-semibold"
                      >
                        {d.direction === "in"
                          ? t.edi.openImport
                          : t.edi.openView}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {openDoc && (
        <div className="mt-6 rounded-2xl border border-line bg-surface p-4 shadow-card">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-lg font-bold">
              {openDoc.number} · {openDoc.date}
            </h2>
            <button
              type="button"
              onClick={() => setOpenDoc(null)}
              className="text-sm text-ink-muted underline"
            >
              {t.common.close}
            </button>
          </div>
          <p className="mt-1 text-sm text-ink-muted">
            {openDoc.partnerName} {openDoc.partnerTin}
          </p>

          {/* ⚠️ Every line is shown, matched or not: a row nobody could match
              is a fact about the invoice worth seeing, and hiding it would make
              a partial import look complete. */}
          <div className="mt-4 overflow-x-auto">
            <table className="w-full min-w-[42rem] border-collapse text-sm">
              <thead>
                <tr className="text-left text-xs uppercase text-ink-muted/70">
                  <th className="py-2 pr-3">{t.edi.lineName}</th>
                  <th className="py-2 pr-3 text-right">{t.edi.qty}</th>
                  <th className="py-2 pr-3 text-right">{t.edi.price}</th>
                  <th className="py-2 pr-3">{t.edi.matchTo}</th>
                </tr>
              </thead>
              <tbody>
                {(openDoc.lines ?? []).map((l) => (
                  <tr key={l.no} className="border-t border-line align-top">
                    <td className="py-2 pr-3">
                      <div>{l.name}</div>
                      <div className="text-xs text-ink-muted tabular-nums">
                        {l.catalogCode}
                      </div>
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums">
                      {l.qty} {l.packageName}
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums">
                      {formatPrice(l.price)}
                    </td>
                    <td className="py-2 pr-3">
                      <select
                        value={picked[l.no] ?? ""}
                        onChange={(e) =>
                          setPicked((prev) => ({
                            ...prev,
                            [l.no]: e.target.value,
                          }))
                        }
                        className="w-full rounded-lg border border-line bg-surface px-2 py-1 text-sm"
                      >
                        <option value="">{t.edi.skipLine}</option>
                        {ingredients.map((i) => (
                          <option key={i.id} value={i.id}>
                            {i.name}
                          </option>
                        ))}
                      </select>
                      {guess[String(l.no)] &&
                        guess[String(l.no)] === picked[l.no] && (
                          <div className="mt-1 text-xs text-ink-muted">
                            {t.edi.guessed}
                          </div>
                        )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {openDoc.direction === "in" && (
            <div className="mt-4 flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={importDoc}
                disabled={busy || matched === 0}
                className="rounded-lg bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
              >
                {t.edi.importAsPurchase}
              </button>
              <span className="text-xs text-ink-muted">
                {t.edi.matchedCount(matched, (openDoc.lines ?? []).length)}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/** The connection itself.
 *
 *  ⚠️ **The credential fields are always blank and an empty one keeps what is
 *  stored** — the rule every settings page here follows, so that an owner
 *  correcting an address does not silently disconnect the integration. */
function EdiSettingsForm({
  settings,
  onSaved,
}: {
  settings: EdiSettings | null;
  onSaved: (s: EdiSettings) => void;
}) {
  const t = useAdminT();
  const [form, setForm] = useState({
    enabled: settings?.enabled ?? false,
    sandbox: settings?.sandbox ?? false,
    tin: settings?.tin ?? "",
    partnerToken: "",
    password: "",
    seller: settings?.seller ?? {
      name: "",
      vatRegCode: "",
      vatRegStatus: 0,
      account: "",
      bankId: "",
      address: "",
      director: "",
      accountant: "",
    },
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setBusy(true);
    setError("");
    try {
      const saved = await api.adminSaveEdiSettings(form);
      onSaved(saved);
      setForm((f) => ({ ...f, partnerToken: "", password: "" }));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const set = (k: string) => (v: string) => setForm((f) => ({ ...f, [k]: v }));
  const setSeller = (k: string) => (v: string) =>
    setForm((f) => ({ ...f, seller: { ...f.seller, [k]: v } }));

  return (
    <div className="mt-5 rounded-2xl border border-line bg-surface p-4 shadow-card">
      <h2 className="text-lg font-bold">{t.edi.settings}</h2>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={(e) =>
              setForm((f) => ({ ...f, enabled: e.target.checked }))
            }
          />
          {t.edi.enabled}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.sandbox}
            onChange={(e) =>
              setForm((f) => ({ ...f, sandbox: e.target.checked }))
            }
          />
          {t.edi.sandbox}
        </label>
        <Field label={t.edi.tin} value={form.tin} onChange={set("tin")} />
        <Field
          label={t.edi.password}
          value={form.password}
          onChange={set("password")}
          type="password"
          hint={settings?.hasPassword ? t.edi.stored : ""}
        />
        <Field
          label={t.edi.partnerToken}
          value={form.partnerToken}
          onChange={set("partnerToken")}
          type="password"
          hint={settings?.hasPartnerKey ? t.edi.stored : ""}
        />
      </div>

      <h3 className="mt-5 text-sm font-semibold">{t.edi.sellerTitle}</h3>
      <p className="mt-1 text-xs text-ink-muted">{t.edi.sellerHint}</p>
      <div className="mt-2 grid gap-3 sm:grid-cols-2">
        <Field
          label={t.edi.name}
          value={form.seller.name}
          onChange={setSeller("name")}
        />
        <Field
          label={t.edi.vatRegCode}
          value={form.seller.vatRegCode}
          onChange={setSeller("vatRegCode")}
        />
        <Field
          label={t.edi.account}
          value={form.seller.account}
          onChange={setSeller("account")}
        />
        <Field
          label={t.edi.bankId}
          value={form.seller.bankId}
          onChange={setSeller("bankId")}
        />
        <Field
          label={t.edi.address}
          value={form.seller.address}
          onChange={setSeller("address")}
        />
        <Field
          label={t.edi.director}
          value={form.seller.director}
          onChange={setSeller("director")}
        />
        <Field
          label={t.edi.accountant}
          value={form.seller.accountant}
          onChange={setSeller("accountant")}
        />
      </div>

      {error && <p className="mt-3 text-sm text-danger">{error}</p>}
      <button
        type="button"
        onClick={save}
        disabled={busy}
        className="mt-4 rounded-lg bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
      >
        {busy ? t.common.saving : t.common.save}
      </button>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  type,
  hint,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  hint?: string;
}) {
  return (
    <label className="block text-sm">
      <span className="text-xs text-ink-muted">{label}</span>
      <input
        type={type ?? "text"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 w-full rounded-lg border border-line bg-surface px-3 py-2"
      />
      {hint && (
        <span className="mt-1 block text-xs text-ink-muted">{hint}</span>
      )}
    </label>
  );
}

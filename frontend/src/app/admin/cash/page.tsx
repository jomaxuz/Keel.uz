"use client";

// The cash drawer: what should be in it, what is, and the difference.
//
// ⚠️ **The difference is the product.** Everything else on this screen — the
// float, the counter takings, the courier handovers, the manual entries —
// exists only to produce a number the drawer can be counted against. A screen
// that shows "expected: 1 240 000", takes what was found and stores only the
// second number has recorded nothing at all: the shortfall it was built to
// surface has been overwritten by the person who might have caused it.
//
// So the count is entered blind-ish and the comparison is shown immediately
// after, `expected` is frozen server-side at the moment of closing, and a
// non-zero difference cannot be saved without a sentence.
//
// ⚠️ **The expected figure deliberately excludes cash couriers are still
// carrying.** Until a courier hands it over the money is in their pocket, not
// in this drawer, and counting both would double every delivery. It is shown
// beside the total anyway, because an owner looking at a short till wants that
// number before they start asking anybody difficult questions.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDateTime, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import type {
  CashEntry,
  CashFigures,
  CashShift,
  ShiftAge,
} from "@/lib/types";

export default function AdminCashPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  const scope = useAdminScope();

  const [shift, setShift] = useState<CashShift | null>(null);
  const [last, setLast] = useState<CashShift | null>(null);
  const [figures, setFigures] = useState<CashFigures | null>(null);
  const [entries, setEntries] = useState<CashEntry[]>([]);
  const [age, setAge] = useState<ShiftAge | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const money = (n: number) => formatPrice(n, "UZS", lang);

  const load = useCallback(() => {
    setLoading(true);
    api
      .cashShift()
      .then((d) => {
        setShift(d.open);
        setLast(d.last ?? null);
        setFigures(d.figures ?? null);
        setEntries(d.entries ?? []);
        setAge(d.age ?? null);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [t]);

  useEffect(load, [load, scope.scopeKey]);

  if (loading) return <p className="text-ink-muted">{t.common.loading}</p>;

  return (
    <div className="space-y-5">
      <header>
        <h1 className="font-display text-2xl font-bold">{t.cash.title}</h1>
        <p className="mt-1 text-sm text-ink-muted">{t.cash.intro}</p>
      </header>

      {error && <p className="text-sm text-danger">{error}</p>}
      {notice && <p className="text-sm text-success">{notice}</p>}

      {/* ⚠️ **Above the figures, not beside them.** Every number on this page is
          computed from a shift that has stopped being a shift: "what should be
          in the drawer" is the sum of several evenings, and the variance at the
          eventual close cannot be attributed to a day or a person. Reading the
          figures first and the caveat afterwards is the wrong order. */}
      {age?.overdue && (
        <p className="rounded-xl border border-danger/40 bg-danger/[0.07] px-3 py-2 text-sm font-semibold text-danger">
          {t.cash.overdue(age.hours, age.maxHours)}
        </p>
      )}

      {shift ? (
        <OpenShift
          shift={shift}
          figures={figures}
          entries={entries}
          money={money}
          onDone={(msg) => {
            setNotice(msg);
            load();
          }}
          onError={setError}
        />
      ) : (
        <ClosedState
          last={last}
          money={money}
          onOpened={load}
          onError={setError}
        />
      )}
    </div>
  );
}

// ---- No shift open ----

function ClosedState({
  last,
  money,
  onOpened,
  onError,
}: {
  last: CashShift | null;
  money: (n: number) => string;
  onOpened: () => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [float_, setFloat] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function open() {
    setBusy(true);
    try {
      await api.openCashShift({
        openingFloat: Number(float_) || 0,
        note: note.trim() || undefined,
      });
      onOpened();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="card p-4">
        <h2 className="font-medium">{t.cash.openTitle}</h2>
        <label className="mt-3 block text-sm">
          <span className="text-ink-muted">{t.cash.openingFloat}</span>
          <input
            className="input mt-1"
            inputMode="numeric"
            value={float_}
            onChange={(e) => setFloat(e.target.value.replace(/\D/g, ""))}
          />
          {/* Said out loud because it is the one figure people miscount: the
              change left in the drawer is already the restaurant's money, not
              takings, and treating it as either changes the variance. */}
          <span className="mt-1 block text-xs text-ink-muted">
            {t.cash.openingFloatHint}
          </span>
        </label>
        <label className="mt-3 block text-sm">
          <span className="text-ink-muted">{t.cash.note}</span>
          <input
            className="input mt-1"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </label>
        <button className="btn btn-primary mt-4" disabled={busy} onClick={open}>
          {busy ? t.common.saving : t.cash.openShift}
        </button>
      </div>

      {/* ⚠️ The last closed shift is the useful answer when nothing is open —
          it says when the till was last counted and how it came out. An empty
          screen would leave the obvious next question unanswered. */}
      {last && (
        <div className="card p-4">
          <h2 className="font-medium">{t.cash.lastShift}</h2>
          <ShiftSummary shift={last} money={money} />
        </div>
      )}
    </div>
  );
}

// ---- A shift is running ----

function OpenShift({
  shift,
  figures,
  entries,
  money,
  onDone,
  onError,
}: {
  shift: CashShift;
  figures: CashFigures | null;
  entries: CashEntry[];
  money: (n: number) => string;
  onDone: (msg: string) => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_20rem]">
      <div className="space-y-4">
        <div className="card p-4">
          <div className="flex items-baseline justify-between gap-2">
            <h2 className="font-medium">{t.cash.expected}</h2>
            <span className="text-xs text-ink-muted">
              {t.cash.openedAt}: {formatDateTime(shift.openedAt)} ·{" "}
              {shift.openedBy}
            </span>
          </div>
          <p className="mt-1 font-display text-3xl font-bold">
            {money(figures?.expected ?? 0)}
          </p>

          <dl className="mt-4 space-y-2 text-sm">
            <Row
              label={t.cash.openingFloat}
              value={money(figures?.openingFloat ?? 0)}
            />
            <Row
              label={`${t.cash.counterCash} (${figures?.counterOrders ?? 0})`}
              value={money(figures?.counterCash ?? 0)}
            />
            <Row
              label={`${t.cash.settlements} (${figures?.settlementCount ?? 0})`}
              value={money(figures?.settlements ?? 0)}
            />
            {(figures?.manualIn ?? 0) > 0 && (
              <Row
                label={t.cash.manualIn}
                value={money(figures?.manualIn ?? 0)}
              />
            )}
            {(figures?.manualOut ?? 0) > 0 && (
              <Row
                label={t.cash.manualOut}
                value={`− ${money(figures?.manualOut ?? 0)}`}
              />
            )}
          </dl>

          {/* ⚠️ Outside the list and visually apart, because it is **not** part
              of the sum. Cash a courier is still carrying is real money that is
              not in this drawer; putting it in the column above would double
              every delivery, and putting it nowhere would leave the first
              question about a short till unanswered. */}
          {(figures?.withCouriers ?? 0) > 0 && (
            <div className="mt-4 rounded-xl bg-ink/5 p-3">
              <div className="flex justify-between text-sm">
                <span className="text-ink-muted">{t.cash.withCouriers}</span>
                <span className="font-medium">
                  {money(figures?.withCouriers ?? 0)}
                </span>
              </div>
              <p className="mt-1 text-xs text-ink-muted">
                {t.cash.withCouriersHint}
              </p>
            </div>
          )}
        </div>

        <EntryList entries={entries} money={money} />
      </div>

      <div className="space-y-4">
        <AddEntry onDone={onDone} onError={onError} />
        <CloseShift
          expected={figures?.expected ?? 0}
          money={money}
          onDone={onDone}
          onError={onError}
        />
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-2">
      <dt className="text-ink-muted">{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function EntryList({
  entries,
  money,
}: {
  entries: CashEntry[];
  money: (n: number) => string;
}) {
  const t = useAdminT();
  if (entries.length === 0) return null;
  return (
    <div className="card p-4">
      <h2 className="font-medium">{t.cash.entries}</h2>
      <ListScroll className="mt-3">
        <ul className="space-y-2">
          {entries.map((e) => (
            <li
              key={e.id}
              className="flex items-start justify-between gap-3 text-sm"
            >
              <div className="min-w-0">
                <div className="truncate font-medium">{e.category}</div>
                <div className="truncate text-xs text-ink-muted">
                  {formatDateTime(e.at)} · {e.by}
                  {e.note ? ` · ${e.note}` : ""}
                </div>
              </div>
              <span
                className={
                  e.kind === "out"
                    ? "shrink-0 text-danger"
                    : "shrink-0 text-success"
                }
              >
                {e.kind === "out" ? "−" : "+"} {money(e.amount)}
              </span>
            </li>
          ))}
        </ul>
      </ListScroll>
    </div>
  );
}

function AddEntry({
  onDone,
  onError,
}: {
  onDone: (msg: string) => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [kind, setKind] = useState<"in" | "out">("out");
  const [category, setCategory] = useState("");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    try {
      await api.addCashEntry({
        kind,
        category: category.trim(),
        amount: Number(amount) || 0,
        note: note.trim() || undefined,
      });
      setCategory("");
      setAmount("");
      setNote("");
      onDone(t.cash.entrySaved);
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card p-4">
      <h2 className="font-medium">{t.cash.addEntry}</h2>
      <div className="mt-3 grid grid-cols-2 gap-2">
        {(["out", "in"] as const).map((k) => (
          <button
            key={k}
            onClick={() => setKind(k)}
            className={`rounded-xl border px-2 py-2 text-sm ${
              kind === k
                ? "border-brand bg-brand/10 font-medium"
                : "border-line"
            }`}
          >
            {k === "out" ? t.cash.kindOut : t.cash.kindIn}
          </button>
        ))}
      </div>
      <label className="mt-3 block text-sm">
        <span className="text-ink-muted">{t.cash.category}</span>
        <input
          className="input mt-1"
          placeholder={t.cash.categoryPh}
          value={category}
          onChange={(e) => setCategory(e.target.value)}
        />
      </label>
      <label className="mt-3 block text-sm">
        <span className="text-ink-muted">{t.cash.amount}</span>
        <input
          className="input mt-1"
          inputMode="numeric"
          value={amount}
          onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
        />
      </label>
      <label className="mt-3 block text-sm">
        <span className="text-ink-muted">{t.cash.note}</span>
        <input
          className="input mt-1"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>
      <button
        className="btn mt-4 w-full"
        disabled={busy || !category.trim() || !amount}
        onClick={save}
      >
        {busy ? t.common.saving : t.common.save}
      </button>
    </div>
  );
}

function CloseShift({
  expected,
  money,
  onDone,
  onError,
}: {
  expected: number;
  money: (n: number) => string;
  onDone: (msg: string) => void;
  onError: (m: string) => void;
}) {
  const t = useAdminT();
  const [counted, setCounted] = useState("");
  const [varianceNote, setVarianceNote] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const typed = counted !== "";
  const variance = (Number(counted) || 0) - expected;
  const needsNote = typed && variance !== 0 && !varianceNote.trim();

  async function close() {
    setBusy(true);
    try {
      const res = await api.closeCashShift({
        counted: Number(counted) || 0,
        varianceNote: varianceNote.trim() || undefined,
        note: note.trim() || undefined,
      });
      // ⚠️ The register's day is asked to end alongside, and it can refuse —
      // usually because receipts are still unfiled. Reported rather than
      // swallowed: the drawer count succeeded either way, and the manager is
      // the only person who can go and get those receipts filed.
      onDone(
        res.fiscalNote ? `${t.cash.closed} — ${res.fiscalNote}` : t.cash.closed,
      );
    } catch (e) {
      onError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="card p-4">
      <h2 className="font-medium">{t.cash.closeTitle}</h2>
      <label className="mt-3 block text-sm">
        <span className="text-ink-muted">{t.cash.counted}</span>
        <input
          className="input mt-1"
          inputMode="numeric"
          value={counted}
          onChange={(e) => setCounted(e.target.value.replace(/\D/g, ""))}
        />
      </label>

      {/* ⚠️ Shown only after a number is typed. Putting the expected figure
          beside an empty box invites it to be copied — and a count that agrees
          because it was read off the screen is the one record this whole page
          exists to prevent. */}
      {typed && (
        <div
          className={`mt-3 rounded-xl p-3 text-sm ${
            variance === 0 ? "bg-success/10" : "bg-danger/10"
          }`}
        >
          <div className="flex justify-between">
            <span className="text-ink-muted">{t.cash.expected}</span>
            <span>{money(expected)}</span>
          </div>
          <div className="mt-1 flex justify-between font-medium">
            <span>{t.cash.variance}</span>
            <span>
              {variance > 0 ? "+" : ""}
              {money(variance)}
            </span>
          </div>
        </div>
      )}

      {typed && variance !== 0 && (
        <label className="mt-3 block text-sm">
          <span className="text-ink-muted">{t.cash.varianceNote}</span>
          <input
            className="input mt-1"
            value={varianceNote}
            onChange={(e) => setVarianceNote(e.target.value)}
          />
          {/* The server refuses it empty too — this is the polite half. */}
          <span className="mt-1 block text-xs text-ink-muted">
            {t.cash.varianceNoteHint}
          </span>
        </label>
      )}

      <label className="mt-3 block text-sm">
        <span className="text-ink-muted">{t.cash.note}</span>
        <input
          className="input mt-1"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>

      <button
        className="btn btn-primary mt-4 w-full"
        disabled={busy || !typed || needsNote}
        onClick={close}
      >
        {busy ? t.common.saving : t.cash.closeShift}
      </button>
    </div>
  );
}

// ---- A closed shift, and what the register said about the same day ----

function ShiftSummary({
  shift,
  money,
}: {
  shift: CashShift;
  money: (n: number) => string;
}) {
  const t = useAdminT();
  return (
    <div className="mt-3 space-y-3 text-sm">
      <div className="text-xs text-ink-muted">
        {formatDateTime(shift.openedAt)} —{" "}
        {shift.closedAt && formatDateTime(shift.closedAt)}
        {shift.closedBy ? ` · ${shift.closedBy}` : ""}
      </div>
      <dl className="space-y-2">
        <Row label={t.cash.expected} value={money(shift.expected)} />
        <Row label={t.cash.counted} value={money(shift.counted)} />
        <div className="flex justify-between gap-2 font-medium">
          <dt>{t.cash.variance}</dt>
          <dd className={shift.variance === 0 ? "text-success" : "text-danger"}>
            {shift.variance > 0 ? "+" : ""}
            {money(shift.variance)}
          </dd>
        </div>
      </dl>
      {shift.varianceNote && (
        <p className="rounded-xl bg-ink/5 p-2 text-xs">{shift.varianceNote}</p>
      )}

      {/* ⚠️ The register's own totals for the same day, kept apart from ours.
          This is a **second, independent count**: the figures above come from
          the orders we recorded, these from the machine that filed them with
          the state. When a drawer is short, the first useful question is which
          of the two the cash agrees with — and it can only be asked if both are
          on the screen. It never corrects the figures above. */}
      {shift.fiscal && (
        <div className="rounded-xl border border-line p-3">
          <div className="flex items-baseline justify-between gap-2">
            <span className="font-medium">{t.cash.fiscalDay}</span>
            {shift.fiscal.number && (
              <span className="text-xs text-ink-muted">
                {t.till.zNumber}: {shift.fiscal.number}
              </span>
            )}
          </div>
          {shift.fiscal.error ? (
            <p className="mt-1 text-xs text-danger">{shift.fiscal.error}</p>
          ) : (
            <dl className="mt-2 space-y-1 text-xs">
              {/* ⚠️ Our figure beside theirs, and **counter cash** rather than
                  the drawer total: the register knows about cash sales, not
                  about the float or what couriers brought back. Comparing it
                  with anything else produces a difference on every shift, which
                  is how a real one becomes invisible. */}
              <Row
                label={t.cash.counterCash}
                value={money(shift.counterCash)}
              />
              <Row
                label={t.till.methodCash}
                value={money(shift.fiscal.saleCash)}
              />
              <Row
                label={t.till.methodCard}
                value={money(shift.fiscal.saleCard)}
              />
              {shift.fiscal.refundTotal > 0 && (
                <Row
                  label={t.till.refunds}
                  value={money(shift.fiscal.refundTotal)}
                />
              )}
              <Row label={t.till.total} value={money(shift.fiscal.saleTotal)} />
            </dl>
          )}
          <p className="mt-2 text-xs text-ink-muted">{t.cash.fiscalDayHint}</p>
        </div>
      )}
    </div>
  );
}

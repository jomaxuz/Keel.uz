"use client";

// What is missing, what it is worth, and who is going to answer for it.
//
// ⚠️ **The arithmetic was finished and nobody could act on it.** A count has
// frozen an expected figure, a counted one and what the difference was worth
// since the stocktake screen shipped — and all of it was reachable only by
// opening one count and reading down forty lines, where the twelve kilos of
// meat sit between two lines of parsley that were out by a gram. This screen
// invents no number. It sorts them by money, names the period each one
// accumulated over, and records an answer once.
//
// ⚠️ **A question, never an accusation.** Nothing here names a suspect, and the
// five answers deliberately do not include one. A shortfall covers every shift
// between two counts — usually a month and everybody who worked it — which is
// the weakest possible evidence about a person and the strongest possible
// evidence about a process. The same line models/alert.go draws.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { sellsGoods } from "@/lib/types";
import { verdictsFor } from "@/lib/shortages";
import type { ShortageQueue, ShortageRow, ShortageVerdict } from "@/lib/types";

/** How far back to look. ⚠️ Ninety days first: a restaurant counting monthly
 *  has three counts in it, which is the shortest window in which a shortfall
 *  can look like a habit rather than an evening. */
const WINDOWS = [30, 90, 180, 365];

/** The pair of ids that names a case — there is no id of its own until it is
 *  answered, and by then the queue no longer needs one. */
const keyOf = (r: ShortageRow) => `${r.stocktakeId}:${r.ingredientId}`;

export default function ShortagesPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  // ⚠️ **Whether this business sells what it bought — the only difference
  // between the two readings of this screen.** The arithmetic is identical in a
  // kitchen and a pharmacy (models/businesstype.go says why the stock module is
  // deliberately not tailored); what changes is the word for the caveat and
  // which answers can possibly be true. See lib/shortages.ts.
  const goods = sellsGoods(scope.brand);
  const verdicts = verdictsFor(scope.brand);
  const [data, setData] = useState<ShortageQueue | null>(null);
  const [days, setDays] = useState(90);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  // Which row is being answered, and what the answer says so far.
  const [open, setOpen] = useState("");
  const [verdict, setVerdict] = useState<ShortageVerdict>("miscount");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminShortages(days)
      .then((d) => {
        setData(d);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [days, t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  async function answer(row: ShortageRow) {
    const text = note.trim();
    if (!text) return;
    setBusy(true);
    try {
      await api.adminCloseShortage({
        stocktakeId: row.stocktakeId,
        ingredientId: row.ingredientId,
        verdict,
        note: text,
      });
      setOpen("");
      setNote("");
      setVerdict("miscount");
      load();
    } catch (e) {
      // ⚠️ The one failure worth naming is "somebody answered it first" — the
      // server refuses the second write and the reload below shows their words
      // in place of this box.
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
      load();
    } finally {
      setBusy(false);
    }
  }

  const rows = data?.rows ?? [];

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.shortages.title}</h1>
      <p className="mt-2 max-w-3xl text-sm text-ink-muted">
        {t.shortages.intro}
      </p>

      <div className="mt-4 flex flex-wrap gap-2">
        {WINDOWS.map((d) => (
          <button
            key={d}
            type="button"
            onClick={() => setDays(d)}
            className={`rounded-lg border px-3 py-1.5 text-sm font-semibold ${
              d === days ? "border-brand bg-brand/10" : "border-line"
            }`}
          >
            {t.shortages.window(d)}
          </button>
        ))}
      </div>

      {/* ⚠️ **The unanswered money leads, not the total.** A total only ever
          grows, and a number that only grows is one nobody reads twice. */}
      {data && (
        <div className="mt-5 grid gap-3 sm:grid-cols-3">
          <div className="rounded-2xl border border-line bg-surface p-4 shadow-card">
            <div className="text-xs uppercase text-ink-muted/70">
              {t.shortages.openValue}
            </div>
            <div className="mt-1 text-2xl font-bold tabular-nums text-danger">
              {formatPrice(data.openValue)}
            </div>
            <div className="mt-1 text-xs text-ink-muted">
              {t.shortages.openCount(data.open)}
            </div>
          </div>
          <div className="rounded-2xl border border-line bg-surface p-4 shadow-card">
            <div className="text-xs uppercase text-ink-muted/70">
              {t.shortages.total}
            </div>
            <div className="mt-1 text-2xl font-bold tabular-nums">
              {formatPrice(data.total)}
            </div>
            <div className="mt-1 text-xs text-ink-muted">
              {formatDate(data.from)} — {formatDate(data.to)}
            </div>
          </div>
          <div className="rounded-2xl border border-line bg-surface p-4 shadow-card">
            <div className="text-xs uppercase text-ink-muted/70">
              {goods ? t.shortages.coverageTitleGoods : t.shortages.coverageTitle}
            </div>
            <div
              className={`mt-1 text-2xl font-bold tabular-nums ${
                data.weak ? "text-amber-600" : ""
              }`}
            >
              {data.coverage}%
            </div>
            <div className="mt-1 text-xs text-ink-muted">
              {goods ? t.shortages.coverageGoods : t.shortages.coverage}
            </div>
          </div>
        </div>
      )}

      {/* ⚠️ **The caveat the whole screen stands on.** A shortfall is "what the
          books expected, less what was found", and the books only expect what a
          tech card told them to. Where half the revenue leaves the shelf with
          no card, this list is confident, precise and invented — and the first
          row an owner investigates costs them their trust in the screen. */}
      {data?.weak && (
        <p className="mt-4 rounded-2xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-700">
          {goods ? t.shortages.weakGoods : t.shortages.weak}
        </p>
      )}

      {error && <p className="mt-4 text-sm text-danger">{error}</p>}

      {loading ? (
        <p className="mt-6 text-sm text-ink-muted">{t.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-6 text-sm text-ink-muted">{t.shortages.empty}</p>
      ) : (
        <div className="mt-6 space-y-3">
          {rows.map((r) => {
            const key = keyOf(r);
            const answered = Boolean(r.verdict);
            return (
              <div
                key={key}
                className={`rounded-3xl border bg-surface p-4 shadow-card ${
                  answered ? "border-line opacity-80" : "border-danger/30"
                }`}
              >
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <div>
                    <span className="text-base font-semibold">{r.name}</span>
                    <span className="ml-2 text-sm text-ink-muted tabular-nums">
                      {r.diff} {r.unit}
                    </span>
                  </div>
                  <div className="text-lg font-bold tabular-nums text-danger">
                    {formatPrice(r.value)}
                  </div>
                </div>

                <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-muted">
                  {/* ⚠️ Where, and over what stretch of time. A shortfall
                      belongs to the period between two counts of that store,
                      not to the day it was found — without this line the whole
                      thing reads as "last night". */}
                  {r.branch && <span>{r.branch}</span>}
                  {r.warehouse && <span>{r.warehouse}</span>}
                  <span>
                    {r.since
                      ? t.shortages.period(
                          formatDate(r.since),
                          formatDate(r.at),
                        )
                      : t.shortages.periodNever(formatDate(r.at))}
                  </span>
                  {r.by && <span>{t.shortages.countedBy(r.by)}</span>}
                  <span className="tabular-nums">
                    {t.shortages.expected(r.expected, r.counted)}
                  </span>
                  {/* How much of the whole count's shortfall this one line is:
                      one thing that happened, or a month of small slippage
                      across forty lines. */}
                  {r.share > 0 && (
                    <span>
                      {t.shortages.shareOfCount(Math.round(r.share * 100))}
                    </span>
                  )}
                  {/* ⚠️ **How fast, and against what.** The same money is
                      shrinkage over a quarter and an event over four days, and
                      twelve kilos out of four hundred is trade while twelve out
                      of fourteen is not — the queue is sorted by money, so the
                      row has to carry what money cannot say. */}
                  {r.perDay ? (
                    <span className="font-medium text-ink-soft">
                      {t.shortages.perDay(formatPrice(r.perDay))}
                    </span>
                  ) : null}
                  {r.pct ? (
                    <span>{t.shortages.pct(Math.round(r.pct * 100))}</span>
                  ) : null}
                  {r.used > 0 && (
                    <span>
                      {t.shortages.used(`${r.used} ${r.unit}`)}
                    </span>
                  )}
                  {/* Twice is a pattern; once is an evening. */}
                  {r.repeat ? (
                    <span className="font-medium text-ink-soft">
                      {t.shortages.repeat(r.repeat)}
                    </span>
                  ) : null}
                </div>

                {/* ⚠️ **The strongest sentence this screen has.** Where no card
                    consumed the ingredient over the period, "expected" is
                    everything that ever arrived and the difference is a gap in
                    the cards — not a loss. The coverage figure at the top
                    cannot say it: that is a fact about the restaurant, and this
                    is a fact about this row. */}
                {r.used === 0 && (
                  <p className="mt-2 rounded-xl bg-amber-500/10 px-3 py-2 text-xs text-amber-700">
                    {goods ? t.shortages.noUseGoods : t.shortages.noUse}
                  </p>
                )}
                {/* The mis-scan, offered as a question. */}
                {r.twin && (
                  <p className="mt-2 text-xs text-ink-muted">
                    {t.shortages.twin(r.twin)}
                  </p>
                )}

                {r.countNote && (
                  <p className="mt-2 text-xs text-ink-muted">
                    {t.shortages.countNote(r.countNote)}
                  </p>
                )}

                {answered ? (
                  <p className="mt-3 rounded-2xl bg-cream px-3 py-2 text-sm">
                    <span className="font-semibold">
                      {t.shortages.verdicts[r.verdict as ShortageVerdict]}
                    </span>
                    {" — "}
                    {r.verdictNote}
                    <span className="ml-2 text-xs text-ink-muted">
                      {r.verdictBy}
                      {r.verdictAt ? ` · ${formatDate(r.verdictAt)}` : ""}
                    </span>
                  </p>
                ) : open === key ? (
                  <div className="mt-3 space-y-2">
                    <select
                      value={verdict}
                      onChange={(e) =>
                        setVerdict(e.target.value as ShortageVerdict)
                      }
                      className="w-full rounded-xl border border-line bg-surface px-3 py-2 text-sm"
                    >
                      {verdicts.map((v) => (
                        <option key={v} value={v}>
                          {t.shortages.verdicts[v]}
                        </option>
                      ))}
                    </select>
                    {/* ⚠️ The sentence is required beside the kind. The five
                        kinds exist so a month of these can be counted, not so a
                        shortfall can be waved away in one click. */}
                    <textarea
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      rows={2}
                      placeholder={
                        goods ? t.shortages.notePhGoods : t.shortages.notePh
                      }
                      className="w-full rounded-xl border border-line bg-surface px-3 py-2 text-sm"
                    />
                    <div className="flex flex-wrap items-center gap-2">
                      <button
                        type="button"
                        disabled={busy || !note.trim()}
                        onClick={() => answer(r)}
                        className="rounded-xl bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                      >
                        {t.shortages.save}
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          setOpen("");
                          setNote("");
                        }}
                        className="rounded-xl border border-line px-4 py-2 text-sm"
                      >
                        {t.common.cancel}
                      </button>
                      {/* ⚠️ Said before the button is pressed, not after: an
                          answer cannot be edited, and finding that out
                          afterwards is finding it out too late. */}
                      <span className="text-xs text-ink-muted">
                        {t.shortages.onceHint}
                      </span>
                    </div>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => {
                      setOpen(key);
                      setNote("");
                      setVerdict("miscount");
                    }}
                    className="mt-3 rounded-xl border border-line px-4 py-2 text-sm font-semibold"
                  >
                    {t.shortages.answer}
                  </button>
                )}
              </div>
            );
          })}
          {/* ⚠️ Said rather than implied: the list is capped, and a screen that
              quietly ends is a screen somebody reads as complete. */}
          {data && data.more > 0 && (
            <p className="text-xs text-ink-muted">
              {t.shortages.more(data.more)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

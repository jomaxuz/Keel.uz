"use client";

// What broke, across every restaurant, before anybody wrote in about it.
//
// ⚠️ **Two panes like the support queue, and for the same reason**: fixing one
// fault means reading it, then the next one, then coming back to compare. A
// screen that navigates away on every click is a screen somebody loses their
// place in twenty times a morning.
//
// ⚠️ **A list of faults, not of occurrences.** One render loop is ten thousand
// identical errors in an afternoon; the platform groups them on arrival, so a
// row here is a bug and the numbers beside it say how much of one.
//
// ⚠️ **Open by default.** A screen that opens on everything ever fixed puts
// today's three faults below four hundred old ones.

import { useCallback, useEffect, useState } from "react";
import {
  reportGroup,
  reportList,
  reportResolve,
  type ErrorGroupRow,
  type ErrorSample,
} from "@/lib/api";

const FILTERS = [
  { id: "open", label: "Ochiq" },
  { id: "resolved", label: "Tuzatilgan" },
  { id: "all", label: "Hammasi" },
];

const APP_LABEL: Record<string, string> = {
  panel: "Panel",
  site: "Sayt",
  till: "Kassa / zal",
  waiter: "Ofitsiant",
  kitchen: "Oshxona",
  courier: "Kuryer",
  server: "Server",
};

// Nobody presses reload on a queue. Slower than support's ten seconds: a crash
// that arrived forty seconds ago is not more urgent than one that arrived now,
// and this list is read rather than worked.
const REFRESH_MS = 30_000;

export default function ReportsPage() {
  const [state, setState] = useState("open");
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<ErrorGroupRow[]>([]);
  const [error, setError] = useState("");

  const [openID, setOpenID] = useState<string | null>(null);
  const [group, setGroup] = useState<
    (ErrorGroupRow & { samples?: ErrorSample[] }) | null
  >(null);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await reportList({ state, q: q.trim() || undefined });
      setRows(r.groups);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Yuklab bo'lmadi");
    }
  }, [state, q]);

  useEffect(() => {
    void load();
    const id = setInterval(() => void load(), REFRESH_MS);
    return () => clearInterval(id);
  }, [load]);

  useEffect(() => {
    if (!openID) {
      setGroup(null);
      return;
    }
    void reportGroup(openID)
      .then((r) => {
        setGroup(r.group);
        setNote(r.group.note ?? "");
      })
      .catch(() => setGroup(null));
  }, [openID]);

  async function toggleResolved() {
    if (!group) return;
    setBusy(true);
    try {
      await reportResolve(group.id, { resolved: !group.resolved, note });
      const r = await reportGroup(group.id);
      setGroup(r.group);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Saqlab bo'lmadi");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      {/* ---- The list ---- */}
      <div className="rounded-2xl border border-line bg-surface">
        <div className="flex flex-wrap items-center gap-2 border-b border-line p-3">
          {FILTERS.map((f) => (
            <button
              key={f.id}
              type="button"
              onClick={() => setState(f.id)}
              className={`rounded-lg px-3 py-1.5 text-sm ${
                state === f.id
                  ? "bg-ink text-surface font-medium"
                  : "text-ink-muted hover:bg-page"
              }`}
            >
              {f.label}
            </button>
          ))}
          <input
            className="ml-auto min-w-40 flex-1 rounded-lg border border-line bg-page px-3 py-1.5 text-sm"
            placeholder="Xato matni, sahifa yoki restoran"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>

        {error && <p className="p-3 text-sm text-danger">{error}</p>}

        {rows.length === 0 && !error && (
          /* ⚠️ Says which list is empty. "Hech narsa yo'q" under a filter
             nobody remembers setting reads as a broken screen. */
          <p className="p-6 text-center text-sm text-ink-muted">
            {state === "open"
              ? "Ochiq xatolik yo'q."
              : "Bu ro'yxatda hech narsa yo'q."}
          </p>
        )}

        <ul className="divide-y divide-line">
          {rows.map((g) => (
            <li key={g.id}>
              <button
                type="button"
                onClick={() => setOpenID(g.id)}
                className={`block w-full px-3 py-3 text-left hover:bg-page ${
                  openID === g.id ? "bg-page" : ""
                }`}
              >
                <div className="flex items-start justify-between gap-3">
                  <span className="min-w-0 flex-1 truncate text-sm font-medium">
                    {g.message}
                  </span>
                  {/* ⚠️ Today's count, not the total. "Forty times" means two
                      entirely different things over an afternoon and over three
                      months, and this list is about what is happening now. */}
                  {g.today > 0 && (
                    <span className="shrink-0 rounded-md bg-danger/10 px-1.5 py-0.5 text-xs font-semibold text-danger">
                      {g.today} bugun
                    </span>
                  )}
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ink-muted">
                  <span className="font-medium">{g.restaurant || g.slug}</span>
                  <span>·</span>
                  <span>{APP_LABEL[g.app] ?? g.app}</span>
                  {g.where && (
                    <>
                      <span>·</span>
                      <span className="truncate font-mono">{g.where}</span>
                    </>
                  )}
                  <span>·</span>
                  {/* ⚠️ How many installs, beside how many times. One cashier
                      with a broken tablet and every cashier in the chain are the
                      same count and completely different mornings. */}
                  <span>
                    {g.count} marta{g.users > 1 ? ` · ${g.users} qurilma` : ""}
                  </span>
                  {g.resolved && (
                    <span className="rounded bg-page px-1.5 py-0.5">
                      tuzatilgan
                    </span>
                  )}
                </div>
              </button>
            </li>
          ))}
        </ul>
      </div>

      {/* ---- One fault ---- */}
      <div className="rounded-2xl border border-line bg-surface">
        {!group ? (
          <p className="p-6 text-center text-sm text-ink-muted">
            Tafsilotlar uchun xatolikni tanlang.
          </p>
        ) : (
          <div className="p-4">
            <p className="text-sm font-semibold">{group.message}</p>
            <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-ink-muted">
              <span>{group.restaurant || group.slug}</span>
              <span>{APP_LABEL[group.app] ?? group.app}</span>
              {group.where && <span className="font-mono">{group.where}</span>}
            </div>

            <dl className="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
              <Fact k="Jami" v={String(group.count)} />
              <Fact k="Bugun" v={String(group.today)} />
              <Fact k="Qurilma" v={String(group.users || 1)} />
              <Fact k="Oxirgi" v={when(group.lastAt)} />
              <Fact k="Birinchi" v={when(group.firstAt)} />
              {/* ⚠️ Both versions, because the first question asked of any of
                  these is "did the deploy fix it" — and that is the two numbers
                  side by side, not one of them. */}
              <Fact k="Ilk versiya" v={group.firstVersion || "—"} />
              <Fact k="Oxirgi versiya" v={group.latestVersion || "—"} />
              {group.resolved && (
                <Fact
                  k="Tuzatilganda"
                  v={`${group.resolvedCount ?? 0} marta edi`}
                />
              )}
            </dl>

            {/* ⚠️ The regression, said out loud. A resolved fault that has
                counted again since the fix is the single most useful row on this
                screen, and it is arithmetic — nobody should have to remember
                what the number was. */}
            {group.resolved &&
              group.count > (group.resolvedCount ?? group.count) && (
                <p className="mt-3 rounded-xl border border-danger/40 bg-danger/5 px-3 py-2 text-sm text-danger">
                  Tuzatilgandan keyin yana{" "}
                  {group.count - (group.resolvedCount ?? 0)} marta takrorlandi —
                  tuzatish ushlamagan.
                </p>
              )}

            <div className="mt-4 space-y-3">
              {(group.samples ?? [])
                .slice()
                .reverse()
                .map((s, i) => (
                  <div key={i} className="rounded-xl border border-line p-3">
                    <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-ink-muted">
                      <span>{when(s.at)}</span>
                      {s.role && <span>{s.role}</span>}
                      {s.branch && <span>{s.branch}</span>}
                      {s.version && <span>v{s.version}</span>}
                      {s.context && <span>{s.context}</span>}
                    </div>
                    {s.platform && (
                      <p className="mt-1 truncate text-xs text-ink-soft">
                        {s.platform}
                      </p>
                    )}
                    {s.stack && (
                      /* Its own scroll box: a stack is wide, and a page that
                         scrolls sideways because of one is a page where nothing
                         else can be read. */
                      <pre className="mt-2 max-h-64 overflow-auto rounded-lg bg-page p-2 text-[11px] leading-relaxed">
                        {s.stack}
                      </pre>
                    )}
                  </div>
                ))}
              {(group.samples ?? []).length === 0 && (
                <p className="text-sm text-ink-muted">
                  Bu xatolikda stek saqlanmagan.
                </p>
              )}
            </div>

            <div className="mt-4 border-t border-line pt-4">
              <label className="block text-sm">
                <span className="font-medium">Nima qilindi</span>
                <input
                  className="mt-1 w-full rounded-lg border border-line bg-page px-3 py-2 text-sm"
                  placeholder="Bir qator — keyingi safar ko'radigan odam uchun"
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </label>
              <button
                type="button"
                disabled={busy}
                onClick={toggleResolved}
                className="mt-3 rounded-lg bg-ink px-4 py-2 text-sm font-medium text-surface disabled:opacity-50"
              >
                {group.resolved ? "Qayta ochish" : "Tuzatildi deb belgilash"}
              </button>
              {group.resolved && group.resolvedBy && (
                <p className="mt-2 text-xs text-ink-muted">
                  {group.resolvedBy} · {when(group.resolvedAt)}
                </p>
              )}
              {/* ⚠️ Said here, because "tuzatildi" reads like "stop watching
                  this" and it is the opposite. */}
              <p className="mt-2 text-xs text-ink-soft">
                Belgilash yig'ishni to'xtatmaydi va hech nimani o'chirmaydi —
                aynan shuning uchun qaytib kelgani ko'rinadi.
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function Fact({ k, v }: { k: string; v: string }) {
  return (
    <div>
      <dt className="text-xs text-ink-muted">{k}</dt>
      <dd className="font-medium">{v}</dd>
    </div>
  );
}

function when(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString("uz-UZ", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

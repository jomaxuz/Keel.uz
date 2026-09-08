"use client";

// Building the restaurant's own Android app.
//
// ⚠️ **Nine minutes, one at a time, on the machine that serves every customer.**
// Gradle and the Kotlin compiler together want more memory than that box has
// spare, so the queue is one deep and the panel says where a build is rather
// than spinning silently — a button that looks stuck is a button people press
// again, and the second press costs another nine minutes for a byte-identical
// file.
//
// ⚠️ **The artifact is deleted the moment it has been downloaded.** Two and a
// half megabytes per build adds up to a directory nobody prunes; the record
// stays, with the version and the hash, because "which version is on the store"
// is asked months later and a filesystem cannot answer it. The panel shows
// history whose files are mostly gone, and says so on each row.
//
// ⚠️ **The format is chosen before the build, not after.** An APK installs on a
// phone this afternoon; an AAB is what Play accepts and cannot be installed at
// all. Guessing costs a build and is discovered at the end of an upload.

import { useCallback, useEffect, useState } from "react";
import {
  appBuilds,
  bytes,
  downloadAppBuild,
  startAppBuild,
  type AppBuild,
} from "@/lib/api";

const LABEL: Record<string, string> = {
  queued: "navbatda",
  building: "qurilmoqda",
  ready: "tayyor",
  taken: "yuklab olingan",
  failed: "xato",
};

export default function AppBuildPanel({ tenantId }: { tenantId: string }) {
  const [builds, setBuilds] = useState<AppBuild[]>([]);
  const [format, setFormat] = useState<"apk" | "aab">("apk");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setBuilds((await appBuilds(tenantId)).builds);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [tenantId]);

  useEffect(() => {
    void load();
  }, [load]);

  // ⚠️ **Polled only while something is unfinished.** A page left open on a
  // tenant nobody is building for should not ask every five seconds forever —
  // and the moment the last build settles, the polling stops on its own.
  const working = builds.some(
    (b) => b.status === "queued" || b.status === "building",
  );
  useEffect(() => {
    if (!working) return;
    const timer = setInterval(() => void load(), 5000);
    return () => clearInterval(timer);
  }, [working, load]);

  async function start() {
    setBusy(true);
    setError("");
    try {
      await startAppBuild(tenantId, format);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function take(build: AppBuild) {
    setError("");
    try {
      await downloadAppBuild(build);
    } catch (e) {
      // ⚠️ Said out loud. The server deletes its copy once the transfer
      // completes, so a silent failure would leave somebody believing they hold
      // a file they do not.
      setError(e instanceof Error ? e.message : String(e));
    }
    await load();
  }

  return (
    <section className="card space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Android ilova</h2>
        <p className="mt-1 text-sm text-ink-soft">
          Restoranning o&apos;z ilovasi: nomi, logosi, rangi va server manzili
          shu mijozdan olinadi. Build ~9 daqiqa oladi va bir vaqtda bittasi
          quriladi.
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="flex flex-wrap items-center gap-2">
        <div className="flex overflow-hidden rounded-xl border border-line">
          {(["apk", "aab"] as const).map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setFormat(f)}
              className={`px-3 py-1.5 text-sm ${
                format === f
                  ? "bg-brand text-white"
                  : "text-ink-soft hover:bg-ink/5"
              }`}
            >
              {f.toUpperCase()}
            </button>
          ))}
        </div>
        <button
          type="button"
          className="btn-primary"
          disabled={busy || working}
          onClick={() => void start()}
        >
          {working ? "Build ketmoqda…" : "Build qilish"}
        </button>
        {/* ⚠️ The difference between the two, where the choice is made. Told
            afterwards it is a wasted build; told here it is one tap. */}
        <span className="text-xs text-ink-muted">
          {format === "apk"
            ? "APK — telefonga to'g'ridan-to'g'ri o'rnatiladi"
            : "AAB — faqat Play Store uchun, telefonga o'rnatib bo'lmaydi"}
        </span>
      </div>

      {builds.length === 0 ? (
        <p className="text-sm text-ink-muted">Hali build qilinmagan.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-ink-muted">
              <tr>
                <th className="py-1.5 pr-3">Versiya</th>
                <th className="py-1.5 pr-3">Turi</th>
                <th className="py-1.5 pr-3">Holati</th>
                <th className="py-1.5 pr-3">Kim</th>
                <th className="py-1.5 text-right">Fayl</th>
              </tr>
            </thead>
            <tbody>
              {builds.map((b) => (
                <tr key={b.id} className="border-t border-line align-top">
                  <td className="py-2 pr-3">
                    <div className="font-medium tabular-nums">
                      {b.versionName ?? "—"}
                    </div>
                    <div className="text-xs text-ink-muted">
                      {new Date(b.createdAt).toLocaleString("uz-UZ")}
                    </div>
                  </td>
                  <td className="py-2 pr-3 uppercase">{b.format}</td>
                  <td className="py-2 pr-3">
                    <span
                      className={
                        b.status === "failed"
                          ? "text-danger"
                          : b.status === "ready"
                            ? "font-medium text-emerald-700 dark:text-emerald-300"
                            : "text-ink-soft"
                      }
                    >
                      {LABEL[b.status] ?? b.status}
                    </span>
                    {b.error && (
                      <pre className="mt-1 max-w-md overflow-x-auto whitespace-pre-wrap text-xs text-ink-muted">
                        {b.error}
                      </pre>
                    )}
                  </td>
                  <td className="py-2 pr-3 text-xs text-ink-muted">
                    {b.by || "—"}
                    {b.downloadedBy && (
                      <div>olgan: {b.downloadedBy}</div>
                    )}
                  </td>
                  <td className="py-2 text-right">
                    {b.status === "ready" ? (
                      <button
                        type="button"
                        className="btn-ghost px-3 py-1 text-xs"
                        onClick={() => void take(b)}
                      >
                        Yuklab olish{b.size ? ` (${bytes(b.size)})` : ""}
                      </button>
                    ) : b.status === "taken" ? (
                      // ⚠️ **Says the file is gone, not that nothing happened.**
                      // A blank cell reads as a build that produced nothing.
                      <span className="text-xs text-ink-muted">
                        olingan — fayl o&apos;chirilgan
                      </span>
                    ) : (
                      <span className="text-xs text-ink-muted">—</span>
                    )}
                    {b.sha256 && (
                      // The hash outlives the file: it is the only way to check
                      // that the APK on somebody's laptop is the one we built.
                      <div
                        className="mt-1 font-mono text-[10px] text-ink-muted"
                        title={b.sha256}
                      >
                        {b.sha256.slice(0, 12)}…
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

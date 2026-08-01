"use client";

// The location permission panel used by both field apps (/staff and /kuryer).
//
// It replaces the old "ruxsat berilmagan" line, which was true and useless: a
// denied permission cannot be re-requested by the page, so a message with no
// instructions leaves the employee stuck holding a phone that will not clock
// them in.

import { useAdminT } from "@/lib/i18n/admin";
import { detectPlatform, type GeoFix, type GeoState } from "@/lib/geo";

interface Props {
  state: GeoState;
  checking: boolean;
  onRequest: () => Promise<GeoFix | null>;
  /** Shown when permission is fine — the caller's own status line. */
  children?: React.ReactNode;
  className?: string;
}

export default function GeoPermission({
  state,
  checking,
  onRequest,
  children,
  className = "",
}: Props) {
  const t = useAdminT();
  const g = t.geo;

  // Everything is in order: let the caller show whatever it wants.
  if (state === "granted") return <>{children}</>;

  const platform = detectPlatform();
  const steps =
    platform === "ios" ? g.stepsIos : platform === "android" ? g.stepsAndroid : g.stepsDesktop;

  // Not asked yet — a button, not an error. This is also what makes iOS work:
  // Safari only shows the prompt in response to a tap.
  if (state === "unknown" || state === "prompt") {
    return (
      <div
        className={`rounded-2xl border border-line-strong bg-ink/[0.03] p-3 ${className}`}
      >
        <p className="text-xs text-ink-muted">{g.needed}</p>
        <button
          type="button"
          disabled={checking}
          onClick={() => void onRequest()}
          className="btn-primary mt-2 w-full py-2.5 text-sm disabled:opacity-50"
        >
          {checking ? g.asking : g.allow}
        </button>
      </div>
    );
  }

  // The page cannot come back from this one — only the person can.
  const denied = state === "denied";
  const title =
    state === "insecure" ? g.insecureTitle : state === "unsupported" ? g.unsupported : g.deniedTitle;

  return (
    <div
      className={`rounded-2xl border border-amber-500/40 bg-amber-50 p-3 dark:bg-amber-500/10 ${className}`}
    >
      <p className="text-xs font-semibold text-amber-800 dark:text-amber-300">
        {title}
      </p>

      {state === "insecure" && (
        <p className="mt-1 text-xs text-amber-800/90 dark:text-amber-300/90">
          {g.insecureHint}
        </p>
      )}

      {denied && (
        <>
          <ol className="mt-2 list-decimal space-y-1 pl-4 text-xs text-amber-800/90 dark:text-amber-300/90">
            {steps.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ol>
          <p className="mt-2 text-[11px] text-amber-800/80 dark:text-amber-300/80">
            {g.afterAllow}
          </p>
          {/* Worth offering even though a denied permission normally fails
              instantly: the user may have just fixed it in settings, and on
              some browsers the stored answer is per-session. */}
          <button
            type="button"
            disabled={checking}
            onClick={() => void onRequest()}
            className="btn-ghost mt-2 w-full py-2 text-xs disabled:opacity-50"
          >
            {checking ? g.asking : g.retry}
          </button>
        </>
      )}
    </div>
  );
}

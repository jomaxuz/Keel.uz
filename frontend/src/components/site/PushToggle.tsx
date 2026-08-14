"use client";

// "Notify me about offers" — the guest's side of the push channel.
//
// ⚠️ **A button, never a prompt on page load.** A browser asks for this
// permission exactly once in its life: after a refusal the dialog never appears
// again and only the person can undo it, from browser settings. A prompt that
// fires while somebody is reading a menu is a prompt most people dismiss, and
// dismissing it spends the only chance the restaurant had. The same lesson the
// geolocation permission taught this codebase.
//
// It lives in the profile rather than on the menu for the same reason: this is
// a decision, and a decision belongs on the page somebody opened to make
// decisions.

import { useEffect, useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { pushState, subscribePush, unsubscribePush, type PushState } from "@/lib/push";

export default function PushToggle() {
  const { t } = useI18n();
  const [state, setState] = useState<PushState | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    pushState()
      .then(setState)
      // A browser that throws while being asked is a browser that cannot do
      // this. Reported as unsupported rather than left spinning.
      .catch(() => setState("unsupported"));
  }, []);

  // Nothing at all while the state is unknown, and nothing at all on a browser
  // that cannot do it. A permanently disabled control teaches people the site
  // is broken; an absent one teaches them nothing, which is correct here —
  // there is no action for them to take.
  if (state === null || state === "unsupported") return null;

  async function toggle() {
    setBusy(true);
    try {
      setState(state === "on" ? await unsubscribePush() : await subscribePush());
    } catch {
      setState("denied");
    } finally {
      setBusy(false);
    }
  }

  const p = t.push;

  return (
    <div className="card p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-semibold text-ink">{p.title}</p>
          <p className="mt-0.5 text-sm text-ink-soft">
            {state === "denied" ? p.deniedHint : p.hint}
          </p>
        </div>
        {state === "denied" ? (
          // ⚠️ No button. Pressing one would do nothing at all — the browser
          // resolves the request immediately with "denied" — and a button that
          // does nothing is worse than an instruction that works.
          <span className="text-xs font-semibold text-ink-muted">{p.denied}</span>
        ) : (
          <button
            type="button"
            onClick={toggle}
            disabled={busy}
            className={state === "on" ? "btn btn-ghost" : "btn btn-primary"}
          >
            {busy ? t.common.loading : state === "on" ? p.turnOff : p.turnOn}
          </button>
        )}
      </div>
    </div>
  );
}

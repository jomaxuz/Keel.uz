"use client";

// Branch setting: the kiosk screen that shows the rotating clock-in code.
//
// Two things live here because they belong together: the switch that makes the
// code mandatory, and the one-time link that turns a spare tablet into the
// screen showing it. Turning the switch on without setting up a screen would
// lock every employee out, so the link is one tap away from the switch.

import { useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import QrCode from "@/components/admin/QrCode";
import type { Branch } from "@/lib/types";

interface Props {
  branch: Branch;
  requireCode: boolean;
  onToggle: (value: boolean) => void;
}

export default function KioskSettings({ branch, requireCode, onToggle }: Props) {
  const t = useAdminT();
  const [link, setLink] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function issue(rotate: boolean) {
    if (rotate && !window.confirm(t.kiosk.rotateConfirm)) return;
    setBusy(true);
    setError(null);
    setNote(null);
    try {
      const res = await api.adminKioskToken(branch.id, rotate);
      setLink(`${window.location.origin}/kiosk?t=${encodeURIComponent(res.token)}`);
      if (rotate) setNote(t.kiosk.rotated);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function copy() {
    if (!link) return;
    try {
      await navigator.clipboard.writeText(link);
      setNote(t.kiosk.linkCopied);
    } catch {
      /* clipboard blocked — the link is on screen to copy by hand */
    }
  }

  return (
    <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
      <p className="text-sm font-semibold">{t.kiosk.title}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">{t.kiosk.hint}</p>

      <label className="mt-3 flex items-start gap-2 text-sm">
        <input
          type="checkbox"
          className="mt-0.5"
          checked={requireCode}
          onChange={(e) => onToggle(e.target.checked)}
        />
        <span>
          {t.kiosk.requireCode}
          <span className="mt-0.5 block text-xs text-ink-muted">
            {t.kiosk.requireCodeHint}
          </span>
        </span>
      </label>

      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void issue(false)}
          className="btn-ghost px-3 py-1.5 text-xs disabled:opacity-50"
        >
          {t.kiosk.openScreen}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => void issue(true)}
          className="px-3 py-1.5 text-xs text-red-600 hover:underline disabled:opacity-50"
          title={t.kiosk.rotateHint}
        >
          {t.kiosk.rotate}
        </button>
      </div>

      {link && (
        <div className="mt-3 rounded-xl border border-line bg-surface p-3">
          {/* Both ways in: scan it with the tablet, or paste the link there. */}
          <div className="flex flex-wrap items-center gap-3">
            <QrCode value={link} size={120} />
            <div className="min-w-0 flex-1">
              <p className="text-xs text-ink-muted">{t.kiosk.linkHint}</p>
              <p className="mt-1 break-all font-mono text-[11px] text-ink-soft">
                {link}
              </p>
              <div className="mt-2 flex gap-2">
                <button
                  type="button"
                  onClick={() => void copy()}
                  className="btn-ghost px-3 py-1 text-xs"
                >
                  {t.kiosk.copyLink}
                </button>
                <a
                  href={link}
                  target="_blank"
                  rel="noreferrer"
                  className="btn-ghost px-3 py-1 text-xs"
                >
                  {t.kiosk.openScreen}
                </a>
              </div>
            </div>
          </div>
        </div>
      )}

      {note && <p className="mt-2 text-xs text-emerald-600">{note}</p>}
      {error && <p className="mt-2 text-xs text-red-600">{error}</p>}
    </div>
  );
}

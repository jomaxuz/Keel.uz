"use client";

// Branch setting: binding a monoblock to this branch.
//
// ⚠️ **A till is set up once and then never signed into.** Nobody types a
// username and a password on a monoblock between two guests, and asking them to
// is how a restaurant ends up with one login shared by everybody and written on
// a note beside the screen. So the machine gets a long-lived branch token from
// this link, and from then on the only thing anybody enters is four digits.
//
// The same shape as the kiosk link next door, and revoked the same way.

import { useState } from "react";
import { ApiError, api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import QrCode from "@/components/admin/QrCode";
import type { Branch } from "@/lib/types";

export default function TillDeviceSettings({ branch }: { branch: Branch }) {
  const t = useAdminT();
  const [link, setLink] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function issue(rotate: boolean) {
    // ⚠️ Rotating kills **every** till on this branch, not just a lost one:
    // the tokens carry no device identity, so there is nothing finer to
    // revoke. Said before it happens, because the fix is walking to each
    // monoblock with a new link.
    if (rotate && !window.confirm(t.tillDevice.rotateConfirm)) return;
    setBusy(true);
    setError(null);
    setNote(null);
    try {
      const res = await api.tillDeviceToken(branch.id, rotate);
      setLink(
        `${window.location.origin}/kassa?t=${encodeURIComponent(res.token)}`,
      );
      if (rotate) setNote(t.tillDevice.rotated);
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
      setNote(t.tillDevice.linkCopied);
    } catch {
      /* clipboard blocked — the link is on screen to copy by hand */
    }
  }

  return (
    <div className="mt-4 rounded-2xl border border-line bg-ink/[0.02] p-4">
      <p className="text-sm font-semibold">{t.tillDevice.title}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">
        {t.tillDevice.hint}
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="btn"
          disabled={busy}
          onClick={() => void issue(false)}
        >
          {t.tillDevice.getLink}
        </button>
        <button
          type="button"
          className="btn-ghost px-3 text-sm"
          disabled={busy}
          onClick={() => void issue(true)}
        >
          {t.tillDevice.rotate}
        </button>
      </div>

      {link && (
        <div className="mt-3 rounded-xl border border-line bg-surface p-3">
          {/* ⚠️ Said before the link, not after: it is shown once and a token
              that was not saved means opening this panel again. */}
          <p className="text-xs font-medium">{t.tillDevice.once}</p>
          <code className="mt-2 block break-all font-mono text-xs">{link}</code>
          <div className="mt-3 flex items-start gap-4">
            {/* The monoblock usually has no keyboard worth typing a token on,
                and the panel is usually open on a different machine — so the
                link travels by camera, exactly as the kiosk one does. */}
            <QrCode value={link} size={132} />
            <div className="text-xs text-ink-muted">
              <p>{t.tillDevice.scan}</p>
              <button
                type="button"
                className="btn mt-2"
                onClick={() => void copy()}
              >
                {t.tillDevice.copy}
              </button>
            </div>
          </div>
        </div>
      )}

      {note && <p className="mt-2 text-xs text-success">{note}</p>}
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
    </div>
  );
}

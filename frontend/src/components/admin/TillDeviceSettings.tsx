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

import { useCallback, useEffect, useState } from "react";
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
  const [devices, setDevices] = useState<
    { id: string; name: string; lastSeenAt?: string; issuedBy?: string }[]
  >([]);
  const [limit, setLimit] = useState(0);

  // ⚠️ **The list is reloaded after every issue and every removal**, because
  // the count it shows is the thing the plan is sold by — a stale "2 / 2" beside
  // a button that has just freed a slot reads as the removal not working.
  const loadDevices = useCallback(async () => {
    try {
      const res = await api.tillDevices(branch.id);
      setDevices(res.devices);
      setLimit(res.limit);
    } catch {
      // ⚠️ Swallowed: this is a list beside the button that matters, and an
      // error here must not stop somebody pairing a till.
    }
  }, [branch.id]);

  useEffect(() => {
    void loadDevices();
  }, [loadDevices]);

  async function remove(id: string, name: string) {
    if (!window.confirm(t.tillDevice.removeConfirm(name))) return;
    setBusy(true);
    setError(null);
    try {
      await api.removeTillDevice(branch.id, id);
      await loadDevices();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function issue(rotate: boolean) {
    // ⚠️ Rotating kills **every** till on this branch, not just a lost one.
    // That is now deliberate rather than forced: a single screen can be
    // unbound from the list below, which is what a replaced tablet needs.
    // This one is for a theft — where the question is "none of them, right
    // now" and nobody should have to work out which row the machine was.
    // Said before it happens, because the fix is walking to each of the
    // others with a new link.
    if (rotate && !window.confirm(t.tillDevice.rotateConfirm)) return;
    setBusy(true);
    setError(null);
    setNote(null);
    try {
      const res = await api.tillDeviceToken(branch.id, rotate);
      await loadDevices();
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

      {/* ⚠️ **What is actually bound, and how many the plan allows.**
          The cap was a sentence in a price list until there was a row per
          machine; shown here so a manager meets it as a list they can act on
          rather than as a refusal they can only ring us about. */}
      {devices.length > 0 && (
        <div className="mt-3">
          <p className="text-xs font-medium text-ink-soft">
            {t.tillDevice.bound}
            {limit > 0 ? ` — ${devices.length} / ${limit}` : ""}
          </p>
          <ul className="mt-2 divide-y divide-line rounded-xl border border-line bg-surface">
            {devices.map((d) => (
              <li
                key={d.id}
                className="flex items-center justify-between gap-3 px-3 py-2 text-xs"
              >
                <span className="min-w-0">
                  <span className="block truncate font-medium">{d.name}</span>
                  {/* ⚠️ "Never used" is the most useful thing this row can say:
                      it is how a manager at their cap tells an abandoned link
                      from the machine somebody is selling on right now. */}
                  <span className="block text-ink-muted">
                    {d.lastSeenAt
                      ? t.tillDevice.lastSeen(d.lastSeenAt.slice(0, 10))
                      : t.tillDevice.neverUsed}
                    {d.issuedBy ? ` · ${d.issuedBy}` : ""}
                  </span>
                </span>
                <button
                  type="button"
                  className="btn-ghost shrink-0 px-2 py-1 text-xs text-danger"
                  disabled={busy}
                  onClick={() => void remove(d.id, d.name)}
                >
                  {t.tillDevice.remove}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {note && <p className="mt-2 text-xs text-success">{note}</p>}
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
    </div>
  );
}

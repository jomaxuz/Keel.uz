"use client";

import { useT } from "@/lib/i18n/client";
import { Field } from "./dash";

/** The first account of the tenant's own panel.
 *
 *  Set here because it has to exist before the container does: the tenant
 *  server seeds its first owner from these on first boot and never asks again.
 *  On the edit form an empty password means "keep the stored one" — the form
 *  cannot show a password it never received, so a blank field must not be read
 *  as "erase it". */
export default function AdminCredentials({
  username,
  password,
  stored,
  onUsername,
  onPassword,
}: {
  username: string;
  password: string;
  /** Edit mode: a password already exists, so this one may be left blank. */
  stored?: boolean;
  onUsername: (v: string) => void;
  onPassword: (v: string) => void;
}) {
  const { t } = useT();

  return (
    <div className="rounded-2xl border border-line bg-raised p-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label={t.dash.adminUsername}
          value={username}
          onChange={(v) => onUsername(v.toLowerCase().replace(/[^a-z0-9._-]/g, ""))}
        />
        <div>
          <label className="text-sm font-medium">
            {t.dash.adminPassword}{" "}
            {stored && (
              <span className="text-xs font-normal text-emerald-700 dark:text-emerald-300">
                · {t.dash.adminStored}
              </span>
            )}
          </label>
          <div className="mt-1 flex gap-2">
            {/* Shown in the clear, on purpose: the operator has to read it out
                to the customer, and a masked field they cannot check is how a
                mistyped password becomes a support call on day one. */}
            <input
              className="input"
              value={password}
              placeholder={stored ? t.dash.adminKeep : ""}
              autoComplete="off"
              onChange={(e) => onPassword(e.target.value)}
            />
            <button
              type="button"
              onClick={() => onPassword(makePassword())}
              className="btn-ghost shrink-0 px-3 py-2 text-xs"
            >
              {t.dash.generate}
            </button>
          </div>
        </div>
      </div>
      <p className="mt-3 text-xs text-ink-muted">{t.dash.adminHint}</p>
    </div>
  );
}

/** A password that survives being read down a phone line.
 *
 *  No l/1/I or O/0, because the first thing that happens to it is somebody
 *  dictating it to a shop owner. */
function makePassword(): string {
  const alphabet = "abcdefghjkmnpqrstuvwxyz23456789";
  const bytes = new Uint32Array(12);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (n) => alphabet[n % alphabet.length]).join("");
}

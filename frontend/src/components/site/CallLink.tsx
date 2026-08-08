"use client";

// A phone number a guest can actually call, including inside Telegram.
//
// ⚠️ **`tel:` links do not reliably work in a mini app**, and this is the shape of
// the problem: Telegram renders the site in a WebView that blocks `tel:`
// navigation outright on iOS and inconsistently on Android, and Telegram's own
// `openLink` accepts only `http`/`https` — there is no "open the dialer" API. So
// the courier's number was a button that visibly did nothing, on the one screen
// where a guest is trying to reach the person holding their food.
//
// What this does instead, in order:
//
//  1. **Copies the number**, always, in the mini app. A number on the clipboard
//     can be pasted into the dialer; a button that did nothing cannot.
//  2. **Still attempts the dialer.** Some Telegram clients do honour `tel:`, and
//     on those the guest gets what they expected. Attempting it after the copy
//     costs nothing and the copy is not lost if it works.
//  3. **Says which happened** — the number itself, echoed back. Silence after a
//     tap is the failure being reported here.
//
// On the open web nothing changes: it is a plain `<a href="tel:">`, which is what
// a phone already handles correctly.

import { useState } from "react";
import { useI18n } from "@/lib/i18n/client";
import { useTelegram } from "@/lib/telegram";

export default function CallLink({
  phone,
  className = "",
  children,
}: {
  phone: string;
  className?: string;
  children?: React.ReactNode;
}) {
  const { inTelegram } = useTelegram();
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);

  // Digits only for the dialer; the label keeps whatever the restaurant typed,
  // because that is the form they recognise on their own receipts.
  const dial = "+" + phone.replace(/\D/g, "");

  async function callInTelegram(e: React.MouseEvent) {
    e.preventDefault();
    try {
      await navigator.clipboard?.writeText(dial);
      setCopied(true);
      // Long enough to read, short enough not to become part of the layout.
      window.setTimeout(() => setCopied(false), 2500);
    } catch {
      // A refused clipboard is not worth an error message: the number is on
      // screen either way, and the dialer attempt below may well succeed.
    }
    // Tried after the copy, deliberately. On a client that honours it the dialer
    // opens and this component's work is invisible — which is the best outcome.
    try {
      window.location.href = `tel:${dial}`;
    } catch {
      /* blocked: the clipboard is the answer */
    }
  }

  if (!inTelegram) {
    return (
      <a href={`tel:${dial}`} className={className}>
        {children ?? phone}
      </a>
    );
  }

  return (
    <span className="inline-flex flex-wrap items-baseline gap-2">
      <a href={`tel:${dial}`} onClick={callInTelegram} className={className}>
        {children ?? phone}
      </a>
      {copied && (
        <span className="text-xs font-semibold text-brand">
          {t.common.copied}
        </span>
      )}
    </span>
  );
}

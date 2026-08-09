"use client";

// "Write to us" on the contact page: a rating, and words if they have any.
//
// ⚠️ **Stars first, comment optional.** A form that only takes text is answered by the
// few people who type; a row of stars is answered by everybody, and the star is the half
// the restaurant can count. The comment is what makes a complaint answerable, so it is
// asked for — just not as the price of answering at all.
//
// ⚠️ **Signed in only, and said so before the form is filled in.** The rule protects the
// restaurant rather than the table: an open form on a public page fills with rubbish
// within a week, and the owner then stops opening the one place complaints arrive.
// Discovering that after typing a paragraph is how somebody leaves without sending it.

import { useState } from "react";
import LocaleLink from "@/components/site/LocaleLink";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { useUser } from "@/lib/user";

export default function ContactFeedback() {
  const { t } = useI18n();
  const { user, loading } = useUser();
  const [rating, setRating] = useState(0);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function send() {
    if (rating < 1) return;
    setBusy(true);
    setError("");
    try {
      await api.submitSiteFeedback({ rating, comment });
      setDone(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  if (done) {
    return (
      <div className="card p-6 sm:p-8">
        <h2 className="font-display text-lg font-bold">{t.contact.title}</h2>
        <p className="mt-3 text-sm text-ink-soft">{t.contact.thanks}</p>
      </div>
    );
  }

  return (
    <div className="card p-6 sm:p-8">
      <h2 className="font-display text-lg font-bold">{t.contact.title}</h2>
      <p className="mt-1 text-sm text-ink-muted">{t.contact.hint}</p>

      {!loading && !user ? (
        <div className="mt-4">
          <p className="text-sm text-ink-soft">{t.contact.needLogin}</p>
          <LocaleLink
            href="/login?next=/about&reason=feedback"
            className="btn btn-primary mt-3 px-5 py-2.5"
          >
            {t.nav.login}
          </LocaleLink>
        </div>
      ) : (
        <>
          <div className="mt-4 flex items-center gap-1.5">
            {[1, 2, 3, 4, 5].map((n) => (
              <button
                key={n}
                type="button"
                onClick={() => setRating(n)}
                aria-label={`${n}`}
                className="p-1"
              >
                <svg
                  viewBox="0 0 24 24"
                  className={`h-8 w-8 ${n <= rating ? "text-brand" : "text-ink-muted"}`}
                  fill={n <= rating ? "currentColor" : "none"}
                  stroke="currentColor"
                  strokeWidth="1.5"
                  aria-hidden
                >
                  <path d="M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 18l-5.8 3 1.1-6.5L2.6 9.8l6.5-.9L12 3Z" />
                </svg>
              </button>
            ))}
          </div>

          <textarea
            className="input mt-3 min-h-24"
            maxLength={1000}
            value={comment}
            placeholder={t.contact.commentPh}
            onChange={(e) => setComment(e.target.value)}
          />

          {error && <p className="mt-2 text-sm text-brand">{error}</p>}

          <button
            type="button"
            onClick={() => void send()}
            disabled={busy || rating < 1}
            className="btn btn-primary mt-3 px-5 py-2.5 disabled:opacity-40"
          >
            {busy ? t.common.saving : t.contact.send}
          </button>
        </>
      )}
    </div>
  );
}

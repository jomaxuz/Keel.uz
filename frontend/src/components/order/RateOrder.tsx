"use client";

// "How was it?" — asked once, on the page the guest is already looking at.
//
// Not by e-mail the next day, which is when people stop answering, and not
// before the food arrives, which would rate nothing.
//
// ⚠️ The comment box used to appear **only for a low rating**, and a five-star
// guest was sent off after one tap with no way to say anything. That threw away
// the reviews a business can actually use — "the courier was lovely", "the
// packaging held up" — and, worse, it meant the only written feedback the owner
// ever saw was complaints. Now everybody gets the box; the label changes, and
// it stays optional, so a happy guest can still finish without typing.

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";

// At or below this, the restaurant treats it as a complaint.
const LOW = 3;

export default function RateOrder({ number }: { number: string }) {
  const { t } = useI18n();
  const [rated, setRated] = useState<number | null>(null);
  const [picked, setPicked] = useState(0);
  const [comment, setComment] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    api
      .orderFeedback(number)
      .then((f) => setRated(f.rated ? (f.rating ?? 0) : null))
      .catch(() => setRated(null))
      .finally(() => setLoaded(true));
  }, [number]);

  async function send(rating: number, withComment: string) {
    setSending(true);
    setError(null);
    try {
      await api.submitFeedback(number, rating, withComment);
      setRated(rating);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.refresh);
    } finally {
      setSending(false);
    }
  }

  if (!loaded) return null;

  if (rated !== null) {
    return (
      <div className="mt-6 rounded-2xl border border-line bg-surface p-5 text-center">
        <p className="text-sm text-ink-muted">{t.rate.thanks}</p>
        <p className="mt-1 text-2xl" aria-label={`${rated}/5`}>
          {"★".repeat(rated)}
          <span className="text-ink-muted/40">{"★".repeat(5 - rated)}</span>
        </p>
      </div>
    );
  }

  return (
    <div className="mt-6 rounded-2xl border border-line bg-surface p-5">
      <p className="text-center font-semibold">{t.rate.question}</p>

      <div className="mt-3 flex justify-center gap-1">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            disabled={sending}
            onClick={() => setPicked(n)}
            aria-label={`${n}/5`}
            className={`px-1.5 text-3xl transition-transform hover:scale-110 disabled:opacity-50 ${
              n <= picked ? "text-brand" : "text-ink-muted/30"
            }`}
          >
            ★
          </button>
        ))}
      </div>

      {/* Shown for every rating, not just a complaint. The question is
          different at each end — one is asking what to fix, the other what to
          keep doing — but a guest who wants to say something must never be
          told they cannot. */}
      {picked > 0 && (
        <div className="mt-4">
          <label className="block text-sm">
            <span className="font-medium">
              {picked <= LOW ? t.rate.whatWentWrong : t.rate.whatWentWell}
            </span>
            <textarea
              className="input mt-1 w-full"
              rows={3}
              maxLength={1000}
              placeholder={picked <= LOW ? t.rate.commentPh : t.rate.commentPhGood}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
            />
          </label>
          {/* Optional, and said so: a happy guest should still be able to
              finish without typing anything. */}
          <p className="mt-1 text-xs text-ink-muted">{t.rate.optional}</p>
          <button
            type="button"
            disabled={sending}
            onClick={() => void send(picked, comment)}
            className="btn-primary mt-3 w-full px-5 py-2.5 disabled:opacity-60"
          >
            {sending ? t.common.loading : t.rate.send}
          </button>
        </div>
      )}

      {error && (
        <p className="mt-3 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
          {error}
        </p>
      )}
    </div>
  );
}

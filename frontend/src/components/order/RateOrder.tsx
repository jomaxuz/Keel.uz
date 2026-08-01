"use client";

// "How was it?" — asked once, on the page the guest is already looking at.
//
// Not by e-mail the next day, which is when people stop answering, and not
// before the food arrives, which would rate nothing. The comment box only
// appears once a low rating is picked: a happy guest should be able to finish
// in one tap, while an unhappy one is exactly who you want to hear from.

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
            onClick={() => {
              setPicked(n);
              // A good rating needs nothing more said: send it and be done.
              if (n > LOW) void send(n, "");
            }}
            aria-label={`${n}/5`}
            className={`px-1.5 text-3xl transition-transform hover:scale-110 disabled:opacity-50 ${
              n <= picked ? "text-brand" : "text-ink-muted/30"
            }`}
          >
            ★
          </button>
        ))}
      </div>

      {/* Only for a complaint — and it is the reason the whole thing exists. */}
      {picked > 0 && picked <= LOW && (
        <div className="mt-4">
          <label className="block text-sm">
            <span className="font-medium">{t.rate.whatWentWrong}</span>
            <textarea
              className="input mt-1 w-full"
              rows={3}
              maxLength={1000}
              placeholder={t.rate.commentPh}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
            />
          </label>
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

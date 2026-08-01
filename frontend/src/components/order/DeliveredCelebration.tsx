"use client";

import { useMemo } from "react";
import { useI18n } from "@/lib/i18n/client";

const COLORS = ["#e11d48", "#f59e0b", "#10b981", "#3b82f6", "#8b5cf6", "#ec4899"];

// Full-screen confetti + a bouncing checkmark shown once an order is delivered.
// Pure CSS (keyframes in globals.css) — no external library.
export default function DeliveredCelebration() {
  const { t } = useI18n();
  // Random confetti pieces generated once on the client (this component only
  // renders after a client-side fetch, so randomness is hydration-safe).
  const pieces = useMemo(
    () =>
      Array.from({ length: 60 }, (_, i) => ({
        id: i,
        left: Math.random() * 100,
        delay: Math.random() * 2,
        duration: 2.5 + Math.random() * 2,
        color: COLORS[i % COLORS.length],
        size: 6 + Math.random() * 6,
      })),
    [],
  );

  return (
    <>
      {/* Confetti overlay */}
      <div className="pointer-events-none fixed inset-0 z-30 overflow-hidden">
        {pieces.map((p) => (
          <span
            key={p.id}
            style={{
              position: "absolute",
              left: `${p.left}%`,
              top: "-5vh",
              width: p.size,
              height: p.size * 0.4,
              background: p.color,
              borderRadius: 2,
              animation: `confetti-fall ${p.duration}s linear ${p.delay}s infinite`,
            }}
          />
        ))}
      </div>

      {/* Checkmark badge */}
      <div className="animate-pop-in flex flex-col items-center py-4">
        <div className="animate-bob flex h-24 w-24 items-center justify-center rounded-full bg-emerald-500 text-5xl text-white shadow-lg shadow-emerald-500/30">
          ✓
        </div>
        <p className="mt-4 font-display text-2xl font-bold text-emerald-600">
          {t.order.deliveredTitle}
        </p>
        <p className="mt-1 text-sm text-ink-muted">{t.order.deliveredText}</p>
      </div>
    </>
  );
}

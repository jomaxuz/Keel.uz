"use client";

// The root layout itself threw. No provider, no dictionary, no theme script —
// see the tenant app's copy of this file for the full reasoning; it applies
// here unchanged.
//
// ⚠️ All three languages at once, on purpose: this is the one screen that cannot
// know what the visitor reads, and guessing would leave two thirds of them with
// gibberish on top of a broken page.

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="uz">
      <body
        style={{
          margin: 0,
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "#f7f5f1",
          color: "#20201e",
          fontFamily: "system-ui, -apple-system, Segoe UI, Roboto, sans-serif",
          padding: "24px",
        }}
      >
        <div style={{ maxWidth: 420, textAlign: "center" }}>
          <svg
            viewBox="0 0 200 140"
            width="180"
            height="126"
            fill="none"
            stroke="#20201e"
            strokeOpacity="0.45"
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden
          >
            <path d="M20 70h34v22a14 14 0 0 0 14 14h6" />
            <path d="M20 58v24" />
            <path d="M180 70h-34V48a14 14 0 0 0-14-14h-6" />
            <path d="M180 58v24" />
            <path d="M88 70h8M104 70h8" stroke="#2563eb" strokeOpacity="0.85" />
          </svg>

          <p style={{ margin: "20px 0 0", fontSize: 17, fontWeight: 700 }}>
            Saytda xatolik yuz berdi
          </p>
          <p style={{ margin: "6px 0 0", fontSize: 15, opacity: 0.7 }}>
            На сайте произошла ошибка
          </p>
          <p style={{ margin: "6px 0 0", fontSize: 15, opacity: 0.7 }}>
            Something went wrong
          </p>
          {/* ⚠️ Said here and nowhere else on this page: whoever is reading it
              is most likely a customer wondering whether their own restaurant is
              down too. It is not — their site is a different container. */}
          <p style={{ margin: "14px 0 0", fontSize: 13, opacity: 0.55 }}>
            Mijozlar saytlari mustaqil ishlaydi · Сайты клиентов работают
            независимо · Customer sites are unaffected
          </p>

          <button
            type="button"
            onClick={reset}
            style={{
              marginTop: 24,
              padding: "12px 24px",
              borderRadius: 12,
              border: "none",
              background: "#20201e",
              color: "#fff",
              fontSize: 15,
              fontWeight: 600,
              cursor: "pointer",
            }}
          >
            Qayta urinish · Повторить · Retry
          </button>

          {error.digest && (
            <p
              style={{
                marginTop: 20,
                fontSize: 12,
                opacity: 0.5,
                fontFamily: "ui-monospace, monospace",
                userSelect: "all",
              }}
            >
              {error.digest}
            </p>
          )}
        </div>
      </body>
    </html>
  );
}

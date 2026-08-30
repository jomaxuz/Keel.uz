"use client";

import { useEffect } from "react";

// The last resort: the root layout itself threw, so React unmounts everything
// including the providers and this file has to render `<html>` and `<body>`.
//
// ⚠️ **Nothing in here may depend on anything.** No dictionary (the language
// provider is inside the layout that just failed), no theme (its script runs in
// that layout's head), no design tokens — a page that referenced `bg-surface`
// would be invisible against an unstyled white page in exactly the situation it
// exists for. So: inline styles, one system font, and no imports beyond React.
//
// ⚠️ **All three languages at once**, which is wrong-looking on purpose. Every
// other screen knows what the visitor reads; this one cannot, and guessing would
// leave two thirds of guests with a page that looks like gibberish on top of
// being broken. Three short lines cost nothing and are read by everybody.

// ⚠️ **Reported by hand, with no import.** Every other screen calls
// `lib/report.ts`; this one cannot, for the reason above — that module imports
// the API client, which imports the token store, and this page exists precisely
// for the case where the module graph is not healthy. A page that crashed while
// reporting a crash reports nothing at all.
//
// So the ten lines are duplicated on purpose, and the duplication is the point:
// nothing here depends on anything. The address is the same-origin API, which
// the edge routes to this restaurant's own server — a native app never renders
// this page, so there is no runtime override to consult.
function reportRootCrash(error: Error & { digest?: string }) {
  try {
    const body = JSON.stringify({
      reports: [
        {
          app: "site",
          message: error?.message || String(error) || "root layout crashed",
          stack: error?.stack?.slice(0, 4000),
          where: "root-layout",
          context: error?.digest,
          platform: navigator.userAgent?.slice(0, 120),
          at: new Date().toISOString(),
        },
      ],
    });
    if (navigator.sendBeacon?.("/api/v1/report", body)) return;
    void fetch("/api/v1/report", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
      keepalive: true,
    }).catch(() => {});
  } catch {
    // Nothing. This is the last screen there is.
  }
}

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  // ⚠️ In an effect, not in the render body. This component renders on the
  // server too — where `navigator` does not exist — and React may render it
  // twice, which would file the same crash twice.
  useEffect(() => {
    reportRootCrash(error);
  }, [error]);

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
          {/* The same boiled-over pot as the ordinary error page, inlined:
              importing the component would import the module graph this page
              cannot assume is healthy. */}
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
            <path d="M56 66h88v34a14 14 0 0 1-14 14H70a14 14 0 0 1-14-14z" />
            <path d="M48 66h104" />
            <path d="M56 78H44M144 78h12" />
            <path d="M74 60l58-10" stroke="#e2483d" strokeOpacity="0.8" />
            <path d="M76 66c2-8 8-8 10-2M104 66c2-10 10-9 12-1" />
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

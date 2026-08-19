import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";

// ⚠️ Imported from the shared tree on purpose: this one import is what proves
// the alias, the TypeScript paths and the bundle all reach `frontend/src`. If
// it ever stops resolving, the build fails here rather than three screens in.
import { formatPrice } from "@/lib/format";

import TitleBar from "./TitleBar";
import { bridge, inWails } from "./bridge";

// ⚠️ **The escape hatch pos-reja.md §2 asks for, with an honest limit.**
// Ctrl+Shift+Q quits from the keyboard, which covers a screen that has lost its
// buttons — a mis-drawn layout, a dialog with nowhere to click. It does **not**
// cover a wedged renderer: this handler is JavaScript, and a webview that has
// stopped running JavaScript will not see the keys either. That case is what
// the window being merely maximised, rather than truly fullscreen, is for —
// Alt+Tab and Win+D still work, so Windows can close a program we cannot.
function useQuitHotkey() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "q") {
        e.preventDefault();
        void bridge()?.Quit();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}

function Boot() {
  useQuitHotkey();
  const [env, setEnv] = useState<Record<string, string>>({});
  useEffect(() => {
    void bridge()?.Env().then(setEnv);
  }, []);

  return (
    <div style={{ font: "16px system-ui", minHeight: "100vh", background: "#fafaf9" }}>
      <TitleBar />
      <main style={{ padding: 32 }}>
        <h1>Keel Kassa</h1>
        <p>Umumiy kod ulandi: {formatPrice(1234500, "UZS", "uz")}</p>
        <p>
          Muhit: {inWails() ? "Wails" : "brauzer"}
          {env.platform ? ` · ${env.platform}/${env.arch}` : ""}
          {env.agent ? ` · agent: ${env.agent}` : ""}
        </p>
        {env.agent === "off" && (
          <p style={{ color: "#b45309" }}>
            till.json to‘ldirilmagan — agent ishlamayapti, chek chiqmaydi.
          </p>
        )}
      </main>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Boot />
  </StrictMode>,
);

import { StrictMode, useCallback, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";

// ⚠️ Imported from the shared tree on purpose: this one import is what proves
// the alias, the TypeScript paths and the bundle all reach `frontend/src`. If
// it ever stops resolving, the build fails here rather than three screens in.
import { formatPrice } from "@/lib/format";

import TitleBar from "./TitleBar";
import Setup from "./Setup";
import { bridge, inWails, type Status } from "./bridge";

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

function App() {
  useQuitHotkey();
  const [status, setStatus] = useState<Status | null>(null);

  const refresh = useCallback(() => {
    const b = bridge();
    if (!b) {
      // An ordinary browser: nothing is paired and nothing can be, so show the
      // till rather than a setup screen that could not finish.
      setStatus({
        paired: true,
        branchName: "",
        server: "",
        agent: false,
        platform: "browser",
        configPath: "",
      });
      return;
    }
    void b.Status().then(setStatus);
  }, []);

  useEffect(refresh, [refresh]);

  if (!status) return null; // one frame, not worth a spinner

  return (
    <div style={{ font: "16px system-ui", minHeight: "100vh", background: "#fafaf9" }}>
      <TitleBar />
      {status.paired ? <Till status={status} /> : <Setup onPaired={refresh} />}
    </div>
  );
}

function Till({ status }: { status: Status }) {
  return (
    <main style={{ padding: 32 }}>
      <h1>Keel Kassa</h1>
      <p>Umumiy kod ulandi: {formatPrice(1234500, "UZS", "uz")}</p>
      <p>
        Muhit: {inWails() ? "Wails" : "brauzer"} · {status.platform}
        {status.branchName && ` · ${status.branchName}`}
      </p>
      {inWails() && !status.agent && (
        // ⚠️ Said out loud rather than left to be discovered. A till whose relay
        // is not running looks completely normal until the first receipt fails
        // to come out, and by then somebody is standing at the counter.
        <p style={{ color: "#b45309" }}>
          Agent ishlamayapti — chek chiqmaydi. Log: {status.configPath}
        </p>
      )}
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

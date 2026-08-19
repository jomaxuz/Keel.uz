import { StrictMode, useCallback, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";

import "./app.css";

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
    // ⚠️ `till` is the scope the whole POS palette lives in (globals.css). The
    // site's tokens follow restaurant.theme; these deliberately do not — two
    // branches of one chain must not have differently coloured tills.
    //
    // ⚠️ The column is fixed height and the work area is what scrolls. A page
    // that scrolls as a whole puts a scrollbar down the side of a monoblock the
    // moment a title bar sits above a full-height screen.
    <div className="till flex h-full flex-col overflow-hidden bg-[rgb(var(--bg))]">
      <TitleBar />
      <div className="min-h-0 flex-1 overflow-y-auto">
        {status.paired ? <Till status={status} /> : <Setup onPaired={refresh} />}
      </div>
    </div>
  );
}

function Till({ status }: { status: Status }) {
  return (
    <main className="p-8">
      <h1 className="font-poppins text-2xl font-semibold tracking-tight">Keel Kassa</h1>
      <p className="mt-2 text-ink-soft">
        Umumiy kod ulandi: <span className="till-num">{formatPrice(1234500, "UZS", "uz")}</span>
      </p>
      <p className="mt-1 text-sm text-ink-muted">
        {inWails() ? "Wails" : "brauzer"} · {status.platform}
        {status.branchName && ` · ${status.branchName}`}
      </p>
      {inWails() && !status.agent && (
        // ⚠️ Said out loud rather than left to be discovered. A till whose relay
        // is not running looks completely normal until the first receipt fails
        // to come out, and by then somebody is standing at the counter.
        <p className="mt-4 text-sm text-danger">
          Agent ishlamayapti — chek chiqmaydi.
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

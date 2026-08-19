import { StrictMode, useCallback, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";

import "./app.css";

// ⚠️ Imported from the shared tree on purpose: this one import is what proves
// the alias, the TypeScript paths and the bundle all reach `frontend/src`. If
// it ever stops resolving, the build fails here rather than three screens in.
import { setTillDeviceToken } from "@/lib/api";
import { LangProvider } from "@/lib/i18n/client";
import { StaffProvider } from "@/lib/staff";
import KassaScreen from "@/app/kassa/page";

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
    // ⚠️ Seeded before the screen renders, not alongside it. Every till call
    // picks its credential at request time (lib/api.ts, tillAuth), so a screen
    // mounted first would fire its opening requests unauthenticated and show a
    // login it cannot complete.
    void (async () => {
      const s = await b.Status();
      if (s.paired) setTillDeviceToken(await b.DeviceToken());
      setStatus(s);
    })();
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
        {/* ⚠️ The language provider wraps both screens, because the setup screen
            is the one place a till is used before anybody has chosen one. Uzbek
            is the base language everywhere else in the codebase; the switcher
            belongs on the till's own screen, not here. */}
        <LangProvider initial="uz">
          {status.paired ? <Till status={status} /> : <Setup onPaired={refresh} />}
        </LangProvider>
      </div>
    </div>
  );
}

// The till itself: the screens the browser till already runs.
//
// ⚠️ **The providers are kassa/layout.tsx's, reproduced rather than imported.**
// A Next layout is a route convention — it takes no props, is composed by the
// router and carries `metadata` and `viewport` exports that mean nothing here.
// What it actually contributes is two providers and a background, and those are
// what is repeated. If a third appears there, it has to be added here too: that
// is the seam this shell has, and it is written down rather than discovered.
function Till({ status }: { status: Status }) {
  return (
    <StaffProvider>
      <KassaScreen />
      {/* The relay is what turns a sale into paper. Silence about it is what
          makes "the printer is broken" the first theory. */}
      {!status.agent && (
        <p className="fixed bottom-3 left-1/2 -translate-x-1/2 rounded-full border border-line bg-surface px-3 py-1.5 text-xs text-danger shadow-card">
          Agent ishlamayapti — chek chiqmaydi
        </p>
      )}
    </StaffProvider>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

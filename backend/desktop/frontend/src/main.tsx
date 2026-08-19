import { StrictMode, useCallback, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";

// ⚠️ **Order matters and is why these are here rather than @import-ed.** The
// shared design system first, this app's rules second, so the second can
// override. They are JavaScript imports because Vite's CSS resolver could not
// follow a relative path out of this project on Windows (wails build failed on
// exactly that); the module resolver knows the `@` alias and is what already
// finds the till screens.
import "@/app/globals.css";
import "@fontsource/poppins/latin-500.css";
import "@fontsource/poppins/latin-600.css";
import "./app.css";

import { setTillDeviceToken } from "@/lib/api";
import { LangProvider } from "@/lib/i18n/client";
import { StaffProvider } from "@/lib/staff";
import KassaScreen from "@/app/kassa/page";

import Setup from "./Setup";
import { bridge, type Status } from "./bridge";

// ⚠️ **There is no title bar, and the window has no frame.** The till draws its
// own header — with the padlock that locks the screen — and a second bar above
// it was both a duplicate and 36px taller than the window, which is what put a
// scrollbar down the side of a screen with nothing below the fold. The till's
// root is already `h-dvh` and manages its own scrolling; anything wrapped
// around it is in the way.
//
// ⚠️ **So the ways out are Ctrl+Shift+Q and Alt+F4**, and they are not the same
// thing. Alt+F4 is Windows closing the window and works even if this program
// has stopped listening; the hotkey below is JavaScript, so it cannot save a
// wedged webview — it covers the ordinary case of a screen that has lost its
// controls. Both belong in the install notes: a monoblock with no visible way
// to close an application gets closed by its power button.
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

  // ⚠️ No wrapper element. Both screens are full-window layouts that carry the
  // `till` palette scope themselves; a container around them can only add
  // height the window does not have.
  return (
    <LangProvider initial="uz">
      {status.paired ? <Till status={status} /> : <Setup onPaired={refresh} />}
    </LangProvider>
  );
}

// The till itself: the screens the browser till already runs.
//
// ⚠️ **The providers are kassa/layout.tsx's, reproduced rather than imported.**
// A Next layout is a route convention — it takes no props, is composed by the
// router and carries `metadata` and `viewport` exports that mean nothing here.
// What it actually contributes is the providers, and those are what is
// repeated. If a third appears there, it has to be added here too: that is the
// seam this shell has, and it is written down rather than discovered.
function Till({ status }: { status: Status }) {
  return (
    <StaffProvider>
      <KassaScreen />
      {/* The relay is what turns a sale into paper. Silence about it is what
          makes "the printer is broken" the first theory. Floated, because the
          till's own layout has no room reserved for us. */}
      {!status.agent && (
        <p className="till fixed bottom-3 left-1/2 z-50 -translate-x-1/2 rounded-full border border-line bg-surface px-3 py-1.5 text-xs text-danger shadow-card">
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

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

import TillShell from "@/components/till/TillShell";
import { setTillDeviceToken } from "@/lib/api";
import { LangProvider } from "@/lib/i18n/client";
import { LANGS, type Lang } from "@/lib/i18n";
import KassaScreen from "@/app/kassa/page";
import ZalScreen from "@/app/zal/page";

import Printer from "./Printer";
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

// ⚠️ **Ctrl+Shift+P, and it belongs in the install notes beside Ctrl+Shift+Q.**
// The printer is chosen once when the machine is installed and again the day it
// is replaced, so a permanent control on the selling screen would sit in a
// cashier's way every evening for a setting nobody touches. A hotkey is
// discoverable only if it is written down — which is the condition, not a
// caveat: an undocumented one is a setting that does not exist.
function usePrinterHotkey(open: () => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "p") {
        e.preventDefault();
        open();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);
}

// ⚠️ **Windows is no longer asked for a keyboard, and that is the fix rather
// than a simplification.** This used to raise TabTip on every focused field —
// correct when the till had no keyboard of its own, and wrong from the moment
// it did: two keyboards competed for the same tap, and the one that won was
// Windows'. It is the wrong shape for a 1024×768 counter, it covers the bottom
// third of the screen including the button somebody is reaching for, it is in
// whatever language Windows was installed in, and it looks nothing like the
// application it appears over — which on a machine sold as an appliance reads
// as the software having crashed into the operating system.
//
// The till's own keyboard is mounted in `Till` below, from the same shared
// component the browser till uses.

// ⚠️ **No context menu.** A long press on a touch screen opens it, and on a
// till it can only offer things that are wrong: reload, back, view source. The
// CSS next door stops the selection bubble; this stops the menu behind it.
// ⚠️ **Ctrl+Shift+M, and it belongs in the install notes beside the other two.**
// Which screen a machine opens is decided when it is installed and again the day
// it is moved — so a permanent control would sit in somebody's way every evening
// for a setting nobody touches, and the same reasoning as the printer hotkey
// applies. It confirms first: pressed by accident during service it would take a
// cashier's till away and leave them looking at a floor plan.
function useModeHotkey(swap: () => void) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === "m") {
        e.preventDefault();
        swap();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [swap]);
}

function useNoContextMenu() {
  useEffect(() => {
    const block = (e: Event) => e.preventDefault();
    document.addEventListener("contextmenu", block);
    return () => document.removeEventListener("contextmenu", block);
  }, []);
}

function App() {
  useQuitHotkey();
  useNoContextMenu();
  const [status, setStatus] = useState<Status | null>(null);
  // ⚠️ **Three states, not a boolean.** "setup" is the last step of pairing and
  // continues into the till; "later" is the same screen opened by the hotkey
  // and closes back to it. A single flag would have the install flow end on a
  // "Close" button that returns to a till nobody has started yet.
  const [printer, setPrinter] = useState<"" | "setup" | "later">("");
  const openPrinter = useCallback(() => setPrinter("later"), []);
  usePrinterHotkey(openPrinter);
  // The other screen this machine could be. Asked rather than done: see the
  // hotkey's note.
  const [swapping, setSwapping] = useState(false);
  useModeHotkey(useCallback(() => setSwapping(true), []));

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
  // ⚠️ Only inside the application: in a browser there is no Go side to ask,
  // and the hotkey would open a screen whose every control is inert.
  if (printer && bridge()) {
    return (
      <LangProvider initial={startLang()}>
        <Printer
          doneLabel={printer === "setup" ? "Kassaga o'tish" : "Yopish"}
          onDone={() => {
            setPrinter("");
            if (printer === "setup") refresh();
          }}
        />
      </LangProvider>
    );
  }

  return (
    <LangProvider initial={startLang()}>
      {swapping && (
        <ModeSwap
          current={status.mode === "zal" ? "zal" : "kassa"}
          onCancel={() => setSwapping(false)}
          onChosen={() => {
            setSwapping(false);
            // ⚠️ **Reloaded, not swapped in place.** The two screens mount
            // different providers and different polls; swapping live would
            // leave the one that was running holding a table it no longer
            // draws. A reload is what pairing already does.
            window.location.reload();
          }}
        />
      )}
      {status.paired ? (
        <Till status={status} />
      ) : (
        // ⚠️ **The printer is the last step of setup, not a screen somebody has
        // to find afterwards.** Whoever is pairing the machine is standing in
        // front of the printer with the paper in reach — which is the only
        // moment a test print can be checked by the person who pressed it.
        <Setup onPaired={() => setPrinter("setup")} />
      )}
    </LangProvider>
  );
}

// The till itself: the screens the browser till already runs.
//
// ⚠️ **The shell is imported, not reproduced.** This file used to list the
// providers `kassa/layout.tsx` mounts, with a note saying a fourth would have
// to be added here too — and when `AskProvider` arrived it was not. The
// browser till started asking its questions in our own box and this one kept
// calling `window.confirm`: the browser's dialog, with the restaurant's domain
// above it, on the machine that is sold as a cash register. Nothing reported
// it, because nothing here was broken — a piece was simply missing, which is
// what a hand-copied list does the first time somebody edits the original.
//
// `TillShell` is that list as a component, so the copy has nothing left to
// fall behind on.
function Till({ status }: { status: Status }) {
  // ⚠️ **One shell, two screens, chosen by the machine.** The floor screen is
  // its own route rather than a mode of the till (`zal/layout.tsx` says why),
  // and the same is true here: what changes is which screen is mounted, not how
  // any of it works. Everything around it — the pairing, the printer, the
  // keyboard, the offline disk, the update — belongs to the machine and is the
  // same for both.
  const floor = status.mode === "zal";
  return (
    <TillShell role={floor ? "ofitsiant" : "kassir"}>
      {floor ? <ZalScreen /> : <KassaScreen />}
      {/* The relay is what turns a sale into paper. Silence about it is what
          makes "the printer is broken" the first theory. Floated, because the
          till's own layout has no room reserved for us. */}
      {!status.agent && (
        <p className="till fixed bottom-3 left-1/2 z-50 -translate-x-1/2 rounded-full border border-line bg-surface px-3 py-1.5 text-xs text-danger shadow-card">
          Agent ishlamayapti — chek chiqmaydi
        </p>
      )}
    </TillShell>
  );
}

/** The question the hotkey asks. */
function ModeSwap({
  current,
  onCancel,
  onChosen,
}: {
  current: "kassa" | "zal";
  onCancel: () => void;
  onChosen: () => void;
}) {
  const other = current === "kassa" ? "zal" : "kassa";
  const [busy, setBusy] = useState(false);
  return (
    <div className="till fixed inset-0 z-[60] grid place-items-center bg-black/40 p-6">
      <div className="w-full max-w-sm rounded-2xl bg-surface p-5 shadow-card">
        <h2 className="text-base font-semibold">
          {other === "zal"
            ? "Zal ekraniga o'tilsinmi?"
            : "Kassaga qaytilsinmi?"}
        </h2>
        <p className="mt-1 text-sm text-ink-muted">
          {other === "zal"
            ? "Bu qurilma ofitsiant ekranini ochadi: stollar va buyurtma. Pul va smena kassada qoladi."
            : "Bu qurilma kassa ekranini ochadi: pul, chek va smena."}
        </p>
        <div className="mt-5 flex gap-2">
          <button
            className="till-btn-ghost flex-1"
            disabled={busy}
            onClick={onCancel}
          >
            Bekor qilish
          </button>
          <button
            className="till-btn-accent flex-1"
            disabled={busy}
            onClick={() => {
              setBusy(true);
              void bridge()
                ?.SetMode(other)
                .then(onChosen)
                .catch(() => setBusy(false));
            }}
          >
            O'tish
          </button>
        </div>
      </div>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

/** Which language this monoblock was set up in.
 *
 *  ⚠️ **Read once, at start.** The shell used to hard-code Uzbek in all three
 *  places it mounts a provider, so the switch on the setup screen lasted until
 *  the window was closed — and a till is closed every night. The provider
 *  writes the choice to its own cookie when it changes; this reads the same
 *  cookie back.
 *
 *  ⚠️ Anything unrecognised is Uzbek, which is what every till has been running
 *  and what a machine with no choice recorded should still open in.
 */
function startLang(): Lang {
  if (typeof document === "undefined") return "uz";
  const m = document.cookie.match(/(?:^|;\s*)lang=([a-z]{2})/);
  const found = m?.[1];
  return (LANGS as readonly string[]).includes(found ?? "") ? (found as Lang) : "uz";
}

"use client";

// The site, running inside Telegram.
//
// The mini app is not a second product: it is this site in Telegram's WebView.
// So this file adds only what that context needs and changes nothing else — a
// guest on the open web must not pay for any of it.
//
// Four things happen here, and the first is the reason the rest is worth doing:
//
//   • **Signing in with no SMS.** Telegram already knows who the person is and
//     says so in a payload signed with the restaurant's own bot token. We hand
//     that to the server, which verifies the signature and gives back one of our
//     sessions (see internal/telegram). The guest is logged in before they have
//     touched anything — no code to wait for, and no paid message to send.
//
//   • **The viewport.** ⚠️ `100vh` is wrong inside Telegram: the WebView reports
//     the full screen while the visible area is smaller, so a full-height layout
//     hides its bottom edge behind Telegram's own chrome — and behind the
//     keyboard the moment somebody types an address. `viewportStableHeight` is
//     the honest number, and it is published here as a CSS variable.
//
//   • **The back button.** Telegram draws one; a mini app that ignores it leaves
//     the guest with a button that does nothing, which reads as a broken app
//     rather than a missing feature.
//
//   • **Closing confirmation**, but only with something in the cart. Asking "are
//     you sure" on an empty cart trains people to dismiss it.
//
// ⚠️ The SDK script is loaded **only inside Telegram**, detected from the launch
// URL. Loading it for every web visitor would add a request to every page to
// serve a feature none of them can use — the opposite of the work just done on
// page weight.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { api, setUserToken } from "./api";
import type { SiteUser } from "./types";

/** The slice of Telegram's WebApp object this app actually uses.
 *
 *  Typed by hand rather than pulled from a package: the surface used here is
 *  small and stable, and a dependency for four fields would be a dependency to
 *  keep in step with Telegram's release notes. Everything is optional because
 *  every field arrived in some Bot API version — an older client simply does not
 *  have it, and the code has to work rather than throw. */
interface TelegramWebApp {
  initData?: string;
  /** Telegram's own parsed copy. Only `start_param` is read from it, and only to
   *  decide where to navigate — **not** who the guest is. That distinction is the
   *  whole reason the signed `initData` above exists: identity comes from the
   *  string the server verified, a destination is just a link. */
  initDataUnsafe?: { start_param?: string };
  platform?: string;
  version?: string;
  viewportStableHeight?: number;
  viewportHeight?: number;
  colorScheme?: "light" | "dark";
  ready?: () => void;
  expand?: () => void;
  onEvent?: (name: string, cb: () => void) => void;
  offEvent?: (name: string, cb: () => void) => void;
  enableClosingConfirmation?: () => void;
  disableClosingConfirmation?: () => void;
  requestContact?: (cb: (ok: boolean, res?: unknown) => void) => void;
  BackButton?: {
    show: () => void;
    hide: () => void;
    onClick: (cb: () => void) => void;
    offClick: (cb: () => void) => void;
  };
  HapticFeedback?: {
    impactOccurred?: (style: string) => void;
    notificationOccurred?: (type: string) => void;
  };
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp };
  }
}

interface TelegramContextValue {
  /** Whether this page is being rendered inside Telegram at all. */
  inTelegram: boolean;
  /** The SDK object, once the script has loaded. */
  webApp: TelegramWebApp | null;
  /** Signed in through Telegram, so the checkout can skip the SMS step. */
  signedIn: boolean;
  /** ⚠️ Signed in, but Telegram gave no phone number — an order cannot be
   *  delivered yet. The checkout asks once; discovering this at "confirm order"
   *  would lose an order already won. */
  needsPhone: boolean;
  /** Ask Telegram for the phone number. Resolves true when one was stored. */
  askPhone: () => Promise<boolean>;
  /** `?startapp=…` from the launch link: which table the guest scanned. */
  startParam: string;
}

const TelegramContext = createContext<TelegramContextValue>({
  inTelegram: false,
  webApp: null,
  signedIn: false,
  needsPhone: false,
  askPhone: async () => false,
  startParam: "",
});

/** The table id inside a `startapp` parameter, or "".
 *
 *  ⚠️ Telegram allows only `A-Za-z0-9_-` in `startapp`, which is why the table is
 *  encoded as `t_<id>` rather than as a query string — a printed QR that Telegram
 *  refuses to open is a card glued to a table for years, doing nothing.
 *
 *  Parsed leniently and validated as a hex id: this value comes from a link
 *  anybody can type, and it ends up in an API call. */
export function parseStartParam(raw: string): { table?: string; branch?: string } {
  const out: { table?: string; branch?: string } = {};
  for (const part of (raw || "").split("-")) {
    const [key, ...rest] = part.split("_");
    const value = rest.join("_");
    if (!/^[a-f0-9]{24}$/i.test(value)) continue;
    if (key === "t") out.table = value;
    if (key === "b") out.branch = value;
  }
  return out;
}

/** Whether this page was opened from Telegram.
 *
 *  Read from the launch URL, which Telegram fills in with `tgWebApp…`
 *  parameters. The user-agent check is a fallback for clients that route
 *  differently; between them a false negative means "the site works normally",
 *  which is the safe way to be wrong. */
function detectTelegram(): boolean {
  if (typeof window === "undefined") return false;
  if (window.Telegram?.WebApp?.initData) return true;
  const hay = window.location.hash + window.location.search;
  if (hay.includes("tgWebApp")) return true;
  return /Telegram/i.test(navigator.userAgent);
}

const SDK_URL = "https://telegram.org/js/telegram-web-app.js";

function loadSdk(): Promise<TelegramWebApp | null> {
  return new Promise((resolve) => {
    if (window.Telegram?.WebApp) return resolve(window.Telegram.WebApp);
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${SDK_URL}"]`,
    );
    if (existing) {
      existing.addEventListener("load", () =>
        resolve(window.Telegram?.WebApp ?? null),
      );
      return;
    }
    const s = document.createElement("script");
    s.src = SDK_URL;
    s.async = true;
    s.onload = () => resolve(window.Telegram?.WebApp ?? null);
    // A failed script is not a failed site: the page keeps working as the web
    // page it also is, which is the whole reason the mini app is this site.
    s.onerror = () => resolve(null);
    document.head.appendChild(s);
  });
}

export function TelegramProvider({
  children,
  onUser,
}: {
  children: ReactNode;
  /** Handed the session so the existing user provider stays the one source of
   *  truth about who is signed in — a second store would drift from it. */
  onUser?: (token: string, user: SiteUser) => void;
}) {
  const [webApp, setWebApp] = useState<TelegramWebApp | null>(null);
  const [inTelegram, setInTelegram] = useState(false);
  const [signedIn, setSignedIn] = useState(false);
  const [needsPhone, setNeedsPhone] = useState(false);
  const [startParam, setStartParam] = useState("");
  // Guards the login against a second run: React mounts effects twice in
  // development, and two logins would mint two sessions for one guest.
  const tried = useRef(false);

  useEffect(() => {
    if (!detectTelegram()) return;
    setInTelegram(true);
    let cancelled = false;

    loadSdk().then(async (app) => {
      if (cancelled || !app) return;
      setWebApp(app);
      setStartParam(app.initDataUnsafe?.start_param ?? "");
      app.ready?.();
      // Opened at full height: a mini app that starts as a half-sheet shows the
      // menu through a letterbox, and the guest has to know to drag it up.
      app.expand?.();
      publishViewport(app);

      if (tried.current || !app.initData) return;
      tried.current = true;
      try {
        const res = await api.telegramLogin(app.initData);
        setUserToken(res.token);
        setSignedIn(true);
        setNeedsPhone(!!res.needsPhone);
        onUser?.(res.token, res.user);
      } catch {
        // The restaurant may not have connected a bot, or the payload may be
        // stale. Either way the site still works — the guest signs in by SMS
        // like everybody on the open web.
      }
    });

    return () => {
      cancelled = true;
    };
  }, [onUser]);

  // Telegram resizes its WebView as its own chrome and the keyboard come and go,
  // and it reports each change. Without this the variable is only right on the
  // first paint, which is exactly when nobody is typing.
  useEffect(() => {
    if (!webApp?.onEvent) return;
    const onResize = () => publishViewport(webApp);
    webApp.onEvent("viewportChanged", onResize);
    return () => webApp.offEvent?.("viewportChanged", onResize);
  }, [webApp]);

  const askPhone = useCallback(async () => {
    if (!webApp?.requestContact) return false;
    return new Promise<boolean>((resolve) => {
      webApp.requestContact!(async (ok, res) => {
        if (!ok) return resolve(false);
        // ⚠️ Passed to the server as it arrived, and verified there. The payload
        // is signed; re-shaping it in the browser would break the signature,
        // and trusting it here would make the phone number a field anybody can
        // type.
        const payload = contactPayload(res);
        if (!payload) return resolve(false);
        try {
          await api.telegramPhone(payload);
          setNeedsPhone(false);
          resolve(true);
        } catch {
          resolve(false);
        }
      });
    });
  }, [webApp]);

  return (
    <TelegramContext.Provider
      value={{ inTelegram, webApp, signedIn, needsPhone, askPhone, startParam }}
    >
      {children}
    </TelegramContext.Provider>
  );
}

export function useTelegram(): TelegramContextValue {
  return useContext(TelegramContext);
}

/** Publishes Telegram's honest viewport height as a CSS variable.
 *
 *  ⚠️ `--tg-viewport` exists because `100vh` lies inside Telegram: the WebView
 *  reports the whole screen, so a full-height element runs under Telegram's
 *  chrome and, once the keyboard opens, under that too. Anything that wants to
 *  fill the screen uses this instead. */
function publishViewport(app: TelegramWebApp) {
  const h = app.viewportStableHeight || app.viewportHeight;
  if (!h) return;
  document.documentElement.style.setProperty("--tg-viewport", `${h}px`);
}

/** Pulls the signed string out of whatever `requestContact` handed back.
 *
 *  ⚠️ Read leniently on purpose. The signature scheme is documented and verified
 *  server-side; the *shape* of this particular response is the one thing in the
 *  Telegram integration that has not been checked against a live bot, so both
 *  the documented wrapper and a bare string are accepted. Being strict here
 *  would reject a real, signed phone number over a field name. */
function contactPayload(res: unknown): string | null {
  if (typeof res === "string") return res;
  if (res && typeof res === "object") {
    const o = res as Record<string, unknown>;
    for (const key of ["response", "responseUnsafe", "data", "contact"]) {
      const v = o[key];
      if (typeof v === "string" && v.includes("hash=")) return v;
    }
  }
  return null;
}

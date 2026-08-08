"use client";

// Mounts the Telegram bridge inside the site's existing providers.
//
// Renders nothing. It exists so `TelegramProvider` can hand the session to
// `useUser()` — the store the rest of the site already reads — rather than
// keeping a second idea of who is signed in. Two stores would disagree the first
// time somebody signed out.
//
// Also wires Telegram's own back button and its closing confirmation, both of
// which are cheap and both of which are noticed only when missing: a back button
// that does nothing reads as a broken app, and a cart lost to a stray swipe reads
// as a lost order.

import { useEffect, useRef } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { TelegramProvider, parseStartParam, useTelegram } from "@/lib/telegram";
import { useUser } from "@/lib/user";
import { useCart } from "@/lib/cart";

export default function TelegramApp({ children }: { children?: React.ReactNode }) {
  const { login } = useUser();
  return (
    <TelegramProvider onUser={login}>
      <TelegramChrome />
      {children}
    </TelegramProvider>
  );
}

function TelegramChrome() {
  const { inTelegram, webApp, startParam } = useTelegram();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const { lines } = useCart();
  const routed = useRef(false);

  // A table QR opened through Telegram arrives as `startapp=t_<id>` — there is no
  // query string to read. Rather than teach the table logic a second source, the
  // parameter is turned into the URL the site already understands, so the brand
  // cookie, the plan lookup, the banner and the dine-in order type all keep
  // working unchanged.
  useEffect(() => {
    if (!inTelegram || routed.current || !startParam) return;
    if (params.get("table")) return; // already there
    const { table, branch } = parseStartParam(startParam);
    if (!table) return;
    routed.current = true;
    const qs = new URLSearchParams({ table });
    if (branch) qs.set("branch", branch);
    router.replace(`/menu?${qs}`);
  }, [inTelegram, startParam, params, router, pathname]);

  // Telegram draws the back button; this makes it mean something.
  useEffect(() => {
    if (!inTelegram || !webApp?.BackButton) return;
    const back = () => router.back();
    webApp.BackButton.onClick(back);
    webApp.BackButton.show();
    return () => {
      webApp.BackButton?.offClick(back);
      webApp.BackButton?.hide();
    };
  }, [inTelegram, webApp, router]);

  // ⚠️ Only with something in the cart. A confirmation on an empty cart teaches
  // people to dismiss it, and then it is not there when it matters.
  useEffect(() => {
    if (!inTelegram || !webApp) return;
    if (lines.length > 0) webApp.enableClosingConfirmation?.();
    else webApp.disableClosingConfirmation?.();
  }, [inTelegram, webApp, lines.length]);

  return null;
}

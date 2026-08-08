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

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { TelegramProvider, useTelegram } from "@/lib/telegram";
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
  const { inTelegram, webApp } = useTelegram();
  const router = useRouter();
  const { lines } = useCart();

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

"use client";

// Registers the courier service worker. Without a service worker the browser
// will not offer "install to home screen", so this is what makes /kuryer a PWA.

import { useEffect } from "react";

export default function RegisterCourierSW() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;
    navigator.serviceWorker
      .register("/courier-sw.js", { scope: "/kuryer" })
      .catch((err) => console.error("[kuryer] SW registration failed:", err));
  }, []);
  return null;
}

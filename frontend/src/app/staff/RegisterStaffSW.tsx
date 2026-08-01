"use client";

// Registers the staff service worker. Without one the browser will not offer
// "install to home screen", so this is what makes /staff a PWA.

import { useEffect } from "react";

export default function RegisterStaffSW() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;
    navigator.serviceWorker
      .register("/staff-sw.js", { scope: "/staff" })
      .catch((err) => console.error("[staff] SW registration failed:", err));
  }, []);
  return null;
}

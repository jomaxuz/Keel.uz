import type { Metadata, Viewport } from "next";
import { Suspense } from "react";
import CrashReporter from "@/components/CrashReporter";

// The kiosk is a screen, not a page of the site: no header, no footer, no
// theme toggle. It is also always light — a wall screen is read from a
// distance, and a dark QR on a dark card is not.
export const metadata: Metadata = {
  title: "Kiosk",
  robots: { index: false, follow: false },
};

export const viewport: Viewport = {
  themeColor: "#ffffff",
  width: "device-width",
  initialScale: 1,
};

export default function KioskLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Suspense>
      <CrashReporter app="till" role="kiosk" />
      {children}
    </Suspense>
  );
}

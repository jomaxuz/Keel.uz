import type { Metadata, Viewport } from "next";
import { Suspense } from "react";
import CrashReporter from "@/components/CrashReporter";
import AskProvider from "@/components/ui/Ask";
import TillAppliance from "@/components/till/TillAppliance";

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
      {/* ⚠️ A screen on a wall, touched by everyone who walks past it and
          owned by nobody standing next to it. Of the four appliance screens
          this is the one where a zoomed-in layout would stay zoomed in for a
          day: there is no cashier to notice, and no obvious control to undo
          it. */}
      <TillAppliance />
      {/* ⚠️ Our own question box, on a screen nobody owns: the browser's
          would sit there with the domain showing until a passer-by pressed
          it. */}
      <div className="appliance">
        <AskProvider look="till">{children}</AskProvider>
      </div>
    </Suspense>
  );
}

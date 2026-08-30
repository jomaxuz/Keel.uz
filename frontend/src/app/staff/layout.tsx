import type { Metadata, Viewport } from "next";
import CrashReporter from "@/components/CrashReporter";
import { StaffProvider } from "@/lib/staff";
import RegisterStaffSW from "./RegisterStaffSW";
import TillAppliance from "@/components/till/TillAppliance";

// The staff app is its own PWA (own manifest and scope) so an employee can
// install just this section to their phone, next to — not instead of — the
// courier app. It deliberately skips the site header/footer: it is a time
// clock, not a page of the restaurant website.
export const metadata: Metadata = {
  title: "Ishchi",
  manifest: "/staff-manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "Ishchi",
  },
};

export const viewport: Viewport = {
  themeColor: "#e2590d",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function StaffLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <StaffProvider>
      <CrashReporter app="kitchen" />
      <RegisterStaffSW />
      {/* ⚠️ **The kitchen display is the reason this is here.** It is a wall
          screen that gets tapped hard and fast with a wet hand all evening, and
          it had none of the till's touch rules — because those rules lived
          inside the till's *palette* class, and the pass does not want cream
          and amber. A stray pinch left the tickets at 140% with the far column
          off the edge, and nothing on screen said how to put it back. */}
      <TillAppliance />
      <div className="appliance min-h-dvh bg-bg">{children}</div>
    </StaffProvider>
  );
}

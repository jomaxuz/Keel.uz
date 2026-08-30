import type { Metadata, Viewport } from "next";
import CrashReporter from "@/components/CrashReporter";
import { StaffProvider } from "@/lib/staff";
import RegisterStaffSW from "./RegisterStaffSW";

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
      <div className="min-h-dvh bg-bg">{children}</div>
    </StaffProvider>
  );
}

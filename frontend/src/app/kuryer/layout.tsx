import type { Metadata, Viewport } from "next";
import CrashReporter from "@/components/CrashReporter";
import { CourierProvider } from "@/lib/courier";
import RegisterCourierSW from "./RegisterCourierSW";

// The courier app is a separate PWA (own manifest and scope) so a courier can
// install just this section to their phone. It deliberately skips the site
// header/footer — it is a tool, not a page of the restaurant website.
export const metadata: Metadata = {
  title: "Kuryer",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "Kuryer",
  },
};

export const viewport: Viewport = {
  themeColor: "#e2590d",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function CourierLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <CourierProvider>
      <CrashReporter app="courier" />
      <RegisterCourierSW />
      <div className="min-h-dvh bg-bg">{children}</div>
    </CourierProvider>
  );
}

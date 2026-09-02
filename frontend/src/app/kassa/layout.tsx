import type { Metadata, Viewport } from "next";
import TillShell from "@/components/till/TillShell";

// The till is its own screen but not its own account system: it runs on the
// same staff token as the pass and the time clock, because it is the same
// people. What separates a cashier from a waiter is a permission the server
// checks per action — see models.Staff.Can — not a second login.
//
// No site header or footer: this is a machine on a counter, used standing up,
// and every pixel that is not a dish or a total is in the way.
export const metadata: Metadata = {
  title: "Kassa",
  // Not indexable and not reachable by accident.
  robots: { index: false, follow: false },
};

export const viewport: Viewport = {
  themeColor: "#e2590d",
  width: "device-width",
  initialScale: 1,
  // The till runs full-screen on a tablet that is usually in a dock; a
  // rubber-band scroll on the whole page moves the buttons under a finger
  // that is already committed to pressing one.
  maximumScale: 1,
  // ⚠️ Says what we mean, and is not trusted to do it: Safari ignores this
  // and desktop browsers never read it at all. The gestures are refused in
  // components/till/NoZoom, which is where the guarantee actually lives.
  userScalable: false,
  viewportFit: "cover",
};

export default function TillLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // ⚠️ **The pieces live in TillShell, not here.** A Next layout cannot be
  // imported — it is a route convention — so the Windows application had to
  // reproduce this list by hand, and the copy fell behind the day a provider
  // was added: every question on that till went back to the browser's own
  // confirm box. One component, three mounts.
  return <TillShell role="kassir">{children}</TillShell>;
}

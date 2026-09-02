import type { Metadata, Viewport } from "next";
import TillShell from "@/components/till/TillShell";

// The floor screen: a tablet carried between tables, not a machine on a
// counter.
//
// ⚠️ **Its own route rather than a mode of the till.** They are held in
// different hands and answer different questions — the till ends a sale, this
// one starts it — and it is this screen that will become the Keel Waiter app on
// a phone (docs/pos-reja.md §3). Sharing a route would mean every change to the
// till had to be checked against a screen nobody was thinking about.
//
// Same accounts and the same device binding as the till: what separates a
// waiter from a cashier is a permission the server checks per action, never a
// second login.
export const metadata: Metadata = {
  title: "Zal",
  robots: { index: false, follow: false },
};

export const viewport: Viewport = {
  themeColor: "#e2590d",
  width: "device-width",
  initialScale: 1,
  // Carried in one hand: a rubber-band scroll moves a button under a thumb
  // that is already committed to pressing it.
  maximumScale: 1,
  // ⚠️ Says what we mean, and is not trusted to do it: Safari ignores this
  // and desktop browsers never read it at all. The gestures are refused in
  // components/till/NoZoom, which is where the guarantee actually lives.
  userScalable: false,
  viewportFit: "cover",
};

export default function FloorLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // ⚠️ **The pieces live in TillShell, not here.** A Next layout cannot be
  // imported — it is a route convention — so the Windows application had to
  // reproduce this list by hand, and the copy fell behind the day a provider
  // was added: every question on that till went back to the browser's own
  // confirm box. One component, three mounts.
  return <TillShell role="ofitsiant">{children}</TillShell>;
}

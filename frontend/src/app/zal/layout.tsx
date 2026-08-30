import type { Metadata, Viewport } from "next";
import CrashReporter from "@/components/CrashReporter";
import { StaffProvider } from "@/lib/staff";
import TillAppliance from "@/components/till/TillAppliance";
import OnScreenKeyboard from "@/components/till/OnScreenKeyboard";

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
  return (
    <StaffProvider>
      <CrashReporter app="till" role="ofitsiant" />
      {/* Three things that make this a machine rather than a web page,
          mounted once for every screen under it: a press that registers where
          the finger landed, nothing that zooms or can be copied off the screen
          (TillAppliance), and our own keyboard instead of the operating
          system's. ⚠️ On the layout, not on the page: a dialog or a toast that
          renders outside `<main>` is exactly the surface the earlier, local
          fixes kept missing. */}
      <TillAppliance />
      {/* ⚠️ The keyboard is inside the class too. It sits outside `<main>` —
          which is exactly why it was missed before — and it is the one surface
          on this screen that is nothing but keys. */}
      <div className="appliance">
        <div className="min-h-dvh bg-bg">{children}</div>
        <OnScreenKeyboard />
      </div>
    </StaffProvider>
  );
}

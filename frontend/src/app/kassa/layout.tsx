import type { Metadata, Viewport } from "next";
import CrashReporter from "@/components/CrashReporter";
import { StaffProvider } from "@/lib/staff";
import TillAppliance from "@/components/till/TillAppliance";
import OnScreenKeyboard from "@/components/till/OnScreenKeyboard";

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
  return (
    <StaffProvider>
      <CrashReporter app="till" role="kassir" />
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

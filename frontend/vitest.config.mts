import { fileURLToPath } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

/**
 * The till and floor screens are the only part of this app with a browser-side
 * test run, and that is deliberate.
 *
 * ⚠️ **Every defect the live trial found was invisible to the Go tests.**
 * `TablesScreen` was imported and never rendered; the option dialog did not
 * exist, so a dish with a required group could not be sold at all. In both
 * cases the server was already correct and already covered — the missing half
 * was on the screen, and no backend test can reach it.
 *
 * So these run the real components against a fake till server: the assertions
 * are about what a cashier can reach with a finger, not about markup.
 */
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    // ⚠️ Named, not a glob over src/. A wide pattern collects nothing else
    // today and starts collecting half-written files later.
    include: [
      "src/app/kassa/*.test.ts?(x)",
      "src/app/zal/*.test.ts?(x)",
      // Shared till pieces that are a screen in their own right rather than a
      // step in a flow — the on-screen keyboard, which both of the above mount.
      "src/components/till/*.test.ts?(x)",
      // ⚠️ Named on its own rather than a glob over src/lib. Printing is not a
      // screen, but it is the one decision on the way out of every screen here
      // — spooler or browser dialog — and it fails silently on the hardware
      // none of these tests run on.
      "src/lib/print.test.ts",
      // The address a printer is stored under: wrong here, and it saves, lists,
      // and never prints.
      "src/lib/printerTarget.test.ts",
      // What a comma means in a quantity. Also not a screen, and it fails the
      // same way: a shelf saved ten times too full, found weeks later at a
      // count and blamed on whoever counted.
      "src/lib/qty.test.ts",
      // The field that applies those rules. Its own half is that it holds the
      // draft: a parent storing a number would otherwise erase the point as it
      // is pressed, which no test of the rules can see.
      "src/components/QtyInput.test.tsx",
      // Which disk the offline queue is written to. Both engines work, so
      // choosing the browser's inside the Windows app fails at no point except
      // the evening the power goes out.
      "src/lib/offline/store.test.ts",
      // Whether the machine's clock may be stamped from at all. The fault it
      // guards is a tax document with the wrong date, written by a till that
      // looks entirely ordinary all evening.
      "src/lib/offline/clock.test.ts",
      // What a scanned marking code may be. Not a screen either, and it fails
      // in front of a guest: a code the tax register refuses stops a payment
      // that has already been started.
      "src/lib/marking.test.ts",
      // Where a token is kept and where the server is — the one seam a second
      // platform needs. Its risk is not that the phone breaks but that the web
      // does, silently, by reading a token under a different name.
      "src/lib/tokenStore.test.ts",
      "src/lib/help/*.test.ts",
      // Turning "osh" into a server address. A second implementation of a Go
      // rule, so the test is what keeps the two honest.
      "src/lib/serverAddress.test.ts",
    ],
    // ⚠️ One at a time. The offline queue is a database shared by the whole
    // run: two files closing checks in parallel would drain each other's
    // payments, and the failure would look like a race in the till rather than
    // in the test setup.
    fileParallelism: false,
  },
});

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
    include: ["src/app/kassa/*.test.tsx", "src/app/zal/*.test.tsx"],
  },
});

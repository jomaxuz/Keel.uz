import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The till screens live in the Next application and are compiled again here.
//
// ⚠️ **The screens are shared, not copied.** `frontend/src/app/kassa` and
// `frontend/src/components/till` are the same files the browser till runs; a
// copy would be two tills within a month, and the second one would be the one
// the restaurant is using when a bug is reported against the first.
//
// The cost of sharing is this file: the shared code imports two Next modules,
// and Vite is given a stand-in for each (see src/shims). Measured before it was
// chosen — across `app/kassa`, `components/till` and `lib`, the entire
// third-party surface is react, react-dom and react-icons.
const shared = fileURLToPath(new URL("../../../frontend/src", import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // Same specifier the shared files already use, pointed at the same tree.
      "@": shared,
      // ⚠️ Order matters: "next/navigation" must be matched before any broader
      // "next" entry, and Vite takes the first alias that matches.
      "next/navigation": fileURLToPath(
        new URL("./src/shims/next-navigation.tsx", import.meta.url),
      ),
      "next/headers": fileURLToPath(
        new URL("./src/shims/next-headers.ts", import.meta.url),
      ),
    },
  },
  // ⚠️ Vite refuses to serve files outside its root in dev; the shared tree is
  // two directories up, so dev mode would 403 on every screen without this.
  server: { fs: { allow: [shared, fileURLToPath(new URL(".", import.meta.url))] } },
  define: {
    // The shared libraries read `process.env.*` because Next replaces it at
    // build time. Vite does not, and an undefined `process` is a blank screen
    // with one line in a console nobody has open.
    "process.env.NEXT_PUBLIC_API_URL": JSON.stringify(
      process.env.NEXT_PUBLIC_API_URL ?? "/api/v1",
    ),
    "process.env.NEXT_PUBLIC_UPLOADS_URL": JSON.stringify(
      process.env.NEXT_PUBLIC_UPLOADS_URL ?? "/uploads",
    ),
    "process.env.TENANT_MODE": JSON.stringify(""),
    "process.env.CONTROL_ORIGIN": JSON.stringify(""),
    "process.env.INTERNAL_API_URL": JSON.stringify(""),
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    // ⚠️ Raised deliberately, not ignored. The warning is about download time
    // over a network, and this bundle is embedded in the executable and read
    // from local disk — there is no network to be slow. Code-splitting the till
    // would trade nothing for a screen that can stall mid-service on a machine
    // whose entire selling point is working when the connection does not.
    chunkSizeWarningLimit: 1500,
  },
});

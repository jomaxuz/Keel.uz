import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";

// ⚠️ **One named file, not a glob.** The same rule the panel's suite follows:
// a pattern picks up whatever somebody drops in the tree, and the list is how a
// test that stopped being run gets noticed.
export default defineConfig({
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    // The article renderer's parser. Pure, so no browser is needed — and the
    // pieces worth pinning are what a YouTube link becomes and what a pasted
    // `javascript:` link does not.
    include: [
      "src/components/blog/article.test.ts",
      // Where each console role lands after signing in — the redirect that keeps
      // sales and support accounts off the owner's overview.
      "src/lib/consoleHome.test.ts",
      // Which role may open which console page — the guard behind the tabs.
      "src/lib/consoleAccess.test.ts",
    ],
  },
});

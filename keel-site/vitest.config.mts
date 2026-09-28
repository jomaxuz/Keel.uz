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
      // Whether an edit can be drawn into the live preview or needs the page
      // rendered again — the difference between an editor that flashes on every
      // click and one that does not.
      "src/lib/designDiff.test.ts",
      // keel.uz/llms.txt and llms-full.txt — including the `URL:` line under
      // every page that the console's watcher splits the file on.
      "src/lib/llms.test.ts",
    ],
  },
});

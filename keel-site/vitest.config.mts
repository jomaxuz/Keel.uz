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
    include: ["src/components/blog/article.test.ts"],
  },
});

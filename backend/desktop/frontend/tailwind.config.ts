import type { Config } from "tailwindcss";
import shared from "../../../frontend/tailwind.config";

// The till's Tailwind setup is the site's, pointed at two source trees.
//
// ⚠️ **Extended, not copied.** The palette, the radius scale and the shadow
// tokens are one design decision per restaurant, and a duplicated config drifts
// the moment somebody tunes an accent — leaving a till whose buttons are a
// slightly different orange from the panel that configures it. The theme comes
// from the shared file; only `content` differs, because the class names live in
// two places now.
const config: Config = {
  ...shared,
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    // The shared screens, which is where nearly every class actually is.
    "../../../frontend/src/app/kassa/**/*.{ts,tsx}",
    "../../../frontend/src/components/**/*.{ts,tsx}",
    "../../../frontend/src/lib/**/*.{ts,tsx}",
  ],
};

export default config;

import { fileURLToPath, URL } from "node:url";
import type { Config } from "tailwindcss";
import shared from "../../../frontend/tailwind.config";

// The till's Tailwind setup is the site's, pointed at two source trees.
//
// ⚠️ **Extended, not copied.** The palette, the radius scale and the shadow
// tokens are one design decision per restaurant, and a duplicated config drifts
// the moment somebody tunes an accent — leaving a till whose buttons are a
// slightly different amber from the panel that configures it. The theme comes
// from the shared file; only `content` differs, because the class names live in
// two places now.
//
// ⚠️ **The globs are absolute.** A relative `../../../` glob is resolved against
// the process's working directory rather than this file, and `wails build` does
// not run npm from where a person runs it — so a relative pattern matches
// nothing on one machine and everything on another, and the symptom is not an
// error but a stylesheet quietly missing every class the shared screens use.
// Same lesson as the CSS import next door, which failed loudly on Windows only.
// ⚠️ **Forward slashes, always.** fileURLToPath returns a backslash path on
// Windows, and fast-glob — what Tailwind scans `content` with — reads a
// backslash as an escape character. The pattern then matches nothing, and the
// failure is silent: a stylesheet with none of the shared screens' classes in
// it, which looks like a broken design rather than a broken path.
const here = (p: string) =>
  fileURLToPath(new URL(p, import.meta.url)).replace(/\\/g, "/");

const config: Config = {
  ...shared,
  content: [
    here("./index.html"),
    here("./src/**/*.{ts,tsx}"),
    // The shared screens, which is where nearly every class actually is.
    here("../../../frontend/src/app/kassa/**/*.{ts,tsx}"),
    here("../../../frontend/src/components/**/*.{ts,tsx}"),
    here("../../../frontend/src/lib/**/*.{ts,tsx}"),
  ],
};

export default config;

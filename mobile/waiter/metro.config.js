// Metro, pointed at the rules the web app already holds.
//
// ⚠️ **Nothing is copied and nothing is moved.** `frontend/src/lib` is around
// 28 000 lines of plain TypeScript — the offline queue, the marking codes, the
// service charge, the clock guard, three languages of text — and every one of
// those rules has to give the same answer on a phone as it does at a counter.
// A copy drifts, and this codebase has written down more than once which of the
// two copies ends up being the one in a restaurant.
//
// ⚠️ **Moving the files out into a package would be the tidier shape and the
// wrong first step.** It touches every import in the web app and the till, and
// doing that before we know the phone build works is a large risky change
// bought with nothing. Metro can watch a folder outside the project; that is
// enough, and it can be revisited once there is something to revisit.
const path = require("path");
const { getDefaultConfig } = require("expo/metro-config");

const config = getDefaultConfig(__dirname);

// The web app's source, two levels up.
const shared = path.resolve(__dirname, "../../frontend/src");

// ⚠️ Metro refuses to serve a file it is not watching, and the error it gives
// ("unable to resolve") reads like a typo in the import rather than a missing
// folder.
config.watchFolders = [shared];

// The same `@/` the web app and the Windows till already use, so a rule
// imported here is written exactly as it is over there.
config.resolver.extraNodeModules = {
  "@": shared,
};

// ⚠️ **The same two stand-ins the Windows till needs**, and they have to be
// swapped in a resolver rather than listed in `extraNodeModules`: that list is
// only consulted for requests originating **inside** the project, and these
// imports come from `frontend/src`, which is a watched folder outside it. The
// symptom is exact and misleading — "unable to resolve next/navigation" with
// the alias sitting right there in the config.
const shims = {
  "next/navigation": path.resolve(__dirname, "src/shims/next-navigation.ts"),
  "next/headers": path.resolve(__dirname, "src/shims/next-headers.ts"),
};

const resolve = config.resolver.resolveRequest;
config.resolver.resolveRequest = (context, moduleName, platform) => {
  const shim = shims[moduleName];
  if (shim) return { type: "sourceFile", filePath: shim };
  return (resolve ?? context.resolveRequest)(context, moduleName, platform);
};

// ⚠️ **One copy of React, and this is the line that guarantees it.** The shared
// folder sits outside this project, so Node's resolution would look for its
// dependencies in `frontend/node_modules` first — giving a second React, a
// second copy of hooks, and the "invalid hook call" that reads as a bug in a
// component rather than in a config file.
config.resolver.nodeModulesPaths = [path.resolve(__dirname, "node_modules")];

module.exports = config;

// A stand-in for `next/headers`, which only exists so the bundler can resolve
// it.
//
// ⚠️ **Every caller is already unreachable in the browser.** `lib/api.ts`
// imports this dynamically inside branches that return early when `window` is
// defined, and each one is wrapped in try/catch. Throwing is therefore the
// honest implementation: it can only be reached if somebody later calls a
// server-only path from the till, and a thrown error names the mistake at the
// call site instead of quietly returning an empty Host and sending the till's
// requests to the wrong tenant.
export function headers(): never {
  throw new Error("next/headers is server-only and unavailable in the till app");
}

export function cookies(): never {
  throw new Error("next/cookies is server-only and unavailable in the till app");
}

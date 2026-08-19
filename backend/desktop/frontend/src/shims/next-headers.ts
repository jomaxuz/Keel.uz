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
//
// ⚠️ The signatures match Next's shape rather than being `never`. Callers write
// `(await headers()).get("host")`, so a `never` return fails to compile in the
// shared code — which would make this shim a reason to edit files that are
// correct.
type ReadonlyHeaders = { get(name: string): string | null };
type ReadonlyCookies = { get(name: string): { value: string } | undefined };

export function headers(): Promise<ReadonlyHeaders> {
  throw new Error("next/headers is server-only and unavailable in the till app");
}

export function cookies(): Promise<ReadonlyCookies> {
  throw new Error("next/cookies is server-only and unavailable in the till app");
}

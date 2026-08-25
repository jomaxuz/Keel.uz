// A stand-in for `next/headers`, which exists only so the bundler can resolve
// it.
//
// ⚠️ **Every caller is already unreachable here.** `lib/api.ts` imports this
// dynamically inside branches that return early when there is no server render,
// and each is wrapped in try/catch. Throwing is therefore the honest
// implementation: reaching it means somebody called a server-only path from a
// phone, and a thrown error names the mistake at the call site instead of
// quietly returning an empty Host and sending requests to the wrong tenant.
type ReadonlyHeaders = { get(name: string): string | null };
type ReadonlyCookies = { get(name: string): { value: string } | undefined };

export function headers(): Promise<ReadonlyHeaders> {
  throw new Error("next/headers is server-only and unavailable in the app");
}

export function cookies(): Promise<ReadonlyCookies> {
  throw new Error("next/cookies is server-only and unavailable in the app");
}

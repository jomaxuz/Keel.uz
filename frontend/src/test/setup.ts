import "@testing-library/jest-dom/vitest";
// ⚠️ jsdom has no IndexedDB, and the till's offline queue is built on it. A
// stand-in rather than a mock of our own code: what has to be tested is that a
// payment survives a real store round-trip, not that a function was called.
import "fake-indexeddb/auto";

import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

import { currentStaff, setSignedInStaff } from "./staffMock";
import { tillApi } from "./tillServer";

// ---- The server ----
//
// ⚠️ Only the `api` object is replaced. `ApiError`, the token helpers and
// `imageUrl` stay real: the screens branch on `err instanceof ApiError` and
// keep the device token in localStorage and the person's in sessionStorage —
// facts these tests are partly here to hold in place.
vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return { ...actual, api: { ...actual.api, ...tillApi } };
});

// ---- The router ----
//
// Nothing under test navigates on purpose; a redirect to the login form is a
// failure the tests assert about, so it is recorded rather than performed.
export const routerCalls = { replace: [] as string[], push: [] as string[] };
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: (url: string) => routerCalls.replace.push(url),
    push: (url: string) => routerCalls.push.push(url),
    refresh: () => {},
    prefetch: () => {},
    back: () => {},
  }),
  usePathname: () => "/kassa",
  useSearchParams: () => new URLSearchParams(),
}));

// ---- The staff session ----
//
// The real provider watches geolocation on a timer, which a jsdom run has no
// business doing.
vi.mock("@/lib/staff", () => ({
  useStaff: () => ({
    staff: currentStaff(),
    workplace: null,
    openShift: null,
    loading: false,
    login: () => {},
    logout: () => {},
    reload: async () => {},
  }),
  StaffProvider: ({ children }: { children: React.ReactNode }) => children,
}));

afterEach(() => {
  cleanup();
  // ⚠️ The device token lives in localStorage and the person's in
  // sessionStorage — deliberately (lib/api.ts). A test that inherited either
  // would start on an unlocked till, which is the one state these tests exist
  // to prove you cannot reach by accident.
  window.localStorage.clear();
  window.sessionStorage.clear();
  // ⚠️ The offline queue outlives a page, which is its whole point — so it also
  // outlives a test unless it is cleared here, and a payment left over from the
  // previous case would make the next one pass for the wrong reason.
  indexedDB.deleteDatabase("keel-till");
  setSignedInStaff(null);
  routerCalls.replace.length = 0;
  routerCalls.push.length = 0;
  vi.clearAllMocks();
});

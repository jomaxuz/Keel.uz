// A stand-in for `next/navigation`, for the till screens running under Wails.
//
// ⚠️ **There is no router here, and that is the whole point.** The window shows
// one application with its own view state (`kassa` switches between the room
// and the menu itself); a history stack would give the monoblock a back button
// that leaves the till. So `push` changes the hash and nothing else listens —
// enough for the two call sites that use it to keep compiling and behaving
// sanely, and deliberately not enough to become a second navigation model.
//
// Used by: app/kassa/page.tsx (useRouter), lib/i18n/client.tsx (useRouter,
// usePathname). If a third caller appears, decide here rather than widening
// this file by reflex.

export type Router = {
  push: (href: string) => void;
  replace: (href: string) => void;
  back: () => void;
  forward: () => void;
  refresh: () => void;
  prefetch: (href: string) => void;
};

const router: Router = {
  push: (href) => {
    window.location.hash = href;
  },
  replace: (href) => {
    window.location.replace(`#${href}`);
  },
  back: () => window.history.back(),
  forward: () => window.history.forward(),
  // ⚠️ A no-op, not a reload. `refresh()` in Next re-fetches server components;
  // reloading the window here would throw away unsent offline sales.
  refresh: () => {},
  prefetch: () => {},
};

export function useRouter(): Router {
  return router;
}

export function usePathname(): string {
  return window.location.hash.replace(/^#/, "") || "/kassa";
}

export function useSearchParams(): URLSearchParams {
  return new URLSearchParams(window.location.search);
}

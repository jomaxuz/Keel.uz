// A stand-in for `next/navigation`, for the shared code running under React
// Native.
//
// ⚠️ **The same shim the Windows till already needs** (`backend/desktop/frontend
// /src/shims/next-navigation.tsx`), for the same reason and with the same
// shape: `lib/i18n` imports the router at module level, so the bundler has to
// resolve it even where nothing renders a router.
//
// ⚠️ **There is no router here, and that is deliberate.** This app holds one
// level of navigation in `App.tsx` — the room, and a check. A history stack
// would give a phone a back button that leaves the screen a waiter is standing
// at. Nothing calls these; they exist so the module graph resolves, and they
// are honest about doing nothing rather than pretending to navigate.

export type Router = {
  push: (href: string) => void;
  replace: (href: string) => void;
  refresh: () => void;
  back: () => void;
};

const router: Router = {
  push: () => {},
  replace: () => {},
  refresh: () => {},
  back: () => {},
};

export function useRouter(): Router {
  return router;
}

export function usePathname(): string {
  return "/";
}

export function useSearchParams(): URLSearchParams {
  return new URLSearchParams();
}

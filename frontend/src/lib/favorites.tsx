"use client";

// The hearts, in one place.
//
// ⚠️ **One provider, not per-card state.** A dish appears on the menu, in the popular
// band, in a category list and in the profile — four cards for the same id. With state
// per card, hearting one leaves the other three showing empty, and the guest's own second
// tap unhearts something that never looked hearted.
//
// ⚠️ **Optimistic, and rolled back on failure.** A heart is the cheapest interaction on
// the site and must feel instant; but a failed write that leaves the heart filled is a
// promise the profile then breaks, so the answer from the server is what stands.
//
// Loaded once for a signed-in guest, and never for anybody else: an anonymous visitor has
// no favourites, and asking for them on every page is a request that can only 401.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api } from "./api";
import { useUser } from "./user";

interface Ctx {
  ids: Set<string>;
  /** Whether this guest can have favourites at all. */
  enabled: boolean;
  toggle: (id: string) => Promise<void>;
}

const FavoritesContext = createContext<Ctx>({
  ids: new Set(),
  enabled: false,
  toggle: async () => {},
});

export function FavoritesProvider({ children }: { children: ReactNode }) {
  const { user } = useUser();
  const [ids, setIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!user) {
      // Signed out: the hearts go with the session rather than lingering as somebody
      // else's on a shared phone.
      setIds(new Set());
      return;
    }
    let cancelled = false;
    api
      .favorites()
      .then((items) => {
        if (!cancelled) setIds(new Set(items.map((i) => i.id)));
      })
      .catch(() => {
        // A failed load leaves the hearts empty rather than guessing. The next toggle
        // still works — the server answers with the whole list.
      });
    return () => {
      cancelled = true;
    };
  }, [user]);

  const toggle = useCallback(
    async (id: string) => {
      if (!user) return;
      const was = ids.has(id);
      setIds((prev) => {
        const next = new Set(prev);
        if (was) next.delete(id);
        else next.add(id);
        return next;
      });
      try {
        const res = await api.toggleFavorite(id);
        // The server's list wins: two devices, or a dish deleted since the page
        // loaded, and the optimistic guess is simply wrong.
        setIds(new Set(res.favorites));
      } catch {
        setIds((prev) => {
          const next = new Set(prev);
          if (was) next.add(id);
          else next.delete(id);
          return next;
        });
      }
    },
    [ids, user],
  );

  return (
    <FavoritesContext.Provider value={{ ids, enabled: !!user, toggle }}>
      {children}
    </FavoritesContext.Provider>
  );
}

export function useFavorites(): Ctx {
  return useContext(FavoritesContext);
}

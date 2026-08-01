"use client";

// Site customer session (phone + one-time SMS code). Persisted via the
// user_token in localStorage (see api.ts). Separate from the admin session.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api, clearUserToken, getUserToken, setUserToken } from "./api";
import type { SiteUser } from "./types";

interface UserContextValue {
  user: SiteUser | null;
  loading: boolean;
  login: (token: string, user: SiteUser) => void;
  logout: () => void;
  // Replace the cached profile after the customer edits it.
  update: (user: SiteUser) => void;
}

const UserContext = createContext<UserContextValue | null>(null);

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SiteUser | null>(null);
  const [loading, setLoading] = useState(true);

  // Restore the session on mount.
  useEffect(() => {
    if (!getUserToken()) {
      setLoading(false);
      return;
    }
    api
      .userMe()
      .then(setUser)
      .catch(() => {
        clearUserToken();
        setUser(null);
      })
      .finally(() => setLoading(false));
  }, []);

  const login = useCallback((token: string, u: SiteUser) => {
    setUserToken(token);
    setUser(u);
  }, []);

  const logout = useCallback(() => {
    clearUserToken();
    setUser(null);
  }, []);

  const update = useCallback((u: SiteUser) => setUser(u), []);

  return (
    <UserContext.Provider value={{ user, loading, login, logout, update }}>
      {children}
    </UserContext.Provider>
  );
}

export function useUser(): UserContextValue {
  const ctx = useContext(UserContext);
  if (!ctx) throw new Error("useUser must be used within a UserProvider");
  return ctx;
}

// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **The address is remembered, the person is remembered, nothing else is.**
// A courier's phone is theirs — signing in every shift would be the fastest way
// to have the password written on the back of the case. The restaurant's
// address is set once, when the phone is handed over or when the courier is
// told which restaurant they now ride for.
//
// ⚠️ **The shift status is *not* kept here.** It lives on the server, because
// it is not a fact about this phone: a dispatcher can take a courier off shift
// from the panel, and a phone that remembered its own answer would keep
// reporting "available" to a screen that had already said otherwise.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError, setCourierToken, clearCourierToken } from "@/lib/api";
import { apiBaseFor, uploadsBaseFor } from "@/lib/serverAddress";
import { setApiBase } from "@/lib/tokenStore";
import type { Courier, CourierStatus } from "@/lib/types";

import { hydrateTokens, readSaved, saveValue, dropValue } from "./tokens";

/** ⚠️ Not a token, so it lives beside them rather than among them: the address
 *  is not a credential and reading it out of the device tells nobody anything
 *  they could not learn by opening the restaurant's website. */
const ADDRESS_KEY = "keel_server_address";

export type Session =
  | { state: "loading" }
  | { state: "noServer" }
  // ⚠️ **Not the same as being signed out.** A launch asks the server who this
  // is; when the request never arrives the answer used to be "signed out", so a
  // courier in a basement, on a dead network or out of data met a password
  // field — types the right password, watches it fail, and blames themselves
  // for a network they cannot see. Nothing on that screen could have helped.
  | { state: "offline"; address: string }
  | { state: "signedOut"; address: string }
  | { state: "ready"; address: string; courier: Courier };

export function useSession() {
  const [session, setSession] = useState<Session>({ state: "loading" });

  /** Ask the server who this is — the one question a launch has to answer.
   *
   *  ⚠️ **Three outcomes, not two.** "The server says no" and "the server did
   *  not answer" look identical from a `catch` and mean opposite things: one is
   *  a sign-in, the other is a network. `ApiError` means the server spoke — any
   *  status, 401 included — and anything else means the request never arrived. */
  const probe = useCallback(async () => {
    // ⚠️ **Before the first request.** A screen that fetched while the store
    // was still empty would read the token as absent and send somebody to a
    // login they had already passed — and only on a cold start, which is the
    // hardest kind of bug to be shown.
    await hydrateTokens();
    const address = readSaved(ADDRESS_KEY);
    if (!address) {
      setSession({ state: "noServer" });
      return;
    }
    setApiBase(apiBaseFor(address), uploadsBaseFor(address));
    try {
      const me = await api.courierMe();
      setSession({ state: "ready", address, courier: me });
    } catch (e) {
      if (!(e instanceof ApiError)) {
        setSession({ state: "offline", address });
        return;
      }
      // ⚠️ A refused token is a sign-in, not an error screen. It expires, or
      // the account was switched off — and both have the same answer for the
      // person holding the phone.
      setSession({ state: "signedOut", address });
    }
  }, []);

  useEffect(() => {
    void probe();
  }, [probe]);

  const useServer = useCallback((address: string) => {
    const base = apiBaseFor(address);
    if (!base) return false;
    saveValue(ADDRESS_KEY, address);
    setApiBase(base, uploadsBaseFor(address));
    setSession({ state: "signedOut", address });
    return true;
  }, []);

  const signIn = useCallback(
    async (address: string, username: string, password: string) => {
      const res = await api.courierLogin(username, password);
      setCourierToken(res.token);
      setSession({ state: "ready", address, courier: res.courier });
    },
    [],
  );

  const signOut = useCallback((address: string) => {
    clearCourierToken();
    setSession({ state: "signedOut", address });
  }, []);

  /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
   *  when a courier changes employer; signing out is the end of a shift, and
   *  answering both with one button would make the common one cost the rare
   *  one's setup. */
  const forgetServer = useCallback(() => {
    clearCourierToken();
    dropValue(ADDRESS_KEY);
    setApiBase("", "");
    setSession({ state: "noServer" });
  }, []);

  /** The shift switch. The server is told first and the screen follows it —
   *  the reverse order shows a courier as available on a phone that the
   *  dispatcher's list still has as off. */
  const setStatus = useCallback(async (status: CourierStatus) => {
    await api.courierSetStatus(status);
    setSession((s) =>
      s.state === "ready"
        ? { ...s, courier: { ...s.courier, status } }
        : s,
    );
  }, []);

  /** Re-read the profile. ⚠️ Called after every list refresh, because the
   *  status can change without this phone doing anything: finishing the last
   *  order flips a courier back to free on the server (`syncCourierBusy`), and
   *  a switch that disagrees with the server is the one thing on this screen
   *  somebody would press twice. */
  const refresh = useCallback(async () => {
    try {
      const me = await api.courierMe();
      setSession((s) => (s.state === "ready" ? { ...s, courier: me } : s));
    } catch {
      // A dropped request is not a sign-out: the courier is riding through a
      // basement, and the next poll will answer.
    }
  }, []);

  return {
    session,
    useServer,
    signIn,
    signOut,
    forgetServer,
    setStatus,
    refresh,
    retry: probe,
  };
}

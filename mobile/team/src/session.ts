// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **The address is remembered, the person is remembered, the unlock is not.**
// A waiter's phone is theirs — signing in every shift would be the fastest way
// to have the password written on the back of the case. The restaurant's
// address is set once when the phone is handed over. What is *not* kept is the
// till session token, which belongs to a sitting at a screen (see
// `lib/tokenStore.ts`).

import { useCallback, useEffect, useState } from "react";

import { api, ApiError, setStaffToken, clearStaffToken } from "@/lib/api";
import { apiBaseFor, uploadsBaseFor } from "@/lib/serverAddress";
import { setApiBase } from "@/lib/tokenStore";
import type { Staff } from "@/lib/types";

import { hydrateTokens, readSaved, saveValue, dropValue } from "./tokens";

/** ⚠️ Not a token, so it lives beside them rather than among them: the address
 *  is not a credential and reading it out of the device tells nobody anything
 *  they could not learn by opening the restaurant's website. */
const ADDRESS_KEY = "keel_server_address";

export type Session =
  | { state: "loading" }
  | { state: "noServer" }
  // ⚠️ **Not the same as being signed out, and telling them apart is the whole
  // of this state.** A phone opened in a basement, on a dead wifi or before the
  // shift's data ran out reached the login screen — where the password is typed
  // correctly, the request fails, and the message is "could not sign in". The
  // waiter then types it again, and again, blaming themselves for a network
  // they cannot see. Nothing here can be fixed by signing in, so nothing here
  // offers to.
  | { state: "offline"; address: string }
  | { state: "signedOut"; address: string }
  | { state: "ready"; address: string; staff: Staff };

export function useSession() {
  const [session, setSession] = useState<Session>({ state: "loading" });

  // ⚠️ **Everything here happens before the first request.** A screen that
  // fetched while the store was still empty would read every token as absent
  // and send somebody to a login they had already passed — and only on a cold
  // start, which is the hardest kind of bug to be shown.
  /** Ask the server who this is. The one question a launch has to answer.
   *
   *  ⚠️ **Three outcomes, not two.** "The server says no" and "the server did
   *  not answer" look identical from a `catch` and mean opposite things: one is
   *  a sign-in, the other is a network. `ApiError` is the server having spoken
   *  — any status, including 401 — and anything else is the request never
   *  having arrived. */
  const probe = useCallback(async () => {
    // Already resolved by App before anything rendered; awaited here so this
    // hook stays correct if it is ever mounted somewhere else.
    await hydrateTokens();
    const address = readSaved(ADDRESS_KEY);
    if (!address) {
      setSession({ state: "noServer" });
      return;
    }
    setApiBase(apiBaseFor(address), uploadsBaseFor(address));
    try {
      const me = await api.staffMe();
      setSession({ state: "ready", address, staff: me.staff });
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
      const res = await api.staffLogin(username, password);
      setStaffToken(res.token);
      setSession({ state: "ready", address, staff: res.staff });
    },
    [],
  );

  const signOut = useCallback((address: string) => {
    clearStaffToken();
    setSession({ state: "signedOut", address });
  }, []);

  /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
   *  when a phone moves to another job; signing out is the end of a shift, and
   *  answering both with one button would make the common one cost the rare
   *  one's setup. */
  const forgetServer = useCallback(() => {
    clearStaffToken();
    dropValue(ADDRESS_KEY);
    setApiBase("", "");
    setSession({ state: "noServer" });
  }, []);

  return { session, useServer, signIn, signOut, forgetServer, retry: probe };
}

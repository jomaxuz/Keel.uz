// What this television is: which restaurant, and whether it has been paired.
//
// ⚠️ **Four states, and the fourth is the one that matters in a restaurant.**
// "Not paired" and "cannot reach the server" look identical from a `catch` and
// mean opposite things: one is answered by somebody typing a code into the
// panel, the other by the wifi coming back. A television that showed a pairing
// code every time the internet blinked would be re-paired by a manager who then
// learns to distrust the screen entirely.

import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError, api, clearTVToken, getTVToken, setTVToken } from "@/lib/api";
import { apiBaseFor, uploadsBaseFor } from "@/lib/serverAddress";
import { setApiBase } from "@/lib/tokenStore";
import type { TVScreenSelf } from "@/lib/types";

import { ADDRESS_KEY, hydrate, installID, readSaved, saveValue } from "./store";

/** How often a paired screen says hello.
 *
 *  ⚠️ **This is also how quickly "unpair" takes effect on the wall**, so it is
 *  a minute rather than an hour — and it is not shorter, because it runs all
 *  day on a restaurant's wifi for months. */
const HEARTBEAT_MS = 60_000;

/** How often an unpaired screen asks for a fresh code, and how often it checks
 *  whether somebody has claimed it.
 *
 *  ⚠️ The code is on a wall in a public room; ten seconds is what makes
 *  photographing it pointless. The poll is quicker because it decides how long
 *  a manager stands in front of the television waiting for it to change. */
const CODE_MS = 10_000;
const POLL_MS = 2_000;

export type TVState =
  | { state: "loading" }
  | { state: "noServer" }
  // Reached the server, not paired yet: this is the code on the screen.
  | { state: "pairing"; address: string; code: string; expiresAt: number }
  // ⚠️ Paired and currently unreachable. **Not** an error screen: the whole
  // point of this app is that a dining room keeps working when the wifi does
  // not, and a paired television with nothing to say should say nothing.
  | { state: "offline"; address: string; screen: TVScreenSelf | null }
  | { state: "paired"; address: string; screen: TVScreenSelf };

export function useTVSession(appVersion: string) {
  const [state, setState] = useState<TVState>({ state: "loading" });
  // The secret this television polls with, for the code currently on screen.
  const poll = useRef<{ install: string; secret: string } | null>(null);
  // The last thing the server told us about this screen, kept across a dropped
  // connection so an offline set can still say which room it belongs to.
  const known = useRef<TVScreenSelf | null>(null);

  /** Point the shared rules at this restaurant's server. */
  const useServer = useCallback((address: string) => {
    const base = apiBaseFor(address);
    if (!base) return false;
    saveValue(ADDRESS_KEY, address);
    setApiBase(base, uploadsBaseFor(address));
    setState({ state: "loading" });
    return true;
  }, []);

  /** Ask for a code to show. */
  const askForCode = useCallback(async (address: string) => {
    try {
      const install = installID();
      const res = await api.tvPairStart({ installId: install, appVersion });
      poll.current = { install, secret: res.pollSecret };
      setState({
        state: "pairing",
        address,
        code: res.code,
        // ⚠️ Counted from *our* clock plus the server's number of seconds,
        // never from the server's timestamp against this device's clock: a
        // cheap television's clock is routinely months out, and the countdown
        // would either sit at zero or never move.
        expiresAt: Date.now() + res.expiresIn * 1000,
      });
    } catch {
      // The server is unreachable. ⚠️ An unpaired television with no server has
      // nothing useful to draw, so it keeps the code it had — a screen that
      // flickered between a code and an error would be unreadable from across a
      // room, which is the only distance it is ever read from.
      setState((prev) =>
        prev.state === "pairing"
          ? prev
          : { state: "offline", address, screen: null },
      );
    }
  }, [appVersion]);

  /** Am I still paired, and who am I? */
  const heartbeat = useCallback(
    async (address: string) => {
      try {
        const res = await api.tvMe(appVersion);
        known.current = res.screen;
        setState({ state: "paired", address, screen: res.screen });
      } catch (e) {
        if (e instanceof ApiError) {
          // ⚠️ **The server spoke, and it said no.** The screen was unpaired
          // from the panel — so the token goes now and the television asks for
          // a code again. Retrying would leave a set that was deliberately
          // taken out of service still playing.
          clearTVToken();
          poll.current = null;
          void askForCode(address);
          return;
        }
        // The request never arrived. Keep playing, keep what we know.
        setState({ state: "offline", address, screen: known.current });
      }
    },
    [appVersion, askForCode],
  );

  // ---- Startup ----
  useEffect(() => {
    let alive = true;
    void (async () => {
      await hydrate();
      if (!alive) return;
      const address = readSaved(ADDRESS_KEY);
      if (!address) {
        setState({ state: "noServer" });
        return;
      }
      setApiBase(apiBaseFor(address), uploadsBaseFor(address));
      if (getTVToken()) {
        await heartbeat(address);
      } else {
        await askForCode(address);
      }
    })();
    return () => {
      alive = false;
    };
  }, [askForCode, heartbeat]);

  // ---- While unpaired: a fresh code, and a poll for the claim ----
  useEffect(() => {
    if (state.state !== "pairing") return;
    const address = state.address;

    const rotate = setInterval(() => void askForCode(address), CODE_MS);
    const check = setInterval(() => {
      const p = poll.current;
      if (!p) return;
      void api
        .tvPairStatus(p.install, p.secret)
        .then((res) => {
          if (res.status === "paired" && res.token && res.screen) {
            setTVToken(res.token);
            known.current = res.screen;
            poll.current = null;
            setState({ state: "paired", address, screen: res.screen });
          }
          // ⚠️ "expired" is not handled here on purpose: the rotation above is
          // already asking for a new code every ten seconds, and reacting to it
          // as well would ask for two.
        })
        .catch(() => {
          // Offline while waiting to be paired. The code on the screen is
          // already dead; the next rotation will say so by failing too, and
          // there is nothing for a person to do about it from the remote.
        });
    }, POLL_MS);

    return () => {
      clearInterval(rotate);
      clearInterval(check);
    };
  }, [state, askForCode]);

  // ---- While paired: say hello, and notice being unpaired ----
  useEffect(() => {
    if (state.state !== "paired" && state.state !== "offline") return;
    const address = state.address;
    const id = setInterval(() => void heartbeat(address), HEARTBEAT_MS);
    return () => clearInterval(id);
  }, [state, heartbeat]);

  return { state, useServer };
}

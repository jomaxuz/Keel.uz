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

import { noteServerTime, restoreClock } from "./clock";
import { ADDRESS_KEY, hydrate, installID, readSaved, saveValue } from "./store";

/** How often a paired screen says hello.
 *
 *  ⚠️ **This is also how quickly "unpair" takes effect on the wall**, so it is
 *  a minute rather than an hour — and it is not shorter, because it runs all
 *  day on a restaurant's wifi for months. */
const HEARTBEAT_MS = 60_000;

/** How often the screen asks whether somebody has claimed it.
 *
 *  ⚠️ Quick, because it decides how long a manager stands in front of the
 *  television after typing the code, wondering whether it worked. */
const POLL_MS = 2_000;

/** How long before a code expires the next one is asked for.
 *
 *  ⚠️ **The rotation is driven by the server's own expiry, not by a second
 *  timer here.** It used to be a constant in this file — ten seconds — while
 *  the server said ninety, so the countdown on the wall was a lie and the code
 *  changed six times inside its own stated lifetime. One number, sent with the
 *  code, and the app's only decision is when to ask for the next one. */
const RENEW_LEAD_MS = 5_000;

export type TVState =
  | { state: "loading" }
  | { state: "noServer" }
  // Reached the server, not paired yet: this is the code on the screen.
  | { state: "pairing"; address: string; code: string; expiresAt: number }
  // ⚠️ Paired and currently unreachable. **Not** an error screen: the whole
  // point of this app is that a dining room keeps working when the wifi does
  // not, and a paired television with nothing to say should say nothing.
  | {
      state: "offline";
      address: string;
      screen: TVScreenSelf | null;
      // ⚠️ Carried across the drop, not reset: the playlist on this set is
      // still the branch's playlist, one revision behind at worst, and clearing
      // this would make the screen re-download everything the moment the wifi
      // came back — over the wifi that has just been struggling.
      contentVersion: number | null;
    }
  // ⚠️ **Never paired, and the address answers nothing.** Deliberately not the
  // same as `offline`: there is nothing to keep playing and nobody to wait for.
  // Somebody is standing in front of this screen right now with a remote, and
  // the only useful thing it can do is show the address back to them — a
  // mistyped restaurant name is by far the likeliest cause, and on a D-pad it
  // has to come back filled in rather than empty.
  | { state: "unreachable"; address: string; answered: boolean }
  | {
      state: "paired";
      address: string;
      screen: TVScreenSelf;
      /** The branch's playlist revision. ⚠️ Null only until the first
       *  heartbeat lands — which is exactly the cold-boot-with-no-network case,
       *  and why the stored playlist is read before anything is asked. */
      contentVersion: number | null;
    };

export function useTVSession(appVersion: string) {
  const [state, setState] = useState<TVState>({ state: "loading" });
  // The secret this television polls with, for the code currently on screen.
  const poll = useRef<{ install: string; secret: string } | null>(null);
  // The last thing the server told us about this screen, kept across a dropped
  // connection so an offline set can still say which room it belongs to.
  const known = useRef<TVScreenSelf | null>(null);
  // The last playlist revision the server named, kept for the same reason.
  const version = useRef<number | null>(null);

  /** Ask for a code to show. */
  const askForCode = useCallback(
    async (address: string) => {
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
      } catch (e) {
        // The server did not answer — or it did, and had nothing to answer with.
        //
        // ⚠️ **Those two are told apart, because they send somebody to different
        // people.** `ApiError` means the address is right and reachable and the
        // *endpoint* refused — on this app that almost always means the
        // restaurant's server has not been updated yet, which nobody standing on
        // a chair can fix. Anything else is a name, a router or a cable, which is
        // exactly what they can.
        const answered = e instanceof ApiError;
        //
        // ⚠️ **A code already on the screen stays there.** A television that
        // flickered between a code and an error would be unreadable from the only
        // distance it is ever read from — and the code may well still be good
        // when the wifi comes back mid-rotation.
        //
        // ⚠️ With no code yet this is **not** treated as being offline: an
        // unpaired set has nothing to keep playing, and the likeliest cause is a
        // mistyped restaurant name — which only a person standing there can fix.
        setState((prev) =>
          prev.state === "pairing"
            ? prev
            : { state: "unreachable", address, answered },
        );
      }
    },
    [appVersion],
  );

  /** Am I still paired, and who am I? */
  const heartbeat = useCallback(
    async (address: string) => {
      try {
        const res = await api.tvMe(appVersion);
        known.current = res.screen;
        version.current = res.contentVersion;
        // ⚠️ **Every heartbeat, not only the playlist fetch.** A dated slide is
        // compared against the restaurant's clock, and this set's own is not
        // usable for it — see clock.ts. The playlist is re-read only when it
        // changes, which on a normal day is never, so this is the only thing
        // that keeps the time honest on a screen left running for a month.
        noteServerTime(res.serverTime);
        setState({
          state: "paired",
          address,
          screen: res.screen,
          contentVersion: res.contentVersion,
        });
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
        setState({
          state: "offline",
          address,
          screen: known.current,
          contentVersion: version.current,
        });
      }
    },
    [appVersion, askForCode],
  );

  /** Start talking to a restaurant's server: the token if we have one, a
   *  pairing code if we do not.
   *
   *  ⚠️ **One function, called from the launch *and* from the address form**,
   *  and it exists because the second caller was missing. Typing the address
   *  set the state to "loading" and stopped: the startup effect had already
   *  run, nothing was listening for the change, and the television sat on the
   *  loading screen — which draws nothing. A black screen, on a wall, with no
   *  code and no way back. It was the first thing anybody saw of this app. */
  const boot = useCallback(
    async (address: string) => {
      setApiBase(apiBaseFor(address), uploadsBaseFor(address));
      if (getTVToken()) {
        await heartbeat(address);
        return;
      }
      await askForCode(address);
    },
    [askForCode, heartbeat],
  );

  /** Point this television at a restaurant. */
  const useServer = useCallback(
    (address: string) => {
      const base = apiBaseFor(address);
      if (!base) return false;
      saveValue(ADDRESS_KEY, address);
      setState({ state: "loading" });
      void boot(address);
      return true;
    },
    [boot],
  );

  // ---- Startup ----
  useEffect(() => {
    let alive = true;
    void (async () => {
      await hydrate();
      if (!alive) return;
      // The last time the server told us, before anything is drawn: the first
      // decision this app makes about a dated slide happens before the first
      // heartbeat, and a set that lost power may have come back in 1970.
      restoreClock();
      const address = readSaved(ADDRESS_KEY);
      if (!address) {
        setState({ state: "noServer" });
        return;
      }
      await boot(address);
    })();
    return () => {
      alive = false;
    };
  }, [boot]);

  // ---- While unpaired: a fresh code, and a poll for the claim ----
  useEffect(() => {
    if (state.state !== "pairing") return;
    const address = state.address;

    // ⚠️ A timeout, not an interval, and rescheduled from every new code: the
    // window is the server's to decide, and a fixed interval here would drift
    // apart from it the first time it changed.
    const wait = Math.max(
      RENEW_LEAD_MS,
      state.expiresAt - Date.now() - RENEW_LEAD_MS,
    );
    const rotate = setTimeout(() => void askForCode(address), wait);
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
            // ⚠️ Null rather than zero: "not asked yet" and "the branch has an
            // empty playlist" are different, and the heartbeat a moment from
            // now is what tells them apart. Zero here would mean a freshly
            // paired screen never fetched its content at all.
            setState({
              state: "paired",
              address,
              screen: res.screen,
              contentVersion: null,
            });
          }
          // ⚠️ "expired" is not handled here on purpose: the rotation above
          // already asks for a new code as this one runs out, and reacting to
          // it here as well would ask for two.
        })
        .catch(() => {
          // Offline while waiting to be paired. The code on the screen is
          // already dead; the next rotation will say so by failing too, and
          // there is nothing for a person to do about it from the remote.
        });
    }, POLL_MS);

    return () => {
      clearTimeout(rotate);
      clearInterval(check);
    };
  }, [state, askForCode]);

  // ---- While unreachable: keep trying, quietly ----
  //
  // ⚠️ **Because the commonest cause is not a typo but a television that was
  // switched on before the router.** Somebody opens the restaurant, everything
  // comes on at once, and this app reaches the network a few seconds early. A
  // screen that needed a person to press something after that would need one
  // every morning.
  useEffect(() => {
    if (state.state !== "unreachable") return;
    const address = state.address;
    const id = setInterval(() => void boot(address), 15_000);
    return () => clearInterval(id);
  }, [state, boot]);

  // ---- While paired: say hello, and notice being unpaired ----
  useEffect(() => {
    if (state.state !== "paired" && state.state !== "offline") return;
    const address = state.address;
    const id = setInterval(() => void heartbeat(address), HEARTBEAT_MS);
    return () => clearInterval(id);
  }, [state, heartbeat]);

  return { state, useServer };
}

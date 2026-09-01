// Where this phone keeps a token, and how the app is told which restaurant it
// belongs to.
//
// ⚠️ **The one adapter, and it carries the asymmetry by itself.** The rules ask
// for a token synchronously — `request()` reads one on the way into every call
// — and every store a phone has is asynchronous. Rather than making thirty call
// sites await something almost always already known, this reads the whole set
// once at startup into memory and writes through behind itself. That is the
// bargain `lib/tokenStore.ts` describes, and this is the file that pays it.
//
// ⚠️ **SecureStore, not AsyncStorage.** This token closes orders and settles
// cash: on Android it lands in the keystore-backed store and on iOS in the
// keychain, which is where a credential belongs. AsyncStorage is a plain file
// readable by anything that gets at the sandbox — acceptable for a remembered
// filter, not for something that can mark a delivery paid for.
//
// ⚠️ **A courier's phone is the one most likely to be lost or resold**, which
// is the other half of the same argument: signing out has to actually remove
// the token, and `remove` below drops the in-memory copy first for that reason.

import * as SecureStore from "expo-secure-store";

import { setSessionStore, setTokenStore } from "@/lib/tokenStore";

/** Every key the rules may ask for.
 *
 *  ⚠️ **Named here rather than discovered.** Hydration happens once, before the
 *  first request, so a key nobody listed would read as absent on launch and
 *  send somebody to a login they had already passed — and only on a cold start,
 *  which is the hardest kind of bug to be shown. The list is short and the
 *  compiler cannot check it, so it is written next to the reason.
 *
 *  ⚠️ Same spellings as the browser's: a token is written by one platform and
 *  read by the other in exactly one case that matters — a QR handoff — and a
 *  divergent name would fail silently there. */
const KEYS = [
  // The courier's own session. ⚠️ Spelled exactly as the browser spells it:
  // the rules in `lib/api.ts` read this name, and a divergent one here would
  // sign somebody in and then send every request unauthenticated.
  "courier_token",
  // ⚠️ Not a token, and hydrated with them anyway: it is read on the same cold
  // start, before the first request, and a second mechanism for one launch
  // would be a second thing to forget.
  "keel_server_address",
  // The interface language and the appearance choice. ⚠️ Hydrated with the rest
  // because the first render has to be in the right language: a screen that
  // painted Uzbek and then switched to Russian is a flash somebody reads as a
  // fault.
  "keel_lang",
  "keel_theme",
] as const;

/** ⚠️ SecureStore keys are restricted to letters, digits, `.`, `-` and `_`,
 *  which every name above already satisfies. Checked rather than assumed,
 *  because the failure is a throw on write — at sign-in, on a real phone,
 *  where a test never looks. */
const VALID = /^[A-Za-z0-9._-]+$/;

const memory = new Map<string, string>();

/** ⚠️ **One hydration for the process, and everybody waits on the same one.**
 *  Two callers each starting their own would race: the second overwrites the
 *  first's store while a screen is already reading from it. */
let hydrating: Promise<void> | null = null;

/** Read what was saved. Awaited before anything reads a token, a language or a
 *  theme — see the note in App.tsx about why that ordering is the whole bug it
 *  once was. */
export function hydrateTokens(): Promise<void> {
  hydrating ??= hydrate();
  return hydrating;
}

async function hydrate(): Promise<void> {
  await Promise.all(
    KEYS.map(async (key) => {
      try {
        const value = await SecureStore.getItemAsync(key);
        if (value !== null) memory.set(key, value);
      } catch {
        // ⚠️ A store that will not open is not a reason to refuse to start:
        // the app asks for the password again, which is the correct outcome
        // and the one somebody standing at a gate can act on.
      }
    }),
  );

  setTokenStore({
    get: (key) => memory.get(key) ?? null,
    set: (key, value) => {
      memory.set(key, value);
      if (!VALID.test(key)) return;
      // ⚠️ Not awaited, and the memory copy is written first. The caller is a
      // sign-in that is about to navigate; the disk write is what makes the
      // next launch remember, not what makes this one work.
      void SecureStore.setItemAsync(key, value).catch(() => {
        // Signed in for this session, forgotten on the next launch.
      });
    },
    remove: (key) => {
      memory.delete(key);
      void SecureStore.deleteItemAsync(key).catch(() => {
        // ⚠️ Failing to delete leaves a token on the device after a sign-out,
        // which is the one direction that is not safe — so the in-memory copy
        // is dropped first and unconditionally. This session is signed out
        // whatever the disk does.
      });
    },
  });

  // ⚠️ **Set even though this app never writes to it.** `lib/api.ts` is shared
  // with the till and the panel, and it reads the session store on paths this
  // app does not take; leaving it unset would make an unrelated helper throw
  // the day somebody imports one more rule from over there.
  const sessionRows = new Map<string, string>();
  setSessionStore({
    get: (key) => sessionRows.get(key) ?? null,
    set: (key, value) => void sessionRows.set(key, value),
    remove: (key) => void sessionRows.delete(key),
  });
}

/** Read something already hydrated. Used for the server address, which is not
 *  a credential but is read on the same cold start. */
export function readSaved(key: string): string | null {
  return memory.get(key) ?? null;
}

/** Save something that must survive a restart. */
export function saveValue(key: string, value: string): void {
  memory.set(key, value);
  if (!VALID.test(key)) return;
  void SecureStore.setItemAsync(key, value).catch(() => {
    // Kept for this launch, asked again on the next.
  });
}

export function dropValue(key: string): void {
  memory.delete(key);
  void SecureStore.deleteItemAsync(key).catch(() => {
    // The in-memory copy is gone either way, so this launch has forgotten it.
  });
}

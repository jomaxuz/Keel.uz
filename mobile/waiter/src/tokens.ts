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
// ⚠️ **SecureStore, not AsyncStorage.** These tokens are a restaurant's till
// and its takings: on Android they land in the keystore-backed store and on iOS
// in the keychain, which is where a credential belongs. AsyncStorage is a plain
// file readable by anything that gets at the sandbox — acceptable for a
// remembered filter, not for something that can open a shift.

import * as SecureStore from "expo-secure-store";

import { setTokenStore } from "@/lib/tokenStore";

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
  "admin_token",
  "user_token",
  "courier_token",
  "staff_token",
  "kiosk_token",
  "keel_till_token",
  "keel_till_device",
] as const;

/** ⚠️ SecureStore keys are restricted to letters, digits, `.`, `-` and `_`,
 *  which every name above already satisfies. Checked rather than assumed,
 *  because the failure is a throw on write — at sign-in, on a real phone,
 *  where a test never looks. */
const VALID = /^[A-Za-z0-9._-]+$/;

const memory = new Map<string, string>();

/** Read what was saved. Called once, and awaited before anything is fetched. */
export async function hydrateTokens(): Promise<void> {
  await Promise.all(
    KEYS.map(async (key) => {
      try {
        const value = await SecureStore.getItemAsync(key);
        if (value !== null) memory.set(key, value);
      } catch {
        // ⚠️ A store that will not open is not a reason to refuse to start: the
        // app asks for the PIN again, which is the correct outcome and the one
        // somebody standing in a dining room can act on.
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
}

// Where this television keeps what it must not forget.
//
// ⚠️ **The same adapter shape as the phone apps** (`mobile/team/src/tokens.ts`):
// the shared rules read a token synchronously on the way into every request,
// and every store a device has is asynchronous. So the whole set is read once
// at startup into memory and written through behind itself.
//
// ⚠️ **Three things survive a restart, and each for a different reason.** The
// server address, because a television is set up once and then nobody touches
// it for months. The token, because a set that had to be re-paired every reboot
// would be re-paired by nobody — it hangs above head height. And the install
// id, because it is what makes a re-pair replace this screen's row instead of
// spending a second paid slot.

import * as Crypto from "expo-crypto";
import * as SecureStore from "expo-secure-store";

import { setSessionStore, setTokenStore } from "@/lib/tokenStore";

/** Everything read on the cold start, before the first request.
 *
 *  ⚠️ Named rather than discovered: hydration happens once, and a key nobody
 *  listed reads as absent on launch — which on this app means a paired
 *  television showing a pairing code again, in a dining room, at opening time. */
const KEYS = ["tv_token", "keel_server_address", "keel_tv_install"] as const;

export const ADDRESS_KEY = "keel_server_address";
export const INSTALL_KEY = "keel_tv_install";

const memory = new Map<string, string>();

/** ⚠️ One hydration for the process, and everybody waits on the same one: two
 *  callers each starting their own would race, and the second would overwrite
 *  the first's store while a screen was already reading from it. */
let hydrating: Promise<void> | null = null;

export function hydrate(): Promise<void> {
  hydrating ??= run();
  return hydrating;
}

async function run(): Promise<void> {
  await Promise.all(
    KEYS.map(async (key) => {
      try {
        const value = await SecureStore.getItemAsync(key);
        if (value !== null) memory.set(key, value);
      } catch {
        // ⚠️ A store that will not open is not a reason to refuse to start. The
        // television shows a pairing code, which is wrong but recoverable by
        // somebody standing in the room — a black screen is neither.
      }
    }),
  );

  setTokenStore({
    get: (key) => memory.get(key) ?? null,
    set: (key, value) => {
      memory.set(key, value);
      void SecureStore.setItemAsync(key, value).catch(() => {
        // Paired for this launch, asked again on the next one.
      });
    },
    remove: (key) => {
      memory.delete(key);
      void SecureStore.deleteItemAsync(key).catch(() => {
        // ⚠️ The memory copy is dropped first and unconditionally: leaving a
        // token on a device after it was unpaired is the one direction that is
        // not safe.
      });
    },
  });

  // Nothing on this device belongs to a "sitting" — there is no person to sign
  // out. The session store is wired to memory so the shared rules have one.
  const perLaunch = new Map<string, string>();
  setSessionStore({
    get: (key) => perLaunch.get(key) ?? null,
    set: (key, value) => void perLaunch.set(key, value),
    remove: (key) => void perLaunch.delete(key),
  });
}

export function readSaved(key: string): string | null {
  return memory.get(key) ?? null;
}

export function saveValue(key: string, value: string): void {
  memory.set(key, value);
  void SecureStore.setItemAsync(key, value).catch(() => {});
}

/** This television's own id, made once and kept for the life of the install.
 *
 *  ⚠️ **Not a device serial or a MAC address**, both of which are either
 *  unavailable, shared between identical cheap sets, or a privacy question we
 *  have no reason to open. A random id answers the only thing we ask of it: is
 *  this the same television that was paired before, so that re-pairing replaces
 *  its row rather than adding one.
 *
 *  ⚠️ It is generated **after** hydration, so a set that already has one keeps
 *  it: a new id on every launch would leave a trail of paid screen rows nobody
 *  can match to a wall. */
export function installID(): string {
  const saved = readSaved(INSTALL_KEY);
  if (saved) return saved;
  const fresh = Crypto.randomUUID();
  saveValue(INSTALL_KEY, fresh);
  return fresh;
}

import * as Application from "expo-application";
import * as Device from "expo-device";
import { Platform } from "react-native";

import { setDevice } from "@/lib/api";

import { readSaved, saveValue } from "./tokens";

// Which install this is.
//
// ⚠️ **The server binds an account to this id**, which is what stops one login
// being shared between two people and one phone being used for two accounts
// (`handlers/logindevice.go`). The whole feature rests on this value being
// stable for as long as the app is installed, and on being *ours*: Android has
// not handed out a device serial to ordinary apps for years, and asking for one
// would be asking for an identifier we have no business keeping.
//
// ⚠️ **A reinstall mints a new one, and that is the accepted cost.** The panel
// has a button that releases a binding, and it exists precisely for this — a
// lock nobody can lift would turn "I reinstalled the app" into a phone call to
// us. Anything more stable than a stored value is either unavailable, unstable
// in its own way (Android ids change on a factory reset anyway), or a privacy
// problem we would rather not have.
//
// ⚠️ **Kept in SecureStore beside the tokens**, hydrated with them: it is read
// before the first request, and a second storage mechanism for one value would
// be a second thing to forget.

const KEY = "keel_device_id";

/** Mint the id if this install has none, and tell `lib/api` who is asking.
 *
 *  Called once at startup, after the tokens are hydrated and before anything
 *  makes a request — the login is exactly the call that has to carry it. */
export function initDevice(app: "owner" | "waiter" | "courier" | "team"): string {
  let id = readSaved(KEY);
  if (!id) {
    id = mint();
    saveValue(KEY, id);
  }
  setDevice({
    id,
    app,
    platform: Platform.OS,
    // ⚠️ A model name, never a serial: the row exists so somebody in the office
    // can say "that is my old phone", and a string nobody recognises would make
    // the release button a guess.
    name: Device.modelName ?? Application.applicationName ?? "",
  });
  return id;
}

/** ⚠️ Not a UUID library. One value, generated once per install, and the only
 *  property that matters is that two phones do not collide — which 96 bits of
 *  `Math.random` plus the clock answers well enough for a restaurant's staff
 *  list. A dependency for this would be a dependency for one line. */
function mint(): string {
  const rnd = () => Math.random().toString(36).slice(2, 10);
  return `${Date.now().toString(36)}-${rnd()}-${rnd()}`;
}

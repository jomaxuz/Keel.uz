import * as Location from "expo-location";
import * as TaskManager from "expo-task-manager";

import { api } from "@/lib/api";
import { apiBaseFor, uploadsBaseFor } from "@/lib/serverAddress";
import { setApiBase } from "@/lib/tokenStore";

import { hydrateTokens, readSaved } from "./tokens";

// Reporting a position while the phone is in a pocket.
//
// ⚠️ **This is the whole reason a courier needs an app rather than the PWA.**
// A browser gives a backgrounded tab no geolocation and eventually no CPU, so
// the position froze the moment the rider put the phone away — and the freeze
// is invisible from both ends until somebody needs it. The visible half is a
// pin stuck on the dispatcher's map. The half that costs money is that
// "delivered" stops opening: the server judges by the last *reported* position
// and refuses one older than ten minutes, so the courier stands at the door
// unable to close the order they just delivered.
//
// ⚠️ **The task runs in its own JavaScript context.** It is not the app: no
// React, no providers, nothing this module did not import — and, crucially, no
// tokens in memory. Android can start it after the app has been killed, which
// means everything it needs has to be re-read from disk on every invocation.
// A version of this that assumed `hydrateTokens()` had already run worked
// perfectly for as long as the app stayed alive, which is exactly the case this
// exists to survive.

export const LOCATION_TASK = "keel-courier-location";

/** ⚠️ Matches the foreground stream and the server's own tolerance: the arrival
 *  check calls a fix older than ten minutes unusable, so a background interval
 *  measured in minutes would leave a rider at a door with a shut button. */
const INTERVAL_MS = 15000;
const DISTANCE_M = 25;

TaskManager.defineTask(LOCATION_TASK, async ({ data, error }) => {
  if (error || !data) return;
  const { locations } = data as { locations: Location.LocationObject[] };
  if (!locations?.length) return;

  // Everything the request needs, re-read rather than assumed — see the note
  // above about which context this runs in.
  await hydrateTokens();
  const address = readSaved("keel_server_address");
  if (!address) return;
  setApiBase(apiBaseFor(address), uploadsBaseFor(address));

  try {
    await api.courierSendLocation(
      locations.map((l) => ({
        lat: l.coords.latitude,
        lng: l.coords.longitude,
        accuracy: l.coords.accuracy ?? 0,
        at: l.timestamp || Date.now(),
      })),
    );
  } catch {
    // ⚠️ **Nothing is buffered here, and that is deliberate.** The foreground
    // app keeps a buffer because it can see whether a send worked; a task woken
    // for a hundred milliseconds cannot own a queue that would need its own
    // storage, its own trimming and its own tests. A dropped background send
    // costs one position, and the next one is fifteen seconds away.
  }
});

/** Start reporting in the background, asking for the permission it needs.
 *
 *  ⚠️ **The background permission is asked separately and only after the
 *  foreground one is granted** — both platforms require that order, and on
 *  Android the "allow all the time" screen is a second, scarier dialog. A
 *  refusal is not a failure: the app keeps the foreground stream, which is what
 *  it had before this existed, and the shift card says as much.
 *
 *  Returns whether background updates are actually running. */
export async function startBackgroundUpdates(): Promise<boolean> {
  const fg = await Location.getForegroundPermissionsAsync();
  if (!fg.granted) return false;

  const bg = await Location.getBackgroundPermissionsAsync();
  let granted = bg.granted;
  if (!granted && bg.canAskAgain) {
    granted = (await Location.requestBackgroundPermissionsAsync()).granted;
  }
  if (!granted) return false;

  if (await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK)) return true;

  await Location.startLocationUpdatesAsync(LOCATION_TASK, {
    accuracy: Location.Accuracy.Balanced,
    timeInterval: INTERVAL_MS,
    distanceInterval: DISTANCE_M,
    // ⚠️ **The notification is the price and it is worth paying.** Android only
    // keeps location running for an app it can show the user is running, and
    // hiding that would be both impossible and wrong: somebody whose phone is
    // reporting where they are should be able to see that it is, and stop it by
    // ending the shift.
    foregroundService: {
      notificationTitle: "Keel Courier — smena ochiq",
      notificationBody: "Joylashuv restoranga yuborilmoqda",
      notificationColor: "#e2590d",
      // Stops the service when the shift ends rather than leaving it to the
      // system; `stopBackgroundUpdates` is what actually ends it.
      killServiceOnDestroy: true,
    },
    pausesUpdatesAutomatically: false,
    // ⚠️ iOS only, and it is what keeps a delivery from being reported from the
    // wrong street: "other navigation" tells the system this is a vehicle
    // moving through a city rather than a fitness track.
    activityType: Location.ActivityType.OtherNavigation,
    showsBackgroundLocationIndicator: true,
  });
  return true;
}

/** Stop reporting. ⚠️ Called when the shift ends and on sign-out — a foreground
 *  service that outlives the shift is a battery complaint and, worse, a phone
 *  still telling a restaurant where somebody is after they have gone home. */
export async function stopBackgroundUpdates(): Promise<void> {
  try {
    if (await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK)) {
      await Location.stopLocationUpdatesAsync(LOCATION_TASK);
    }
  } catch {
    // The task was never registered on this launch; nothing to stop.
  }
}

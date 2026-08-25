import { useEffect, useRef } from "react";
import { Platform } from "react-native";
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import Constants from "expo-constants";

import { api } from "@/lib/api";

// Being told the food is ready.
//
// ⚠️ **The one thing a waiter cannot find out by looking.** Every other screen
// here is something they can open — the room, the check, their hours. Food
// reaching the pass happens in another part of the building, and the
// alternatives are a bell, a shout, or walking over to check. That walk is what
// this removes.
//
// ⚠️ **Permission is asked after signing in, not at launch.** A prompt on the
// first screen is asked before anybody knows what the app is for, and the
// answer to a question you do not understand is "no" — which on iOS is close to
// permanent, because the second ask has to happen in the system settings.

/** The channel the server names in every message. ⚠️ The two spellings have to
 *  agree: a mismatched channel on Android arrives silent and unranked, which
 *  looks exactly like a notification nobody sent. */
const CHANNEL = "kitchen";

Notifications.setNotificationHandler({
  // ⚠️ Shown even while the app is open. A waiter with the room on screen is
  // the person most likely to be walking, and the whole message is "go to the
  // pass" — swallowing it because the app happens to be foregrounded is the one
  // case where it is least likely to be read some other way.
  handleNotification: async () => ({
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

/** Register this phone, and hand back a way to drop it on sign-out. */
export function usePushRegistration(
  signedIn: boolean,
  onOpenCheck: (checkId: string) => void,
) {
  const token = useRef<string | null>(null);

  useEffect(() => {
    if (!signedIn) return;
    let alive = true;

    void (async () => {
      if (Platform.OS === "android") {
        // Created before the first notification arrives, or Android files it
        // under a default channel with no sound.
        await Notifications.setNotificationChannelAsync(CHANNEL, {
          name: "Oshxona",
          importance: Notifications.AndroidImportance.HIGH,
          vibrationPattern: [0, 250, 250, 250],
          sound: "default",
        });
      }

      // ⚠️ A simulator has no push transport, and asking there produces an
      // error that reads like a broken app to whoever is testing.
      if (!Device.isDevice) return;

      const existing = await Notifications.getPermissionsAsync();
      let granted = existing.granted;
      if (!granted && existing.canAskAgain) {
        granted = (await Notifications.requestPermissionsAsync()).granted;
      }
      // ⚠️ **Refusal is a real answer and the app keeps working.** Somebody who
      // says no still has the room, the check and their hours; nagging them
      // every launch would be the reason the app gets uninstalled.
      if (!granted || !alive) return;

      const projectId =
        Constants.expoConfig?.extra?.eas?.projectId ??
        Constants.easConfig?.projectId;
      if (!projectId) return;

      const value = (
        await Notifications.getExpoPushTokenAsync({ projectId })
      ).data;
      if (!alive) return;
      token.current = value;
      // Registered on every launch: the token can be re-issued after a
      // reinstall, and the server keys on it so a phone handed to somebody else
      // moves to them.
      await api.staffRegisterPush(value, Platform.OS);
    })().catch(() => {
      // ⚠️ Silently. Every reason this fails — no transport, a refused
      // permission, no network — leaves an app that works, and an error banner
      // about notifications on a screen somebody is taking an order on is worse
      // than the missing feature.
    });

    return () => {
      alive = false;
    };
  }, [signedIn]);

  // A tap opens the table rather than the room: somebody reading this on the
  // move has already decided where they are going.
  useEffect(() => {
    const sub = Notifications.addNotificationResponseReceivedListener((res) => {
      const data = res.notification.request.content.data as {
        checkId?: string;
      };
      if (data?.checkId) onOpenCheck(data.checkId);
    });
    return () => sub.remove();
  }, [onOpenCheck]);

  /** Drop this phone's registration. ⚠️ Awaited by the sign-out path before the
   *  token is cleared, or the request goes out unauthenticated and the row
   *  stays. */
  return async function forget() {
    const value = token.current;
    token.current = null;
    if (!value) return;
    try {
      await api.staffForgetPush(value);
    } catch {
      // Signed out either way. The server prunes a dead token on the next
      // event it fails to deliver.
    }
  };
}

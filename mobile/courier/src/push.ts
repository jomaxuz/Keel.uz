import { useCallback, useEffect, useRef, useState } from "react";
import { Platform } from "react-native";
import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import Constants from "expo-constants";

import { api, ApiError } from "@/lib/api";

import type { Lang } from "./i18n";

// Being told what happened to an order while the phone was away.
//
// ⚠️ **A courier is the one person here who is not near a screen.** The waiter
// walks a floor, the kitchen has a monitor, the owner has the panel — a courier
// is on a bike with the phone in a pocket, and every event that concerns them
// happens while they are not looking. Before this the app answered by polling
// every twenty seconds: a battery cost paid all evening to learn something that
// happens twice.
//
// ⚠️ **Permission is asked after signing in, not at launch.** A prompt on the
// first screen is asked before anybody knows what the app is for, and the
// answer to a question you do not understand is "no" — which on iOS is close to
// permanent, because the second ask has to happen in the system settings.

/** The channel the server names in every message. ⚠️ The two spellings have to
 *  agree (`push.DeliveryChannel` in the Go code): a mismatched channel on
 *  Android arrives silent and unranked, which looks exactly like a notification
 *  nobody sent. */
const CHANNEL = "delivery";

Notifications.setNotificationHandler({
  // ⚠️ Shown even while the app is open. A courier with the list on screen is
  // the person most likely to be riding, and every one of these messages is an
  // instruction — swallowing it because the app happens to be foregrounded is
  // the case where it is least likely to be read some other way.
  handleNotification: async () => ({
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

/** What happened when this phone tried to subscribe.
 *
 *  ⚠️ **Not shown as an error, and available where somebody would ask.** An app
 *  that puts a notification problem on the screen a courier is reading an
 *  address from is worse than the missing feature — but with nothing recorded
 *  anywhere, "the restaurant says they sent me an order and nothing arrived"
 *  has five causes and no way to tell them apart. So it lives on the settings
 *  screen. */
export type PushState =
  | "working"
  | "asking"
  | "denied"
  | "noDevice"
  | "noProject"
  | "failed";

/** ⚠️ **Why "failed" was not enough.** The first courier to run this on a real
 *  phone read "not registered · could not reach the internet or the server",
 *  pressed retry, and got the same line — which is true of at least three
 *  completely different faults: this build has no push credentials, the server
 *  is older than the endpoint, or the phone is offline. None of them is fixed
 *  from the phone, and the person who *can* fix them needs to know which one it
 *  is. So the failure carries the sentence it failed with. */
export function usePush(
  signedIn: boolean,
  lang: Lang,
  /** A tap on a notification. The payload names the order, so the app can put
   *  the list in front of somebody who has already decided to look at it. */
  onOpen: (orderId: string | undefined) => void,
) {
  const token = useRef<string | null>(null);
  const [state, setState] = useState<PushState>("asking");
  /** The raw reason, for the settings screen. Deliberately not translated: it
   *  is a diagnostic, it is read by whoever is fixing the install, and a
   *  translated HTTP status is a status somebody cannot search for. */
  const [detail, setDetail] = useState<string | null>(null);
  // Bumped by `retry`, so the effect runs again without remounting anything.
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    if (!signedIn) return;
    let alive = true;

    void (async () => {
      if (Platform.OS === "android") {
        // Created before the first notification arrives, or Android files it
        // under a default channel with no sound.
        await Notifications.setNotificationChannelAsync(CHANNEL, {
          name: "Yetkazish",
          importance: Notifications.AndroidImportance.HIGH,
          vibrationPattern: [0, 250, 250, 250],
          sound: "default",
        });
      }

      // ⚠️ A simulator has no push transport, and asking there produces an
      // error that reads like a broken app to whoever is testing.
      if (!Device.isDevice) {
        setState("noDevice");
        return;
      }

      const existing = await Notifications.getPermissionsAsync();
      let granted = existing.granted;
      if (!granted && existing.canAskAgain) {
        granted = (await Notifications.requestPermissionsAsync()).granted;
      }
      // ⚠️ **Refusal is a real answer and the app keeps working.** Somebody who
      // says no still has the list, the map links and the shift switch; nagging
      // them every launch is how an app gets uninstalled.
      if (!granted) {
        setState("denied");
        return;
      }
      if (!alive) return;

      const projectId =
        Constants.expoConfig?.extra?.eas?.projectId ??
        Constants.easConfig?.projectId;
      if (!projectId) {
        // ⚠️ Not hypothetical: on the waiter app the id was cleared from
        // app.json while an EAS project was being re-linked, and the app went
        // on looking entirely normal while subscribing to nothing.
        setState("noProject");
        return;
      }

      // ⚠️ **Two failures live here and they are not the same fault.** Asking
      // the operating system for a token fails when the *build* has no push
      // credentials — on Android, FCM was never configured for this EAS
      // project — and no amount of retrying on the phone will change it.
      // Registering fails when the *server* refuses, and a 404 means it is
      // older than the endpoint. Both used to read "could not reach the
      // internet or the server".
      let value: string;
      try {
        value = (await Notifications.getExpoPushTokenAsync({ projectId })).data;
      } catch (e) {
        if (alive) {
          setState("failed");
          setDetail(`token: ${e instanceof Error ? e.message : String(e)}`);
        }
        return;
      }
      if (!alive) return;
      token.current = value;
      // ⚠️ Registered on every launch **and on every language change**: the
      // token can be re-issued after a reinstall, the server keys on it so a
      // phone handed to somebody else moves with them, and the notification
      // text is written server-side in whichever language this row carries.
      try {
        await api.courierRegisterPush(value, Platform.OS, lang);
      } catch (e) {
        if (alive) {
          setState("failed");
          setDetail(
            e instanceof ApiError
              ? `server ${e.status}: ${e.message}`
              : `server: ${e instanceof Error ? e.message : String(e)}`,
          );
        }
        return;
      }
      if (alive) {
        setState("working");
        setDetail(null);
      }
    })().catch((e) => {
      // Anything left: the channel, the permission call, `Device.isDevice`.
      setState("failed");
      setDetail(e instanceof Error ? e.message : String(e));
      // Silently on screen: every reason this fails leaves an app that works.
    });

    return () => {
      alive = false;
    };
  }, [signedIn, lang, nonce]);

  useEffect(() => {
    const sub = Notifications.addNotificationResponseReceivedListener((res) => {
      const data = res.notification.request.content.data as {
        orderId?: string;
      };
      onOpen(data?.orderId);
    });
    return () => sub.remove();
  }, [onOpen]);

  /** What the settings screen shows, and a way to try again. */
  const retry = useCallback(() => {
    setState("asking");
    setDetail(null);
    setNonce((n) => n + 1);
  }, []);

  /** Drop this phone's registration. ⚠️ Awaited by the sign-out path **before**
   *  the token is cleared, or the request goes out unauthenticated and the row
   *  stays — sending the next rider's addresses to a phone that has left. */
  const forget = useCallback(async () => {
    const value = token.current;
    token.current = null;
    if (!value) return;
    try {
      await api.courierForgetPush(value);
    } catch {
      // Signed out either way. The server prunes a dead token on the next
      // event it fails to deliver.
    }
  }, []);

  return { state, detail, retry, forget };
}

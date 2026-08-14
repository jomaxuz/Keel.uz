// Subscribing a browser to the restaurant's notifications.
//
// ⚠️ **The permission is asked for once in the life of a browser, and there is
// no second chance.** After a guest says no, `Notification.requestPermission`
// resolves immediately with "denied" and the dialog never appears again — the
// page cannot bring it back, only the person can, from browser settings. The
// same rule the geolocation permission taught this codebase (`lib/geo.tsx`),
// and it has the same consequence: the prompt must never fire on page load,
// because a dialog nobody asked for is a dialog most people dismiss, and
// dismissing it spends the only chance there was.
//
// So this module never asks. It reports what the state is and offers a
// `subscribe()` the caller wires to a button the guest actually pressed.

import { api } from "@/lib/api";

export type PushState =
  /** This browser cannot do it at all — no service worker, or an insecure
   *  origin (a phone opening `http://192.168.x.x` gets no push and no
   *  explanation). */
  | "unsupported"
  /** Allowed, and a subscription is registered with the server. */
  | "on"
  /** Never asked. The only state in which pressing a button shows a dialog. */
  | "prompt"
  /** The guest said no. Nothing this page does can change it. */
  | "denied";

/** Whether this browser can receive push notifications at all.
 *
 *  ⚠️ `isSecureContext` is checked separately from the API's existence: the
 *  service worker API is missing entirely on an insecure origin, so without
 *  this the developer testing from a phone over the LAN sees "your browser does
 *  not support notifications" on a browser that supports them perfectly. */
export function pushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    window.isSecureContext &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window
  );
}

/** What state this browser is in right now. */
export async function pushState(): Promise<PushState> {
  if (!pushSupported()) return "unsupported";
  if (Notification.permission === "denied") return "denied";
  if (Notification.permission === "default") return "prompt";

  // Granted — but that is the *permission*, not a subscription. They come
  // apart in a way that matters: clearing site data leaves the permission
  // granted and destroys the subscription, so a page trusting the permission
  // alone would show "notifications are on" to somebody who will never receive
  // one again.
  const reg = await navigator.serviceWorker.getRegistration("/push-sw.js");
  const sub = await reg?.pushManager.getSubscription();
  return sub ? "on" : "prompt";
}

/** Registers this browser, asking for the permission if it has not been asked.
 *
 *  Must be called from a real click: Safari only shows the dialog in response
 *  to a user gesture, and an `await` before the call is enough to lose it. */
export async function subscribePush(): Promise<PushState> {
  if (!pushSupported()) return "unsupported";

  const permission = await Notification.requestPermission();
  if (permission !== "granted") return permission === "denied" ? "denied" : "prompt";

  const reg = await navigator.serviceWorker.register("/push-sw.js");
  // The worker has to be active before `pushManager` will do anything. On a
  // first registration it is still installing, and subscribing against it
  // fails with an error naming neither the worker nor the reason.
  await navigator.serviceWorker.ready;

  const { publicKey } = await api.pushKey();
  if (!publicKey) return "prompt";

  const existing = await reg.pushManager.getSubscription();
  // ⚠️ An existing subscription is reused rather than replaced. Unsubscribing
  // and re-subscribing issues a new endpoint, which leaves the old row in the
  // database until the push service eventually answers 410 — and until then
  // the guest is two recipients and gets every campaign twice.
  const sub =
    existing ??
    (await reg.pushManager.subscribe({
      // Required by every browser; a subscription without it cannot be sent to.
      userVisibleOnly: true,
      applicationServerKey: decodeKey(publicKey),
    }));

  const json = sub.toJSON();
  await api.pushSubscribe({
    endpoint: sub.endpoint,
    keys: { p256dh: json.keys?.p256dh ?? "", auth: json.keys?.auth ?? "" },
    device: navigator.userAgent.slice(0, 120),
  });
  return "on";
}

/** Stops notifications on this browser, and tells the server. */
export async function unsubscribePush(): Promise<PushState> {
  if (!pushSupported()) return "unsupported";

  const reg = await navigator.serviceWorker.getRegistration("/push-sw.js");
  const sub = await reg?.pushManager.getSubscription();
  // ⚠️ The server is told **before** the browser forgets. The endpoint is the
  // only handle on that row, and unsubscribing first would leave a subscription
  // nobody can name — messaged on every campaign until the push service gives
  // up on it, months later.
  if (sub) {
    await api.pushUnsubscribe(sub.endpoint).catch(() => undefined);
    await sub.unsubscribe();
  } else {
    await api.pushUnsubscribe().catch(() => undefined);
  }
  // The permission stays granted; only the subscription is gone. That is the
  // honest state, and it is why turning it back on later needs no dialog.
  return "prompt";
}

/** Turns the server's base64url key into the `Uint8Array` the browser wants.
 *
 *  ⚠️ Base64**url**, and it has to be converted rather than passed through:
 *  `atob` rejects `-` and `_`, so handing it the key straight from the API
 *  throws an "InvalidCharacterError" that names the decoder rather than the
 *  key. */
function decodeKey(base64url: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64url.length % 4)) % 4);
  const base64 = (base64url + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  // Backed by a plain ArrayBuffer explicitly: `applicationServerKey` will not
  // take a view that might sit on a SharedArrayBuffer, and the default
  // `Uint8Array` type is wide enough to include one.
  const out = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

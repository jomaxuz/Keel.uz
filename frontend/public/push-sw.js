// Service worker for the customer site's browser notifications.
//
// Deliberately tiny, and deliberately **not** the courier one. That worker
// caches an app shell so the courier app survives a flaky mobile connection;
// this one caches nothing at all. Registering a caching worker on the public
// site would put a copy of the menu between the restaurant and its guests —
// and the day the owner changes a price, the price on the phone would be
// yesterday's, with no way for anybody to explain it.
//
// So: two jobs, both of them about a notification that has already arrived.

// The message the server encrypted. Everything below assumes it is JSON of the
// shape { title, body, url, icon } — but the parse is guarded anyway, because a
// service worker that throws here shows the browser's own "This site has been
// updated in the background" notification instead, which is a message from
// nobody about nothing.
self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = {};
  }

  const title = data.title || "";
  const options = {
    body: data.body || "",
    icon: data.icon || "/icon.svg",
    // The small monochrome badge Android puts in the status bar.
    badge: "/icon.svg",
    data: { url: data.url || "/" },
    // ⚠️ A tag, so a second campaign replaces the first rather than stacking.
    // Without it a guest who was offline for a day comes back to a column of
    // identical restaurant notifications, and clears all of them at once
    // without reading any.
    tag: "keel-campaign",
    renotify: true,
  };

  // ⚠️ `waitUntil` is not optional. The browser kills the worker the moment
  // this handler returns, so without it the notification is a race the worker
  // usually loses — and it loses it silently, on the guest's phone, where
  // nobody can see that anything was supposed to happen.
  event.waitUntil(self.registration.showNotification(title, options));
});

// Clicking it opens the site.
//
// ⚠️ An existing tab is focused rather than a new one opened. A guest who
// already has the menu open does not want a second copy of it, and on a phone
// the second tab is the one holding their half-filled cart.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = (event.notification.data && event.notification.data.url) || "/";

  event.waitUntil(
    self.clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clients) => {
        for (const client of clients) {
          // Same origin only — the worker's scope guarantees it, but the URL
          // came out of a push payload and is treated as untrusted anyway.
          if (client.url && "focus" in client) {
            client.focus();
            if ("navigate" in client) client.navigate(target);
            return undefined;
          }
        }
        return self.clients.openWindow(target);
      }),
  );
});

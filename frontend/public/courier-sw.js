// Service worker for the courier PWA. Its only job is to make the app
// installable and to keep the shell reachable on a flaky mobile connection.
//
// NOTE: it deliberately does NOT track location. Browsers do not grant
// geolocation to service workers, so positions are reported by the page while
// it is open (see useCourierTracking).

const CACHE = "kuryer-v1";
const SHELL = ["/kuryer", "/kuryer/login", "/manifest.webmanifest", "/courier-icon.svg"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(SHELL)).then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);
  if (url.origin !== self.location.origin) return;
  // Never cache API traffic — orders and positions must always be live.
  if (url.pathname.startsWith("/api/")) return;

  event.respondWith(
    fetch(req)
      .then((res) => {
        if (res.ok && url.pathname.startsWith("/kuryer")) {
          const copy = res.clone();
          caches.open(CACHE).then((c) => c.put(req, copy));
        }
        return res;
      })
      .catch(() => caches.match(req).then((hit) => hit || caches.match("/kuryer"))),
  );
});

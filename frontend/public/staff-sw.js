// Service worker for the staff PWA. Its only job is to make the app
// installable and to keep the shell reachable on a flaky connection.
//
// NOTE: it deliberately does NOT touch location. Browsers do not grant
// geolocation to service workers, and this app only needs a position at the
// moment a button is pressed — which is always with the page in front of
// someone (see lib/staff.tsx).

const CACHE = "staff-v1";
const SHELL = [
  "/staff",
  "/staff/login",
  "/staff-manifest.webmanifest",
  "/staff-icon.svg",
];

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
  // Never cache API traffic: a cached shift would tell someone they are
  // clocked in when they are not.
  if (url.pathname.startsWith("/api/")) return;

  event.respondWith(
    fetch(req)
      .then((res) => {
        if (res.ok && url.pathname.startsWith("/staff")) {
          const copy = res.clone();
          caches.open(CACHE).then((c) => c.put(req, copy));
        }
        return res;
      })
      .catch(() => caches.match(req).then((hit) => hit || caches.match("/staff"))),
  );
});

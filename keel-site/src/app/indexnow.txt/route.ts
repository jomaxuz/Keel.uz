// The IndexNow ownership key, served as a plain file.
//
// ⚠️ **This is what makes "tell the search engines a page changed" possible at
// all.** IndexNow is a push protocol: we POST a list of URLs and the engine
// fetches them within minutes instead of waiting for its next crawl. The only
// proof of ownership is that the key we send back is readable on the domain —
// so if this route stops answering, every submission is refused and the failure
// is a 403 in a log nobody is reading.
//
// ⚠️ **Bing and Yandex, not Google.** Google does not participate in IndexNow
// and has removed its own sitemap ping; there is no public way to push a page
// to it, and the Indexing API is documented for job postings and live streams
// only. That is not a gap this file can close — see the console's SEO screen,
// which says so rather than implying otherwise. Yandex on its own justifies
// this: it is a large share of search in Uzbekistan.
//
// ⚠️ Served at a fixed path rather than at `/<key>.txt`, which is the usual
// arrangement: the protocol allows `keyLocation` to point anywhere on the host,
// and a fixed path means rotating the key is one environment variable rather
// than a code change.

export const dynamic = "force-dynamic";

export function GET() {
  const key = process.env.INDEXNOW_KEY ?? "";
  if (!key) {
    // ⚠️ 404 rather than an empty 200. An empty body would be accepted as a
    // key file whose contents match nothing, and the submissions would fail
    // later with a reason that points at the wrong thing.
    return new Response("not configured", { status: 404 });
  }
  return new Response(key, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "public, max-age=3600",
    },
  });
}

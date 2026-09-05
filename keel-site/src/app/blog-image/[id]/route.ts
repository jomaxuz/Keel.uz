// A picture from a blog post.
//
// ⚠️ **This route exists because the edge does not proxy `/internal`.** The
// control plane serves the bytes at `/internal/blog/image/{id}`, which is
// reachable from inside the docker network and from nowhere else — so a post
// whose pictures pointed there rendered every one of them as a broken box, for
// every reader, while looking perfectly correct in the editor. Found by asking
// the live site for the address rather than by reading the code.
//
// ⚠️ **A proxy, not a redirect.** A redirect would send the reader's browser to
// an address that does not answer publicly, which is the same failure with an
// extra request in front of it.

import { CONTROL, INTERNAL } from "@/lib/partners";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;
  // ⚠️ Hex only. The id goes into a URL we then fetch, and anything else is a
  // request somebody else composed — there is no reason to pass it on.
  if (!/^[0-9a-f]{24}$/.test(id)) {
    return new Response("not found", { status: 404 });
  }
  let res: Response;
  try {
    res = await fetch(`${CONTROL}${INTERNAL}/blog/image/${id}`, {
      // The bytes never change under an id, so this is cached hard on the way
      // in as well as on the way out.
      next: { revalidate: 31536000 },
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    return new Response("unavailable", { status: 502 });
  }
  if (!res.ok) return new Response("not found", { status: 404 });

  return new Response(res.body, {
    headers: {
      "Content-Type": res.headers.get("Content-Type") ?? "image/jpeg",
      // ⚠️ Immutable: the id *is* the content, so a reader's browser never has
      // to ask about the same picture twice.
      "Cache-Control": "public, max-age=31536000, immutable",
    },
  });
}

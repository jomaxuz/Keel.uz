// One reading, forwarded.
//
// ⚠️ **The reader's browser cannot reach the control plane**, so this is the
// same hop `/blog-image` makes and for the same reason: `/internal` answers
// inside the docker network and nowhere else.

import { CONTROL, INTERNAL } from "@/lib/partners";

export async function POST(
  _req: Request,
  { params }: { params: Promise<{ slug: string }> },
) {
  const { slug } = await params;
  try {
    await fetch(
      `${CONTROL}${INTERNAL}/blog/${encodeURIComponent(slug)}/view`,
      { method: "POST", signal: AbortSignal.timeout(3000) },
    );
  } catch {
    // ⚠️ Swallowed. A counter is not worth an error in front of somebody who
    // is reading — the words are already on their screen, and the number this
    // failed to add is one nobody will miss.
  }
  return new Response(null, { status: 204 });
}

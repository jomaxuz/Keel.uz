// keel.uz/h/<kod> — a referrer's own address.
//
// ⚠️ **A path, not a query string**, and that is why this route exists at all.
// `keel.uz/?ref=fiskal` is the obvious build and it does not survive the
// journey these links actually take: read off a leaflet, typed into a phone,
// pasted into a Telegram message and copied back out — a query string is the
// part people trim when a link looks long. A path segment reads as the address
// itself and comes through intact.
//
// ⚠️ **A route handler rather than a page, because a page cannot set a cookie.**
// Next only allows that in a route handler or a server action, and the first
// build here was a page that looked correct, compiled, and returned a 500 to
// every visitor arriving on a printed link — the one audience that cannot
// report it, because they do not know what they were supposed to see.
//
// ⚠️ The redirect below *does* carry a query string, and that is a different
// thing: it has to survive one hop between two pages of ours, not a person.

import { NextResponse } from "next/server";
import {
  REF_COOKIE,
  REF_COOKIE_MAX_AGE,
  REF_PARAM,
  cleanRefCode,
} from "@/lib/referral";

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ code: string }> },
) {
  const { code } = await params;
  const clean = cleanRefCode(code);

  // ⚠️ **The landing itself, not a page of its own.** A referral link that
  // opens a different, thinner page is a page that has to be maintained twice
  // and will be updated once. The only difference is the strip at the top
  // naming who sent them.
  //
  // ⚠️ **A relative Location, and this is the whole bug that shipped.** The
  // first version built an absolute URL from `req.url`, which is what every
  // example does — and inside a container `req.url` is the container's own
  // hostname. Scanning the printed QR sent people to
  // `https://76c98175c198:3100/?h=…`, an address that exists on no network in
  // the world. It worked perfectly in local development, which is exactly why
  // it reached paper.
  //
  // HTTP allows a relative `Location`, and the browser resolves it against the
  // address it actually used. That is the correct answer here in a way an
  // absolute URL never is: this route must send the visitor back to *the host
  // they came in on*, whatever it is — keel.uz, a staging domain, localhost —
  // and it must never need to be told what that host is.
  const to = clean ? `/?${REF_PARAM}=${clean}` : "/";
  const res = new NextResponse(null, {
    status: 307,
    headers: { Location: to },
  });

  if (clean) {
    // ⚠️ A year, because the gap between "saw the leaflet" and "wrote to us" is
    // measured in weeks: it gets left on a counter and picked up when the
    // current way of doing things annoys somebody enough.
    res.cookies.set(REF_COOKIE, clean, {
      path: "/",
      maxAge: REF_COOKIE_MAX_AGE,
      sameSite: "lax",
    });
  }
  return res;
}

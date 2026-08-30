// The card Telegram, WhatsApp and Twitter draw when somebody sends this link.
//
// It matters more here than the layout of any section on the page: this product
// is sold by one restaurant owner forwarding the link to another, and until now
// the site declared `summary_large_image` with **no image anywhere**. That is
// not a neutral default — the platforms honour the declaration and render a
// large blank card, which reads as a dead or half-built link. A small card with
// nothing would have been better than what was there.
//
// Generated rather than checked in as a PNG so it stays in the same three
// languages as the page, and so the headline cannot drift away from the one on
// the page it previews.

import { ImageResponse } from "next/og";
import { dicts } from "@/lib/i18n/dict";
import { getLang } from "@/lib/i18n/server";
import { plain } from "@/lib/accent";

export const alt = "Keel";
// The size every platform crops from. Anything smaller is upscaled and looks it.
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function Image() {
  const t = dicts[await getLang()];

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          // The hull, the same deep marine as the site. A dark card stands out
          // in a chat list of mostly white previews.
          background: "#05101A",
          padding: 72,
          fontFamily: "sans-serif",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 20 }}>
          {/* The mark, drawn inline: this renderer loads no external asset, and
              a missing logo would be the same blank-card failure one level in. */}
          <svg width="64" height="64" viewBox="0 0 32 32">
            <g
              fill="none"
              stroke="#F5A524"
              strokeWidth="2.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 6c0 9.5 4.4 14.5 11 14.5S27 15.5 27 6" />
              <path d="M16 20.5V29" />
            </g>
          </svg>
          <span style={{ color: "#F5A524", fontSize: 44, fontWeight: 700 }}>Keel</span>
        </div>

        {/* The headline the page actually leads with, not a slogan invented for
            the card. A preview that promises something the page does not say is
            a bounce. */}
        <div
          style={{
            display: "flex",
            color: "#ECF1F5",
            fontSize: 60,
            lineHeight: 1.15,
            fontWeight: 700,
            maxWidth: 980,
          }}
        >
          {plain(t.hero.title)}
        </div>

        {/* The two numbers the whole pitch rests on, side by side. They survive
            the thumbnail size a chat preview is actually rendered at, which the
            paragraph above would not. */}
        <div style={{ display: "flex", gap: 56, alignItems: "flex-end" }}>
          <Stat label={t.hero.stat1} value={t.hero.stat1v} accent />
          <Stat label={t.hero.stat2} value={t.hero.stat2v} />
          <Stat label={t.hero.stat3} value={t.hero.stat3v} />
        </div>
      </div>
    ),
    size,
  );
}

function Stat({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
      <span style={{ color: "#8094A4", fontSize: 26 }}>{label}</span>
      <span style={{ color: accent ? "#F5A524" : "#ECF1F5", fontSize: 52, fontWeight: 700 }}>
        {value}
      </span>
    </div>
  );
}

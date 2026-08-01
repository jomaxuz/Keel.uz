"use client";

// A QR code rendered as inline SVG — no network, no canvas, no image host.
//
// It exists for one job: a link that only works on a phone (an app deeplink,
// e.g. Yandex Go's delivery flow) has to travel from the desktop panel to the
// operator's phone. Scanning is faster than typing, and it keeps every
// prefilled coordinate intact.

import { useMemo } from "react";
import qrcode from "qrcode-generator";

export default function QrCode({
  value,
  size = 148,
  className = "",
}: {
  value: string;
  size?: number;
  className?: string;
}) {
  // Type 0 = pick the smallest version that fits; "M" error correction is the
  // usual trade-off between density and tolerance for a screen-to-camera scan.
  const path = useMemo(() => {
    try {
      const qr = qrcode(0, "M");
      qr.addData(value);
      qr.make();
      const count = qr.getModuleCount();
      const parts: string[] = [];
      for (let row = 0; row < count; row++) {
        for (let col = 0; col < count; col++) {
          if (qr.isDark(row, col)) parts.push(`M${col} ${row}h1v1h-1z`);
        }
      }
      return { d: parts.join(""), count };
    } catch {
      // Too much data for any QR version — the link is shown as text anyway.
      return null;
    }
  }, [value]);

  if (!path) return null;

  return (
    <svg
      viewBox={`-1 -1 ${path.count + 2} ${path.count + 2}`}
      width={size}
      height={size}
      className={`rounded-lg bg-white p-1 ${className}`}
      role="img"
      aria-hidden="true"
      shapeRendering="crispEdges"
    >
      <path d={path.d} fill="#000" />
    </svg>
  );
}

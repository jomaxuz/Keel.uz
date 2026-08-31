"use client";

// A QR code as inline SVG — no network, no canvas, no image host.
//
// ⚠️ **Copied from the tenant panel rather than imported**, the same decision
// the chart palette and the Keel mark already carry: these are two separate
// builds and two separate images, and a shared component would tie keel.uz's
// deploy to the restaurant app's. Twenty lines is a cheaper price than that
// coupling.
//
// ⚠️ Inline SVG specifically because this one is printed. A canvas renders at
// screen resolution and comes out of a printer as a soft grey square that a
// phone camera has to be coaxed into reading — on a leaflet whose only job is
// to be scanned.

import { useMemo } from "react";
import qrcode from "qrcode-generator";

export default function Qr({
  value,
  size = 132,
  className = "",
}: {
  value: string;
  size?: number;
  className?: string;
}) {
  const path = useMemo(() => {
    try {
      // Type 0 picks the smallest version that fits. "M" error correction is
      // the usual screen-to-camera trade-off, and a leaflet gets creased.
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
      return null;
    }
  }, [value]);

  if (!path) return null;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${path.count} ${path.count}`}
      className={className}
      // The quiet zone is the white border a scanner needs to find the code.
      // Printed edge to edge against a coloured block, a QR simply does not
      // read — and the failure looks like a broken phone, not a bad layout.
      style={{ background: "#fff", padding: 4, boxSizing: "content-box" }}
      role="img"
      aria-label={value}
    >
      <path d={path.d} fill="#000" />
    </svg>
  );
}

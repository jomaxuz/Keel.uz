"use client";

// A printable QR card: background, title, description, the code itself.
//
// The QR that goes on a table is not a bare square of pixels — it is a small
// poster a guest has to understand at a glance ("scan for the menu", table 7).
// So the card is drawn on a canvas: one canvas is both what the admin sees and
// exactly what downloads, with no screenshot-the-screen step in between, and it
// prints at a resolution that survives a laminator.

import { useEffect, useRef } from "react";
import qrcode from "qrcode-generator";

export interface PosterStyle {
  bg: string;
  fg: string;
  accent: string;
  /** Rounded plate behind the code, so a dark background still scans. */
  plate: string;
}

export const POSTER_STYLES: Record<string, PosterStyle> = {
  cream: { bg: "#faf6f0", fg: "#231b16", accent: "#e2590d", plate: "#ffffff" },
  charcoal: { bg: "#1c1917", fg: "#f5f0ea", accent: "#f59e0b", plate: "#ffffff" },
  brand: { bg: "#e2590d", fg: "#ffffff", accent: "#ffe8d5", plate: "#ffffff" },
  mint: { bg: "#eaf6f0", fg: "#12291f", accent: "#0f8a5f", plate: "#ffffff" },
};

export interface PosterContent {
  title: string;
  subtitle: string;
  description: string;
  /** Big number on the card ("7"), empty for the restaurant-wide code. */
  badge: string;
  footer: string;
}

// Card geometry in device pixels: A6-ish at ~300dpi, which prints crisply and
// is still small enough to draw dozens of at once.
const W = 1200;
const H = 1700;

function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

/** Wraps text to `maxWidth`, returning the lines actually drawn. */
function wrap(
  ctx: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
  maxLines: number,
): string[] {
  const words = text.split(/\s+/).filter(Boolean);
  const lines: string[] = [];
  let line = "";
  for (const word of words) {
    const next = line ? `${line} ${word}` : word;
    if (ctx.measureText(next).width > maxWidth && line) {
      lines.push(line);
      line = word;
      if (lines.length === maxLines) return lines;
    } else {
      line = next;
    }
  }
  if (line && lines.length < maxLines) lines.push(line);
  return lines;
}

/** Draws the whole card. Exported so "download all" can reuse it off-screen. */
export function drawPoster(
  canvas: HTMLCanvasElement,
  url: string,
  content: PosterContent,
  style: PosterStyle,
): void {
  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  canvas.width = W;
  canvas.height = H;

  ctx.fillStyle = style.bg;
  ctx.fillRect(0, 0, W, H);

  // A thin inset frame: it makes a plain background look deliberate, and gives
  // whoever cuts the card a line to follow.
  ctx.strokeStyle = style.accent;
  ctx.lineWidth = 6;
  roundRect(ctx, 40, 40, W - 80, H - 80, 48);
  ctx.stroke();

  ctx.textAlign = "center";

  let y = 190;
  if (content.badge) {
    // The table number, big enough to read from standing height.
    ctx.fillStyle = style.accent;
    roundRect(ctx, W / 2 - 110, y - 78, 220, 130, 40);
    ctx.fill();
    ctx.fillStyle = style.bg;
    ctx.font = "bold 92px Georgia, 'Times New Roman', serif";
    ctx.fillText(content.badge, W / 2, y + 12);
    y += 150;
  }

  ctx.fillStyle = style.fg;
  ctx.font = "bold 84px Georgia, 'Times New Roman', serif";
  for (const line of wrap(ctx, content.title, W - 200, 2)) {
    ctx.fillText(line, W / 2, y);
    y += 96;
  }

  if (content.subtitle) {
    ctx.fillStyle = style.accent;
    ctx.font = "600 44px Arial, Helvetica, sans-serif";
    y += 10;
    for (const line of wrap(ctx, content.subtitle, W - 220, 2)) {
      ctx.fillText(line, W / 2, y);
      y += 56;
    }
  }

  // ---- the code, on a white plate so any background still scans ----
  const plate = 720;
  const plateY = Math.max(y + 40, 620);
  ctx.fillStyle = style.plate;
  roundRect(ctx, (W - plate) / 2, plateY, plate, plate, 48);
  ctx.fill();

  try {
    const qr = qrcode(0, "M");
    qr.addData(url);
    qr.make();
    const count = qr.getModuleCount();
    const quiet = 4; // modules of margin the spec asks for
    const cell = (plate - 96) / (count + quiet * 2);
    const originX = (W - plate) / 2 + 48 + quiet * cell;
    const originY = plateY + 48 + quiet * cell;
    ctx.fillStyle = "#000000";
    for (let row = 0; row < count; row++) {
      for (let col = 0; col < count; col++) {
        if (!qr.isDark(row, col)) continue;
        // +1 closes the hairline gaps that fractional cells leave behind.
        ctx.fillRect(
          Math.floor(originX + col * cell),
          Math.floor(originY + row * cell),
          Math.ceil(cell) + 1,
          Math.ceil(cell) + 1,
        );
      }
    }
  } catch {
    /* the URL is longer than any QR version can hold */
  }

  let textY = plateY + plate + 90;
  if (content.description) {
    ctx.fillStyle = style.fg;
    ctx.font = "42px Arial, Helvetica, sans-serif";
    for (const line of wrap(ctx, content.description, W - 220, 3)) {
      ctx.fillText(line, W / 2, textY);
      textY += 56;
    }
  }

  if (content.footer) {
    ctx.fillStyle = style.fg;
    ctx.globalAlpha = 0.55;
    ctx.font = "34px Arial, Helvetica, sans-serif";
    ctx.fillText(content.footer, W / 2, H - 90);
    ctx.globalAlpha = 1;
  }
}

/** Live preview of one card; the same drawing that downloads. */
export default function QrPoster({
  url,
  content,
  style,
  className = "",
  canvasRef,
}: {
  url: string;
  content: PosterContent;
  style: PosterStyle;
  className?: string;
  canvasRef?: React.RefObject<HTMLCanvasElement | null>;
}) {
  const ownRef = useRef<HTMLCanvasElement>(null);
  const ref = canvasRef ?? ownRef;

  useEffect(() => {
    if (ref.current) drawPoster(ref.current, url, content, style);
  }, [ref, url, content, style]);

  return (
    <canvas
      ref={ref}
      className={`h-auto w-full rounded-2xl border border-line shadow-card ${className}`}
    />
  );
}

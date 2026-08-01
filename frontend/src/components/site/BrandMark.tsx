"use client";

// The restaurant's logo. Falls back to an initial-letter tile when no logo has
// been uploaded in the admin panel — which is also what a fresh deploy looks
// like, so the header never renders an empty box.

import { imageUrl } from "@/lib/api";

export default function BrandMark({
  name,
  logoUrl,
  className = "h-9 w-9",
}: {
  name: string;
  logoUrl?: string | null;
  className?: string;
}) {
  const src = imageUrl(logoUrl ?? "");
  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={name}
        className={`shrink-0 rounded-xl object-cover ${className}`}
      />
    );
  }
  return (
    <span
      className={`flex shrink-0 items-center justify-center rounded-xl bg-brand font-display text-lg font-bold text-white ${className}`}
    >
      {name.trim().charAt(0).toUpperCase() || "R"}
    </span>
  );
}

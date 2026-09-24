"use client";

// One code sample with a copy button — the only interactive part of the
// developers page, so the only client component on it.
//
// ⚠️ **Dark in both themes.** A code sample that flips to white in light mode
// stops looking like something to paste; the whole page around it can follow
// the theme, the terminal does not.

import { useState } from "react";

export default function CodeBlock({
  code,
  label,
  copy,
  copied,
}: {
  code: string;
  label?: string;
  copy: string;
  copied: string;
}) {
  const [done, setDone] = useState(false);
  async function onCopy() {
    try {
      await navigator.clipboard.writeText(code);
      setDone(true);
      window.setTimeout(() => setDone(false), 1500);
    } catch {
      // A browser that refuses the clipboard still shows the text to select.
    }
  }
  return (
    <div className="overflow-hidden rounded-xl border border-hull-700 bg-hull-950">
      <div className="flex items-center justify-between border-b border-hull-800 px-4 py-1.5">
        <span className="text-xs font-medium text-[#8aa3b8]">{label ?? ""}</span>
        <button
          type="button"
          onClick={() => void onCopy()}
          className="rounded-md px-2 py-1 text-xs text-[#8aa3b8] transition hover:bg-hull-800 hover:text-white"
        >
          {done ? copied : copy}
        </button>
      </div>
      <pre className="overflow-x-auto px-4 py-3 text-[13px] leading-relaxed text-[#dbe7f0]">
        <code className="font-mono">{code}</code>
      </pre>
    </div>
  );
}

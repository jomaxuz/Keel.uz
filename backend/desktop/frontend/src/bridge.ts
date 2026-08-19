// The Go side, as the screen sees it.
//
// ⚠️ **Reached through `window.go`, not through the generated wailsjs imports.**
// Wails writes those bindings during a Windows build, so importing them would
// make `tsc` and `vite build` fail everywhere else — including on the Linux
// machine where the shared till screens are actually developed. Going through
// the runtime object costs one hand-written interface and keeps every other
// environment able to compile and run this code.
//
// The same choice is what lets the till screens keep working in an ordinary
// browser: outside Wails `bridge()` is null, and callers fall back to what they
// already did.

export type PrintOptions = {
  target: string;
  charset?: "latin" | "cyrillic";
  feedLines?: number;
  cut?: boolean;
  fullCut?: boolean;
  openDrawer?: boolean;
};

type Bridge = {
  PrintLines: (lines: string[], o: PrintOptions) => Promise<void>;
  Quit: () => Promise<void>;
  Env: () => Promise<Record<string, string>>;
};

declare global {
  interface Window {
    go?: { main?: { App?: Bridge } };
  }
}

/** The Go side, or null when this is a plain browser. */
export function bridge(): Bridge | null {
  return window.go?.main?.App ?? null;
}

/** Whether the till is running inside the Windows application. */
export function inWails(): boolean {
  return bridge() !== null;
}

/**
 * Print already-laid-out lines on this machine's printer.
 *
 * ⚠️ Returns false rather than throwing when there is no Go side, so the caller
 * can fall through to the browser's print dialog. A till that refuses to print
 * because it is in the wrong window is worse than one that opens a dialog.
 */
export async function printLines(
  lines: string[],
  o: PrintOptions,
): Promise<boolean> {
  const b = bridge();
  if (!b) return false;
  await b.PrintLines(lines, o);
  return true;
}

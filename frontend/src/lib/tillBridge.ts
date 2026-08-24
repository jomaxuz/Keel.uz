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

export type Status = {
  paired: boolean;
  branchName: string;
  server: string;
  agent: boolean;
  platform: string;
  configPath: string;
};

export type BranchView = { id: string; name: string };

/** One printer this machine can reach. */
export type Installed = { name: string; default: boolean };

/** The printer this till puts paper out of, and what it could choose instead. */
export type PrintConfig = {
  /** What was chosen, or "" for whatever Windows prints to. */
  target: string;
  /** What that resolves to right now — filled in even when nothing was chosen,
   *  so the screen can say which printer will actually be used. */
  effective: string;
  charset?: "latin" | "cyrillic";
  feedLines?: number;
  cut?: boolean;
  fullCut?: boolean;
  drawer?: boolean;
  printers: Installed[];
};
export type ConnectResult = { server: string; branches: BranchView[] };

type Bridge = {
  PrintLines: (lines: string[], o: PrintOptions) => Promise<void>;
  Quit: () => Promise<void>;
  Status: () => Promise<Status>;
  DeviceToken: () => Promise<string>;
  ShowKeyboard: () => Promise<void>;
  Connect: (
    address: string,
    username: string,
    password: string,
  ) => Promise<ConnectResult>;
  Pair: (branchId: string) => Promise<void>;
  Printers: () => Promise<Installed[]>;
  PrintConfig: () => Promise<PrintConfig>;
  SavePrintConfig: (c: PrintConfig) => Promise<void>;
  TestPrint: () => Promise<void>;
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
  try {
    await b.PrintLines(lines, o);
    return true;
  } catch (e) {
    // ⚠️ **Swallowed into `false`, deliberately.** Every reason this throws —
    // no printer installed, a name that no longer resolves, a spooler that
    // refused — has the same correct answer at the counter: put the browser's
    // dialog up so the cashier can still hand the guest their bill. Rethrowing
    // would turn a printer problem into a till that stops mid-sale, and the
    // reason belongs in the log, not in front of a queue.
    console.warn("local print failed, falling back to the dialog", e);
    return false;
  }
}

/** Whether this till can print without the browser's dialog. */
export function canPrintLocally(): boolean {
  return bridge() !== null;
}

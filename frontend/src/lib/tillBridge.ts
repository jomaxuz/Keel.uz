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
  /** Which screen this machine opens: the till, or the waiter's floor screen.
   *
   *  ⚠️ A property of the machine rather than of the person unlocking it: one
   *  is bolted to a counter with a drawer under it, the other is carried
   *  through a dining room, and the person changes every shift. */
  mode?: "kassa" | "zal";
};

export type BranchView = { id: string; name: string };

/** One printer this machine can reach, as Windows describes it. */
export type Installed = {
  name: string;
  default: boolean;
  /** The port Windows has it on: "USB001", "192.168.1.50", "COM3". Shown so two
   *  similarly named rows can be told apart. */
  port: string;
  /** Where to send bytes, worked out from the port — so nobody types an address.
   *  Empty when the port says nothing useful. */
  target: string;
};

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
  Unpair: () => Promise<void>;
  Printers: () => Promise<Installed[]>;
  PrintConfig: () => Promise<PrintConfig>;
  SavePrintConfig: (c: PrintConfig) => Promise<void>;
  TestPrint: () => Promise<void>;
  /** The machine's own database — SQLite with WAL and `synchronous=FULL`.
   *  See `lib/offline/store.ts` for why the browser's storage is not enough
   *  and what the till falls back to when this is not there. */
  StoreReady: () => Promise<boolean>;
  StorePut: (store: string, key: string, value: string) => Promise<void>;
  StoreAll: (store: string) => Promise<string[]>;
  StoreRemove: (store: string, key: string) => Promise<void>;
  StoreClear: () => Promise<void>;
  /** Choose which screen this machine opens. The screen reloads afterwards —
   *  see the note on the Go side. */
  SetMode: (mode: string) => Promise<void>;
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

/**
 * Take this machine out of service and return to the setup screen.
 *
 * ⚠️ **Returns false in a browser so the caller navigates instead.** A URL is
 * the right way out of a tab and the wrong way out of this window: the till is
 * a bundled application, so navigating leaves it and the monoblock is left
 * showing a bare webview pointed at localhost — a black screen with an address
 * bar, in a restaurant, at the moment somebody has just retired the till.
 */
export async function unpair(): Promise<boolean> {
  const b = bridge();
  if (!b) return false;
  try {
    await b.Unpair();
  } catch (e) {
    // ⚠️ Swallowed, and the screen still goes back. Somebody who pressed this
    // has decided the machine is leaving; refusing because a file would not
    // save strands them holding a till they can neither use nor retire. The
    // pairing row is removable from the panel either way.
    console.warn("unpair failed, returning to setup anyway", e);
  }
  // ⚠️ A reload rather than React state: everything that makes this window a
  // till also lives in memory (the menu, the floor plan, the open checks, the
  // session), and the shell decides what to show from Status() on mount. This
  // is the one reset that cannot miss a piece, and it stays inside the bundle.
  window.location.reload();
  return true;
}

/** Whether this till can print without the browser's dialog. */
export function canPrintLocally(): boolean {
  return bridge() !== null;
}

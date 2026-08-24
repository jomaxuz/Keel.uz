import { afterEach, describe, expect, it, vi } from "vitest";
import { printReceipt } from "./print";
import type { PrintOptions } from "./tillBridge";

// Which of the two printing paths a receipt takes.
//
// ⚠️ **The failure this seals is silent and only happens on a monoblock**: the
// Windows till prints through the spooler, and the moment it also opens the
// browser dialog the cashier is tapping through a modal for every sale — on a
// touch screen, with a guest waiting. Nothing errors, the paper still comes
// out, and it looks like the application "always did that".

function fakeBridge(impl: Partial<Record<string, unknown>> = {}) {
  // ⚠️ Typed like the binding it stands in for. An untyped mock records its
  // calls as an empty tuple, so the assertions below — which are the whole
  // point, since they check *what* was sent — would not compile.
  const PrintLines = vi.fn(async (_lines: string[], _o: PrintOptions) => {});
  (window as unknown as { go?: unknown }).go = {
    main: { App: { PrintLines, ...impl } },
  };
  return PrintLines;
}

afterEach(() => {
  delete (window as unknown as { go?: unknown }).go;
  vi.restoreAllMocks();
});

describe("printReceipt", () => {
  it("prints through the till and never opens the dialog", async () => {
    const PrintLines = fakeBridge();
    const frames = vi.spyOn(document.body, "appendChild");

    printReceipt(["chek"], 80, "");
    await vi.waitFor(() => expect(PrintLines).toHaveBeenCalledOnce());

    expect(frames).not.toHaveBeenCalled();
    // ⚠️ The target is left empty on purpose: the Go side resolves the
    // machine's own setting and then the Windows default. A target invented
    // here would override a choice somebody made standing at the printer.
    expect(PrintLines.mock.calls[0]![1]).toMatchObject({ target: "" });
  });

  it("falls back to the dialog when the local printer fails", async () => {
    fakeBridge({
      PrintLines: vi.fn(async () => {
        throw new Error("printer topilmadi");
      }),
    });
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const frames = vi.spyOn(document.body, "appendChild");

    printReceipt(["chek"], 80, "");

    // ⚠️ The guest still gets their bill. Every reason the spooler refuses has
    // the same right answer at the counter, and a till that stops mid-sale
    // because a printer is unplugged is worse than one that opens a dialog.
    await vi.waitFor(() => expect(frames).toHaveBeenCalled());
  });

  it("opens the drawer only when the caller asks", async () => {
    const PrintLines = fakeBridge();

    printReceipt(["chek"], 80, "", { drawer: true });
    await vi.waitFor(() => expect(PrintLines).toHaveBeenCalledOnce());
    expect(PrintLines.mock.calls[0]![1]).toMatchObject({ openDrawer: true });

    // ⚠️ Absent means closed, never "whatever was configured". The drawer opens
    // for the till's own copy of a sale and nothing else — a bill or a kitchen
    // ticket that kicks it leaves the drawer standing open all evening.
    printReceipt(["chek"], 80, "");
    await vi.waitFor(() => expect(PrintLines).toHaveBeenCalledTimes(2));
    expect(PrintLines.mock.calls[1]![1]).toMatchObject({ openDrawer: false });
  });

  it("uses the browser dialog when there is no till application", () => {
    const frames = vi.spyOn(document.body, "appendChild");
    printReceipt(["chek"], 80, "");
    // The browser till is real — a tablet on the floor, a laptop at a demo.
    expect(frames).toHaveBeenCalled();
  });
});

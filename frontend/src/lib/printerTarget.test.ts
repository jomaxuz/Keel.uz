import { describe, expect, it } from "vitest";
import { buildTarget, connectionOf, readTarget } from "./printerTarget";

// ⚠️ **This is the part that fails invisibly.** A wrong address saves, appears
// in the list, and never prints — the spooler discards a job for a name it does
// not know, and a socket to a machine that is not there fails by timing out.
// Nothing on any screen says so until a real ticket does not come out, at the
// worst moment of the evening.

describe("building the stored address", () => {
  it("wraps a Windows name and keeps its spelling exactly", () => {
    // Real driver names: spaces, brackets, dots. The spooler wants this string
    // character for character.
    expect(buildTarget("usb", { name: "XP-58" })).toBe("usb://XP-58");
    expect(buildTarget("usb", { name: "EPSON TM-T20III Receipt" })).toBe(
      "usb://EPSON TM-T20III Receipt",
    );
    expect(buildTarget("usb", { name: "XP-58 (Copy 1)" })).toBe(
      "usb://XP-58 (Copy 1)",
    );
  });

  it("always writes the port, including the default one", () => {
    // ⚠️ The line is read later by somebody deciding whether a printer that
    // stopped working is on a different port. Leaving 9100 off makes them look
    // it up.
    expect(buildTarget("lan", { ip: "192.168.1.50" })).toBe(
      "tcp://192.168.1.50:9100",
    );
    expect(buildTarget("lan", { ip: "192.168.1.50", port: 9101 })).toBe(
      "tcp://192.168.1.50:9101",
    );
  });

  it("returns nothing when the answer is missing, rather than a broken line", () => {
    // ⚠️ An empty target is refused by the server (cleanPrinters drops it), so
    // half an answer must not become "usb://" — a printer that saves and prints
    // nothing is worse than one the form would not let you save.
    expect(buildTarget("usb", { name: "  " })).toBe("");
    expect(buildTarget("lan", { ip: "" })).toBe("");
  });
});

describe("reading an address back into the form", () => {
  it("round-trips everything the form can build", () => {
    const cases: Array<[Parameters<typeof buildTarget>[0], Record<string, unknown>]> = [
      ["usb", { name: "EPSON TM-T20III Receipt" }],
      ["lan", { ip: "192.168.1.50", port: 9100 }],
      ["lan", { ip: "10.0.0.9", port: 9101 }],
    ];
    for (const [how, v] of cases) {
      const target = buildTarget(how, v);
      const back = readTarget(target);
      expect(buildTarget(back.how, back)).toBe(target);
    }
  });

  it("opens a printer added from the panel without rewriting it", () => {
    // ⚠️ The panel has always been one free-text box, so this has to recognise
    // every shape the server accepts — not only what this form produces.
    // Otherwise a printer typed there comes back as "other" and the next save
    // from the till silently changes an address nobody touched.
    expect(readTarget("192.168.1.50")).toMatchObject({
      how: "lan",
      ip: "192.168.1.50",
      port: 9100,
    });
    expect(readTarget("tcp://192.168.1.50:9100")).toMatchObject({
      how: "lan",
      ip: "192.168.1.50",
    });
    expect(readTarget("share://XP-58")).toMatchObject({ how: "usb", name: "XP-58" });
  });

  it("keeps a line it cannot take apart, instead of losing it", () => {
    // A COM port and a UNC share are both real and neither is a question this
    // form asks. They open as "other" with the text intact.
    for (const t of ["serial://COM3", "\\\\PC\\XP-58", "device:///dev/usb/lp0"]) {
      const back = readTarget(t);
      expect(back.how).toBe("other");
      expect(buildTarget(back.how, back)).toBe(t);
    }
  });
});

describe("the badge in the list", () => {
  it("names how each one is attached", () => {
    expect(connectionOf("usb://XP-58")).toBe("usb");
    expect(connectionOf("tcp://192.168.1.50:9100")).toBe("lan");
    expect(connectionOf("serial://COM3")).toBe("other");
  });
});

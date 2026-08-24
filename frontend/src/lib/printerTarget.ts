// Turning "how is it plugged in" into the one line the server stores.
//
// ⚠️ **A form of three shapes over a field of one string, and the string is
// still the truth.** `models.Printer.Target` is deliberately one box the owner
// can paste into — but "paste the line from the self-test page" is advice for
// somebody who has the self-test page, and the person connecting a kitchen
// printer at eight in the evening has an IP address written on a sticker. So
// the form asks the question they can answer and builds the line for them.
//
// ⚠️ **Here rather than inside the component, because it is the part that can
// be wrong invisibly.** A mistyped scheme is a printer that saves, appears in
// the list, tests green in every way the screen can see, and never prints. The
// server parses the same string (internal/printer.Parse) and the tests below
// hold both ends of that agreement.

/** How somebody says a printer is attached. */
export type Connection = "usb" | "lan" | "other";

/** The port every ESC/POS printer listens on, and the one thing about network
 *  printers that is genuinely universal. */
export const DEFAULT_PORT = 9100;

/** Build the stored address from what the form asked.
 *
 *  ⚠️ **Nothing is trimmed off a Windows name.** Printers are called "EPSON
 *  TM-T20III Receipt" and "XP-58 (Copy 1)"; the spooler wants that spelling
 *  exactly, and a name one character out is a receipt that never prints with
 *  no error anybody sees — the job is discarded. */
export function buildTarget(
  how: Connection,
  v: { name?: string; ip?: string; port?: number; raw?: string },
): string {
  switch (how) {
    case "usb": {
      const name = (v.name ?? "").trim();
      return name ? `usb://${name}` : "";
    }
    case "lan": {
      const ip = (v.ip ?? "").trim();
      if (!ip) return "";
      // ⚠️ The port is always written, even when it is the default. The line is
      // read later by a person deciding whether a printer that stopped working
      // is on a different port, and "tcp://192.168.1.50" makes them look it up.
      const port = v.port && v.port > 0 ? v.port : DEFAULT_PORT;
      return `tcp://${ip}:${port}`;
    }
    default:
      return (v.raw ?? "").trim();
  }
}

/** Read a stored address back into the form's three questions.
 *
 *  ⚠️ **A printer added from the panel must open correctly here**, and the
 *  panel has always been a single free-text box — so this has to recognise
 *  every shape internal/printer.Parse accepts, not only the ones buildTarget
 *  produces. Anything it cannot take apart opens as "other" with the line
 *  intact, which is the honest answer and never loses what was typed. */
export function readTarget(target: string): {
  how: Connection;
  name: string;
  ip: string;
  port: number;
  raw: string;
} {
  const t = (target ?? "").trim();
  const base = { how: "other" as Connection, name: "", ip: "", port: DEFAULT_PORT, raw: t };

  const usb = /^(usb|share|printer):\/\/(.+)$/i.exec(t);
  if (usb) return { ...base, how: "usb", name: usb[2]!.trim() };

  const lan = /^(tcp|net|socket|lan):\/\/([^/]+)$/i.exec(t);
  if (lan) {
    const [host, port] = splitHostPort(lan[2]!);
    return { ...base, how: "lan", ip: host, port };
  }
  // ⚠️ A bare host with no scheme is a network printer — Parse reads it that
  // way, so the form must too, or a printer typed in the panel as
  // "192.168.1.50" would come back as "other" and be rewritten on the next save
  // into something the person never chose.
  if (/^[\d.]+(:\d+)?$/.test(t)) {
    const [host, port] = splitHostPort(t);
    return { ...base, how: "lan", ip: host, port };
  }
  return base;
}

function splitHostPort(s: string): [string, number] {
  const i = s.lastIndexOf(":");
  if (i < 0) return [s, DEFAULT_PORT];
  const port = Number(s.slice(i + 1));
  if (!Number.isFinite(port) || port <= 0) return [s.slice(0, i), DEFAULT_PORT];
  return [s.slice(0, i), port];
}

/** A one-word badge for the list: how this printer is attached.
 *
 *  ⚠️ Derived from the address rather than stored beside it. A stored copy is a
 *  second fact that disagrees with the first the day somebody edits the line —
 *  and the badge is what a person scans to find the kitchen printer. */
export function connectionOf(target: string): Connection {
  return readTarget(target).how;
}

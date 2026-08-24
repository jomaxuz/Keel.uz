import KeelMark from "@/components/till/KeelMark";
import PrinterSettings from "@/components/till/PrinterSettings";

// The printer step of setting a monoblock up.
//
// ⚠️ **The panel itself is the till's**, imported rather than written again:
// the same controls appear in the till's settings, and a second copy would be
// two printer screens within a month — with the one being described on a
// support call always the other one. This file is the *placement*: page chrome,
// and a button that continues into the till.
//
// ⚠️ **Here, at the end of pairing, because of who is standing there.** Whoever
// binds the machine is next to the printer with paper in reach, which is the
// only moment a test print can be checked by the person who pressed it. Later
// the same panel is reached from the till's own settings, or with Ctrl+Shift+P
// when nobody can unlock the screen.
export default function Printer({
  onDone,
  doneLabel,
}: {
  onDone: () => void;
  /** "Continue" during setup, "Close" when opened by the hotkey — the same
   *  screen answering two different questions about what happens next. */
  doneLabel: string;
}) {
  return (
    <div className="till h-dvh overflow-y-auto bg-cream">
      <div className="grid min-h-full place-items-center p-6">
        <div className="w-full max-w-[26rem]">
          {/* ⚠️ Dressed as the setup screen it follows and the lock screen it
              comes before: these are the same machine talking about itself, and
              a step in another style reads as a different program. */}
          <div className="mb-6 flex flex-col items-center gap-3">
            <KeelMark className="h-10 w-10" />
            <h1 className="font-display text-xl text-ink">Kassa printeri</h1>
          </div>

          <div className="card p-5">
            <PrinterSettings />
          </div>

          <button
            type="button"
            className="btn btn-dark mt-4 w-full"
            onClick={onDone}
          >
            {doneLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

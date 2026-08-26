"use client";

// What became of a print, said where somebody has to see it.
//
// ⚠️ **A dialog rather than a toast, for these three buttons only.** An X
// report, a Z report and a reprint have nothing else to show for themselves:
// the entire result of pressing them is paper, so a message that fades after
// three seconds is indistinguishable from no message at all — which is exactly
// how these were reported as broken. Everywhere else in the till a toast is
// right, because something visible happened on screen as well.
//
// ⚠️ **The wording never claims paper came out when it cannot know.** A local
// printer that accepted the job is "printed"; the browser's dialog being opened
// is "sent", because the browser does not tell us what happened next. Claiming
// otherwise is the one lie a person standing next to a silent printer catches
// immediately.

import { LuCircleAlert, LuCircleCheck, LuPrinter } from "react-icons/lu";

import { useAdminT } from "@/lib/i18n/admin";
import type { PrintOutcome } from "@/lib/print";

export default function PrintResultDialog({
  outcome,
  onClose,
}: {
  outcome: PrintOutcome;
  onClose: () => void;
}) {
  const t = useAdminT();
  const ok = outcome === "printed";
  const sent = outcome === "browser";

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      // ⚠️ Tapping the backdrop closes it. A cashier between two guests should
      // not have to find a button, and there is nothing here to lose.
      onClick={onClose}
    >
      <div
        className="w-full max-w-[22rem] rounded-2xl bg-surface p-5 text-center"
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className={`mx-auto grid h-14 w-14 place-items-center rounded-full text-3xl ${
            ok
              ? "bg-emerald-500/10 text-emerald-600"
              : sent
                ? "bg-sky-500/10 text-sky-600"
                : "bg-danger/10 text-danger"
          }`}
        >
          {ok ? <LuCircleCheck /> : sent ? <LuPrinter /> : <LuCircleAlert />}
        </div>

        <h2 className="mt-3 text-lg font-bold">
          {ok
            ? t.printResult.printed
            : sent
              ? t.printResult.sent
              : t.printResult.failed}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">
          {ok
            ? t.printResult.printedHint
            : sent
              ? t.printResult.sentHint
              : t.printResult.failedHint}
        </p>

        <button
          type="button"
          onClick={onClose}
          className="till-btn-primary mt-4 w-full justify-center"
        >
          {t.common.close}
        </button>
      </div>
    </div>
  );
}

"use client";

// Asking, and telling, in our own words.
//
// ⚠️ **`window.confirm` is not our screen.** It is drawn by the browser, in the
// browser's font, with the browser's buttons — and above them, in the till's
// case, the domain name of the restaurant's own site. On a machine that is
// meant to look like a cash register, and to a customer who is buying a
// product rather than a web page, that is the moment the illusion drops. It is
// also unstyleable, untranslatable beyond the two words the browser chooses for
// OK and Cancel, and it **blocks the JavaScript thread**: nothing repaints, no
// poll runs, no toast appears until somebody answers it.
//
// ⚠️ **One primitive for both surfaces, in two looks.** The till is pressed
// with a thumb — its buttons are large and its dialog is the `till-dialog` the
// rest of that screen already uses; the panel is read with a mouse. Two
// components would drift, and the drift would be in the confirmation of exactly
// the actions nobody can undo.
//
// ⚠️ **Promise-based on purpose.** Every call site was `if (!confirm(...))
// return;` — the shape a person reading the code understands at a glance. This
// keeps that shape (`if (!(await ask(...))) return;`) rather than asking twenty
// screens to be rewritten around a callback.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { useAdminT } from "@/lib/i18n/admin";

export type AskOptions = {
  /** The question, in one line. */
  title: string;
  /** What it will do, when that is not obvious from the question. */
  body?: string;
  /** The word on the button that does it. Defaults to "Ha"/"OK". */
  confirmLabel?: string;
  cancelLabel?: string;
  /** Whether the action destroys something. Colours the button, nothing else. */
  danger?: boolean;
};

type Ask = (o: AskOptions) => Promise<boolean>;
type Tell = (o: Omit<AskOptions, "cancelLabel" | "danger">) => Promise<void>;

/** ⚠️ **The fallback is the browser's own dialog, and it must never be
 *  reached.** A surface that forgets to mount the provider keeps working —
 *  losing a confirmation because a layout was missed would be the worse bug —
 *  and it looks wrong enough to be reported. */
const Ctx = createContext<{ ask: Ask; tell: Tell }>({
  ask: async (o) =>
    window.confirm([o.title, o.body].filter(Boolean).join("\n\n")),
  tell: async (o) => {
    window.alert([o.title, o.body].filter(Boolean).join("\n\n"));
  },
});

export function useAsk() {
  return useContext(Ctx);
}

type Pending = AskOptions & {
  kind: "ask" | "tell";
  resolve: (v: boolean) => void;
};

export default function AskProvider({
  look = "panel",
  children,
}: {
  /** Which machine this is: a counter pressed with a thumb, or the panel. */
  look?: "till" | "panel";
  children: React.ReactNode;
}) {
  const t = useAdminT();
  const [open, setOpen] = useState<Pending | null>(null);
  // ⚠️ Held in a ref as well, so the keyboard handler below answers the dialog
  // that is actually on screen rather than the one that was there when the
  // listener was attached.
  const live = useRef<Pending | null>(null);
  live.current = open;

  const answer = useCallback((v: boolean) => {
    const p = live.current;
    live.current = null;
    setOpen(null);
    p?.resolve(v);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!live.current) return;
      if (e.key === "Escape") answer(false);
      // ⚠️ Enter confirms only what is not destructive. A panel where Return
      // deletes a dish is a panel that deletes a dish while somebody is typing.
      if (e.key === "Enter" && !live.current.danger) answer(true);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [answer]);

  const value = useMemo(() => {
    const push = (kind: "ask" | "tell", o: AskOptions) =>
      new Promise<boolean>((resolve) => {
        // ⚠️ A second question while one is open answers the first with "no".
        // It cannot be queued: the code waiting on it is holding a lock, a
        // spinner or a half-written check, and two stacked dialogs is how a
        // cashier confirms something they never read.
        const prev = live.current;
        if (prev) prev.resolve(false);
        const next = { ...o, kind, resolve };
        live.current = next;
        setOpen(next);
      });
    return {
      ask: (o: AskOptions) => push("ask", o),
      tell: async (o: Omit<AskOptions, "cancelLabel" | "danger">) => {
        await push("tell", o);
      },
    };
  }, []);

  const till = look === "till";

  return (
    <Ctx.Provider value={value}>
      {children}
      {open && (
        <div
          // ⚠️ Above the on-screen keyboard, which is `z-[60]`. A question about
          // a discount, drawn *behind* the pad the discount was typed on, is
          // the one dialog nobody can answer.
          className="fixed inset-0 z-[70] flex items-end justify-center bg-ink/50 p-4 sm:items-center"
          // ⚠️ The backdrop cancels, like every other dialog in this product —
          // and like the browser's own, which closes on Escape. It never
          // confirms: a stray press on the dark area must not be an answer.
          onClick={() => answer(false)}
          role="presentation"
        >
          <div
            role="alertdialog"
            aria-modal="true"
            className={
              till
                ? "till-dialog w-full max-w-sm p-5"
                : "w-full max-w-sm rounded-2xl border border-line bg-surface p-5 shadow-card"
            }
            onClick={(e) => e.stopPropagation()}
          >
            <h2
              className={
                till
                  ? "font-display text-lg font-bold"
                  : "font-display text-lg font-semibold text-ink"
              }
            >
              {open.title}
            </h2>
            {open.body && (
              <p className="mt-2 text-sm text-ink-soft">{open.body}</p>
            )}

            <div className="mt-5 flex gap-2">
              {/* ⚠️ Cancel on the left and confirm on the right, the order the
                  rest of these screens already use. A dialog that swaps them is
                  a dialog answered by muscle memory and read afterwards. */}
              {open.kind === "ask" && (
                <button
                  type="button"
                  className={till ? "till-btn flex-1" : "btn-ghost flex-1"}
                  onClick={() => answer(false)}
                >
                  {open.cancelLabel ?? t.common.cancel}
                </button>
              )}
              <button
                type="button"
                autoFocus
                className={
                  till
                    ? `${open.danger ? "till-btn-danger" : "till-btn-primary"} flex-1`
                    : `${
                        open.danger
                          ? "btn border border-rose-600 bg-rose-600 text-white hover:bg-rose-700"
                          : "btn-primary"
                      } flex-1`
                }
                onClick={() => answer(true)}
              >
                {open.confirmLabel ??
                  (open.kind === "tell" ? t.common.close : t.common.yes)}
              </button>
            </div>
          </div>
        </div>
      )}
    </Ctx.Provider>
  );
}

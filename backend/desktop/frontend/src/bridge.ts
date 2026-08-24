// The Go side, as the screen sees it.
//
// ⚠️ **Defined in the shared tree, re-exported here.** The till screens print,
// and they live in `frontend/src` — so the interface to the Go side has to be
// reachable from there or `lib/print.ts` would carry a second hand-written copy
// of it. Two copies of a hand-written FFI signature is the drift that produces
// a method renamed on one side only, and the symptom is a till that silently
// stops printing.
//
// This file stays because the shell's own screens import from it, and because
// the path says what it is.
export * from "@/lib/tillBridge";

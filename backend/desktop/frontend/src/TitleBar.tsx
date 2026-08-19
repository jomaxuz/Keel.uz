import { LuX } from "react-icons/lu";
import { bridge } from "./bridge";

// The window's own controls, because the frame is gone.
//
// ⚠️ **Drag and close, and deliberately no minimise.** Behind the till there is
// nothing to reveal, and a minimised till is a support call that begins "the
// cash register disappeared" (pos-reja.md §2).
//
// ⚠️ Closing asks nothing here. The plan is explicit: if the app survives a
// power cut, it survives being closed, and a scary confirmation on every exit
// teaches people to click through confirmations. A reminder belongs here only
// once there are open checks to name.
export default function TitleBar() {
  const b = bridge();
  if (!b) return null; // an ordinary browser tab has its own chrome
  return (
    <header
      className="till-chrome flex h-9 shrink-0 items-center justify-between pl-4 pr-1"
      // Wails reads this property to decide what drags the window.
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <span className="font-poppins text-sm font-medium tracking-tight text-ink-muted">
        Keel
      </span>
      <button
        onClick={() => void b.Quit()}
        aria-label="Yopish"
        className="till-btn-ghost min-h-7 px-2"
        // ⚠️ Excluded from dragging, or the button moves the window instead of
        // being pressed — and the till cannot be closed at all.
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        <LuX className="h-4 w-4" aria-hidden />
      </button>
    </header>
  );
}

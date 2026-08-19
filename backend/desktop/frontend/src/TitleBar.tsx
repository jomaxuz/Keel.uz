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
      // Wails reads this property to decide what drags the window.
      style={{
        // @ts-expect-error -- a Wails custom property, not in React's types
        "--wails-draggable": "drag",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        height: 36,
        padding: "0 8px 0 16px",
        background: "#111",
        color: "#fff",
        userSelect: "none",
      }}
    >
      <span style={{ fontSize: 13, letterSpacing: 0.3 }}>Keel Kassa</span>
      <button
        onClick={() => void b.Quit()}
        aria-label="Yopish"
        style={{
          // ⚠️ Excluded from dragging, or the button moves the window instead
          // of being pressed — and the till cannot be closed at all.
          // @ts-expect-error -- a Wails custom property, not in React's types
          "--wails-draggable": "no-drag",
          width: 44,
          height: 28,
          border: 0,
          borderRadius: 6,
          background: "transparent",
          color: "#fff",
          fontSize: 16,
          cursor: "pointer",
        }}
      >
        ✕
      </button>
    </header>
  );
}

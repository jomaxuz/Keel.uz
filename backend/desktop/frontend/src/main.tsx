import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

// ⚠️ Imported from the shared tree on purpose: this one import is what proves
// the alias, the TypeScript paths and the bundle all reach `frontend/src`. If
// it ever stops resolving, the build fails here rather than three screens in.
import { formatPrice } from "@/lib/format";

function Boot() {
  return (
    <main style={{ font: "16px system-ui", padding: 32 }}>
      <h1>Keel Kassa</h1>
      <p>Wails o‘rami tayyor. Umumiy kod ulandi: {formatPrice(1234500, "UZS", "uz")}</p>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Boot />
  </StrictMode>,
);

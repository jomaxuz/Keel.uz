import { useState } from "react";
import { bridge, type BranchView } from "./bridge";

// The first screen a monoblock ever shows, and ideally the only time anybody
// sees it.
//
// ⚠️ **Two steps, not one.** Signing in proves the person is allowed to bind
// this machine; choosing the branch is a separate press because binding to the
// wrong one sends this till's receipts to another kitchen and its sales to
// another report — and nothing on this screen would look wrong afterwards.
export default function Setup({ onPaired }: { onPaired: () => void }) {
  const [address, setAddress] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [branches, setBranches] = useState<BranchView[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function connect(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await bridge()!.Connect(address, username, password);
      setBranches(res.branches);
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function pair(id: string) {
    setBusy(true);
    setError("");
    try {
      await bridge()!.Pair(id);
      onPaired();
    } catch (err) {
      setError(String(err));
      setBusy(false);
    }
  }

  return (
    <div style={S.wrap}>
      <div style={S.card}>
        <h1 style={S.title}>Kassani ulash</h1>

        {branches === null ? (
          <form onSubmit={connect}>
            <p style={S.hint}>
              Bu kompyuter qaysi restoranga tegishli? Bir marta ulanadi — keyin
              faqat PIN so‘raladi.
            </p>
            <label style={S.label}>
              Restoran manzili
              <input
                style={S.input}
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="osh"
                autoFocus
                required
              />
            </label>
            {/* The one thing somebody might get subtly wrong, so it is answered
                before it is asked rather than in a support call. */}
            <p style={S.note}>
              Saytingiz manzili. Faqat nomni yozsangiz yetarli — <code>osh</code>{" "}
              → <code>osh.keel.uz</code>. O‘z domeningiz bo‘lsa to‘liq yozing.
            </p>
            <label style={S.label}>
              Login
              <input
                style={S.input}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </label>
            <label style={S.label}>
              Parol
              <input
                style={S.input}
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </label>
            <p style={S.note}>
              Ega yoki menejer hisobi. Parol bu kompyuterda saqlanmaydi.
            </p>
            {error && <p style={S.error}>{error}</p>}
            <button style={S.button} disabled={busy}>
              {busy ? "Ulanmoqda…" : "Davom etish"}
            </button>
          </form>
        ) : (
          <div>
            <p style={S.hint}>Bu kassa qaysi filialda turibdi?</p>
            {error && <p style={S.error}>{error}</p>}
            <div style={{ display: "grid", gap: 8, marginTop: 16 }}>
              {branches.map((b) => (
                <button
                  key={b.id}
                  style={S.branch}
                  disabled={busy}
                  onClick={() => void pair(b.id)}
                >
                  {b.name}
                </button>
              ))}
            </div>
            <button
              style={{ ...S.button, background: "transparent", color: "#57534e", marginTop: 16 }}
              disabled={busy}
              onClick={() => setBranches(null)}
            >
              Orqaga
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

const S: Record<string, React.CSSProperties> = {
  wrap: {
    minHeight: "100vh",
    display: "grid",
    placeItems: "center",
    background: "#fafaf9",
    font: "16px system-ui",
  },
  card: {
    width: 420,
    padding: 32,
    background: "#fff",
    borderRadius: 16,
    boxShadow: "0 1px 3px rgba(0,0,0,.08), 0 8px 24px rgba(0,0,0,.06)",
  },
  title: { margin: "0 0 8px", fontSize: 24 },
  hint: { margin: "0 0 20px", color: "#57534e", lineHeight: 1.5 },
  label: { display: "block", marginBottom: 4, fontSize: 14, color: "#44403c" },
  input: {
    display: "block",
    width: "100%",
    marginTop: 6,
    marginBottom: 4,
    padding: "12px 14px",
    fontSize: 16,
    border: "1px solid #d6d3d1",
    borderRadius: 10,
    boxSizing: "border-box",
  },
  note: { margin: "0 0 16px", fontSize: 13, color: "#78716c", lineHeight: 1.5 },
  error: {
    margin: "0 0 16px",
    padding: "10px 12px",
    background: "#fef2f2",
    color: "#b91c1c",
    borderRadius: 8,
    fontSize: 14,
  },
  button: {
    width: "100%",
    padding: "13px 16px",
    fontSize: 16,
    border: 0,
    borderRadius: 10,
    background: "#1c1917",
    color: "#fff",
    cursor: "pointer",
  },
  branch: {
    padding: "14px 16px",
    fontSize: 16,
    textAlign: "left",
    border: "1px solid #d6d3d1",
    borderRadius: 10,
    background: "#fff",
    cursor: "pointer",
  },
};

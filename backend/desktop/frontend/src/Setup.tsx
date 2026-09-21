import { useState } from "react";
import { LuBuilding2, LuChevronRight } from "react-icons/lu";
import KeelMark from "@/components/till/KeelMark";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { LANGS, type Lang } from "@/lib/i18n";
import { bridge, type BranchView } from "./bridge";

// The first screen a monoblock ever shows, and ideally the only time anybody
// sees it.
//
// ⚠️ **Two steps, not one.** Signing in proves the person is allowed to bind
// this machine; choosing the branch is a separate press because binding to the
// wrong one sends this till's receipts to another kitchen and its sales to
// another report — and nothing on this screen would look wrong afterwards.
//
// ⚠️ **Dressed as the lock screen it comes before.** The mark, the wordmark and
// the panel are the ones PinPad uses, because these two screens are the same
// machine talking about itself — and a setup screen in some other style reads
// as a different program, on the one occasion somebody is deciding whether to
// trust it with the restaurant's money.
export default function Setup({ onPaired }: { onPaired: () => void }) {
  // ⚠️ **Three languages on the first screen too.** Every other screen in the
  // till has had them all along; this one was written inside the shell, where
  // the dictionary was out of reach, and stayed Uzbek. It is the screen a
  // machine is set up on — often by whoever delivers it, who may not read
  // Uzbek — and the one where somebody decides whether to trust this program
  // with the restaurant's money.
  const t = useAdminT().till.setup;
  const [address, setAddress] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [branches, setBranches] = useState<BranchView[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // Set once the branch is chosen: the last question before the printer.
  const [pairedTo, setPairedTo] = useState("");

  async function connect(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      setBranches((await bridge()!.Connect(address, username, password)).branches);
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
      setPairedTo(id);
      // ⚠️ **Cleared on success, and forgetting it was a dead screen.** This
      // used to end by unmounting — `onPaired()` was called here — so a `busy`
      // left true was harmless. Putting the "till or floor?" question between
      // pairing and the printer left the screen mounted with the flag still
      // set, and both buttons on it are `disabled={busy}`: the setup showed a
      // choice nobody could make, on every machine installed since.
      setBusy(false);
    } catch (err) {
      setError(String(err));
      setBusy(false);
    }
  }

  // ⚠️ **Asked once, here, and not by the person who unlocks the machine.**
  // Which screen this is — the counter or the dining room — is a fact about
  // where it stands, and it does not change between shifts. Asking a waiter
  // forty times an evening would be the same question with a worse answer.
  async function chooseMode(mode: "kassa" | "zal") {
    setBusy(true);
    try {
      await bridge()!.SetMode(mode);
      onPaired();
    } catch (err) {
      setError(String(err));
      setBusy(false);
    }
  }

  return (
    // ⚠️ The full window, and it carries the `till` palette scope itself: there
    // is no wrapper any more, because a wrapper is what put a scrollbar down
    // the side of the till. The scroll lives here rather than on the page so a
    // short screen can still reach the button, without the window ever growing
    // taller than itself.
    <div className="till h-dvh overflow-y-auto bg-cream">
      <div className="grid min-h-full place-items-center p-6">
        <div className="w-full max-w-[26rem]">
        {/* ⚠️ Our colour and our type, not the restaurant's — the same reasoning
            as the lock screen: `text-brand` and the theme fonts would draw a
            different Keel in every install. */}
        {/* ⚠️ **Above the mark, not in a settings screen.** Whoever is
            standing here has no account yet and nowhere else to go: a language
            they cannot read is the whole screen, and a switch they have to
            find first is a switch that is not there. */}
        <LanguagePicker label={t.language} />

        <div className="mb-7 flex items-center justify-center gap-3">
          <KeelMark className="h-11 w-11 text-keel-deep" />
          <span className="font-poppins text-3xl font-semibold tracking-tight text-ink">
            Keel
          </span>
        </div>

        <div className="till-panel p-6">
          {branches === null ? (
            <form onSubmit={connect}>
              <h1 className="text-base font-semibold">{t.title}</h1>
              <p className="mt-1 text-sm text-ink-muted">{t.lead}</p>

              <label className="till-label mt-6 block" htmlFor="address">
                {t.address}
              </label>
              <input
                id="address"
                className="till-input mt-1.5"
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                placeholder="osh"
                autoFocus
                required
              />
              {/* The one thing somebody might get subtly wrong, answered before
                  it is asked rather than in a support call. */}
              <p className="mt-1.5 text-xs text-ink-muted">
                {t.addressHint}{" "}
                <span className="till-num">osh</span> →{" "}
                <span className="till-num">osh.keel.uz</span>
              </p>

              <label className="till-label mt-5 block" htmlFor="username">
                {t.login}
              </label>
              <input
                id="username"
                className="till-input mt-1.5"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />

              <label className="till-label mt-4 block" htmlFor="password">
                {t.password}
              </label>
              <input
                id="password"
                type="password"
                className="till-input mt-1.5"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
              <p className="mt-1.5 text-xs text-ink-muted">
                Ega yoki menejer hisobi. Parol bu kompyuterda saqlanmaydi.
              </p>

              {error && <Problem text={error} />}

              {/* ⚠️ Exactly one accent control per screen (globals.css): it is
                  always the thing that moves the setup forward. */}
              <button className="till-btn-accent mt-6 w-full" disabled={busy}>
                {busy ? t.connecting : t.connect}
              </button>
            </form>
          ) : pairedTo ? (
            <div>
              <h1 className="text-base font-semibold">{t.whatTitle}</h1>
              <p className="mt-1 text-sm text-ink-muted">
                Keyin ham o'zgartirish mumkin (Ctrl+Shift+M).
              </p>

              {error && <Problem text={error} />}

              {/* ⚠️ Quiet, not accent, for the reason the branch list is: there
                  is no recommended answer — a restaurant that bought one
                  machine wants the till, one that bought four wants three floor
                  screens, and an accent on either row would make the choice
                  look already made. */}
              <div className="mt-5 grid gap-2">
                <button
                  className="till-btn-quiet w-full px-4 py-3 text-left"
                  disabled={busy}
                  onClick={() => void chooseMode("kassa")}
                >
                  <span className="block font-medium">{t.kassa}</span>
                  <span className="block text-xs text-ink-muted">{t.kassaHint}</span>
                </button>
                <button
                  className="till-btn-quiet w-full px-4 py-3 text-left"
                  disabled={busy}
                  onClick={() => void chooseMode("zal")}
                >
                  <span className="block font-medium">{t.zal}</span>
                  <span className="block text-xs text-ink-muted">{t.zalHint}</span>
                </button>
              </div>
            </div>
          ) : (
            <div>
              <h1 className="text-base font-semibold">{t.branchTitle}</h1>
              <p className="mt-1 text-sm text-ink-muted">{t.branchHint}</p>

              {error && <Problem text={error} />}

              {/* ⚠️ Quiet, not accent: there is no "recommended" branch, and an
                  accent on every row would make the choice look already made. */}
              <div className="mt-5 grid gap-2">
                {branches.map((b) => (
                  <button
                    key={b.id}
                    className="till-btn-quiet w-full justify-between px-4 py-3 text-left"
                    disabled={busy}
                    onClick={() => void pair(b.id)}
                  >
                    <span className="flex items-center gap-2.5">
                      <LuBuilding2
                        className="h-4 w-4 text-[rgb(var(--till-dim))]"
                        aria-hidden
                      />
                      {b.name}
                    </span>
                    <LuChevronRight
                      className="h-4 w-4 text-[rgb(var(--till-dim))]"
                      aria-hidden
                    />
                  </button>
                ))}
              </div>

              <button
                className="till-btn-ghost mt-4 w-full"
                disabled={busy}
                onClick={() => setBranches(null)}
              >
                Orqaga
              </button>
            </div>
          )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ⚠️ Named in red, not filled in it — the same rule as till-btn-danger. A panel
// that turns red says the machine is broken; this says one attempt did not work.
function Problem({ text }: { text: string }) {
  return (
    <p
      className="mt-5 rounded-[11px] border px-3 py-2.5 text-sm text-danger"
      style={{ borderColor: "rgb(var(--till-accent-line))" }}
      role="alert"
    >
      {text}
    </p>
  );
}


/** Three letters, on the one screen that has no other way to change them.
 *
 *  ⚠️ The choice is kept by `setLang` in the provider's own cookie, which is
 *  what the shell reads when it starts — so a till set up in Russian opens in
 *  Russian tomorrow, rather than asking again every morning. */
function LanguagePicker({ label }: { label: string }) {
  const { lang, setLang } = useI18n();
  return (
    <div className="mb-4 flex items-center justify-center gap-2">
      <span className="sr-only">{label}</span>
      {LANGS.map((l: Lang) => (
        <button
          key={l}
          type="button"
          onClick={() => setLang(l)}
          aria-pressed={l === lang}
          className={`rounded-full px-3 py-1 text-xs font-semibold uppercase transition ${
            l === lang
              ? "bg-ink text-cream"
              : "text-ink-muted hover:text-ink"
          }`}
        >
          {l}
        </button>
      ))}
    </div>
  );
}

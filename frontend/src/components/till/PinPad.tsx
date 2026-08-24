"use client";

import { useEffect, useState } from "react";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import { LuDelete, LuLock } from "react-icons/lu";

import { api, ApiError, imageUrl, setTillToken } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import LangSwitch from "@/components/site/LangSwitch";
import SubscriptionCorner from "@/components/till/SubscriptionCorner";
import type { TillPerson, TillSession } from "@/lib/types";

import KeelMark from "./KeelMark";

/**
 * The lock screen: who is standing at this till.
 *
 * ⚠️ **A till is a shared screen**, and until this existed one account stayed
 * signed in all evening — so every void, every discount and every closed check
 * was recorded against whoever unlocked it at six. Those records exist to
 * answer one question, and the login model was quietly answering it wrong.
 *
 * ⚠️ **The PIN is not the authentication.** The monoblock already holds a token
 * for its branch, obtained once with a real username and password. The four
 * digits only say *who*. That is why they can be four digits at all — and why
 * the token they buy is weaker than the one behind it.
 *
 * ---- The shape of it ----
 *
 * ⚠️ **Two panels, and only one of them is the till.** This is the first screen
 * of the evening and the one a guest at the counter can see over the cashier's
 * shoulder, so the half that faces the room is the restaurant's own picture and
 * the half under the hand is nothing but the code. A pad centred on a 15" panel
 * is typed at with a raised arm several hundred times a shift; a pad in its own
 * column is typed at with a resting one.
 *
 * ⚠️ **The left panel disappears below `md`, the pad never does.** The same URL
 * gets opened on a phone to check something, and a lock screen that cannot be
 * got past is a till that cannot be sold from. The branding is the part that is
 * allowed to go.
 *
 * ⚠️ **This screen comes up on every till, including one where nobody has been
 * given a code yet.** It used to be skipped entirely on a branch with no PINs,
 * which meant the machine that most needed asking "who is standing here" was
 * the one machine that never asked — and the screen only appeared on the day
 * somebody finally set a code, as a surprise mid-service. A lock screen that is
 * sometimes there is a lock screen nobody has a habit for.
 *
 * ⚠️ **So there has to be a way through it, and `fallback` is that way.** A
 * branch with no codes is handed a named button rather than a pad it cannot
 * answer: the person is already signed in with a real staff login, so the till
 * knows who they are and only lacks the four digits. Without it this change
 * would lock every no-PIN till out of its own evening.
 */
/** How many digits a till code has. Fixed, so the pad can draw exactly that
 *  many dots and submit by itself — see pinDigits on the server. */
const PIN_DIGITS = 4;

export default function PinPad({
  onUnlock,
  session,
  online = true,
  fallback,
}: {
  onUnlock: (person: TillPerson) => void;
  /** What this monoblock knows about itself — the brand it belongs to, the
   *  branch it stands in, and the pictures the owner put on this screen. Null
   *  while the first answer is still in flight. */
  session?: TillSession | null;
  /** Whether the last question we asked the server got an answer. Drawn as one
   *  dot, because it is the only thing on a locked till that can be wrong. */
  online?: boolean;
  /** The way past a pad nobody has a code for — see the header.
   *
   *  ⚠️ Passed only when the branch has issued **no** codes at all. The moment
   *  one person has a PIN, everybody types one: an escape hatch that survives
   *  alongside real codes is the hatch the whole shift uses, and every void
   *  goes back to being anonymous. */
  fallback?: { name: string; onContinue: () => void };
}) {
  const t = useAdminT();
  const [pin, setPin] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  // ⚠️ **Submits itself on the fourth digit.** A confirming tap after every
  // code is a tap added to the busiest screen in the building, and the pad
  // knows exactly when the code is complete because the length is fixed.
  useEffect(() => {
    if (pin.length === PIN_DIGITS) void submit(pin);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pin]);

  async function submit(code: string) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const res = await api.tillUnlock(code);
      setTillToken(res.token);
      setPin("");
      onUnlock(res.staff);
    } catch (e) {
      // ⚠️ The server's own words. It distinguishes "wrong code" from "too many
      // tries, wait N seconds", and a cashier who cannot tell those apart
      // retypes the same PIN and extends their own lockout.
      setError(e instanceof ApiError ? e.message : t.till.retry);
      setPin("");
    } finally {
      setBusy(false);
    }
  }

  function tap(d: string) {
    if (busy) return;
    setError("");
    setPin((p) => (p.length >= PIN_DIGITS ? p : p + d));
  }

  return (
    // ⚠️ **No card. The screen is the layout.**
    //
    // This was drawn as a floating panel on a cream page, and it was wrong in a
    // way that is hard to see in a browser and obvious on the counter: a
    // monoblock is a fixed 1024×768 sheet of glass in a fixed frame, and a card
    // centred on it is a second frame drawn inside the first. Everything the
    // card cost — its own margins, its own corners, its own shadow — came out
    // of the two things this screen is for, and the picture ended up a
    // postcard. So the page *is* the card: white ground, the restaurant's
    // poster down the left, the keypad down the right.
    //
    // ⚠️ `overflow-y-auto` as a floor, not a design: a monoblock is 768px tall
    // and this fits, but the same screen is opened on a phone to check
    // something, and a lock screen whose last row of keys is off the edge
    // cannot be got past at all.
    <div className="till relative flex min-h-dvh w-full overflow-y-auto bg-surface md:h-dvh md:overflow-hidden">
      {/* ⚠️ **The language switch belongs on this screen, not behind it.** The
          till is a shared machine and the person who set it up is not the
          person standing at it now; a cashier who reads Russian met an
          Uzbek-only lock screen and had no way past it to the switch that would
          have fixed the whole shift. */}
      {/* ⚠️ **The one screen where the notice gets a sentence.** The lock
          screen is the only surface in this app with room to spare and the only
          one somebody reads while *not* mid-transaction — a monoblock waking up
          between guests, a waiter picking the tablet off the pass. The corner
          badge on the chrome bar catches a glance; this catches a read, and it
          is the same fact from the same field. */}
      <div className="absolute right-4 top-4 z-10 flex items-start gap-2">
        <SubscriptionCorner notice={session?.subscription} size="full" />
        <LangSwitch />
      </div>

      {/* ---- The restaurant's half ----

          ⚠️ **Half the screen, inset from the glass rather than bled to it.**
          Padding and a radius are the whole difference between "the restaurant
          put a poster on this machine" and "this machine has a photograph
          stuck to its edge" — a full-bleed picture fights the bezel and looks
          like a rendering fault on a monoblock with a dark frame.

          ⚠️ The width is the *window's* half, so the picture is 46% of the
          glass once the inset is taken — which is what makes it read as a
          poster beside the keypad rather than as decoration behind it. */}
      <div className="hidden w-1/2 shrink-0 p-5 md:block">
        <BannerPanel banners={session?.banners ?? []} />
      </div>

      {/* ---- The till's half ---- */}
      <div className="flex min-w-0 flex-1 flex-col px-6 py-5 sm:px-10">

            {/* ⚠️ **Pinned to the top of its own column, not stacked on the
                pad.** Whose machine this is and what to type into it are two
                different statements: run together as one centred block they
                read as a heading for the keypad, and the mark stops being the
                first thing the eye lands on. Sitting at the top it labels the
                whole screen, which is what a mark is for.

                ⚠️ Here on every width. It used to appear only on the phone
                layout, so the wide screen — the one this actually runs on — had
                no mark over the keys at all: the picture panel carried it, half
                a screen away from the thing being typed into. */}
            {/* ⚠️ Held off the top edge, and not only for the look: the
                language switch floats in that corner, and a mark level with it
                reads as one row of chrome with a logo parked in it. */}
            <div className="flex shrink-0 items-center justify-center gap-2.5 pt-4 md:pt-10">
              <KeelMark className="h-14 w-14 text-keel-deep" />
              {/* ⚠️ Our own type, not the restaurant's. The theme fonts dress
                  the restaurant — its menu, its site, its receipts — and
                  letting them reset our name would make the wordmark different
                  in every install. */}
              <span className="font-poppins text-[38px] font-semibold tracking-tight text-ink">
                Keel
              </span>
            </div>

            {/* ⚠️ **Centred in what the mark leaves, with a little weight
                towards the top.** A group centred in the whole column sits low
                enough that the pad drifts under the middle of a 768px panel —
                below the natural resting height of a hand on a counter, which
                is the one measurement this screen is laid out around. */}
            <div className="flex min-h-0 flex-1 flex-col items-center justify-center pb-[6%] pt-6">
            {/* ⚠️ **The closed padlock, and it is the other half of the
                header's open one.** The button that got you here shows an open
                lock; this screen shows it shut. Two states of one object say
                "this machine is locked now" faster than a sentence does. */}
            <h1 className="flex items-center justify-center gap-2 text-[21px] font-bold tracking-tight">
              <LuLock className="h-4 w-4 text-ink-muted" aria-hidden />
              {t.till.pinTitle}
            </h1>
            <p className="mt-1 text-center text-sm text-ink-muted">
              {t.till.pinHint}
            </p>

            {/* Dots rather than digits: the pad is at head height in a room
                with guests and colleagues in it. */}
            <div className="mt-5 flex justify-center gap-4">
              {Array.from({ length: PIN_DIGITS }, (_, i) => (
                <span
                  key={i}
                  className={`h-4 w-4 rounded-full transition-all ${
                    i < pin.length ? "scale-110 bg-keel-deep" : "bg-ink/[0.12]"
                  }`}
                />
              ))}
            </div>

            {/* ⚠️ The row is always there, empty or not: a message that appears
                pushes the whole pad down, and the key under the finger changes
                between the tap that failed and the retry. */}
            <p className="mt-2.5 h-5 text-center text-sm font-medium text-danger">
              {error}
            </p>

            {/* Big targets: this is tapped hundreds of times a day, often with
                a wet hand, on a screen at arm's length. */}
            <div className="mt-1 grid w-[16.5rem] grid-cols-3 gap-2.5">
              {["1", "2", "3", "4", "5", "6", "7", "8", "9"].map((d) => (
                <PadKey key={d} onClick={() => tap(d)}>
                  {d}
                </PadKey>
              ))}
              {/* No confirm key: the pad submits on the fourth digit, so a tick
                  would be a control that is never the right thing to press. */}
              <span />
              <PadKey onClick={() => tap("0")}>0</PadKey>
              <PadKey
                onClick={() => setPin("")}
                disabled={pin.length === 0}
                label={t.till.pinClear}
              >
                <LuDelete className="h-[1.35rem] w-[1.35rem]" aria-hidden />
              </PadKey>
            </div>

            {/* ---- No codes on this branch yet ----

                ⚠️ **Under the pad, not instead of it.** The pad stays because
                the first person to be given a code must find it where they will
                find it every day afterwards; the button is the exception, and
                it looks like one. A screen that replaced the pad with a
                "continue" button would teach the room that this till has no
                lock — right up to the evening it suddenly does. */}
            {fallback && (
              <div className="mt-6 w-[16.5rem] border-t border-line pt-4 text-center">
                <p className="text-xs leading-snug text-ink-muted">
                  {t.till.pinNoneHint}
                </p>
                <button
                  type="button"
                  onClick={fallback.onContinue}
                  className="till-btn-quiet mt-2.5 w-full justify-center"
                >
                  {t.till.pinContinueAs(fallback.name)}
                </button>
              </div>
            )}
            </div>

        {/* ⚠️ Along the bottom of the till's half, not across the screen: the
            other half is the restaurant's poster, and a status bar laid over it
            would be our chrome printed on their picture. */}
        <StatusStrip session={session} online={online} />
      </div>
    </div>
  );
}

/**
 * Which machine this is, what time it is, and whether it can reach us.
 *
 * ⚠️ **Along the bottom of the card, across both columns.** These are facts
 * about the *machine*, not about the code — putting them beside the pad would
 * make somebody read them on their way to typing, several hundred times a
 * shift, and putting them in a corner of the page would leave them floating
 * next to nothing.
 *
 * ⚠️ **The clock is not decoration.** The till runs fullscreen with no OS
 * chrome, so between locking and unlocking there is nowhere else on the machine
 * to read the time — and the first thing somebody arriving for a shift wants to
 * know is whether they are late. The header carries the same clock for the same
 * reason.
 *
 * ⚠️ **The name is the branch's, and it is here to catch a specific mistake.**
 * A chain sets its monoblocks up from a link, and a link opened on the wrong
 * machine binds that machine to the wrong branch — after which every check,
 * every shift and every stop-list edit lands in another building's books. The
 * only moment anybody would notice is this one, before the first sale, and only
 * if the screen says which branch it thinks it is.
 */
function StatusStrip({
  session,
  online,
}: {
  session?: TillSession | null;
  online: boolean;
}) {
  const t = useAdminT();
  return (
    <div className="flex shrink-0 items-center gap-2.5 pt-3 text-[13px] sm:gap-3">
      {/* ⚠️ **A light, and the only one on this screen.** Everything else here
          is always true; this is the one fact that can be wrong, and it is the
          answer to the question a cashier asks when a sale will not save.

          ⚠️ **The words are gone from the glass, not from the screen.** Green
          against red is the distinction roughly one man in twelve cannot make,
          so dropping the label outright would have left those cashiers a dot
          that means nothing. Two things carry it instead: the state is in the
          accessible name and the tooltip, and the offline dot is drawn
          *differently* — bigger, with a ring — so it is told apart by size as
          well as by hue, which is the part that survives colour blindness and a
          glance from two metres. */}
      <span
        className={`shrink-0 rounded-full ${online ? "h-2.5 w-2.5" : "h-3 w-3"}`}
        style={{
          background: online ? "rgb(var(--till-ok))" : "rgb(var(--till-late))",
          // A halo rather than a bigger dot alone: at 2.5mm on a matte panel
          // the colour is easy to miss from where the cashier stands.
          boxShadow: online
            ? "0 0 0 3px rgb(var(--till-ok) / 0.18)"
            : "0 0 0 4px rgb(var(--till-late) / 0.22)",
        }}
        role="img"
        aria-label={online ? t.till.linkOnline : t.till.linkOffline}
        title={online ? t.till.linkOnline : t.till.linkOffline}
      />

      {/* ⚠️ Truncated rather than wrapped. A brand name is free text and a long
          one used to push the clock off the end of a 1024px card — and the
          clock is the half of this strip somebody actually looks at. */}
      <span className="min-w-0 truncate font-semibold text-ink">
        {session?.brandName || "—"}
      </span>
      {/* ⚠️ Dropped when it repeats the brand. A single-branch restaurant names
          its one branch after itself, so this line read "Maracanda · Maracanda"
          on the majority install — and a label that says the same thing twice
          is one nobody reads the second half of, including on the chain where
          the second half is the whole point. */}
      {session?.branchName && session.branchName !== session.brandName && (
        <span className="min-w-0 truncate text-ink-muted">
          · {session.branchName}
        </span>
      )}

      <span className="flex-1" />
      <StripClock />
    </div>
  );
}

/** ⚠️ Rendered only after mount, like the header's: the server has no idea what
 *  time it is on the counter, and a server-rendered clock is a hydration
 *  mismatch reported as an error on the busiest screen in the building. */
function StripClock() {
  const [now, setNow] = useState<Date | null>(null);
  useEffect(() => {
    setNow(new Date());
    // Once a second would repaint sixty times a minute for a display that only
    // changes once; the interval keeps it roughly on the minute.
    const timer = setInterval(() => setNow(new Date()), 15_000);
    return () => clearInterval(timer);
  }, []);
  // The row keeps its height while the clock is missing, so the strip does not
  // grow a pixel on hydration.
  if (!now) return <span className="h-5" />;
  const p = (n: number) => String(n).padStart(2, "0");
  return (
    <span className="flex shrink-0 items-baseline gap-2.5">
      {/* Built by hand rather than with toLocaleDateString, which follows the
          device language and prints "8/23/2026" on an English tablet — the same
          rule as everywhere else on the till (see lib/format). */}
      <span className="till-num text-ink-muted">
        {p(now.getDate())}.{p(now.getMonth() + 1)}.{now.getFullYear()}
      </span>
      <span className="till-num text-[17px] font-bold text-ink">
        {p(now.getHours())}:{p(now.getMinutes())}
      </span>
    </span>
  );
}

/**
 * The restaurant's own poster, on the machine in its own dining room.
 *
 * ⚠️ **It rotates because a lock screen is looked at every few minutes for a
 * whole shift**, and one picture stops being seen by the second hour. Four is
 * about the point where it stops being a slot and starts being something the
 * owner has to keep filling, which is why the settings page recommends three to
 * four rather than accepting any number silently.
 *
 * ⚠️ **Nothing here is a control.** The banner is not tappable and the dots are
 * not buttons: this sits on a *locked* screen, and the only thing a tap on it
 * may do is bring the keypad to the finger. A picture that navigated somewhere
 * would be a way past the lock — small, but a lock with a small way past it is
 * not a lock. The server drops the link on a till banner for the same reason.
 */
function BannerPanel({ banners }: { banners: string[] }) {
  const t = useAdminT();
  const [i, setI] = useState(0);
  const n = banners.length;

  useEffect(() => {
    // ⚠️ One picture does not rotate, and a timer for it would repaint the
    // panel every six seconds for the rest of the evening.
    if (n < 2) return;
    setI((x) => (x < n ? x : 0));
    const timer = setInterval(() => setI((x) => (x + 1) % n), ROTATE_MS);
    return () => clearInterval(timer);
  }, [n]);

  // ⚠️ **The fallback is ours, not an empty box.** A restaurant that has not
  // uploaded anything — which is every restaurant on its first day — gets the
  // Keel panel rather than a grey rectangle that reads as a picture failing to
  // load. Drawn rather than photographed: the till has to come up with no
  // network at all, since it is the screen that sells when the wifi is out.
  if (n === 0) {
    return (
      <div className="relative flex h-full w-full flex-col justify-end overflow-hidden rounded-[22px] bg-[linear-gradient(150deg,#F5A524_0%,#E89412_45%,#C97A08_100%)] p-8 text-white">
        <KeelMark className="pointer-events-none absolute -bottom-20 -right-16 h-[24rem] w-[24rem] text-white/15" />
        <p className="relative max-w-[16rem] text-[19px] font-semibold leading-snug">
          {t.till.pinPanelHint}
        </p>
      </div>
    );
  }

  return (
    // ⚠️ **Fills the card rather than holding a fixed ratio.** A monoblock, a
    // 1280×800 tablet and a phone in a stand are three different shapes, so
    // there is no one aspect that fits the slot on all of them — and a picture
    // locked to 4:5 left white margins on two of the three. `object-cover`
    // instead, which is why the settings page says to keep anything that
    // matters away from the edges rather than promising an exact crop.
    <div className="relative h-full w-full overflow-hidden rounded-[22px] bg-[rgb(var(--till-quiet))]">
      {banners.map((src, idx) => (
        // ⚠️ Cross-faded by stacking all of them, not by swapping one `src`.
        // Changing the source makes the browser fetch the next picture at the
        // moment it is needed, so the panel goes blank for as long as the
        // restaurant's wifi takes — on the screen whose whole job is to look
        // finished.
        <img
          key={src}
          src={imageUrl(src, 1200) ?? ""}
          alt=""
          className="absolute inset-0 h-full w-full object-cover transition-opacity duration-700"
          style={{ opacity: idx === i ? 1 : 0 }}
        />
      ))}
      {n > 1 && (
        <div className="absolute inset-x-0 bottom-4 flex justify-center gap-1.5">
          {banners.map((src, idx) => (
            // Dashes rather than dots: on a photograph a round dot disappears
            // into whatever is behind it, and this strip has to survive every
            // picture a restaurant might upload.
            <span
              key={src}
              className={`h-1 rounded-full transition-all ${
                idx === i ? "w-6 bg-white" : "w-3 bg-white/45"
              }`}
              style={{ boxShadow: "0 1px 3px rgb(0 0 0 / 0.35)" }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/** How long each banner holds the panel.
 *
 *  ⚠️ Slow on purpose. Nobody is watching this — it is glanced at on the way
 *  past — and a carousel that moves while somebody is typing their code is
 *  motion in the corner of the eye of the person using the screen. */
const ROTATE_MS = 7000;

function PadKey({
  children,
  onClick,
  disabled,
  label,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  label?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      // ⚠️ No drop shadow: on a matte monoblock it reads as a smudge, and the
      // first thing anybody does about a smudge is wipe the screen. The key is
      // separated by a hairline and a press state instead.
      className="flex h-[3.6rem] items-center justify-center rounded-[13px] border border-line bg-surface font-display text-[26px] font-bold text-ink transition hover:bg-ink/[0.03] active:scale-[0.96] active:bg-ink/10 disabled:opacity-30"
    >
      {children}
    </button>
  );
}

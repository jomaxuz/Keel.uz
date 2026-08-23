"use client";

/**
 * The till's own keyboard.
 *
 * ⚠️ **A monoblock is not a phone, and the OS keyboard makes it look like
 * one.** Android's and Windows' virtual keyboards arrive with their own colours,
 * their own suggestion strip, their own emoji key and their own idea of how
 * tall they should be — half a screen on a 1024×768 panel, over the very total
 * the cashier is typing next to. The first thing that happens is the guest sees
 * a browser wearing a phone's keyboard, and the second is that somebody presses
 * the emoji key into a price field.
 *
 * ⚠️ **`inputMode="none"` is the whole mechanism.** It is the one thing that
 * tells every browser "this field is typed into by something else" while
 * keeping the element focused, the caret visible and selection working. Making
 * the field `readOnly` instead would suppress the keyboard too — and also the
 * caret, the selection and every browser's idea that this is an editable
 * control, which is how a "the field is broken" support call starts. The
 * original `inputMode` is remembered and put back on blur: it is what decides
 * whether *our* pad comes up numeric.
 *
 * ⚠️ **Written through the native setter, not by assignment.** React installs
 * its own `value` setter on the input prototype and remembers the last value it
 * wrote; assigning `el.value` directly leaves React believing nothing changed,
 * so the next re-render puts the old text back — a keyboard whose keys work
 * once and then silently stop. The prototype setter plus a bubbling `input`
 * event is exactly what a real keystroke does.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { LuChevronUp, LuDelete, LuGlobe } from "react-icons/lu";

import { useAdminT } from "@/lib/i18n/admin";

/** What a field wants typed into it. Decided from the field's own `inputMode`
 *  and `type`, so nothing has to be marked up specially — the money fields
 *  already say `inputMode="numeric"` because they always needed to. */
type Pad = "numeric" | "text";

/** Which set of letters. ⚠️ Both, because the staff are not one language: a
 *  Russian-speaking cashier searching the menu for "Лагман" on a Latin-only pad
 *  finds nothing and concludes the search is broken. Same key as a Mac's. */
type Script = "latin" | "cyrillic";

type Editable = HTMLInputElement | HTMLTextAreaElement;

/** Fields the pad stays out of.
 *
 *  ⚠️ **The PIN pad is not one of these.** It draws its own keys and never
 *  focuses an input at all — there is nothing here to suppress. Listed for the
 *  ones that would otherwise be caught: a native picker draws its own UI, and a
 *  file input has no text. */
const SKIP_TYPES = new Set([
  "checkbox",
  "radio",
  "file",
  "range",
  "color",
  "date",
  "time",
  "datetime-local",
  "month",
  "week",
  "submit",
  "button",
  "reset",
  "image",
]);

function isEditable(el: Element | null): el is Editable {
  if (!el) return false;
  if (el instanceof HTMLTextAreaElement) return !el.readOnly && !el.disabled;
  if (!(el instanceof HTMLInputElement)) return false;
  if (SKIP_TYPES.has(el.type)) return false;
  return !el.readOnly && !el.disabled;
}

/** Numeric unless the field says otherwise.
 *
 *  ⚠️ Read from the field's **remembered** mode, not from the live attribute:
 *  by the time this is asked the live one has already been set to "none" to
 *  keep the OS pad away, and reading it back would make every field textual —
 *  including every price on the screen. */
function padFor(el: Editable, original: string | null): Pad {
  const mode = (original ?? "").toLowerCase();
  if (mode === "numeric" || mode === "decimal" || mode === "tel") return "numeric";
  if (el instanceof HTMLInputElement && (el.type === "number" || el.type === "tel")) {
    return "numeric";
  }
  return "text";
}

/** Type into the field the way a keystroke does — see the header. */
function setValue(el: Editable, next: string, caret: number) {
  const proto =
    el instanceof HTMLTextAreaElement
      ? HTMLTextAreaElement.prototype
      : HTMLInputElement.prototype;
  const setter = Object.getOwnPropertyDescriptor(proto, "value")?.set;
  if (setter) setter.call(el, next);
  else el.value = next;
  // Before the event: a controlled field re-renders on it, and a caret restored
  // beforehand survives because React does not touch it when the value matches.
  try {
    el.setSelectionRange(caret, caret);
  } catch {
    // Some input types refuse selection (email, number in some browsers). The
    // text is in; the caret lands at the end, which is where it was going.
  }
  el.dispatchEvent(new Event("input", { bubbles: true }));
}

export default function OnScreenKeyboard() {
  const t = useAdminT();
  const [target, setTarget] = useState<Editable | null>(null);
  const [pad, setPad] = useState<Pad>("text");
  const [script, setScript] = useState<Script>("latin");
  const [shift, setShift] = useState(false);
  const [symbols, setSymbols] = useState(false);
  // The field's own inputMode, borrowed for as long as we hold the field.
  const original = useRef<string | null>(null);
  // ⚠️ **The same fact as `target`, in a ref, because focus does not wait for
  // React.** Moving from one field to the next fires the new field's `focusin`
  // before any re-render, so the handler has to be able to hand the *previous*
  // field back from inside itself — and reading it from state would read the
  // value from before the last focus.
  const held = useRef<Editable | null>(null);
  const board = useRef<HTMLDivElement | null>(null);

  /** Give a field back exactly what we took from it.
   *
   *  ⚠️ **Removing the attribute is not the same as clearing it.** A field with
   *  no `inputMode` of its own must end with none: setting it to "" leaves the
   *  attribute present, `padFor` reads "" as textual, and every price field on
   *  the till comes up with letters — after the first time somebody typed into
   *  it, which is the shape of bug nobody can reproduce on request. */
  const giveBack = useCallback((el: Editable | null) => {
    if (!el) return;
    if (original.current === null) el.removeAttribute("inputmode");
    else el.setAttribute("inputmode", original.current);
    original.current = null;
  }, []);

  /** Hand the field back and stop drawing. */
  const release = useCallback(() => {
    giveBack(held.current);
    held.current = null;
    setTarget(null);
    setShift(false);
    setSymbols(false);
  }, [giveBack]);

  // ---- Which field is being typed into ----
  useEffect(() => {
    function onFocus(e: FocusEvent) {
      const el = e.target as Element | null;
      if (!isEditable(el)) return;
      // An opt-out for anything that must keep the OS pad — a barcode field
      // driven by a scanner, say. Nothing uses it yet; it exists so that the
      // answer to "this one field needs the real keyboard" is an attribute
      // rather than a fork of this component.
      if (el.closest("[data-osk='off']")) return;
      // The field we were on, before we take the next one — see `held`.
      if (held.current && held.current !== el) giveBack(held.current);
      held.current = el;
      original.current = el.getAttribute("inputmode");
      setPad(padFor(el, original.current));
      el.setAttribute("inputmode", "none");
      setTarget(el);
      setShift(false);
      setSymbols(false);
    }
    document.addEventListener("focusin", onFocus);
    return () => document.removeEventListener("focusin", onFocus);
  }, [giveBack]);

  // ---- Putting it away ----
  //
  // ⚠️ **Anything that is not the field or the pad closes it**, which is the
  // rule the cashier already believes: they press a button and expect the
  // keyboard to be gone. Tying it to `blur` alone would not do it — a tap on a
  // plain `<div>` does not move focus in Safari, so the pad would sit over the
  // total until something focusable was pressed.
  useEffect(() => {
    if (!target) return;
    function onDown(e: PointerEvent) {
      const n = e.target as Node | null;
      if (!n) return;
      if (board.current?.contains(n)) return;
      if (target?.contains(n) || n === target) return;
      // Moving straight to another field: that field's own focus handler takes
      // over on the next tick, so the pad stays up and simply re-aims.
      release();
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") release();
    }
    // Capture, so a button that stops propagation still puts the pad away: the
    // dish grid's tiles do exactly that.
    document.addEventListener("pointerdown", onDown, true);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown, true);
      document.removeEventListener("keydown", onKey);
    };
  }, [target, release]);

  // A field removed from the page while it was being typed into — a dialog
  // closed by its own Cancel — leaves the pad pointing at nothing.
  useEffect(() => {
    if (!target) return;
    const check = setInterval(() => {
      if (!target.isConnected) release();
    }, 500);
    return () => clearInterval(check);
  }, [target, release]);

  // ⚠️ **The page gets shorter, it does not get covered.** A pad drawn over the
  // bottom of a dialog hides the button the typing was for, and the cashier
  // types the amount and then cannot find "To'lash". The class is on the root
  // so every till screen reacts to it without knowing this component exists.
  useEffect(() => {
    const root = document.documentElement;
    if (target) root.classList.add("osk-open");
    else root.classList.remove("osk-open");
    return () => root.classList.remove("osk-open");
  }, [target]);

  // Keep the field the finger is in above the pad.
  useEffect(() => {
    if (!target) return;
    const id = window.setTimeout(() => {
      target.scrollIntoView({ block: "center", behavior: "smooth" });
    }, 60);
    return () => window.clearTimeout(id);
  }, [target]);

  // ---- The keys ----

  const press = useCallback(
    (ch: string) => {
      const el = target;
      if (!el) return;
      const start = el.selectionStart ?? el.value.length;
      const end = el.selectionEnd ?? start;
      const next = el.value.slice(0, start) + ch + el.value.slice(end);
      setValue(el, next, start + ch.length);
      // One capital, then back to lower case — a Mac's shift, not caps lock.
      if (shift) setShift(false);
    },
    [target, shift],
  );

  const backspace = useCallback(() => {
    const el = target;
    if (!el) return;
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? start;
    if (start === end) {
      if (start === 0) return;
      setValue(el, el.value.slice(0, start - 1) + el.value.slice(end), start - 1);
      return;
    }
    setValue(el, el.value.slice(0, start) + el.value.slice(end), start);
  }, [target]);

  /** Return. ⚠️ On a single-line field this is "done", and it also submits: the
   *  forms on this screen are one field and a button, and a cashier who has
   *  just typed the float expects the pad to get out of the way. A textarea
   *  gets a real newline — a comment for the kitchen is often two lines. */
  const enter = useCallback(() => {
    const el = target;
    if (!el) return;
    if (el instanceof HTMLTextAreaElement) {
      press("\n");
      return;
    }
    el.form?.requestSubmit?.();
    el.blur();
    release();
  }, [target, press, release]);

  if (!target) return null;

  return (
    <div
      ref={board}
      // ⚠️ Nothing here takes focus. A key that stole it would blur the field
      // mid-word, and the next key would have nowhere to type.
      onPointerDown={(e) => e.preventDefault()}
      // Exempts the pad from the rule that shortens every `.till` to make room
      // for it — see globals.css, `html.osk-open`.
      data-osk-board=""
      className="till fixed inset-x-0 bottom-0 z-[60] select-none border-t border-line bg-[rgb(var(--till-quiet))] px-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] pt-2 shadow-[0_-8px_28px_rgb(0_0_0/0.10)]"
      role="group"
      aria-label={t.till.keyboard}
    >
      <div className="mx-auto flex w-full max-w-[62rem] flex-col gap-1.5">
        {/* One row of chrome: what is being typed, and the way out. The tick is
            duplicated on the pad itself; this one is for the hand that is
            already up here. */}
        <div className="flex items-center gap-2 px-1 pb-0.5">
          <span className="truncate text-[12px] font-semibold text-[rgb(var(--till-dim))]">
            {fieldLabel(target) || t.till.keyboard}
          </span>
          <span className="flex-1" />
          <button
            type="button"
            onClick={release}
            className="flex h-8 items-center gap-1.5 rounded-[9px] px-2.5 text-[12px] font-bold text-[rgb(var(--till-mid))] hover:bg-surface"
          >
            <LuChevronUp className="h-4 w-4 rotate-180" aria-hidden />
            {t.till.keyboardHide}
          </button>
        </div>

        {pad === "numeric" ? (
          <NumericPad
            onKey={press}
            onBack={backspace}
            onDone={enter}
            doneLabel={t.till.keyboardDone}
          />
        ) : (
          <TextPad
            script={script}
            shift={shift}
            symbols={symbols}
            onKey={press}
            onBack={backspace}
            onDone={enter}
            onShift={() => setShift((s) => !s)}
            onSymbols={() => setSymbols((s) => !s)}
            onScript={() => setScript((s) => (s === "latin" ? "cyrillic" : "latin"))}
            doneLabel={t.till.keyboardDone}
            spaceLabel={t.till.keyboardSpace}
          />
        )}
      </div>
    </div>
  );
}

/** What the cashier is typing into, so the pad is not an anonymous slab.
 *
 *  ⚠️ **The placeholder wins only when it is a sentence.** On this screen the
 *  good ones are the whole question ("Chek, stol yoki ofitsiant") — but the
 *  money fields use "0" as a placeholder, and a keyboard captioned "0" is a
 *  keyboard captioned nothing while the label above it says "Kassadagi
 *  boshlang'ich pul". Anything under three characters is a hint about the
 *  format, not a name for the field. */
function fieldLabel(el: Editable): string {
  const ph = (el.getAttribute("placeholder") ?? "").trim();
  if (ph.length > 2) return ph;
  const aria = (el.getAttribute("aria-label") ?? "").trim();
  if (aria) return aria;
  const label = el.closest("label")?.textContent?.trim();
  if (label) {
    // The label of a wrapping <label> includes the field's own text; the name
    // is the first line of it.
    return label.split("\n")[0]!.trim().slice(0, 60);
  }
  return ph;
}

// ---- Key shapes ----

function Key({
  children,
  onPress,
  className = "",
  wide,
  tone = "light",
  label,
}: {
  children: React.ReactNode;
  onPress: () => void;
  className?: string;
  wide?: boolean;
  tone?: "light" | "dark" | "accent";
  label?: string;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      // ⚠️ `onPointerUp`, not `onClick`. The container cancels `pointerdown` to
      // protect the focus, and a cancelled press never becomes a click in
      // Safari — the keys worked on Android and did nothing on an iPad.
      onPointerUp={onPress}
      className={`flex h-[3.1rem] flex-1 items-center justify-center rounded-[10px] border text-[17px] font-semibold transition active:scale-[0.94] ${
        tone === "dark"
          ? "border-line-strong bg-[rgb(var(--till-quiet))] text-[rgb(var(--till-mid))] active:bg-ink/10"
          : tone === "accent"
            ? "border-transparent bg-[rgb(var(--till-accent))] text-ink active:brightness-95"
            : "border-line bg-surface text-ink shadow-[0_1px_0_rgb(0_0_0/0.06)] active:bg-ink/[0.06]"
      } ${wide ? "flex-[2.2]" : ""} ${className}`}
    >
      {children}
    </button>
  );
}

function Row({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-1.5">{children}</div>;
}

// ---- Numbers ----

/** ⚠️ Phone order (1 at the top), not calculator order.
 *
 *  Every other pad these hands touch is a phone — including the PIN screen two
 *  taps away, which is the one they use most. A till that flips the rows costs
 *  a mis-key on the first day and a wrong price on a bad one. */
function NumericPad({
  onKey,
  onBack,
  onDone,
  doneLabel,
}: {
  onKey: (c: string) => void;
  onBack: () => void;
  onDone: () => void;
  doneLabel: string;
}) {
  return (
    <div className="mx-auto flex w-full max-w-[26rem] flex-col gap-1.5">
      {[
        ["1", "2", "3"],
        ["4", "5", "6"],
        ["7", "8", "9"],
      ].map((row) => (
        <Row key={row[0]}>
          {row.map((d) => (
            <Key key={d} onPress={() => onKey(d)} className="text-[22px]">
              {d}
            </Key>
          ))}
        </Row>
      ))}
      <Row>
        {/* ⚠️ No decimal point: money here is whole so'm and always has been —
            see the currency rule. A key that types a character the server
            strips is a key that teaches the cashier the pad is lying. */}
        <Key onPress={() => onKey("000")} tone="dark" className="text-[18px]">
          000
        </Key>
        <Key onPress={() => onKey("0")} className="text-[22px]">
          0
        </Key>
        <Key onPress={onBack} tone="dark" label="Backspace">
          <LuDelete className="h-5 w-5" aria-hidden />
        </Key>
      </Row>
      <Row>
        <Key onPress={onDone} tone="accent" className="text-[16px]">
          {doneLabel}
        </Key>
      </Row>
    </div>
  );
}

// ---- Letters ----

const LATIN = [
  ["q", "w", "e", "r", "t", "y", "u", "i", "o", "p"],
  ["a", "s", "d", "f", "g", "h", "j", "k", "l"],
  ["z", "x", "c", "v", "b", "n", "m"],
];

/** ⚠️ ЙЦУКЕН, not a transliteration of QWERTY. A Russian-speaking cashier types
 *  by muscle memory from every other keyboard they have ever used. */
const CYRILLIC = [
  ["й", "ц", "у", "к", "е", "н", "г", "ш", "щ", "з", "х"],
  ["ф", "ы", "в", "а", "п", "р", "о", "л", "д", "ж", "э"],
  ["я", "ч", "с", "м", "и", "т", "ь", "б", "ю"],
];

const SYMBOLS = [
  ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"],
  ["-", "/", ":", ";", "(", ")", "so'm", "&", "@"],
  [".", ",", "?", "!", "'", "ʻ", "%", "+"],
];

function TextPad({
  script,
  shift,
  symbols,
  onKey,
  onBack,
  onDone,
  onShift,
  onSymbols,
  onScript,
  doneLabel,
  spaceLabel,
}: {
  script: Script;
  shift: boolean;
  symbols: boolean;
  onKey: (c: string) => void;
  onBack: () => void;
  onDone: () => void;
  onShift: () => void;
  onSymbols: () => void;
  onScript: () => void;
  doneLabel: string;
  spaceLabel: string;
}) {
  const rows = symbols ? SYMBOLS : script === "latin" ? LATIN : CYRILLIC;
  const cap = (c: string) => (shift && !symbols ? c.toUpperCase() : c);

  return (
    <div className="flex flex-col gap-1.5">
      {rows.map((row, i) => (
        <Row key={i}>
          {/* The middle rows are inset the way a real keyboard's are: it is what
              makes the three rows read as a keyboard rather than as a grid, and
              the eye finds "a" faster for it. */}
          {i > 0 && !symbols && <span className="w-4 shrink-0 sm:w-7" />}
          {i === 2 && !symbols && (
            <Key onPress={onShift} tone={shift ? "accent" : "dark"} label="Shift">
              ⇧
            </Key>
          )}
          {row.map((c) => (
            <Key key={c} onPress={() => onKey(cap(c))} className={c.length > 1 ? "text-[14px]" : ""}>
              {cap(c)}
            </Key>
          ))}
          {i === 2 && (
            <Key onPress={onBack} tone="dark" label="Backspace">
              <LuDelete className="h-5 w-5" aria-hidden />
            </Key>
          )}
          {i > 0 && !symbols && <span className="w-4 shrink-0 sm:w-7" />}
        </Row>
      ))}
      <Row>
        <Key onPress={onSymbols} tone="dark" className="text-[14px]">
          {symbols ? "ABC" : "?123"}
        </Key>
        {/* ⚠️ The globe changes the alphabet, not the interface language: the
            till's own language is the switch in the header, and one control
            doing both would mean a cashier who wanted "Лагман" got a Russian
            till — or worse, the other way round mid-shift. */}
        <Key onPress={onScript} tone="dark" label="Alifbo">
          <LuGlobe className="h-5 w-5" aria-hidden />
        </Key>
        <Key onPress={() => onKey(" ")} className="flex-[6] text-[13px] font-medium text-[rgb(var(--till-dim))]">
          {spaceLabel}
        </Key>
        <Key onPress={onDone} tone="accent" wide className="text-[15px]">
          {doneLabel}
        </Key>
      </Row>
    </div>
  );
}

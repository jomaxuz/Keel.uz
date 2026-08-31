"use client";

// What to write to a restaurant that has never heard of us.
//
// ⚠️ **The messages are written by hand and picked here, not generated.** A
// model writes in a rhythm this market now recognises on sight, and the whole
// problem this screen exists to solve is that these messages get ignored. See
// `lib/outreach.ts` for the rest of that reasoning.
//
// ⚠️ **The one line that matters is left empty on purpose.** "Nima
// ko'rdingiz" is where the sender writes the thing they actually noticed about
// this particular place. It is the entire difference between a message that
// gets a reply and a template, and nothing here can supply it — it is a fact
// about that restaurant that only the person writing knows.

import { useMemo, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { growthDict } from "@/lib/i18n/growth";
import {
  RUNNING_SYSTEMS,
  outreachText,
  variantCount,
  type OutreachKind,
  type OutreachLang,
  type OutreachStage,
} from "@/lib/outreach";

const KINDS: OutreachKind[] = ["telegram", "opening", "running"];

export default function OutreachPage() {
  // ⚠️ **Two languages on one screen, and they are not the same language.**
  // `lang` is the console's — the words around the box. `msgLang` is the
  // message's — the words that go to the restaurant. A Russian-speaking sender
  // still writes Uzbek to an Uzbek street, so tying one to the other would take
  // away the choice this screen exists to offer.
  const { lang } = useT();
  const d = growthDict(lang).out;

  const [kind, setKind] = useState<OutreachKind>("telegram");
  const [msgLang, setMsgLang] = useState<OutreachLang>("uz");
  const [stage, setStage] = useState<OutreachStage>("first");
  const [variant, setVariant] = useState(0);
  const [name, setName] = useState("");
  const [current, setCurrent] = useState("iiko");
  const [note, setNote] = useState("");
  const [copied, setCopied] = useState(false);

  const total = variantCount(kind, msgLang, stage);
  const text = useMemo(
    () => outreachText(kind, msgLang, stage, variant, { name, current, note }),
    [kind, msgLang, stage, variant, name, current, note],
  );

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      // ⚠️ Silent failure would look like a dead button. The textarea is
      // editable and selectable, so the fallback is telling them to select it
      // rather than pretending the copy worked.
      setCopied(false);
      alert(d.copyFailed);
    }
  }

  function pick(next: OutreachKind) {
    setKind(next);
    setVariant(0);
  }

  const tab = (on: boolean) =>
    `rounded-xl px-3 py-2 text-sm font-semibold transition ${
      on ? "bg-ink text-page" : "border border-line text-ink-muted hover:text-ink"
    }`;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="h-display text-2xl">{d.title}</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-muted">
          {d.leadA} <strong className="text-ink">{d.leadStrong}</strong>.{" "}
          {d.leadB}
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
        {/* ---- Choices ---- */}
        <div className="space-y-5">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
              {d.who}
            </p>
            <div className="mt-2 space-y-2">
              {KINDS.map((k) => (
                <button
                  key={k}
                  onClick={() => pick(k)}
                  className={`w-full rounded-2xl border p-3 text-left transition ${
                    kind === k
                      ? "border-signal-500 bg-signal-500/[0.07]"
                      : "border-line hover:bg-raised"
                  }`}
                >
                  <span className="block text-sm font-semibold text-ink">
                    {k === "telegram"
                      ? d.kindTelegram
                      : k === "opening"
                        ? d.kindOpening
                        : d.kindRunning}
                  </span>
                  <span className="mt-0.5 block text-xs text-ink-muted">
                    {k === "telegram"
                      ? d.kindTelegramHint
                      : k === "opening"
                        ? d.kindOpeningHint
                        : d.kindRunningHint}
                  </span>
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-wrap gap-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {d.lang}
              </p>
              <div className="mt-2 flex gap-2">
                <button
                  className={tab(msgLang === "uz")}
                  onClick={() => {
                    setMsgLang("uz");
                    setVariant(0);
                  }}
                >
                  {d.langUz}
                </button>
                <button
                  className={tab(msgLang === "ru")}
                  onClick={() => {
                    setMsgLang("ru");
                    setVariant(0);
                  }}
                >
                  {d.langRu}
                </button>
              </div>
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                {d.stage}
              </p>
              {/* ⚠️ The follow-up is here rather than buried, because it is the
                  message that actually gets replies — and the one people skip
                  writing because the first felt like enough. */}
              <div className="mt-2 flex gap-2">
                <button
                  className={tab(stage === "first")}
                  onClick={() => {
                    setStage("first");
                    setVariant(0);
                  }}
                >
                  {d.stageFirst}
                </button>
                <button
                  className={tab(stage === "follow")}
                  onClick={() => {
                    setStage("follow");
                    setVariant(0);
                  }}
                >
                  {d.stageFollow}
                </button>
              </div>
            </div>
          </div>

          <label className="block text-sm">
            <span className="text-ink-muted">{d.name}</span>
            <input
              className="input mt-1"
              placeholder={d.namePh}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>

          {kind === "running" && (
            <label className="block text-sm">
              <span className="text-ink-muted">{d.current}</span>
              <input
                className="input mt-1"
                list="running-systems"
                value={current}
                onChange={(e) => setCurrent(e.target.value)}
              />
              <datalist id="running-systems">
                {RUNNING_SYSTEMS.map((s) => (
                  <option key={s} value={s} />
                ))}
              </datalist>
            </label>
          )}

          <label className="block text-sm">
            <span className="text-ink-muted">
              {d.note}{" "}
              <span className="text-ink-muted/70">{d.noteOptional}</span>
            </span>
            <textarea
              className="input mt-1 min-h-[5rem]"
              /* ⚠️ Written in the message's language, not the console's: it is
                 an example of the sentence being asked for, and an example in
                 the wrong language teaches the wrong thing. */
              placeholder={
                msgLang === "uz"
                  ? "Instagramda menyungizni ko'rdim, yetkazib berish Telegram orqali ekan."
                  : "Видел ваше меню в Instagram, доставка идёт через Telegram."
              }
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {d.noteHint}
            </span>
          </label>
        </div>

        {/* ---- The message ---- */}
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
              {d.message}
            </p>
            <div className="flex items-center gap-2">
              {total > 1 && (
                <>
                  <span className="text-xs text-ink-muted">
                    {(variant % total) + 1} / {total}
                  </span>
                  {/* ⚠️ Variants matter more than they look: sending one
                      identical message to thirty places is what gets an account
                      limited, and it reads as a mailshot to anybody who
                      compares notes with a neighbour. */}
                  <button
                    className="btn-ghost text-sm"
                    onClick={() => setVariant((v) => v + 1)}
                  >
                    {d.variant}
                  </button>
                </>
              )}
              {/* ⚠️ **Not copyable while the name is blank.** The message
                  renders "[restoran nomi]" where the name goes, and a bracketed
                  blank is only a warning if something stops it being sent —
                  otherwise it goes out at eleven at night unread. */}
              <button
                className="btn-primary"
                disabled={!name.trim()}
                onClick={copy}
              >
                {copied ? d.copied : d.copy}
              </button>
            </div>
          </div>

          {!name.trim() && (
            <p className="text-sm text-signal-600 dark:text-signal-400">
              {d.needName}
            </p>
          )}

          {/* ⚠️ A textarea, not a read-only block: the last edit before sending
              belongs to the person sending it, and making them paste elsewhere
              to change one word is how a good message goes out unchanged. */}
          <textarea
            className="input min-h-[26rem] whitespace-pre-wrap font-normal leading-7"
            value={text}
            onChange={() => {
              /* Editing here is fine and expected; it is re-derived when a
                 choice above changes, and that is the intended behaviour —
                 the choices are the source, this box is the draft. */
            }}
            readOnly
          />

          <div className="rounded-2xl border border-line bg-raised px-4 py-3 text-sm text-ink-soft">
            <p className="font-medium text-ink">{d.checkTitle}</p>
            <ul className="mt-1.5 space-y-1 text-ink-muted">
              <li>• {d.check1}</li>
              <li>• {d.check2}</li>
              <li>• {d.check3}</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}

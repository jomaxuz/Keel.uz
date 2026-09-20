import { readFileSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

// The advisor thread's one rule, tested by reading the source: the failure it
// had is invisible from any assertion about rendered output that does not also
// stub a model call.
const src = readFileSync(
  join(process.cwd(), "src/components/admin/SupportWidget.tsx"),
  "utf8",
);

function body(from: string): string {
  const at = src.indexOf(from);
  expect(at, `${from} is gone from the source`).toBeGreaterThan(-1);
  const rest = src.slice(at);
  const end = rest.indexOf("\n  }\n");
  return rest.slice(0, end === -1 ? undefined : end);
}

describe("the advisor thread", () => {
  // ⚠️ **The question appears when it is sent, not when it is answered.** It
  // used to be appended only on success: typing a question emptied the box and
  // left the thread unchanged for the several seconds a model takes, and a
  // failed question vanished without trace. Every chat anybody has used shows
  // their own message immediately; one that does not reads as a send button
  // that did nothing.
  it("adds the question before the answer is asked for", () => {
    const ask = body("async function ask(question: string)");
    const shown = ask.indexOf("pending: true");
    const sent = ask.indexOf("api.advisorAsk");
    expect(shown, "the question is no longer shown while it is pending").toBeGreaterThan(-1);
    expect(sent).toBeGreaterThan(-1);
    expect(
      shown < sent,
      "the question reaches the thread only after the request — the panel " +
        "shows nothing while the model is working",
    ).toBe(true);
  });

  // ⚠️ **The pending question is not sent back as context.** It has no answer
  // yet, and handing the model an empty reply attributed to itself is how a
  // conversation starts arguing with a blank.
  it("builds the history from answered turns only", () => {
    const ask = body("async function ask(question: string)");
    expect(ask).toContain("filter((x) => x.answer)");
    const built = ask.indexOf("const history =");
    const appended = ask.indexOf("setTurns((prev) => [...prev,");
    expect(built).toBeGreaterThan(-1);
    expect(
      built < appended,
      "the history is built after the question joins the thread, so the " +
        "unanswered question is sent to the model as context",
    ).toBe(true);
  });

  // ⚠️ **A refusal stays with the question that caused it.** A banner at the
  // bottom is about "the last thing that happened" and is wrong the moment
  // anything else happens.
  it("keeps a refusal on its own turn", () => {
    const ask = body("async function ask(question: string)");
    expect(ask).toContain("settle({ failed:");
    expect(
      src.includes("{note && <p"),
      "the bottom banner is back beside the per-turn reason: the same " +
        "sentence twice",
    ).toBe(false);
  });
});

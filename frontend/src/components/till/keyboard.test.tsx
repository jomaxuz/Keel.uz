/**
 * The till's keyboard, tested at the two places it silently fails.
 *
 * ⚠️ **Both of these look fine in a browser and are wrong in production.** A
 * keyboard that writes with `el.value = …` types perfectly into an uncontrolled
 * input and does nothing at all into a React-controlled one, which is every
 * field on the till; and a keyboard that leaves `inputMode` behind turns every
 * field on the screen numeric the moment somebody has typed a price into one.
 * Neither shows up as an error, and both are only visible on the machine.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { LangProvider } from "@/lib/i18n/client";
import { adminUz as t } from "@/lib/i18n/admin";

import OnScreenKeyboard from "./OnScreenKeyboard";

/** Two controlled fields, because a controlled field is the case that breaks:
 *  React writes the value back on every render, so a keyboard that does not go
 *  through the native setter has its keystroke undone before it is seen. */
function Screen() {
  const [text, setText] = useState("");
  const [money, setMoney] = useState("");
  return (
    <>
      <input
        aria-label="izoh"
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <input
        aria-label="summa"
        inputMode="numeric"
        value={money}
        onChange={(e) => setMoney(e.target.value)}
      />
      <button type="button">Boshqa tugma</button>
      <OnScreenKeyboard />
    </>
  );
}

function draw() {
  return render(
    <LangProvider initial="uz">
      <Screen />
    </LangProvider>,
  );
}

/** A key press, the way the pad receives one: `pointerUp`, because the board
 *  cancels `pointerdown` to keep the focus on the field and a cancelled press
 *  never becomes a click. */
function tap(name: string) {
  fireEvent.pointerUp(screen.getByRole("button", { name }));
}

describe("the till keyboard", () => {
  it("does not exist until a field is focused", () => {
    draw();
    expect(screen.queryByRole("group", { name: t.till.keyboard })).toBeNull();
  });

  it("types into a controlled field, and the field keeps the text", () => {
    draw();
    const field = screen.getByLabelText("izoh") as HTMLInputElement;
    fireEvent.focusIn(field);

    tap("k");
    tap("a");
    tap("m");

    expect(field.value).toBe("kam");
  });

  it("keeps the OS keyboard away while it holds the field, and gives the field back", () => {
    draw();
    const field = screen.getByLabelText("izoh") as HTMLInputElement;
    const money = screen.getByLabelText("summa") as HTMLInputElement;

    fireEvent.focusIn(field);
    // The one mechanism: the browser is told something else is typing here.
    expect(field.getAttribute("inputmode")).toBe("none");
    // ⚠️ The money field had a mode of its own before we borrowed it, and it is
    // what decides that *our* pad comes up numeric. Losing it would make every
    // price on the till a letter pad.
    fireEvent.focusIn(money);
    expect(field.hasAttribute("inputmode")).toBe(false);
    expect(money.getAttribute("inputmode")).toBe("none");
  });

  it("shows digits for a money field and letters for a text one", () => {
    draw();
    fireEvent.focusIn(screen.getByLabelText("summa"));
    // The 000 key exists only on the numeric pad, and no letter does.
    expect(screen.getByRole("button", { name: "000" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "q" })).toBeNull();

    fireEvent.focusIn(screen.getByLabelText("izoh"));
    expect(screen.getByRole("button", { name: "q" })).toBeTruthy();
  });

  it("puts itself away when anything else on the screen is pressed", () => {
    draw();
    fireEvent.focusIn(screen.getByLabelText("izoh"));
    expect(screen.getByRole("group", { name: t.till.keyboard })).toBeTruthy();

    // ⚠️ A plain button, not another field: pressing one of these does not
    // move focus in every browser, so a pad tied to `blur` alone would sit over
    // the total for the rest of the sale.
    fireEvent.pointerDown(screen.getByRole("button", { name: "Boshqa tugma" }));

    expect(screen.queryByRole("group", { name: t.till.keyboard })).toBeNull();
    // …and the field is handed back exactly as it was found.
    expect(
      (screen.getByLabelText("izoh") as HTMLInputElement).hasAttribute(
        "inputmode",
      ),
    ).toBe(false);
  });

  it("stays up while moving from one field to the next", () => {
    draw();
    fireEvent.focusIn(screen.getByLabelText("izoh"));
    fireEvent.focusIn(screen.getByLabelText("summa"));
    expect(screen.getByRole("group", { name: t.till.keyboard })).toBeTruthy();
    expect(screen.getByRole("button", { name: "000" })).toBeTruthy();
  });

  it("deletes one character at a time", () => {
    draw();
    const field = screen.getByLabelText("izoh") as HTMLInputElement;
    fireEvent.focusIn(field);
    tap("o");
    tap("s");
    tap("h");
    tap("Backspace");
    expect(field.value).toBe("os");
  });

  it("closes on Escape, because a pad with no visible way out is a trap", () => {
    draw();
    fireEvent.focusIn(screen.getByLabelText("izoh"));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("group", { name: t.till.keyboard })).toBeNull();
  });
});

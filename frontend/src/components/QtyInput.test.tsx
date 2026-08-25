import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { QtyInput } from "./QtyInput";
import { qtyNumber } from "@/lib/qty";

// ⚠️ **The rules are in `lib/qty.ts`; what is tested here is the field holding
// its own draft.** A parent that stores a number and re-prints it on every
// keystroke turns "9." into "9" the moment the point is pressed — so the
// fraction can never be typed, on exactly the screens that exist for
// fractions. It is invisible in a unit test of the rules and obvious with a
// thumb, which is the combination that ships.

/** A parent of the harder kind: it keeps a number, not the text. */
function NumberParent({ start = 0 }: { start?: number }) {
  const [qty, setQty] = useState(start);
  return (
    <div>
      <QtyInput
        aria-label="qty"
        value={qty}
        onValue={(v) => setQty(qtyNumber(v))}
      />
      <output>{qty}</output>
    </div>
  );
}

describe("typing a fraction into a field whose parent stores a number", () => {
  it("keeps the point on screen while the rest is still being typed", async () => {
    const user = userEvent.setup();
    render(<NumberParent />);
    const field = screen.getByLabelText("qty") as HTMLInputElement;

    await user.type(field, "9.");
    // The parent already reads 9 — and the field must not be re-printed as it.
    expect(field.value).toBe("9.");
    expect(screen.getByRole("status").textContent).toBe("9");

    await user.type(field, "4");
    expect(field.value).toBe("9.4");
    expect(screen.getByRole("status").textContent).toBe("9.4");
  });

  it("accepts a comma as that point", async () => {
    const user = userEvent.setup();
    render(<NumberParent />);
    const field = screen.getByLabelText("qty") as HTMLInputElement;

    await user.type(field, "9,4");
    // What a number field could never do: the browser handed it "" and the
    // digits disappeared.
    expect(field.value).toBe("9.4");
    expect(screen.getByRole("status").textContent).toBe("9.4");
  });

  it("refuses what a quantity cannot contain, as it is typed", async () => {
    const user = userEvent.setup();
    render(<NumberParent />);
    const field = screen.getByLabelText("qty") as HTMLInputElement;

    await user.type(field, "-9kg");
    expect(field.value).toBe("9");
  });

  it("takes a new value from the parent when the parent really changes it", () => {
    // A form reset, or a row replaced by another ingredient. The draft belongs
    // to the field only while the two still agree as numbers.
    const { rerender } = render(
      <QtyInput aria-label="qty" value={5} onValue={() => {}} />,
    );
    expect((screen.getByLabelText("qty") as HTMLInputElement).value).toBe("5");

    rerender(<QtyInput aria-label="qty" value={12} onValue={() => {}} />);
    expect((screen.getByLabelText("qty") as HTMLInputElement).value).toBe("12");
  });

  it("opens empty rather than on a zero somebody has to delete first", () => {
    render(<NumberParent />);
    expect((screen.getByLabelText("qty") as HTMLInputElement).value).toBe("");
  });
});

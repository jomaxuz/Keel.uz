import { describe, expect, it } from "vitest";

import {
  fromOptionDrafts,
  optionProblems,
  type OptionGroupDraft,
} from "./OptionsEditor";

function group(p: Partial<OptionGroupDraft> = {}): OptionGroupDraft {
  return {
    name: "Hajm",
    nameRu: "",
    nameEn: "",
    required: true,
    multiple: false,
    choices: [
      { name: "Kichik", nameRu: "", nameEn: "", priceDelta: "0", recipe: [] },
      { name: "Katta", nameRu: "", nameEn: "", priceDelta: "5000", recipe: [] },
    ],
    ...p,
  };
}

const blankChoice = {
  name: "",
  nameRu: "",
  nameEn: "",
  priceDelta: "0",
  recipe: [],
};

describe("option groups", () => {
  // ⚠️ **The bug this was reported as.** An owner filled in the two sizes,
  // left the question name blank, pressed Saqlash — and the dish saved with the
  // variants gone. Nothing errored: the filter is right, an unnamed question
  // with no name is not a question. A correct filter applied without telling
  // anybody is indistinguishable from a save that did not work.
  it("names a question that has answers but no name", () => {
    const problems = optionProblems([group({ name: "  " })]);
    expect(problems).toHaveLength(1);
    expect(problems[0].needName).toBe(true);
    // And it really would have been dropped.
    expect(fromOptionDrafts([group({ name: "  " })])).toHaveLength(0);
  });

  it("names a question that has a name but no answers", () => {
    const problems = optionProblems([group({ choices: [blankChoice] })]);
    expect(problems).toHaveLength(1);
    expect(problems[0].needChoice).toBe(true);
  });

  // ⚠️ The row the "add question" button just created is not a problem.
  // Complaining about it would mean refusing to save a dish because somebody
  // opened the form and changed their mind — and a warning that appears before
  // anything is typed is a warning people learn to ignore.
  it("says nothing about a question nobody has started", () => {
    expect(
      optionProblems([
        group({ name: "", choices: [blankChoice, { ...blankChoice }] }),
      ]),
    ).toHaveLength(0);
  });

  it("says nothing about a finished question", () => {
    expect(optionProblems([group()])).toHaveLength(0);
    const out = fromOptionDrafts([group()]);
    expect(out).toHaveLength(1);
    expect(out[0].choices.map((c) => c.name)).toEqual(["Kichik", "Katta"]);
    // The surcharge is a number by the time it leaves the form.
    expect(out[0].choices[1].priceDelta).toBe(5000);
  });
});

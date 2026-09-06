import { describe, expect, it } from "vitest";

import { parse } from "./Article";

/**
 * ⚠️ **A YouTube link arrives in whatever shape the app copied it in**, and the
 * one thing that must not happen is the URL being handed to an iframe whole: a
 * copied link carries a playlist, a start time and a tracking parameter, and
 * embedding all of it hands a page of ours to whatever those mean this month.
 */
describe("a YouTube link becomes a player", () => {
  const cases = [
    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "https://youtu.be/dQw4w9WgXcQ",
    "https://m.youtube.com/watch?v=dQw4w9WgXcQ&t=42s",
    "youtube.com/shorts/dQw4w9WgXcQ",
    "https://www.youtube.com/embed/dQw4w9WgXcQ",
  ];
  for (const url of cases) {
    it(url, () => {
      const blocks = parse(url);
      expect(blocks).toHaveLength(1);
      expect(blocks[0]).toEqual({ kind: "video", id: "dQw4w9WgXcQ" });
    });
  }

  it("leaves a link inside a sentence alone", () => {
    // ⚠️ A player in the middle of a paragraph is not what somebody typing a
    // sentence about a video meant.
    const blocks = parse("Buni ko'ring: https://youtu.be/dQw4w9WgXcQ — zo'r.");
    expect(blocks[0].kind).toBe("p");
  });
});

describe("the rest of the small language", () => {
  it("keeps paragraphs, headings, lists and quotes apart", () => {
    const blocks = parse(
      [
        "## Sarlavha",
        "",
        "Birinchi qator",
        "ikkinchi qator",
        "",
        "- bir",
        "- ikki",
        "",
        "> iqtibos",
      ].join("\n"),
    );
    expect(blocks.map((b) => b.kind)).toEqual(["h2", "p", "ul", "quote"]);
    // ⚠️ Two lines with no blank between them are one paragraph, which is how
    // anybody types into a textarea that wraps.
    expect(blocks[1]).toEqual({ kind: "p", text: "Birinchi qator ikkinchi qator" });
  });

  it("reads a picture on its own line", () => {
    const blocks = parse("![Kassa](/internal/blog/image/abc)");
    expect(blocks[0]).toEqual({
      kind: "img",
      alt: "Kassa",
      src: "/internal/blog/image/abc",
    });
  });

  it("lifts a picture out of the sentence it was written in", () => {
    // ⚠️ **The bug that shipped.** The editor inserts at the cursor, so a
    // writer who uploads mid-paragraph leaves words on the same line — and an
    // anchored `^…$` match meant the reader was shown `![](/blog-image/…)` as
    // text, with no picture anywhere on the page.
    const blocks = parse("oldin ![](/blog-image/abc)keyin\nyana");
    expect(blocks.map((b) => b.kind)).toEqual(["p", "img", "p"]);
    expect(blocks[0]).toEqual({ kind: "p", text: "oldin" });
    expect(blocks[1]).toEqual({ kind: "img", alt: "", src: "/blog-image/abc" });
    expect(blocks[2]).toEqual({ kind: "p", text: "keyin yana" });
  });

  it("reads two pictures on one line", () => {
    const blocks = parse("![a](/x) ![b](/y)");
    expect(blocks).toEqual([
      { kind: "img", alt: "a", src: "/x" },
      { kind: "img", alt: "b", src: "/y" },
    ]);
  });

  it("survives Windows line endings", () => {
    // ⚠️ The text arrives from a textarea on somebody's laptop; a "\r" left in
    // place turns every heading into a paragraph that merely looks like one.
    const blocks = parse("## Sarlavha\r\n\r\nMatn\r\n");
    expect(blocks.map((b) => b.kind)).toEqual(["h2", "p"]);
  });
});

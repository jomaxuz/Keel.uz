import { describe, expect, it } from "vitest";

import { buildLlmsFull, buildLlmsIndex, URL_MARKER, type LlmsPost } from "./llms";
import { ALL_SLUGS } from "./help";

const posts: LlmsPost[] = [
  {
    slug: "tarozi",
    title: "Tarozi bilan sotish",
    excerpt: "Do'konda tarozi qanday ulanadi.",
    body: "Birinchi xatboshi.\n\n## Muallif sarlavhasi\n\n![rasm](/blog-image/abc)",
    publishedAt: "2026-09-01T05:00:00Z",
  },
];

describe("llms.txt", () => {
  it("has the llmstxt.org shape: title, summary, sections of links", () => {
    const md = buildLlmsIndex("uz", posts);
    const lines = md.split("\n");
    expect(lines[0]).toBe("# Keel");
    expect(lines[2].startsWith("> ")).toBe(true);
    expect(md).toContain("## Optional");
    expect(md).toContain("(https://keel.uz/llms-full.txt)");
  });

  it("lists every help article and every post", () => {
    const md = buildLlmsIndex("uz", posts);
    for (const slug of ALL_SLUGS) expect(md).toContain(`(https://keel.uz/help/${slug})`);
    expect(md).toContain("(https://keel.uz/blog/tarozi)");
  });

  // ⚠️ The prefixed file has to point at prefixed pages: a Russian index that
  // links to Uzbek articles sends a Russian reader to a language nobody offered.
  it("links to its own language and names the others", () => {
    const md = buildLlmsIndex("ru", []);
    expect(md).toContain("(https://keel.uz/ru/help/");
    expect(md).not.toContain("(https://keel.uz/help/");
    expect(md).toContain("(https://keel.uz/llms.txt)");
    expect(md).toContain("(https://keel.uz/en/llms.txt)");
  });
});

describe("llms-full.txt", () => {
  const md = buildLlmsFull("uz", posts);
  const lines = md.split("\n");

  // ⚠️ The contract with the control plane's watcher (handlers/llms.go): every
  // page is `## heading`, a blank line, then `URL: <address>`. Break it and
  // every change reads as "the whole site changed".
  it("puts a URL line two lines under every page heading", () => {
    const urls = lines.filter((l) => l.startsWith(URL_MARKER));
    expect(urls.length).toBe(4 + ALL_SLUGS.length + posts.length);
    lines.forEach((l, i) => {
      if (!l.startsWith(URL_MARKER)) return;
      expect(lines[i - 1]).toBe("");
      expect(lines[i - 2].startsWith("## ")).toBe(true);
    });
    expect(new Set(urls).size).toBe(urls.length);
  });

  it("carries the words, not just the titles", () => {
    expect(md).toContain("Birinchi xatboshi.");
    // A help article's steps survive as a numbered list.
    expect(md).toMatch(/\n1\. \*\*/);
    // Site-relative pictures get the host.
    expect(md).toContain("](https://keel.uz/blog-image/abc)");
  });

  it("writes the Russian file with Russian addresses", () => {
    const ru = buildLlmsFull("ru", []);
    expect(ru).toContain(`${URL_MARKER}https://keel.uz/ru/help/`);
    expect(ru).not.toContain(`${URL_MARKER}https://keel.uz/help/`);
  });
});

import { describe, expect, it } from "vitest";

import { apiBaseFor, uploadsBaseFor } from "./serverAddress";

// ⚠️ **The same examples the Go side is held to**
// (`backend/desktop/config_windows.go` → `apiBase`). Two implementations of one
// rule is the drift this codebase warns about most; where a boundary makes it
// unavoidable, the test is what keeps them honest — the same bargain
// `serviceOn` strikes.

describe("what a restaurant actually types", () => {
  it("turns a bare name into a Keel subdomain", () => {
    // The whole reason this exists: somebody setting up a phone knows their
    // restaurant as "osh".
    expect(apiBaseFor("osh")).toBe("https://osh.keel.uz/api/v1");
    expect(apiBaseFor("  OSH  ")).toBe("https://osh.keel.uz/api/v1");
  });

  it("takes anything with a dot as its own host", () => {
    expect(apiBaseFor("kassa.restoran.uz")).toBe(
      "https://kassa.restoran.uz/api/v1",
    );
  });

  it("accepts what somebody pasted out of the panel", () => {
    for (const typed of [
      "https://osh.keel.uz",
      "http://osh.keel.uz/",
      "https://osh.keel.uz/api/v1",
      "osh.keel.uz/api/v1/",
    ]) {
      expect(apiBaseFor(typed)).toBe("https://osh.keel.uz/api/v1");
    }
  });

  it("refuses what cannot be a host rather than building a dead address", () => {
    // ⚠️ A dead address fails as "no internet", which sends somebody to their
    // router instead of to this field.
    expect(apiBaseFor("")).toBe("");
    expect(apiBaseFor("   ")).toBe("");
    expect(apiBaseFor("mening restoranim")).toBe("");
    expect(apiBaseFor("osh.keel.uz/admin/menu")).toBe("");
  });

  it("derives the picture host from the same answer", () => {
    // Two fields would be two chances to mistype one thing.
    expect(uploadsBaseFor("osh")).toBe("https://osh.keel.uz/uploads");
    expect(uploadsBaseFor("")).toBe("");
  });
});

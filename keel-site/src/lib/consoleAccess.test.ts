import { readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import type { Me } from "./api";
import { CONSOLE_SECTIONS, canOpen, permissionFor } from "./consoleAccess";
import { consoleHome } from "./consoleHome";

const none: Me["can"] = {
  allTenants: false, tenants: false, stats: false, overview: false, staff: false,
  log: false, provision: false, billing: false, support: false, partners: false,
  seo: false, blog: false, release: false,
};

// What `/me` answers for each role on the server (models.Permissions).
const agent: Me["can"] = { ...none, tenants: true };
const admin: Me["can"] = { ...none, allTenants: true, tenants: true, stats: true, provision: true, billing: true, blog: true };
const support: Me["can"] = { ...none, support: true };
const owner: Me["can"] = Object.fromEntries(Object.keys(none).map((k) => [k, true])) as Me["can"];

describe("console page guard", () => {
  // ⚠️ The report that started this: an agent typing the address of a section
  // their tabs do not show.
  it("keeps an agent out of every section that is not selling", () => {
    for (const path of [
      "/console", "/console/seo", "/console/blog", "/console/referrers",
      "/console/support", "/console/reports", "/console/staff", "/console/staff/abc",
    ]) {
      expect(canOpen(path, agent), path).toBe(false);
    }
    for (const path of [
      "/console/tenants", "/console/tenants/abc", "/console/tenants/abc/design",
      "/console/visits", "/console/outreach", "/console/leaflet?ref=x",
    ]) {
      expect(canOpen(path, agent), path).toBe(true);
    }
  });

  it("gives support only the queue and the errors", () => {
    expect(canOpen("/console/support", support)).toBe(true);
    expect(canOpen("/console/reports", support)).toBe(true);
    for (const path of ["/console", "/console/tenants", "/console/visits", "/console/seo", "/console/leaflet"]) {
      expect(canOpen(path, support), path).toBe(false);
    }
  });

  it("gives admin the blog and the customers, not search, partners or staff", () => {
    expect(canOpen("/console/blog", admin)).toBe(true);
    expect(canOpen("/console/tenants", admin)).toBe(true);
    for (const path of ["/console", "/console/seo", "/console/referrers", "/console/staff", "/console/support"]) {
      expect(canOpen(path, admin), path).toBe(false);
    }
  });

  it("opens everything for the owner", () => {
    for (const s of CONSOLE_SECTIONS) {
      expect(canOpen(`/console/${s}`, owner), s).toBe(true);
    }
    expect(canOpen("/console", owner)).toBe(true);
  });

  it("closes an unknown page and does not match on a prefix of a word", () => {
    expect(permissionFor("/console/seo-tools")).toBeNull();
    expect(permissionFor("/console/nothing")).toBeNull();
    expect(canOpen("/console/nothing", owner)).toBe(false);
    expect(permissionFor("/console/")).toBe("overview");
  });

  // ⚠️ A redirect target the account cannot open would bounce forever.
  it("sends every role home to a page that role can open", () => {
    for (const can of [agent, admin, support, owner]) {
      expect(canOpen(consoleHome({ can }), can)).toBe(true);
    }
  });

  // ⚠️ **The rule that makes "unknown means closed" safe to have.** Every page
  // directory under the console must be listed, or it ships unreachable — and
  // a test failing today is better than an owner asking tomorrow why a new
  // page answers "no access".
  it("lists every console page directory", () => {
    const dir = fileURLToPath(new URL("../app/console", import.meta.url));
    const pages = readdirSync(dir, { withFileTypes: true })
      .filter((e) => e.isDirectory() && e.name !== "login")
      .map((e) => e.name);
    for (const page of pages) {
      expect(CONSOLE_SECTIONS, `app/console/${page} has no permission in consoleAccess.ts`).toContain(page);
    }
  });
});

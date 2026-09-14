import { describe, expect, it } from "vitest";
import type { Me } from "./api";
import { consoleHome } from "./consoleHome";

const none: Me["can"] = {
  allTenants: false, tenants: false, stats: false, overview: false, staff: false,
  log: false, provision: false, billing: false, support: false, partners: false,
  seo: false, blog: false,
};

describe("consoleHome", () => {
  it("sends the owner to the overview", () => {
    expect(consoleHome({ can: { ...none, overview: true, tenants: true, support: true } })).toBe("/console");
  });
  it("sends admin, manager and agent to the customers", () => {
    expect(consoleHome({ can: { ...none, tenants: true, stats: true } })).toBe("/console/tenants");
  });
  it("sends support to the queue", () => {
    expect(consoleHome({ can: { ...none, support: true } })).toBe("/console/support");
  });
  it("prefers customers when an agent also answers support", () => {
    expect(consoleHome({ can: { ...none, tenants: true, support: true } })).toBe("/console/tenants");
  });
});

// A till server small enough to hold in your head, so the screens can be tested
// the way they are used.
//
// ⚠️ **The point is the flow, not the markup.** Every defect the live trial
// found — the floor screen never rendered, the option dialog missing, the back
// button absent — was a step a cashier could not reach with a finger while the
// Go tests stayed green. So this fake answers the same calls the real server
// does, in the same shapes, and the tests drive the real components through it.
//
// What it deliberately does **not** do: re-implement pricing rules. Option
// deltas are added here only so the check panel has a number to draw; the real
// server prices everything again on the way in (see resolveOptions), and a fake
// that argued with it about money would be testing itself.

import type {
  Check,
  FloorShape,
  FloorTable,
  MenuGroup,
  MenuItem,
  OrderItemOption,
  TableZone,
} from "@/lib/types";

/** The refusal the screens branch on.
 *
 *  ⚠️ Imported at call time, not at the top. This module is what the mocked
 *  `@/lib/api` is built from, so a static import would be a cycle — and the
 *  screens genuinely test `err instanceof ApiError`, so a look-alike error
 *  class here would make every refusal read as "qayta urinish". */
async function refuse(status: number, message: string): Promise<never> {
  const { ApiError } = await import("@/lib/api");
  throw new ApiError(status, message);
}

export const PIN = "1234";

/** The dish that adds in one tap. */
export const PLAIN_DISH = "Lag'mon";
/** The dish that has to ask "which size" first — the one that was unsellable. */
export const OPTION_DISH = "Osh";

function category(id: string, name: string, sort: number) {
  return { id, name, slug: id, sortOrder: sort, isActive: true, imageUrl: "" };
}

function dish(over: Partial<MenuItem> & { id: string; name: string }): MenuItem {
  return {
    categoryId: "c1",
    description: "",
    price: 30000,
    oldPrice: null,
    imageUrl: "",
    images: null,
    isAvailable: true,
    isPopular: false,
    sortOrder: 0,
    options: null,
    tags: null,
    updatedAt: "2026-08-18T09:00:00Z",
    ...over,
  } as MenuItem;
}

export const MENU: MenuGroup[] = [
  {
    category: category("c1", "Taomlar", 0),
    items: [
      // ⚠️ Not 30 000: that is what the option dish costs with the large
      // portion chosen, and two tiles reading the same number make the
      // dialog's total impossible to assert without also matching the menu
      // behind it.
      dish({ id: "m1", name: PLAIN_DISH, price: 32000 }),
      dish({
        id: "m2",
        name: OPTION_DISH,
        price: 25000,
        options: [
          {
            name: "Hajm",
            required: true,
            multiple: false,
            choices: [
              { name: "Kichik", priceDelta: 0 },
              { name: "Katta", priceDelta: 5000 },
            ],
          },
          {
            name: "Qo'shimcha",
            required: false,
            multiple: true,
            choices: [{ name: "Pishloq", priceDelta: 3000 }],
          },
        ],
      }),
      // Off the menu today: the grid must not offer it, and the tests read the
      // dish count.
      dish({ id: "m3", name: "Norin", isAvailable: false }),
    ],
  },
  {
    category: category("c2", "Ichimliklar", 1),
    items: [dish({ id: "m4", name: "Choy", categoryId: "c2", price: 3000 })],
  },
];

export const ZONES: TableZone[] = [
  { id: "z1", name: "Zal", bookable: true, layout: "map", sort: 0 },
  { id: "z2", name: "Saboy", bookable: false, layout: "list", sort: 1 },
];

/** A drawn table.
 *
 *  ⚠️ **With real coordinates**, because zero is what the screens read as "this
 *  room was never drawn" and fall back to the list for. A fixture full of
 *  zeroes would have tested only the fallback, which is the path most
 *  restaurants are not on. */
function table(
  id: string,
  number: string,
  zoneId: string,
  x: number,
  y: number,
): FloorTable {
  return {
    id,
    number,
    seats: 4,
    shape: "rect",
    x,
    y,
    w: 140,
    h: 110,
    isActive: true,
    note: "",
    zoneId,
  };
}

/** The plan's own coordinate space, as the panel's editor writes it. */
export const PLAN = { width: 1000, height: 700 };

/** A wall and a named area, the two things an owner draws around the tables. */
export const SHAPES: FloorShape[] = [
  { kind: "area", label: "Bar", x: 40, y: 40, w: 180, h: 90 },
];

export const TABLES: FloorTable[] = [
  table("t1", "7", "z1", 320, 200),
  table("t2", "8", "z1", 540, 200),
  table("t3", "101", "z2", 320, 420),
];

export interface TillServerOptions {
  /** Does this branch lock its screens? */
  pinsUsed?: boolean;
  /** Is the drawer already open when the screen loads? */
  shiftOpen?: boolean;
  canCashier?: boolean;
  canWaiter?: boolean;
  /** Whether this person may open the drawer themselves. */
  canShift?: boolean;
}

export function createTillServer(opts: TillServerOptions = {}) {
  const {
    pinsUsed = true,
    shiftOpen = true,
    canCashier = true,
    canWaiter = true,
    canShift = true,
  } = opts;

  let shift = shiftOpen ? openShift(0) : null;
  const checks = new Map<string, Check>();
  let seq = 0;

  /** What the screens asked for, so a test can prove the request carried what
   *  the screen claims it did — `mine` on the floor, the option names on the
   *  wire. */
  const calls = {
    unlock: [] as string[],
    checksMine: [] as boolean[],
    addLines: [] as {
      checkId: string;
      menuItemId: string;
      qty: number;
      options?: OrderItemOption[];
    }[],
    openCheck: [] as { tableId?: string; guests?: number }[],
    openShift: [] as number[],
  };

  function openShift(float: number) {
    return {
      id: "shift-1",
      openedAt: "2026-08-18T09:00:00Z",
      openedBy: "Nodira",
      openingFloat: float,
      expected: float,
      counterCash: 0,
      counted: 0,
      variance: 0,
    };
  }

  function itemById(id: string): MenuItem | undefined {
    for (const g of MENU) for (const it of g.items) if (it.id === id) return it;
    return undefined;
  }

  function retotal(check: Check) {
    check.subtotal = check.lines
      .filter((l) => !l.void)
      .reduce((s, l) => s + l.sum, 0);
    check.total = check.subtotal;
    check.unfired = check.lines.filter((l) => !l.fired && !l.void).length;
  }

  const api = {
    // ---- The lock screen ----
    tillSession: async () => ({ pinsUsed }),
    tillUnlock: async (pin: string) => {
      calls.unlock.push(pin);
      if (pin !== PIN) return refuse(401, "Kod noto'g'ri");
      return {
        token: "person-token",
        staff: {
          id: "s1",
          name: "Nodira",
          canWaiter,
          canCashier,
        },
      };
    },

    // ---- The drawer ----
    tillCashShift: async () => ({ open: shift, canShift }),
    tillOpenCashShift: async (body: { openingFloat: number }) => {
      calls.openShift.push(body.openingFloat);
      shift = openShift(body.openingFloat);
      return shift;
    },

    // ---- The room and the menu ----
    getMenu: async () => MENU,
    getRestaurant: async () => ({
      restaurant: {
        currency: "UZS",
        booking: {
          tables: TABLES,
          zones: ZONES,
          shapes: SHAPES,
          width: PLAN.width,
          height: PLAN.height,
        },
      },
      isOpenNow: true,
    }),

    // ---- Checks ----
    tillChecks: async (mine = false) => {
      calls.checksMine.push(mine);
      return { checks: [...checks.values()] };
    },
    tillOpenCheck: async (body: { tableId?: string; guests?: number }) => {
      calls.openCheck.push(body);
      seq += 1;
      const tb = TABLES.find((t) => t.id === body.tableId);
      const check: Check = {
        id: `chk-${seq}`,
        number: `6XGC-ZNH${seq}`,
        status: "pending",
        tableId: body.tableId || undefined,
        tableNumber: tb?.number,
        guests: body.guests,
        openedAt: "2026-08-18T09:05:00Z",
        openMin: 0,
        lines: [],
        subtotal: 0,
        unfired: 0,
        total: 0,
      };
      checks.set(check.id, check);
      return { ...check };
    },
    tillAddLines: async (
      id: string,
      items: { menuItemId: string; qty: number; options?: OrderItemOption[] }[],
    ) => {
      const check = checks.get(id);
      if (!check) return refuse(404, "Chek topilmadi");
      for (const it of items) {
        calls.addLines.push({ checkId: id, ...it });
        const menu = itemById(it.menuItemId);
        if (!menu) return refuse(400, "Taom topilmadi");
        // The server's own rule, kept because a test that could add an
        // unanswered required group would pass while the till stayed broken.
        for (const g of menu.options ?? []) {
          if (g.required && !it.options?.some((o) => o.name === g.name)) {
            return refuse(400, `${menu.name}: "${g.name}" tanlanmagan`);
          }
        }
        const unit =
          menu.price +
          (it.options ?? []).reduce((s, o) => s + (o.priceDelta || 0), 0);
        // The server's own merge rule, kept because the screens are read
        // through it: the same dish tapped twice is one line of two, unless the
        // kitchen already has it or the choices differ.
        const same = check.lines.find(
          (l) =>
            !l.void &&
            !l.fired &&
            l.name === menu.name &&
            sameOptions(l.options, it.options),
        );
        if (same) {
          same.qty += it.qty;
          same.sum = same.price * same.qty;
        } else {
          check.lines.push({
            lineId: `ln-${check.lines.length + 1}`,
            name: menu.name,
            price: unit,
            qty: it.qty,
            sum: unit * it.qty,
            options: it.options,
            fired: false,
          });
        }
      }
      retotal(check);
      return { ...check };
    },
    tillLineQty: async (id: string, lineId: string, qty: number) => {
      const check = checks.get(id);
      if (!check) return refuse(404, "Chek topilmadi");
      const line = check.lines.find((l) => l.lineId === lineId);
      if (!line) return refuse(404, "Qator topilmadi");
      // The server refuses both of these, and the screens must not be the only
      // thing standing between a thumb and the kitchen.
      if (line.fired) return refuse(409, "allaqachon oshxonaga yuborilgan");
      if (qty < 1 || qty > 99) return refuse(400, "soni 1 dan 99 gacha");
      line.qty = qty;
      line.sum = line.price * qty;
      retotal(check);
      return { ...check };
    },
    tillFire: async (id: string) => {
      const check = checks.get(id)!;
      for (const l of check.lines) if (!l.void) l.fired = true;
      retotal(check);
      return { ...check };
    },
    tillVoidLine: async (id: string, lineId: string) => {
      const check = checks.get(id)!;
      const line = check.lines.find((l) => l.lineId === lineId);
      if (line) line.void = { at: "2026-08-18T09:10:00Z", reason: "xato" };
      retotal(check);
      return { ...check };
    },
    tillUpdateCheck: async (id: string) => ({ ...checks.get(id)! }),
    tillCommentLine: async (id: string) => ({ ...checks.get(id)! }),
    tillClose: async (id: string) => {
      const check = checks.get(id)!;
      checks.delete(id);
      return { ...check, status: "delivered" as const };
    },
    tillCancel: async (id: string) => {
      const check = checks.get(id)!;
      checks.delete(id);
      return { ...check, status: "cancelled" as const };
    },

    // ---- Fiscal: off, which is the state of every restaurant without a
    // register and the one where these screens must still sell food.
    tillFiscalStatus: async () => ({ enabled: false }),
    tillUnfiledChecks: async () => ({ checks: [] as Check[] }),
  };

  return { api, calls, checks };
}

/** Same dish, same choices — order-insensitive, like the server's. */
function sameOptions(a?: OrderItemOption[], b?: OrderItemOption[]): boolean {
  const x = a ?? [];
  const y = b ?? [];
  if (x.length !== y.length) return false;
  const left = [...y];
  return x.every((want) => {
    const i = left.findIndex(
      (got) => got.name === want.name && got.choice === want.choice,
    );
    if (i < 0) return false;
    left.splice(i, 1);
    return true;
  });
}

export type TillServer = ReturnType<typeof createTillServer>;

// The live instance the mocked module delegates to. A test swaps it in
// `beforeEach`; the mock factory is hoisted and cannot capture it.
//
// ⚠️ **Kept on globalThis, and that is not paranoia.** The setup file and the
// test file reach this module by different specifiers ("./tillServer" and
// "@/test/tillServer") and vitest gives them **two module instances** — so a
// plain module-level variable meant the server the test configured was not the
// one the screen talked to. Everything still ran: the screen got the defaults
// and the assertions failed on the far side of a full flow, which reads as a
// broken screen rather than a broken harness.
const SLOT = Symbol.for("keel.test.tillServer");
type Slot = { [SLOT]?: TillServer };

export function installTillServer(opts?: TillServerOptions): TillServer {
  const server = createTillServer(opts);
  (globalThis as Slot)[SLOT] = server;
  return server;
}

function live(): TillServer {
  return ((globalThis as Slot)[SLOT] ??= createTillServer());
}

/** The `api` object the screens see: every till call forwarded to whichever
 *  server the current test installed.
 *
 *  ⚠️ **Resolved when the call is made, not when the property is read.** The
 *  mock spreads this object once, at mock time; binding the server at that
 *  moment meant every test ran against the one built first and configured its
 *  own into the void. */
export const tillApi = new Proxy({} as Record<string, unknown>, {
  get(_t, prop: string) {
    if (!(prop in live().api)) return undefined;
    return (...args: unknown[]) => {
      const server = live();
      const fn = (server.api as Record<string, unknown>)[prop] as (
        ...a: unknown[]
      ) => unknown;
      return fn.apply(server.api, args);
    };
  },
  has(_t, prop: string) {
    return prop in live().api;
  },
  ownKeys() {
    return Reflect.ownKeys(live().api);
  },
  getOwnPropertyDescriptor() {
    return { enumerable: true, configurable: true };
  },
});

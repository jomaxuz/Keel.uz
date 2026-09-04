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
  Printer,
  StopListItem,
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
/** A second plain dish, so a check can be divided without emptying it. */
export const TEA_DISH = "Choy";

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
  /** Whether this person runs the restaurant rather than the counter — the
   *  server sends `void` under this name, and it gates both retiring the
   *  screen and the settings section. */
  canExit?: boolean;
  /** What this room adds to a table's bill. */
  servicePercent?: number;
  /** Dishes the branch has run out of today, by name. */
  soldOut?: string[];
  /** The stored device token is dead — expired, or revoked when a monoblock
   *  left the building. The real server answers 401. */
  deviceRejected?: boolean;
  /** The rails this restaurant has signed up for, beyond cash. */
  paymentMethods?: string[];
  /** What the branch has already connected. */
  printers?: Printer[];
  /** The dishes the stop-list screen shows. */
  stopList?: StopListItem[];
}

/** What a printer can be asked to print — the server's list, mirrored here so
 *  the screen is driven by the same four words it is in production. */
const PRINT_KINDS = ["kitchen", "till", "customer", "precheck"];

/** The browser's copy of the server's rounding — see lib/offline/checks.ts. */
function serviceOn(payable: number, percent: number): number {
  if (payable <= 0 || percent <= 0) return 0;
  return Math.floor((payable * percent + 50) / 100);
}

export function createTillServer(opts: TillServerOptions = {}) {
  const {
    pinsUsed = true,
    shiftOpen = true,
    canCashier = true,
    canWaiter = true,
    canShift = true,
    // ⚠️ Default false, matching the majority of a real staff list: a till test
    // that silently ran as a manager would assert nothing about the gate.
    canExit = false,
    servicePercent = 0,
    deviceRejected = false,
    soldOut = [] as string[],
    paymentMethods = ["cash", "card", "transfer", "debt"],
  } = opts;

  // Whether the provider has confirmed the outstanding payment. A test flips
  // it the way a callback does — from outside, between two polls.
  let onlinePaid = false;

  let printers: Printer[] = opts.printers ?? [];
  let stopRows: StopListItem[] = opts.stopList ?? [];

  let shift = shiftOpen ? openShift(0) : null;
  const checks = new Map<string, Check>();
  let seq = 0;

  /** What the screens asked for, so a test can prove the request carried what
   *  the screen claims it did — `mine` on the floor, the option names on the
   *  wire. */
  const calls = {
    unlock: [] as string[],
    savePrinters: [] as Printer[][],
    setLimit: [] as { menuItemId: string; limit: number }[],
    /** ⚠️ The hold is on the wire, not in component state: the screen's whole
     *  contract here is that it sends a **duration** and lets the server turn
     *  it into a moment. A test reading the rendered countdown would pass on a
     *  screen that computed the deadline in the browser. */
    setSoldOut: [] as {
      menuItemId: string;
      soldOut: boolean;
      hold?: { minutes?: number; untilClose?: boolean };
    }[],
    checksMine: [] as boolean[],
    addLines: [] as {
      checkId: string;
      menuItemId: string;
      qty: number;
      options?: OrderItemOption[];
    }[],
    openCheck: [] as { tableId?: string; guests?: number }[],
    openShift: [] as number[],
    /** Sales handed over after an outage. */
    sync: [] as { clientId: string; lines: unknown[]; servicePercent?: number }[],
    split: [] as { checkId: string; lineIds: string[] }[],
    /** Every close, with how it was paid — including who owes it. */
    payOnline: [] as { checkId: string; provider: string }[],
    payDebt: [] as { orderId: string; method: string }[],
    close: [] as {
      checkId: string;
      paymentMethod?: string;
      userId?: string;
      debtNote?: string;
    }[],
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
    // ⚠️ The real server adds the room's service to a **table's** check and
    // never to a counter sale, and this fake exists to answer the way it does:
    // a screen tested against a fake that skipped it would look correct here
    // and read a different number to the guest in the restaurant.
    check.servicePercent = check.tableId ? servicePercent : 0;
    check.service = serviceOn(check.subtotal, check.servicePercent);
    check.total = check.subtotal + check.service;
    check.unfired = check.lines.filter((l) => !l.fired && !l.void).length;
  }

  const api = {
    // ---- The lock screen ----
    tillSession: async () => {
      // ⚠️ Mirrors what the client does with a dead device token: the real
      // `api.tillSession` clears it and retries with the staff login, so the
      // fake has to refuse the first call the same way or the test would pass
      // against a screen that never met the failure.
      if (deviceRejected && window.localStorage.getItem("keel_till_device")) {
        window.localStorage.removeItem("keel_till_device");
      }
      return { pinsUsed };
    },
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
          canExit,
        },
      };
    },

    // ---- The stop list ----
    tillStopList: async () => ({ items: stopRows, branch: "Chilonzor" }),
    tillSetSoldOut: async (
      menuItemId: string,
      soldOut: boolean,
      hold?: { minutes?: number; untilClose?: boolean },
    ) => {
      calls.setSoldOut.push({ menuItemId, soldOut, hold });
      // The real server turns the duration into an instant on its own clock,
      // so the fake does too — a fake that echoed the minutes back would let a
      // screen that renders "120" pass.
      const until =
        soldOut && hold?.minutes
          ? new Date(Date.now() + hold.minutes * 60_000).toISOString()
          : undefined;
      stopRows = stopRows.map((r) =>
        r.menuItemId === menuItemId ? { ...r, manual: soldOut, until } : r,
      );
      return { ok: true, menuItemId, soldOut, until };
    },
    tillSetDailyLimit: async (menuItemId: string, limit: number) => {
      calls.setLimit.push({ menuItemId, limit });
      const row = stopRows.find((r) => r.menuItemId === menuItemId);
      const sold = row?.sold ?? 0;
      // ⚠️ The server recomputes the stop from the orders, so raising the
      // number can put a dish back on sale. The fake does the same arithmetic
      // or the test would pass on a screen that only ever stops things.
      const limitOff = limit > 0 && sold >= limit;
      stopRows = stopRows.map((r) =>
        r.menuItemId === menuItemId ? { ...r, limit, limitOff } : r,
      );
      return { menuItemId, limit, sold, limitOff };
    },

    // ---- The branch's printers ----
    //
    // ⚠️ Stored on the server in the real thing, so the fake keeps them in one
    // place too: a test that saved into component state would pass while the
    // screen forgot everything on reload.
    tillPrinters: async () => ({ printers, kinds: PRINT_KINDS }),
    // ⚠️ The **client's** signature, not the request body: the proxy forwards
    // the arguments `api.tillSavePrinters(...)` was called with. Mirroring the
    // JSON instead gives a fake that fails with "cannot read properties of
    // undefined", which reads as a bug in the screen.
    tillSavePrinters: async (next: Printer[]) => {
      // The server mints ids and drops an address it cannot parse. Both are
      // asserted against, so the fake has to do them.
      printers = next
        .filter((p) => p.target.trim() !== "")
        .map((p, i) => ({ ...p, id: p.id || `p${printers.length + i + 1}` }));
      calls.savePrinters.push(printers);
      return { printers };
    },
    tillTestPrinter: async (printerId: string) => {
      const p = printers.find((x) => x.id === printerId);
      if (!p) return refuse(404, "printer topilmadi");
      // ⚠️ Zero when it prints nothing — the case the screen has to tell apart
      // from success, because a printer that queues nothing looks broken.
      return { queued: p.kinds.length > 0 && !p.disabled ? 1 : 0 };
    },

    // ---- The drawer ----
    tillCashShift: async () => ({ open: shift, canShift }),
    tillOpenCashShift: async (body: { openingFloat: number }) => {
      calls.openShift.push(body.openingFloat);
      shift = openShift(body.openingFloat);
      return shift;
    },

    // ---- The room and the menu ----
    getMenu: async () =>
      // ⚠️ The real menu carries the branch's stop list on the dish, so the
      // fake has to as well — a grid tested against dishes that are never sold
      // out would look correct here and hand the cashier a refusal in the
      // restaurant.
      MENU.map((g) => ({
        ...g,
        items: g.items.map((i) =>
          soldOut.includes(i.name) ? { ...i, soldOut: true } : i,
        ),
      })),
    // The till reads its own branch, not the public profile.
    tillBranch: async () => ({
      id: "b1",
      name: "Maracanda",
      currency: "UZS",
      servicePercent,
      booking: {
        tables: TABLES,
        zones: ZONES,
        shapes: SHAPES,
        width: PLAN.width,
        height: PLAN.height,
      },
    }),
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
      items: {
        menuItemId: string;
        qty: number;
        options?: OrderItemOption[];
        guest?: number;
        course?: number;
      }[],
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
            (l.guest ?? 0) === (it.guest ?? 0) &&
            (l.course ?? 0) === (it.course ?? 0) &&
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
            ...(it.guest ? { guest: it.guest } : {}),
            ...(it.course ? { course: it.course } : {}),
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
    tillPrint: async (
      id: string,
      kind: "kitchen" | "till" | "customer" | "precheck",
    ) => {
      const check = checks.get(id)!;
      // The bill records that the table asked for it — the floor draws that.
      if (kind === "precheck" && !check.precheckAt) {
        check.precheckAt = "2026-08-18T21:40:00Z";
      }
      return {
        lines: ["MARACANDA", "#" + check.number, "HISOB — fiskal chek emas"],
        widthMM: 80,
        // No printer configured, which is the state a restaurant is in until
        // somebody sets one up — and the case the browser fallback exists for.
        queued: 0,
        check: { ...check },
      };
    },
    tillFire: async (id: string, course?: number) => {
      const check = checks.get(id)!;
      for (const l of check.lines) {
        if (l.void) continue;
        // The server's rule: a course is sent on its own, no course means all.
        if (course !== undefined && (l.course ?? 0) !== course) continue;
        l.fired = true;
      }
      retotal(check);
      return { ...check };
    },
    tillLineGuest: async (id: string, lineId: string, guest: number) => {
      const check = checks.get(id)!;
      const line = check.lines.find((l) => l.lineId === lineId);
      // ⚠️ Allowed after firing, unlike everything else: a table decides how to
      // split the bill when the plates are cleared.
      if (line) line.guest = guest;
      return { ...check };
    },
    tillMoveLines: async (id: string, lineIds: string[], toCheckId: string) => {
      const from = checks.get(id)!;
      const to = checks.get(toCheckId)!;
      const moved = from.lines.filter(
        (l) => lineIds.includes(l.lineId) && !l.void,
      );
      from.lines = from.lines.filter((l) => !moved.includes(l));
      to.lines.push(...moved);
      retotal(from);
      retotal(to);
      return { ...from };
    },
    tillSplit: async (id: string, lineIds: string[]) => {
      const from = checks.get(id)!;
      const moved = from.lines.filter(
        (l) => lineIds.includes(l.lineId) && !l.void,
      );
      from.lines = from.lines.filter((l) => !moved.includes(l));
      seq += 1;
      const split: Check = {
        ...from,
        id: `chk-split-${seq}`,
        number: `SPL-${seq}`,
        lines: moved.map((l) => ({ ...l, guest: 0 })),
      };
      retotal(from);
      retotal(split);
      checks.set(split.id, split);
      calls.split.push({ checkId: id, lineIds });
      return { check: { ...from }, split: { ...split } };
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
    tillClose: async (
      id: string,
      body?: { paymentMethod?: string; userId?: string; debtNote?: string },
    ) => {
      calls.close.push({ checkId: id, ...(body ?? {}) });
      const check = checks.get(id)!;
      checks.delete(id);
      return { ...check, status: "delivered" as const };
    },
    tillCancel: async (id: string) => {
      const check = checks.get(id)!;
      checks.delete(id);
      return { ...check, status: "cancelled" as const };
    },

    // Today's sales. One paid, one refunded — the two rows the list has to
    // draw differently, and the pair a cashier comes looking for.
    tillClosedChecks: async () => ({
      checks: [
        {
          id: "closed-1",
          number: "A-0011",
          status: "delivered",
          tableNumber: "3",
          openedAt: "2026-08-18T11:00:00Z",
          openMin: 40,
          lines: [
            {
              lineId: "l-1",
              name: PLAIN_DISH,
              price: 42000,
              qty: 1,
              sum: 42000,
              fired: true,
            },
          ],
          subtotal: 42000,
          unfired: 0,
          total: 42000,
          closedAt: "2026-08-18T11:40:00Z",
          closedBy: "Nodira",
          paymentMethod: "cash" as const,
          paymentStatus: "paid",
        },
        {
          id: "closed-2",
          number: "A-0012",
          status: "delivered",
          openedAt: "2026-08-18T12:00:00Z",
          openMin: 10,
          lines: [],
          subtotal: 90000,
          unfired: 0,
          total: 90000,
          closedAt: "2026-08-18T12:10:00Z",
          paymentMethod: "card" as const,
          paymentStatus: "refunded",
          refund: {
            at: "2026-08-18T12:30:00Z",
            reason: "taom sovuq edi",
            amount: 90000,
          },
        },
      ] as Check[],
      total: 42000,
      refunded: 90000,
      owed: 0,
      count: 2,
    }),
    tillPaymentMethods: async () => ({ methods: paymentMethods }),
    // One number owes something, the rest owe nothing — the case the screen
    // has to say out loud rather than leave blank.
    tillDebts: async (phone: string) =>
      phone === "998901234567"
        ? {
            name: "Aziz Karimov",
            phone,
            debts: [
              {
                orderId: "d-1",
                number: "A-0007",
                at: "2026-08-17T14:00:00Z",
                total: 120000,
                note: "juma kuni to'laydi",
              },
            ],
            total: 120000,
          }
        : { name: "", phone, debts: [], total: 0 },
    tillPayDebt: async (orderId: string, method: string) => {
      calls.payDebt.push({ orderId, method });
      return { ok: true };
    },
    tillStartPayment: async (id: string, provider: string) => {
      calls.payOnline.push({ checkId: id, provider });
      return {
        url: `https://checkout.paycom.uz/${id}`,
        provider,
        total: checks.get(id)?.total ?? 0,
        number: checks.get(id)?.number ?? "",
      };
    },
    tillPaymentStatus: async () => ({
      status: onlinePaid ? "paid" : "pending",
      method: "payme",
      paid: onlinePaid,
    }),

    // The customer a debt is written against. One phone finds somebody, the
    // rest find nobody — which is the case the till has to refuse in.
    adminLookup: async (phone: string) => ({
      phone,
      user:
        phone === "998901234567"
          ? { id: "u-1", firstName: "Aziz", lastName: "Karimov", phone }
          : null,
    }),
    // ⚠️ **The till's own lookup, not the panel's.** Writing a debt used to
    // call `/admin/lookup`, which needs an administrator's token: on the
    // desktop app that failed silently, and in a browser it worked only
    // because somebody had signed into the panel on that machine. Narrow on
    // purpose — a name and an id, never the customer card.
    tillCustomer: async (phone: string) => ({
      user:
        phone === "998901234567"
          ? { id: "u-1", name: "Aziz Karimov", phone, creditAllowed: true }
          : // ⚠️ A second regular, found and **not** allowed to owe: the till
            // has to tell the two apart, because refusing everybody who is not
            // in the database is a different rule from refusing this guest.
            phone === "998907654321"
            ? {
                id: "u-2",
                name: "Bek Yusupov",
                phone,
                creditAllowed: false,
              }
            : null,
    }),

    // ---- Fiscal: off, which is the state of every restaurant without a
    // register and the one where these screens must still sell food.
    tillFiscalStatus: async () => ({ enabled: false }),
    tillUnfiledChecks: async () => ({ checks: [] as Check[] }),
    // No bookings by default: the strip draws nothing at all in that case,
    // which is the state a restaurant that takes no bookings is always in.
    tillReservations: async () => ({ reservations: [] }),
    tillSyncChecks: async (batch: { clientId: string; lines: unknown[] }[]) => {
      calls.sync.push(...batch);
      return {
        results: batch.map((c) => ({ clientId: c.clientId, id: "srv-1" })),
        serverTime: new Date().toISOString(),
      };
    },
  };

  return {
    api,
    calls,
    checks,
    /** What the provider's callback does, from outside the screen. */
    confirmPayment: () => {
      onlinePaid = true;
    },
  };
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

"use client";

// "Which table am I sitting at?"
//
// Every table carries a printed QR code (see the admin QR generator). Scanning
// it opens the site with `?table=<id>`, and from then on the guest is *at that
// table*: the cart offers to serve the order to the table instead of asking for
// an address, and the kitchen sees the table number on the receipt.
//
// The choice is remembered in sessionStorage, not localStorage, on purpose: it
// belongs to this visit. A guest who goes home and orders delivery from the
// same phone must not have last week's table quietly attached to it.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import {
  clearBranchCookie,
  writeBranchCookie,
  writeBrandCookie,
} from "@/lib/siteBrand";

const STORAGE_KEY = "table_v1";

export interface TableSession {
  id: string;
  number: string;
  /** Which branch's room this table is in — printed on the QR card. */
  branchId?: string;
}

interface TableContextValue {
  table: TableSession | null;
  /** Forget the table (the guest is ordering delivery after all). */
  clear: () => void;
}

const TableContext = createContext<TableContextValue | null>(null);

export function TableProvider({ children }: { children: ReactNode }) {
  const params = useSearchParams();
  const router = useRouter();
  const scanned = params.get("table");
  // A printed QR card carries the brand and the branch alongside the table:
  // the card is glued to that table for years, so it has to say which menu and
  // which kitchen it belongs to, not rely on whatever the browser remembers.
  const scannedBrand = params.get("brand");
  const scannedBranch = params.get("branch");
  const [table, setTable] = useState<TableSession | null>(null);

  // A scanned card moves the guest into that brand and that branch. Written
  // before the table lookup below, so the lookup already asks the right room.
  useEffect(() => {
    if (!scannedBrand && !scannedBranch) return;
    if (scannedBrand) writeBrandCookie(scannedBrand);
    if (scannedBranch) writeBranchCookie(scannedBranch);
    // The menu and the theme are server-rendered from those cookies.
    router.refresh();
  }, [scannedBrand, scannedBranch, router]);

  useEffect(() => {
    // A QR scan wins over whatever was remembered — the guest moved tables.
    if (scanned) {
      // The number shown to the guest comes from the floor plan, so a table
      // that was renumbered or removed cannot produce a wrong receipt.
      api
        .bookingPlan(undefined, {
          brand: scannedBrand ?? undefined,
          branchId: scannedBranch ?? undefined,
        })
        .then((plan) => {
          const found = plan.booking.tables.find((t) => t.id === scanned);
          if (!found) return;
          const next: TableSession = {
            id: found.id,
            number: found.number,
            branchId: scannedBranch ?? plan.branchId,
          };
          setTable(next);
          window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        })
        .catch(() => {
          /* offline: the guest can still browse, just not as "at table 7" */
        });
      return;
    }
    try {
      const raw = window.sessionStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw) as Partial<TableSession>;
      if (typeof parsed?.id === "string" && typeof parsed?.number === "string") {
        setTable({
          id: parsed.id,
          number: parsed.number,
          branchId: typeof parsed.branchId === "string" ? parsed.branchId : undefined,
        });
      }
    } catch {
      /* corrupt value — behave as if nothing was scanned */
    }
  }, [scanned, scannedBrand, scannedBranch]);

  const clear = useCallback(() => {
    setTable(null);
    // Leaving the table also releases the branch it pinned: a guest ordering
    // delivery from home must be served by whichever branch covers their
    // address, not by the one whose chair they sat in this afternoon.
    clearBranchCookie();
    try {
      window.sessionStorage.removeItem(STORAGE_KEY);
    } catch {
      /* nothing to forget */
    }
  }, []);

  const value = useMemo(() => ({ table, clear }), [table, clear]);
  return (
    <TableContext.Provider value={value}>{children}</TableContext.Provider>
  );
}

export function useTable(): TableContextValue {
  return useContext(TableContext) ?? { table: null, clear: () => {} };
}

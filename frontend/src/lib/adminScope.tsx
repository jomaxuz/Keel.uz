"use client";

// Which brand and branch the panel is currently looking at.
//
// A company may run several brands, each with several branches; every list in
// the panel is read through this lens. The rule that keeps the product usable
// for a single restaurant: when there is one brand and one branch, the switcher
// is not rendered at all and every screen behaves exactly as it did before.
//
// The choice is remembered per browser, so a branch manager's laptop opens on
// their own branch each morning.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api, setAdminScope } from "@/lib/api";
import { hasId } from "@/lib/id";
import type { AdminUser, Brand, Branch } from "@/lib/types";

const BRAND_KEY = "admin_brand";
const BRANCH_KEY = "admin_branch";

interface ScopeValue {
  brands: Brand[];
  branches: Branch[];
  /** Branches of the selected brand only. */
  brandBranches: Branch[];
  brand: Brand | null;
  branch: Branch | null;
  /** True while the first load is in flight — screens should wait for it. */
  loading: boolean;
  /**
   * Changes whenever the lens moves. Screens put it in their effect deps to
   * reload; it is a plain string, so it works as a `key` too.
   */
  scopeKey: string;
  /** More than one of either: the panel shows the switcher. */
  multi: boolean;
  /** The signed-in panel account, once loaded. */
  me: AdminUser | null;
  /** Owner: may edit the company, the brands and the list of branches. */
  isOwner: boolean;
  /** Pinned to one branch by their account — the lens cannot be moved. */
  pinned: boolean;
  setBrand: (id: string) => void;
  /** "" means every branch of the brand (an owner's overview). */
  setBranch: (id: string) => void;
  reload: () => void;
}

const ScopeContext = createContext<ScopeValue | null>(null);

export function AdminScopeProvider({ children }: { children: ReactNode }) {
  const [brands, setBrands] = useState<Brand[]>([]);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [brandId, setBrandId] = useState<string>("");
  const [branchId, setBranchId] = useState<string>("");
  const [me, setMe] = useState<AdminUser | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([
      api.adminBrands().catch(() => [] as Brand[]),
      api.adminBranches().catch(() => [] as Branch[]),
      api.me().catch(() => null),
    ])
      .then(([bs, brs, account]) => {
        setBrands(bs);
        setBranches(brs);
        setMe(account);

        // A manager pinned to a branch has no choice to make: the server clamps
        // them to it whatever the panel asks for. So the lens opens there and
        // stays there — a switcher that silently does nothing is worse than no
        // switcher at all.
        // hasId, not a truthiness check: an unset id arrives as a string of
        // zeros and would pin every admin to a branch that does not exist.
        if (hasId(account?.branchId)) {
          const own = brs.find((b) => b.id === account.branchId);
          setBranchId(account.branchId);
          setBrandId(own?.brandId ?? bs[0]?.id ?? "");
          return;
        }

        // Restore the last choice, but never point at something deleted.
        const storedBrand = window.localStorage.getItem(BRAND_KEY) ?? "";
        const storedBranch = window.localStorage.getItem(BRANCH_KEY) ?? "";
        const brand = bs.find((b) => b.id === storedBrand) ?? bs[0];
        setBrandId(brand?.id ?? "");
        const branch = brs.find(
          (b) => b.id === storedBranch && b.brandId === brand?.id,
        );
        // A brand with a single branch is always "in" that branch; with several,
        // the panel opens on the overview.
        const only = brs.filter((b) => b.brandId === brand?.id);
        setBranchId(branch?.id ?? (only.length === 1 ? only[0].id : ""));
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(load, [load]);

  const setBrand = useCallback(
    (id: string) => {
      setBrandId(id);
      window.localStorage.setItem(BRAND_KEY, id);
      const only = branches.filter((b) => b.brandId === id);
      const next = only.length === 1 ? only[0].id : "";
      setBranchId(next);
      window.localStorage.setItem(BRANCH_KEY, next);
    },
    [branches],
  );

  const setBranch = useCallback((id: string) => {
    setBranchId(id);
    window.localStorage.setItem(BRANCH_KEY, id);
  }, []);

  // The API layer holds the lens for every admin call (see setAdminScope).
  // Applied during render, not in an effect: an effect would let the first
  // fetch of a freshly mounted screen go out unscoped and show another
  // branch's orders for a blink.
  setAdminScope({ brandId, branchId });

  const value = useMemo<ScopeValue>(() => {
    const brandBranches = branches.filter((b) => b.brandId === brandId);
    return {
      brands,
      branches,
      brandBranches,
      brand: brands.find((b) => b.id === brandId) ?? null,
      branch: branches.find((b) => b.id === branchId) ?? null,
      loading,
      scopeKey: `${brandId}:${branchId}`,
      multi: brands.length > 1 || branches.length > 1,
      me,
      isOwner: me?.role === "owner",
      pinned: hasId(me?.branchId),
      setBrand,
      setBranch,
      reload: load,
    };
  }, [
    brands,
    branches,
    brandId,
    branchId,
    loading,
    me,
    setBrand,
    setBranch,
    load,
  ]);

  return (
    <ScopeContext.Provider value={value}>{children}</ScopeContext.Provider>
  );
}

export function useAdminScope(): ScopeValue {
  const ctx = useContext(ScopeContext);
  if (!ctx) {
    // Outside the provider (a stray render) the panel behaves single-brand.
    return {
      brands: [],
      branches: [],
      brandBranches: [],
      brand: null,
      branch: null,
      loading: false,
      scopeKey: "",
      multi: false,
      me: null,
      isOwner: false,
      pinned: false,
      setBrand: () => {},
      setBranch: () => {},
      reload: () => {},
    };
  }
  return ctx;
}

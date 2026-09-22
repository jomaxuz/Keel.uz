// Which set of words the **panel** uses for the thing this business sells.
//
// ⚠️ **The sidebar was fixed and the screens were not.** `navLabel` already
// turns "Masalliqlar" into "Tovarlar" for a chemist, on the reasoning that a
// person reading a kitchen's word over a shelf of paracetamol concludes the
// panel was built for somebody else. Everything one click further in still said
// it: a pharmacy opened its catalogue to "+ Yangi taom", read "5 ta taom" under
// every category, cancelled an order with the preset "Taom qolmagan" and found
// a column headed "Taom" in its sales report. One wrong word is forgiven; the
// same wrong word on every screen is the sentence the sidebar was fixed to stop
// saying.
//
// ⚠️ **Two readings, not twelve** — the rule `siteWords` and `settingsTabLabel`
// already follow. A pharmacy is a shop, a boutique is a shop and an online
// store is a shop; a dictionary per business type would be a dozen copies of
// every string to keep in step for a gain nobody can see.
//
// ⚠️ **Words, never a second screen.** Two screens are two things to keep in
// step and two places for a fix to be applied once — and the difference between
// a shop's catalogue row and a kitchen's is a noun.
//
// ⚠️ **Not a permission and not a rule about data.** What a business *may* do
// is decided by `needsMet` and by the server; this file only decides what the
// screen calls it. A restaurant that starts selling bottled water keeps every
// field it had.

import { useMemo } from "react";

import { useAdminScope } from "@/lib/adminScope";
import { useAdminT, type AdminDict } from "@/lib/i18n/admin";
import { sellsGoods } from "@/lib/types";

export type BrandLike = { businessType?: string } | null | undefined;

/** Everything the panel needs a noun for.
 *
 *  ⚠️ **One flat object, built in one place.** The alternative — each screen
 *  asking `sellsGoods` and picking between two dictionary keys inline — is the
 *  state this replaced, and it is how the sidebar came to be right while the
 *  screens under it were wrong. */
export interface PanelWords {
  /** The catalogue section: "Menyu" or "Katalog". */
  catalog: string;
  item: string;
  items: string;
  count: (n: number) => string;
  add: string;
  addNew: string;
  newTitle: string;
  editTitle: string;
  confirmDelete: (name: string) => string;
  noItems: string;
  empty: string;
  needCategoryNotice: string;
  categoryDeleteFull: (name: string) => string;
  search: string;
  uncostedCount: (n: number) => string;
  uncostedShowAll: string;
  costHint: string;
  topSelling: string;
  stopLead: string;
  stopNoItems: string;
  stopSynced: (n: number) => string;
  stopPosSynced: (n: number) => string;
  stopPosLocked: string;
  comboHint: string;
  comboAdd: string;
  comboMissing: string;
  comboNothingToAdd: string;
  comboEmpty: string;
  cancelReasonPh: string;
  /** The cancel preset a dispatcher presses most: "Taom qolmagan". */
  cancelPresetOut: string;
  recommendTitle: string;
  recommendSearch: string;
  optionsLead: string;
  optionsEmpty: string;
  reportItem: string;
  reportCount: string;
  /** The till, which a grocery's cashier stands at all day. */
  tillSearch: string;
  tillEmptyCheck: string;
  tillAdd: string;
  tillTotalLabel: string;
  tillNeedLines: string;
  tillComment: string;
  tillMoveHint: string;
  tillStopSearch: string;
  tillStopEmpty: string;
  tillStopOffCount: (n: number) => string;
  tillStopConfirmOff: string;
}

/** The words this business's panel uses.
 *
 *  ⚠️ **An unknown business type reads as a restaurant**, the fallback every
 *  predicate in lib/types.ts makes: a panel meeting a type from a newer console
 *  keeps saying what it said yesterday rather than saying nothing. */
export function panelWords(t: AdminDict, brand: BrandLike): PanelWords {
  const g = t.goods;
  if (sellsGoods(brand)) {
    return {
      catalog: g.catalog,
      item: g.item,
      items: g.items,
      count: g.count,
      add: g.add,
      addNew: g.addNew,
      newTitle: g.newTitle,
      editTitle: g.editTitle,
      confirmDelete: g.confirmDelete,
      noItems: g.noItems,
      empty: g.empty,
      needCategoryNotice: g.needCategoryNotice,
      categoryDeleteFull: g.categoryDeleteFull,
      search: g.search,
      uncostedCount: g.uncostedCount,
      uncostedShowAll: g.uncostedShowAll,
      costHint: g.costHint,
      topSelling: g.topSelling,
      stopLead: g.stopLead,
      stopNoItems: g.stopNoItems,
      stopSynced: g.stopSynced,
      stopPosSynced: g.stopPosSynced,
      stopPosLocked: g.stopPosLocked,
      comboHint: g.comboHint,
      comboAdd: g.comboAdd,
      comboMissing: g.comboMissing,
      comboNothingToAdd: g.comboNothingToAdd,
      comboEmpty: g.comboEmpty,
      cancelReasonPh: g.cancelReasonPh,
      cancelPresetOut: g.cancelPresetOut,
      recommendTitle: g.recommendTitle,
      recommendSearch: g.recommendSearch,
      optionsLead: g.optionsLead,
      optionsEmpty: g.optionsEmpty,
      reportItem: g.reportItem,
      reportCount: g.reportCount,
      tillSearch: g.tillSearch,
      tillEmptyCheck: g.tillEmptyCheck,
      tillAdd: g.tillAdd,
      tillTotalLabel: g.tillTotalLabel,
      tillNeedLines: g.tillNeedLines,
      tillComment: g.tillComment,
      tillMoveHint: g.tillMoveHint,
      tillStopSearch: g.tillStopSearch,
      tillStopEmpty: g.tillStopEmpty,
      tillStopOffCount: g.tillStopOffCount,
      tillStopConfirmOff: g.tillStopConfirmOff,
    };
  }
  return {
    catalog: t.menu.title,
    item: t.menu.kindDish,
    items: t.dashboard.menuDishes,
    count: t.categories.itemsCount,
    add: t.menu.add,
    addNew: t.menu.addNew,
    newTitle: t.menu.newTitle,
    editTitle: t.menu.editTitle,
    confirmDelete: t.menu.confirmDelete,
    noItems: t.menu.noItems,
    empty: t.menu.empty,
    needCategoryNotice: t.menu.needCategoryNotice,
    categoryDeleteFull: t.categories.confirmDeleteFull,
    search: t.menu.comboSearch,
    uncostedCount: t.menu.uncostedCount,
    uncostedShowAll: t.menu.uncostedShowAll,
    costHint: t.menu.costHint,
    topSelling: t.dashboard.topDishes,
    stopLead: t.stopList.subtitle,
    stopNoItems: t.stopList.noItems,
    stopSynced: t.stopList.stockSynced,
    stopPosSynced: t.stopList.posSynced,
    stopPosLocked: t.stopList.posLocked,
    comboHint: t.menu.comboHint,
    comboAdd: t.menu.comboAddDish,
    comboMissing: t.menu.comboMissingDish,
    comboNothingToAdd: t.menu.comboNothingToAdd,
    comboEmpty: t.menu.comboEmpty,
    cancelReasonPh: t.orders.cancelReasonPh,
    cancelPresetOut: t.orders.cancelPresets[1],
    recommendTitle: t.menu.recommend.title,
    recommendSearch: t.menu.recommend.searchPh,
    optionsLead: t.options.lead,
    optionsEmpty: t.options.empty,
    reportItem: t.reports.dish,
    reportCount: t.reports.sales.items,
    tillSearch: t.till.search,
    tillEmptyCheck: t.till.emptyCheck,
    tillAdd: t.till.addDish,
    tillTotalLabel: t.till.dishesTotal,
    tillNeedLines: t.till.payNeedsLines,
    tillComment: t.till.commentTitle,
    tillMoveHint: t.till.moveLinesHint,
    tillStopSearch: t.till.stopSearch,
    tillStopEmpty: t.till.stopEmpty,
    tillStopOffCount: t.till.stopOffCount,
    tillStopConfirmOff: t.till.stopConfirmOffTitle,
  };
}

/** The words for the brand the panel is currently looking at.
 *
 *  ⚠️ **Reads the lens, not the login.** A company running a restaurant and a
 *  shop switches between them in the same session, and the words have to follow
 *  the switch — which is the same reason every list on these screens is read
 *  through `useAdminScope`. Outside the provider it answers as a restaurant,
 *  exactly as the scope itself does. */
export function usePanelWords(): PanelWords {
  const t = useAdminT();
  const { brand } = useAdminScope();
  return useMemo(() => panelWords(t, brand), [t, brand]);
}

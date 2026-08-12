"use client";

// Customers registered on the site (phone + one-time SMS code), with how many
// orders each of them placed.

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { useRouter } from "next/navigation";
import { formatDate, formatDateTime, formatPrice, formatUzPhone } from "@/lib/format";
import { timeAgo } from "@/lib/orderFlow";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { SEGMENTS, SegmentBadge } from "@/components/admin/Segments";
import type { AdminUserRow, CustomerSegment } from "@/lib/types";

export default function AdminUsersPage() {
  const router = useRouter();
  const [users, setUsers] = useState<AdminUserRow[]>([]);
  const [loading, setLoading] = useState(true);
  const paged = usePaged(users, 25);
  const [q, setQ] = useState("");
  // Which group is being looked at. Empty = everyone. The counts next to each
  // chip come from the unfiltered list, so the owner can see the size of a
  // group before deciding to open it.
  const [segment, setSegment] = useState<CustomerSegment | "">("");
  const [tag, setTag] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [all, setAll] = useState<AdminUserRow[]>([]);
  const t = useAdminT();

  useEffect(() => {
    // Debounce the search so typing doesn't hammer the API.
    const id = setTimeout(() => {
      setLoading(true);
      api
        .adminUsers({
          q: q.trim() || undefined,
          segment: segment || undefined,
          tag: tag || undefined,
        })
        .then(setUsers)
        .catch(() => setUsers([]))
        .finally(() => setLoading(false));
    }, 250);
    return () => clearTimeout(id);
  }, [q, segment, tag]);

  // The unfiltered base, for the chip counts and the tag list.
  useEffect(() => {
    api.adminUsers().then(setAll).catch(() => setAll([]));
    api.adminTags().then(setTags).catch(() => setTags([]));
  }, []);

  const segmentCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const u of all) {
      for (const s of u.segments ?? []) counts[s] = (counts[s] ?? 0) + 1;
    }
    return counts;
  }, [all]);

  const totals = useMemo(
    () => ({
      users: users.length,
      ordered: users.filter((u) => u.ordersCount > 0).length,
      withAddress: users.filter((u) => (u.addresses?.length ?? 0) > 0).length,
      revenue: users.reduce((s, u) => s + u.ordersTotal, 0),
    }),
    [users],
  );

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="font-display text-2xl font-bold">{t.users.title}</h1>
        <input
          className="input max-w-xs"
          placeholder={t.users.searchPh}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      {/* Groups worth acting on. "Sleeping" is the one that makes money: a
          customer who used to order and stopped is cheaper to bring back than
          a new one is to find. */}
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => setSegment("")}
          className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition-colors ${
            segment === ""
              ? "border-brand bg-brand-tint text-brand-dark"
              : "border-line-strong text-ink-soft hover:border-brand"
          }`}
        >
          {t.users.segAll} · {all.length}
        </button>
        {SEGMENTS.map((seg) => (
          <button
            key={seg}
            type="button"
            onClick={() => setSegment(segment === seg ? "" : seg)}
            title={t.users.segHint[seg]}
            className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition-colors ${
              segment === seg
                ? "border-brand bg-brand-tint text-brand-dark"
                : "border-line-strong text-ink-soft hover:border-brand"
            }`}
          >
            {t.users.segment[seg]} · {segmentCounts[seg] ?? 0}
          </button>
        ))}
      </div>

      {tags.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <span className="text-xs text-ink-muted">{t.users.tagsLabel}</span>
          {tags.map((x) => (
            <button
              key={x}
              type="button"
              onClick={() => setTag(tag === x ? "" : x)}
              className={`rounded-full border px-2.5 py-1 text-xs font-semibold transition-colors ${
                tag === x
                  ? "border-brand bg-brand-tint text-brand-dark"
                  : "border-line-strong text-ink-muted hover:border-brand"
              }`}
            >
              {x}
            </button>
          ))}
        </div>
      )}

      <div className="mt-6 grid gap-4 sm:grid-cols-4">
        <Stat label={t.users.statTotal} value={String(totals.users)} />
        <Stat label={t.users.statOrdered} value={String(totals.ordered)} />
        <Stat label={t.users.statAddress} value={String(totals.withAddress)} />
        <Stat label={t.users.statRevenue} value={formatPrice(totals.revenue)} />
      </div>

      <div className="mt-6 overflow-hidden rounded-3xl border border-line bg-surface shadow-card">
        {loading ? (
          <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
        ) : users.length === 0 ? (
          <p className="py-10 text-center text-ink-muted/70">
            {t.users.empty}
          </p>
        ) : (
          <ListScroll className="overflow-x-auto" max="max-h-[68vh]">
            <table className="w-full min-w-[900px] text-sm">
              <thead className="sticky top-0 z-10 border-b border-line bg-surface text-left text-xs uppercase tracking-wider text-ink-muted">
                <tr>
                  <th className="px-4 py-3 font-semibold">{t.users.colUser}</th>
                  <th className="px-4 py-3 font-semibold">{t.users.colPhone}</th>
                  <th className="px-4 py-3 font-semibold">
                    {t.users.colAddresses}
                  </th>
                  <th className="px-4 py-3 font-semibold">
                    {t.users.colLastOrder}
                  </th>
                  <th className="px-4 py-3 text-right font-semibold">
                    {t.users.colOrders}
                  </th>
                  <th className="px-4 py-3 text-right font-semibold">
                    {t.users.colSum}
                  </th>
                  <th className="px-4 py-3 font-semibold">
                    {t.users.colRegistered}
                  </th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {paged.pageItems.map((u) => (
                  <tr
                    key={u.id}
                    onClick={() => router.push(`/admin/users/${u.id}`)}
                    className="cursor-pointer hover:bg-ink/[0.02]"
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-tint text-xs font-bold text-brand">
                          {(u.firstName || u.phone || "?")
                            .charAt(0)
                            .toUpperCase()}
                        </span>
                        <div className="min-w-0">
                          <p className="font-medium">
                            {[u.firstName, u.lastName]
                              .filter(Boolean)
                              .join(" ") || "—"}
                          </p>
                          {/* The labels, right next to the name: the point of
                              the list is to see at a glance who needs what. */}
                          {(u.segments?.length || u.tags?.length) && (
                            <div className="mt-1 flex flex-wrap gap-1">
                              {(u.segments ?? []).map((sg) => (
                                <SegmentBadge key={sg} segment={sg} />
                              ))}
                              {(u.tags ?? []).map((x) => (
                                <span
                                  key={x}
                                  className="rounded-full border border-line-strong px-2 py-0.5 text-xs text-ink-muted"
                                >
                                  {x}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      {u.phone ? (
                        <a
                          href={`tel:+${u.phone.replace(/\D/g, "")}`}
                          className="hover:text-brand"
                        >
                          {formatUzPhone(u.phone)}
                        </a>
                      ) : (
                        <span className="text-ink-muted/70">—</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-ink-muted">
                      {u.addresses?.length ? (
                        <span title={u.addresses.map((a) => a.text).join("\n")}>
                          {t.users.addressCount(u.addresses.length)}
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td className="px-4 py-3 text-ink-muted">
                      {u.lastOrderAt ? (
                        <span title={formatDateTime(u.lastOrderAt)}>
                          {timeAgo(u.lastOrderAt, t.common.timeAgo)}
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td className="px-4 py-3 text-right font-semibold">
                      {u.ordersCount}
                    </td>
                    <td className="px-4 py-3 text-right">
                      {u.ordersTotal > 0 ? formatPrice(u.ordersTotal) : "—"}
                    </td>
                    <td className="px-4 py-3 text-ink-muted">
                      {formatDate(u.createdAt)}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        href={`/admin/users/${u.id}`}
                        onClick={(e) => e.stopPropagation()}
                        className="text-xs font-semibold text-brand hover:underline"
                      >
                        {t.common.details} →
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </ListScroll>
        )}
        <Pager
          page={paged.page}
          pageCount={paged.pageCount}
          from={paged.from}
          to={paged.to}
          total={paged.total}
          onPage={paged.setPage}
        />
      </div>


    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-3xl border border-line bg-surface p-4 shadow-card">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p className="mt-1 font-display text-xl font-bold">{value}</p>
    </div>
  );
}

import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { Search, ChevronRight, ChevronsUpDown, ArrowUp, ArrowDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { docsUrlFor } from "@/lib/docsLinks";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { TableSkeleton } from "./LoadingSkeleton";

/**
 * DataTable v2 — the canonical list-page table (DASHBOARD_REDESIGN.md Stage 4).
 *
 * Columns config:
 *   [{ key, header,
 *      cell?: (row) => ReactNode,
 *      align?: "left"|"right"|"center",
 *      sortable?: boolean,             // header becomes a real sort button
 *      sortValue?: (row) => any,       // client-sort accessor (default row[key])
 *      hideBelow?: "sm"|"md"|"lg",     // column priority: hidden under bp
 *      minWidth?: number|string,       // px min-width for the column
 *      className?, headerClassName? }]
 *
 * Props:
 *  - columns, data (required)
 *  - loading, error, onRetry
 *  - onRowClick(row) — row activation. v2 semantics: the FIRST column's
 *    content is wrapped in a real <button> that carries keyboard focus and
 *    AT semantics; the <tr> itself keeps only a mouse-convenience onClick
 *    (no role="button", no tabIndex). This fixes the double-fire bug (Enter
 *    on a nested action button no longer also activates the row) and stops
 *    flattening row/cell semantics for screen readers. A trailing chevron
 *    signals clickability (rowChevron={false} to opt out).
 *  - rowHref(row) — real link semantics: the first cell renders a <Link>
 *    (⌘-click, middle-click, copy-address all work) and a plain row click
 *    navigates. Prefer this over onRowClick wherever the object has a URL.
 *  - sort / onSortChange — controlled (server) sorting: sort={{key,dir}}.
 *    Omit both and mark columns sortable for built-in CLIENT sorting; only
 *    do that on fully-loaded lists (sorting one server page misleads).
 *  - getRowId(row) (defaults to row.id)
 *  - search: { value, onChange, placeholder }
 *  - toolbar: ReactNode (wraps on narrow screens)
 *  - density: "comfortable" (default) | "compact"
 *  - footer: ReactNode rendered in a <TableFooter> (totals rows)
 *  - renderExpanded(row) + expandedId: single-row expansion slot
 *  - empty: { icon, title, description, action, learnMoreHref? }
 *  - docsLink: false to suppress the auto guide link on empty
 *  - pagination:
 *      { page, pageSize?, total?, onPageChange? }   (preferred contract)
 *      { page, onPrev, onNext, hasNext, total? }    (legacy, still supported)
 *    With total+pageSize it renders "start–end of total" and computes the
 *    Next boundary exactly (no more Next-into-an-empty-page).
 */
export function DataTable({
  columns,
  data = [],
  loading = false,
  error = null,
  onRetry,
  onRowClick,
  rowHref,
  rowChevron = true,
  sort,
  onSortChange,
  getRowId = (row) => row.id,
  search,
  toolbar,
  density = "comfortable",
  footer,
  renderExpanded,
  expandedId,
  empty = {},
  docsLink = true,
  pagination,
  className,
}) {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const [internalSort, setInternalSort] = useState(null);
  const alignClass = {
    left: "text-left",
    right: "text-right",
    center: "text-center",
  };
  const hideClass = {
    sm: "hidden sm:table-cell",
    md: "hidden md:table-cell",
    lg: "hidden lg:table-cell",
  };

  const sortState = sort ?? internalSort;
  const clientSorted = !onSortChange && Boolean(internalSort);

  const toggleSort = (col) => {
    const dir =
      sortState?.key === col.key && sortState.dir === "asc" ? "desc"
      : sortState?.key === col.key && sortState.dir === "desc" ? null
      : "asc";
    const next = dir ? { key: col.key, dir } : null;
    if (onSortChange) onSortChange(next);
    else setInternalSort(next);
  };

  const rows = useMemo(() => {
    if (!clientSorted) return data;
    const col = columns.find((c) => c.key === internalSort.key);
    if (!col) return data;
    const acc = col.sortValue || ((row) => row[col.key]);
    const mul = internalSort.dir === "desc" ? -1 : 1;
    return [...data].sort((a, b) => {
      const va = acc(a);
      const vb = acc(b);
      if (va == null && vb == null) return 0;
      if (va == null) return 1; // nulls last regardless of direction
      if (vb == null) return -1;
      if (typeof va === "number" && typeof vb === "number") return (va - vb) * mul;
      return String(va).localeCompare(String(vb)) * mul;
    });
  }, [data, clientSorted, internalSort, columns]);

  // New-row reveal: a row that wasn't present on the previous render animates
  // in. The whole table never animates on first mount (the ref starts null) —
  // the page-level reveal already covers first appearance. Sorting keeps ids
  // (rows just reposition), so only genuinely new rows flash: a loosened
  // filter, a page turn, or appended data.
  const prevIdsRef = useRef(null);
  const isNewRow = (id) =>
    prevIdsRef.current !== null && id != null && !prevIdsRef.current.has(id);
  useEffect(() => {
    prevIdsRef.current = new Set(rows.map(getRowId));
  });

  const showToolbar = Boolean(search || toolbar);
  const interactive = Boolean(onRowClick || rowHref);
  const activateRow = (row) => (rowHref ? navigate(rowHref(row)) : onRowClick(row));
  const showChevron = interactive && rowChevron;
  const cellPad = density === "compact" ? "[&_td]:py-2 [&_th]:h-9" : "";

  // Contextual guide link on the "nothing here yet" state — but not while the
  // user is filtering (a search that returns nothing isn't a getting-started
  // moment). An explicit empty.learnMoreHref always wins.
  const searching = Boolean(search?.value);
  const learnMoreHref =
    empty.learnMoreHref ?? (docsLink && !searching ? docsUrlFor(pathname) : undefined);

  // Pagination normalization: prefer the {page,pageSize,total,onPageChange}
  // contract; fall back to the legacy onPrev/onNext shape.
  const pg = pagination;
  const totalPages =
    pg?.total != null && pg?.pageSize ? Math.max(1, Math.ceil(pg.total / pg.pageSize)) : null;
  const canPrev = pg && pg.page > 1;
  const canNext = pg
    ? totalPages != null
      ? pg.page < totalPages
      : pg.hasNext !== false
    : false;
  const goPrev = () => (pg.onPageChange ? pg.onPageChange(pg.page - 1) : pg.onPrev?.());
  const goNext = () => (pg.onPageChange ? pg.onPageChange(pg.page + 1) : pg.onNext?.());
  const rangeText = () => {
    if (pg.total != null && pg.pageSize) {
      const start = (pg.page - 1) * pg.pageSize + 1;
      const end = Math.min(pg.page * pg.pageSize, pg.total);
      return `${start.toLocaleString()}–${end.toLocaleString()} of ${pg.total.toLocaleString()}`;
    }
    if (pg.total != null) return `${pg.total.toLocaleString()} total`;
    return `Page ${pg.page}`;
  };

  const sortIconFor = (col) => {
    if (sortState?.key !== col.key)
      return <ChevronsUpDown className="h-3.5 w-3.5 text-subtle/60" aria-hidden="true" />;
    return sortState.dir === "asc" ? (
      <ArrowUp className="h-3.5 w-3.5" aria-hidden="true" />
    ) : (
      <ArrowDown className="h-3.5 w-3.5" aria-hidden="true" />
    );
  };
  const ariaSortFor = (col) => {
    if (!col.sortable) return undefined;
    if (sortState?.key !== col.key) return "none";
    return sortState.dir === "asc" ? "ascending" : "descending";
  };

  return (
    <div className={cn("space-y-4", className)}>
      {showToolbar && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          {search && (
            <div className="relative w-full sm:max-w-xs">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-subtle" />
              <Input
                type="search"
                value={search.value}
                onChange={(e) => search.onChange(e.target.value)}
                placeholder={search.placeholder || "Search..."}
                className="pl-9"
              />
            </div>
          )}
          {/* min-w-0 + wrap: fixed-width filter selects must never force the
              page sideways (audit R3). */}
          {toolbar && (
            <div className="flex min-w-0 flex-wrap items-center gap-2">{toolbar}</div>
          )}
        </div>
      )}

      <Card className="overflow-hidden" aria-busy={loading || undefined}>
        {error ? (
          <ErrorState message={error} onRetry={onRetry} />
        ) : loading ? (
          <TableSkeleton rows={6} columns={columns.length} />
        ) : rows.length === 0 ? (
          <EmptyState
            icon={empty.icon}
            title={empty.title || "No results"}
            description={empty.description}
            action={empty.action}
            learnMoreHref={learnMoreHref}
            learnMoreLabel={empty.learnMoreLabel}
          />
        ) : (
          <Table className={cn(cellPad)}>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                {columns.map((col) => (
                  <TableHead
                    key={col.key}
                    aria-sort={ariaSortFor(col)}
                    style={col.minWidth ? { minWidth: col.minWidth } : undefined}
                    className={cn(
                      alignClass[col.align || "left"],
                      col.hideBelow && hideClass[col.hideBelow],
                      col.headerClassName
                    )}
                  >
                    {col.sortable ? (
                      <button
                        type="button"
                        onClick={() => toggleSort(col)}
                        className={cn(
                          "inline-flex items-center gap-1 rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                          col.align === "right" && "flex-row-reverse"
                        )}
                      >
                        {col.header}
                        {sortIconFor(col)}
                      </button>
                    ) : (
                      col.header
                    )}
                  </TableHead>
                ))}
                {showChevron && <TableHead className="w-8" aria-hidden="true" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const id = getRowId(row);
                return (
                  <RowGroup key={id}>
                    <TableRow
                      onClick={interactive ? () => activateRow(row) : undefined}
                      className={cn(
                        interactive && "cursor-pointer",
                        isNewRow(id) && "animate-motion-reveal"
                      )}
                      data-state={expandedId === id ? "expanded" : undefined}
                    >
                      {columns.map((col, i) => (
                        <TableCell
                          key={col.key}
                          className={cn(
                            alignClass[col.align || "left"],
                            col.hideBelow && hideClass[col.hideBelow],
                            col.className
                          )}
                        >
                          {interactive && i === 0 ? (
                            // The row's ONE keyboard/AT activation point:
                            // a real <Link> when the object has a URL
                            // (⌘-click works), else a real <button>. Never
                            // role=button on the <tr>; no double-fire with
                            // nested action buttons.
                            rowHref ? (
                              <Link
                                to={rowHref(row)}
                                onClick={(e) => e.stopPropagation()}
                                className="block w-full rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                              >
                                {col.cell ? col.cell(row) : row[col.key]}
                              </Link>
                            ) : (
                              <button
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  onRowClick(row);
                                }}
                                className="w-full rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                              >
                                {col.cell ? col.cell(row) : row[col.key]}
                              </button>
                            )
                          ) : col.cell ? (
                            col.cell(row)
                          ) : (
                            row[col.key]
                          )}
                        </TableCell>
                      ))}
                      {showChevron && (
                        <TableCell className="w-8 pr-4 text-subtle/60">
                          <ChevronRight className="h-4 w-4" aria-hidden="true" />
                        </TableCell>
                      )}
                    </TableRow>
                    {renderExpanded && expandedId === id && (
                      <TableRow className="hover:bg-transparent">
                        <TableCell colSpan={columns.length + (showChevron ? 1 : 0)}>
                          {renderExpanded(row)}
                        </TableCell>
                      </TableRow>
                    )}
                  </RowGroup>
                );
              })}
            </TableBody>
            {footer && <TableFooter>{footer}</TableFooter>}
          </Table>
        )}
      </Card>

      {pagination && !loading && !error && rows.length > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-sm tabular-nums text-muted-foreground">{rangeText()}</p>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={goPrev} disabled={!canPrev}>
              Previous
            </Button>
            <span className="text-sm tabular-nums text-muted-foreground">
              {totalPages != null ? `${pagination.page} / ${totalPages}` : pagination.page}
            </span>
            <Button variant="outline" size="sm" onClick={goNext} disabled={!canNext}>
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// Keyed fragment so a row and its expansion stay siblings inside <tbody>.
function RowGroup({ children }) {
  return <>{children}</>;
}

export default DataTable;

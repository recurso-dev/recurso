import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { Search, ChevronRight, ChevronsUpDown, ArrowUp, ArrowDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { docsUrlFor } from "@/lib/docsLinks";
import { sortRows } from "@/lib/tableSort";
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
import { Checkbox } from "@/components/ui/checkbox";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { TableSkeleton } from "./LoadingSkeleton";

// Stable so a caller that doesn't pass getRowId doesn't recreate it each render
// (which would defeat the selection-prune memo below).
const defaultGetRowId = (row) => row.id;

// The select-all header checkbox needs the DOM `indeterminate` property (no HTML
// attribute exists), so it's set imperatively via a ref.
function IndeterminateCheckbox({ indeterminate, ...props }) {
  const ref = useRef(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = Boolean(indeterminate) && !props.checked;
  });
  return <Checkbox ref={ref} {...props} />;
}

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
 *  - isFetching — react-query's background-refetch flag. While rows are already
 *    on screen it shows a thin progress line (+ aria-busy) instead of the
 *    skeleton, so a refetch/page turn is visible without a flash. Ignored while
 *    loading, errored or empty (those states already speak for themselves).
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
 *  - selectable + selectedIds (a Set) + onSelectionChange(nextSet): opt-in row
 *    selection. Semantics are deliberately PAGE-SCOPED — the header checkbox is
 *    "select all rows on this page", never "all matching records" (no backend
 *    supports that). Selection is pruned to the current result set whenever it
 *    changes (page/filter/search), so a stale id can never be actioned; a refetch
 *    of the same page keeps the selection. Row checkboxes stopPropagation so
 *    selecting never triggers row navigation.
 *  - renderBulkActions(selectedIds, clear): the action buttons for the bulk bar,
 *    which DataTable renders above the table whenever the selection is non-empty.
 */
export function DataTable({
  columns,
  data = [],
  loading = false,
  isFetching = false,
  error = null,
  onRetry,
  onRowClick,
  rowHref,
  rowChevron = true,
  sort,
  onSortChange,
  getRowId = defaultGetRowId,
  search,
  toolbar,
  density = "comfortable",
  footer,
  renderExpanded,
  expandedId,
  empty = {},
  docsLink = true,
  pagination,
  selectable = false,
  selectedIds,
  onSelectionChange,
  renderBulkActions,
  className,
  // Accessible name (Batch D): by default the table names itself from the page's
  // visible <h1> (PageHeader renders id="page-title") via aria-labelledby — no
  // duplicate hidden title, no generic "Data table". Pass `ariaLabel` for a table
  // that has no page heading to reference.
  ariaLabelledby = "page-title",
  ariaLabel,
}) {
  const { pathname, search: locationSearch } = useLocation();
  // The originating list URL (with its filters/search/page/sort) rides along as
  // navigation state so the object page's back-link can return here exactly —
  // see ObjectHeader/useListBackDestination. Direct object opens have no state
  // and fall back to the static list root.
  const fromUrl = pathname + locationSearch;
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

  // Built-in client sort (uncontrolled). Uses the SAME comparator as the shared
  // sortRows() the fully-loaded list pages call, so controlled and uncontrolled
  // sorts order identically. Only reorders `data` as given — safe only on
  // fully-loaded lists (see the sort/onSortChange doc above).
  const rows = useMemo(
    () => (clientSorted ? sortRows(data, internalSort, columns) : data),
    [data, clientSorted, internalSort, columns]
  );

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
  const refreshing = isFetching && !loading && !error && rows.length > 0;
  // Sticky header (Batch D): the table body scrolls inside the bounded wrapper
  // (stickyWrap), so each header cell pins to the top. Opaque bg so scrolling
  // rows don't bleed through; z above rows; a travelling bottom border for the
  // divider. Pure CSS position: sticky — no scroll listeners.
  const stickyHead = "sticky top-0 z-20 bg-muted border-b border-border";
  const stickyWrap = "max-h-[calc(100vh-15rem)]";
  const activateRow = (row) =>
    rowHref ? navigate(rowHref(row), { state: { from: fromUrl } }) : onRowClick(row);
  const showChevron = interactive && rowChevron;
  const cellPad = density === "compact" ? "[&_td]:py-2 [&_th]:h-9" : "";

  // Row selection (opt-in, page-scoped). The header selects only the visible
  // rows; selection is pruned to the current result set so a stale id (from a
  // page/filter/search that has moved on) can never be actioned.
  const selectionOn = selectable && selectedIds != null && Boolean(onSelectionChange);
  const selection = selectionOn ? selectedIds : null;
  const visibleIds = useMemo(() => rows.map((r) => getRowId(r)), [rows, getRowId]);
  const allVisibleSelected =
    selectionOn && visibleIds.length > 0 && visibleIds.every((id) => selection.has(id));
  const someVisibleSelected = selectionOn && visibleIds.some((id) => selection.has(id));

  useEffect(() => {
    if (!selectionOn || selection.size === 0) return;
    const valid = new Set(visibleIds);
    const next = new Set();
    let changed = false;
    selection.forEach((id) => (valid.has(id) ? next.add(id) : (changed = true)));
    if (changed) onSelectionChange(next);
  }, [selectionOn, selection, visibleIds, onSelectionChange]);

  const toggleAllVisible = () => {
    const next = new Set(selection);
    if (allVisibleSelected) visibleIds.forEach((id) => next.delete(id));
    else visibleIds.forEach((id) => next.add(id));
    onSelectionChange(next);
  };
  const toggleRow = (id) => {
    const next = new Set(selection);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onSelectionChange(next);
  };
  const clearSelection = () => onSelectionChange(new Set());
  const selectedCount = selectionOn ? selection.size : 0;

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

      {selectionOn && selectedCount > 0 && (
        <div
          role="region"
          aria-label={`${selectedCount} selected`}
          className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-primary/20 bg-primary/5 px-4 py-2.5"
        >
          <div className="flex items-center gap-3 text-sm">
            <span className="font-medium text-foreground">
              {selectedCount.toLocaleString()} selected{" "}
              <span className="font-normal text-muted-foreground">on this page</span>
            </span>
            <button
              type="button"
              onClick={clearSelection}
              className="rounded text-muted-foreground underline-offset-2 hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              Clear
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {renderBulkActions?.(selection, clearSelection)}
          </div>
        </div>
      )}

      <Card className="relative overflow-hidden" aria-busy={loading || refreshing || undefined}>
        {refreshing && (
          <div
            data-testid="datatable-refreshing"
            aria-hidden="true"
            className="absolute inset-x-0 top-0 z-30 h-0.5 animate-pulse bg-primary/70"
          />
        )}
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
          <Table
            className={cn(cellPad, "transition-opacity", refreshing && "opacity-70")}
            wrapperClassName={stickyWrap}
            aria-labelledby={ariaLabel ? undefined : ariaLabelledby}
            aria-label={ariaLabel}
          >
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                {selectionOn && (
                  <TableHead className={cn(stickyHead, "w-10 pl-4")}>
                    <IndeterminateCheckbox
                      checked={allVisibleSelected}
                      indeterminate={someVisibleSelected}
                      onChange={toggleAllVisible}
                      aria-label="Select all rows on this page"
                    />
                  </TableHead>
                )}
                {columns.map((col) => (
                  <TableHead
                    key={col.key}
                    aria-sort={ariaSortFor(col)}
                    style={col.minWidth ? { minWidth: col.minWidth } : undefined}
                    className={cn(
                      stickyHead,
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
                {showChevron && <TableHead className={cn(stickyHead, "w-8")} aria-hidden="true" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {/* eslint-disable-next-line react-hooks/refs -- isNewRow reads prevIdsRef during render on purpose (new-row reveal, see above) */}
              {rows.map((row) => {
                const id = getRowId(row);
                return (
                  <RowGroup key={id}>
                    <TableRow
                      onClick={interactive ? () => activateRow(row) : undefined}
                      className={cn(
                        interactive && "cursor-pointer",
                        selectionOn && selection.has(id) && "bg-primary/5",
                        isNewRow(id) && "animate-motion-reveal"
                      )}
                      data-state={
                        selectionOn && selection.has(id)
                          ? "selected"
                          : expandedId === id
                            ? "expanded"
                            : undefined
                      }
                    >
                      {selectionOn && (
                        <TableCell className="w-10 pl-4">
                          <Checkbox
                            checked={selection.has(id)}
                            onChange={() => toggleRow(id)}
                            onClick={(e) => e.stopPropagation()}
                            aria-label={`Select row ${id}`}
                          />
                        </TableCell>
                      )}
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
                                state={{ from: fromUrl }}
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
                        <TableCell colSpan={columns.length + (showChevron ? 1 : 0) + (selectionOn ? 1 : 0)}>
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

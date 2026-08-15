import { useCallback } from "react";
import { useUrlState } from "./useUrlState";

/**
 * Shared client-sort primitives for fully-loaded list pages (Batch F3).
 *
 * These exist so worklist sorting is HONEST: they sort the COMPLETE row set a
 * page holds in memory, never a single server page. Server-paginated lists
 * (page in the query key) must NOT use these — sorting one page misleads; they
 * need a backend ORDER BY contract instead (documented as a gap).
 *
 * The mechanism reuses DataTable's *controlled* sort mode (sort/onSortChange)
 * plus useTableSort() to persist the sort in the URL, so returning from an
 * object page restores the exact ordering (Batch F1). The comparator is the
 * SAME one DataTable's built-in client sort uses (compareValues below), so
 * controlled and uncontrolled sorts order identically.
 */

// Null-safe compare: numbers numerically, everything else via localeCompare,
// nulls always last regardless of direction. `mul` is +1 asc / -1 desc.
export function compareValues(va, vb, mul) {
  if (va == null && vb == null) return 0;
  if (va == null) return 1; // nulls last regardless of direction
  if (vb == null) return -1;
  if (typeof va === "number" && typeof vb === "number") return (va - vb) * mul;
  return String(va).localeCompare(String(vb)) * mul;
}

// Sort a fully-loaded row set by the active column, reading the column's
// `sortValue` accessor (or row[key]). Returns the original array untouched when
// there is no active sort or the column is unknown/not sortable. Callers pass
// the COMPLETE set (never a server page), so the ordering spans everything.
export function sortRows(rows, sort, columns) {
  if (!sort) return rows;
  const col = columns.find((c) => c.key === sort.key);
  if (!col || !col.sortable) return rows;
  const acc = col.sortValue || ((row) => row[col.key]);
  const mul = sort.dir === "desc" ? -1 : 1;
  return [...rows].sort((a, b) => compareValues(acc(a), acc(b), mul));
}

// The URL param is a single `sort=key:dir` (empty = unsorted).
export function parseSort(raw) {
  if (!raw) return null;
  const [key, dir] = raw.split(":");
  return key && (dir === "asc" || dir === "desc") ? { key, dir } : null;
}

/**
 * URL-persisted table sort for client-side / fully-loaded lists. Stores one
 * `sort=key:dir` param (default empty is omitted from the URL). Returns:
 *  - sort:    { key, dir } | null   — pass to DataTable's controlled `sort`
 *  - onSortChange(next)             — pass to DataTable's `onSortChange`
 *  - sortKey: the raw "key:dir" string, a stable value for effect/memo deps
 *             (e.g. useResetPageOnChange(setPage, [search, status, sortKey])).
 *
 * Pair with sortRows() over the complete set — NOT DataTable's built-in client
 * sort, which would only reorder the current page slice.
 */
export function useTableSort() {
  const [raw, setRaw] = useUrlState("sort", "");
  const sort = parseSort(raw);
  const onSortChange = useCallback(
    (next) => setRaw(next ? `${next.key}:${next.dir}` : ""),
    [setRaw],
  );
  return { sort, sortKey: raw, onSortChange };
}

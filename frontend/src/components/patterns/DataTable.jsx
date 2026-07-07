import { Search } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  Table,
  TableBody,
  TableCell,
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
 * DataTable — the canonical list-page table. Copy this usage for every list.
 *
 * Columns config:
 *   [{ key, header, cell?: (row) => ReactNode, align?: "left"|"right"|"center",
 *      className?, headerClassName? }]
 *
 * Props:
 *  - columns, data (required)
 *  - loading, error, onRetry
 *  - onRowClick(row)
 *  - getRowId(row)                 (defaults to row.id)
 *  - search: { value, onChange, placeholder }   (omit to hide search box)
 *  - toolbar: ReactNode            (filter chips / selects, rendered right of search)
 *  - empty: { icon, title, description, action }
 *  - pagination: { page, onPrev, onNext, hasNext, total? }
 */
export function DataTable({
  columns,
  data = [],
  loading = false,
  error = null,
  onRetry,
  onRowClick,
  getRowId = (row) => row.id,
  search,
  toolbar,
  empty = {},
  pagination,
  className,
}) {
  const alignClass = {
    left: "text-left",
    right: "text-right",
    center: "text-center",
  };

  const showToolbar = Boolean(search || toolbar);

  return (
    <div className={cn("space-y-4", className)}>
      {showToolbar && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          {search && (
            <div className="relative w-full sm:max-w-xs">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <Input
                type="search"
                value={search.value}
                onChange={(e) => search.onChange(e.target.value)}
                placeholder={search.placeholder || "Search..."}
                className="pl-9"
              />
            </div>
          )}
          {toolbar && <div className="flex items-center gap-2">{toolbar}</div>}
        </div>
      )}

      <Card className="overflow-hidden">
        {error ? (
          <ErrorState message={error} onRetry={onRetry} />
        ) : loading ? (
          <TableSkeleton rows={6} columns={columns.length} />
        ) : data.length === 0 ? (
          <EmptyState
            icon={empty.icon}
            title={empty.title || "No results"}
            description={empty.description}
            action={empty.action}
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/40 hover:bg-muted/40">
                {columns.map((col) => (
                  <TableHead
                    key={col.key}
                    className={cn(alignClass[col.align || "left"], col.headerClassName)}
                  >
                    {col.header}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((row) => (
                <TableRow
                  key={getRowId(row)}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  className={cn(onRowClick && "cursor-pointer")}
                >
                  {columns.map((col) => (
                    <TableCell
                      key={col.key}
                      className={cn(alignClass[col.align || "left"], col.className)}
                    >
                      {col.cell ? col.cell(row) : row[col.key]}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {pagination && !loading && !error && data.length > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {pagination.total != null
              ? `${pagination.total} total`
              : `Page ${pagination.page}`}
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={pagination.onPrev}
              disabled={pagination.page <= 1}
            >
              Previous
            </Button>
            <span className="text-sm tabular-nums text-muted-foreground">
              {pagination.page}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={pagination.onNext}
              disabled={pagination.hasNext === false}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default DataTable;

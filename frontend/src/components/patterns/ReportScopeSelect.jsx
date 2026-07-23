import { useQuery } from "@tanstack/react-query";
import { Building2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectGroup,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SCOPE_ALL, SCOPE_CONSOLIDATED } from "@/components/patterns/reportScope";

// ReportScopeSelect chooses how a finance report is scoped across legal entities
// (Multi-Entity Books): all entities (each line tagged with its entity), a
// tenant-wide consolidated rollup, or a single entity. Renders nothing for
// single-entity tenants — there is only one ledger to report on.
//
// hideConsolidated drops the "Consolidated" option for SCALAR reports (e.g. MRR)
// where a total across entities is identical to "All entities" — showing both
// would offer two controls that produce the same number.
export function ReportScopeSelect({ value, onChange, hideConsolidated = false }) {
  const { data: entities = [] } = useQuery({
    queryKey: ["entities"],
    queryFn: async () => (await endpoints.getEntities()).data?.data || [],
  });

  if (entities.length <= 1) return null;

  return (
    <div className="flex items-center gap-2">
      <Building2 className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
      <label htmlFor="report-scope" className="text-sm text-muted-foreground">
        Scope
      </label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="report-scope" className="w-[240px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={SCOPE_ALL}>All entities</SelectItem>
          {!hideConsolidated && (
            <SelectItem value={SCOPE_CONSOLIDATED}>Consolidated</SelectItem>
          )}
          <SelectGroup>
            <SelectLabel>By entity</SelectLabel>
            {entities.map((e) => (
              <SelectItem key={e.id} value={e.id}>
                {e.name}
                {e.is_primary ? " (primary)" : ""}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  );
}

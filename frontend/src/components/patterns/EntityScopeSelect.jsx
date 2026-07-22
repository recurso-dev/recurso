import { useQuery } from "@tanstack/react-query";
import { Building2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// Radix Select forbids an empty-string item value, so the primary/default entity
// is represented by a sentinel internally and mapped back to "" for the API
// (where a missing entity_id means the tenant's primary/default config).
const PRIMARY = "__primary__";

// EntityScopeSelect chooses which legal entity's tax config is being viewed or
// edited (Multi-Entity Books). `value` is the selected entity id, with "" meaning
// the tenant's primary/default config. It renders nothing for single-entity
// tenants — there is only one config to manage, so no scoping control is shown.
export function EntityScopeSelect({ value, onChange }) {
  const { data: entities = [] } = useQuery({
    queryKey: ["entities"],
    queryFn: async () => (await endpoints.getEntities()).data?.data || [],
  });

  if (entities.length <= 1) return null;

  const primary = entities.find((e) => e.is_primary);
  const others = entities.filter((e) => !e.is_primary);

  return (
    <div className="flex items-center gap-2">
      <Building2 className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
      <label htmlFor="entity-scope" className="text-sm text-muted-foreground">
        Legal entity
      </label>
      <Select
        value={value || PRIMARY}
        onValueChange={(v) => onChange(v === PRIMARY ? "" : v)}
      >
        <SelectTrigger id="entity-scope" className="w-[240px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={PRIMARY}>
            {primary ? `${primary.name} (primary)` : "Primary entity"}
          </SelectItem>
          {others.map((e) => (
            <SelectItem key={e.id} value={e.id}>
              {e.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";

import { endpoints } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";

// Action is "METHOD /v1/route/:template" — same split the Audit Log page uses.
function splitAction(action = "") {
  const [method, ...rest] = action.split(" ");
  return { method, path: rest.join(" ") };
}

const METHOD_VARIANTS = { POST: "success", PUT: "info", PATCH: "info", DELETE: "destructive" };

/**
 * AuditTrail — this object's slice of the immutable audit log
 * (GET /audit-logs?entity_type&entity_id), for object-page rails.
 *
 * Props:
 *  - entityType: plural resource segment as the middleware records it
 *    ("customers", "subscriptions", …)
 *  - entityId:   the object's uuid
 *  - limit:      max entries shown (default 8)
 */
export function AuditTrail({ entityType, entityId, limit = 8 }) {
  const { data: logs, isLoading, error } = useQuery({
    queryKey: ["auditTrail", entityType, entityId],
    queryFn: async () =>
      (
        await endpoints.getAuditLogs({
          entity_type: entityType,
          entity_id: entityId,
          limit,
        })
      ).data.data || [],
    enabled: Boolean(entityType && entityId),
  });

  if (isLoading) {
    return (
      <div className="space-y-3" aria-busy="true">
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-2/3" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    );
  }
  if (error) {
    return (
      <p className="text-sm text-muted-foreground" role="status">
        Couldn&apos;t load the audit trail.
      </p>
    );
  }
  if (!logs?.length) {
    return (
      <p className="text-sm text-muted-foreground">
        No configuration changes recorded for this object.
      </p>
    );
  }

  return (
    <div>
      <ol className="space-y-3">
        {logs.map((l) => {
          const { method, path } = splitAction(l.action);
          return (
            <li key={l.id} className="flex items-start gap-2.5">
              <Badge
                variant={METHOD_VARIANTS[method] || "neutral"}
                className="mt-0.5 shrink-0 font-mono text-[10px]"
              >
                {method}
              </Badge>
              <div className="min-w-0">
                <div className="truncate font-mono text-xs text-foreground" title={path}>
                  {path}
                </div>
                <div className="mt-0.5 text-xs text-muted-foreground">
                  {formatDateTime(l.created_at)}
                  {l.actor === "api_key" ? " · via API key" : ""}
                </div>
              </div>
            </li>
          );
        })}
      </ol>
      <Link
        to="/audit-log"
        className="mt-4 inline-block text-xs font-medium text-primary underline-offset-2 hover:underline"
      >
        View full audit log
      </Link>
    </div>
  );
}

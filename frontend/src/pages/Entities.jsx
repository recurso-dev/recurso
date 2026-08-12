import { useQuery } from "@tanstack/react-query";
import { Building2, TrendingUp, CircleDollarSign } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { docsUrlFor } from "@/lib/docsLinks";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Entities is the multi-entity control tower: each legal entity's recurring
// revenue and open receivables side by side. It's only meaningful for tenants
// with more than one entity; single-entity tenants get a pointer to the
// consolidated dashboards.
export default function Entities() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["entities-overview"],
    queryFn: async () => (await endpoints.getEntitiesOverview()).data?.data ?? null,
  });

  const rows = data?.entities ?? [];
  const cur = data?.reporting_currency || "USD";
  const money = (n) => formatCurrency(n || 0, cur);
  const loadError = isError
    ? error?.response?.data?.error?.message || "Failed to load the entities overview"
    : null;
  const multiEntity = rows.length > 1;

  return (
    <div>
      <PageHeader
        title="Entities"
        description="Recurring revenue and open receivables for each legal entity, in your reporting currency."
      />

      {loadError && (
        <p className="mb-4 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive" role="alert">
          {loadError} — refresh to retry.
        </p>
      )}

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !multiEntity ? (
        <Card>
          <CardContent className="py-10">
            <EmptyState
              icon={Building2}
              title="Single-entity account"
              description="This view compares multiple legal entities. Your account has one entity, so the consolidated dashboards (Overview, Invoice Aging, MRR) already show the full picture."
              learnMoreHref={docsUrlFor("/finance/entities")}
              learnMoreLabel="Set up multiple entities"
            />
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard label="Legal entities" value={rows.length.toLocaleString()} icon={Building2} />
            <StatCard label="Total MRR" value={money(data?.total_mrr)} icon={TrendingUp} />
            <StatCard label="Open receivables" value={money(data?.total_ar_outstanding)} icon={CircleDollarSign} />
          </div>

          <Card className="mt-6">
            <CardContent className="px-0 py-0">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/40 hover:bg-muted/40">
                      <TableHead className="pl-6">Entity</TableHead>
                      <TableHead className="text-right">MRR</TableHead>
                      <TableHead className="text-right">ARR</TableHead>
                      <TableHead className="text-right">Open AR</TableHead>
                      <TableHead className="pr-6 text-right">Active subs</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rows.map((e) => (
                      <TableRow key={e.entity_id} className="hover:bg-muted/20">
                        <TableCell className="pl-6">
                          <span className="flex items-center gap-2 font-medium text-foreground">
                            {e.entity_name || "Unnamed entity"}
                            {e.is_primary && (
                              <Badge variant="outline" className="text-[10px] uppercase">
                                primary
                              </Badge>
                            )}
                          </span>
                        </TableCell>
                        <TableCell className="text-right tabular-nums font-medium">{money(e.mrr)}</TableCell>
                        <TableCell className="text-right tabular-nums text-muted-foreground">{money(e.arr)}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          <span className={e.ar_outstanding > 0 ? "text-warning" : "text-muted-foreground"}>
                            {money(e.ar_outstanding)}
                          </span>
                        </TableCell>
                        <TableCell className="pr-6 text-right tabular-nums text-muted-foreground">
                          {(e.subscriptions || 0).toLocaleString()}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

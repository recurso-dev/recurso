import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { endpoints } from "@/lib/api";
import { formatCurrency } from "@/lib/utils";
import {
  getFounderToken,
  setFounderToken,
  clearFounderToken,
} from "@/lib/founderToken";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// Recurso Cloud — founder operator view. Cross-tenant, so it lives OUTSIDE the
// tenant dashboard and is gated by the FOUNDER_TOKEN (never the tenant session).
// Read-only: it shows the signup funnel and the money-free charge dry-run (what
// each tenant WOULD pay this month). Nothing here bills anyone.
export default function Platform() {
  const [hasToken, setHasToken] = useState(() => !!getFounderToken());
  const [tokenInput, setTokenInput] = useState("");

  const { data, error, isLoading, isFetching, refetch } = useQuery({
    queryKey: ["platform-metrics"],
    queryFn: () =>
      endpoints.platformMetrics(getFounderToken()).then((r) => r.data),
    enabled: hasToken,
    retry: false,
  });

  const status = error?.response?.status;
  const unauthorized = status === 401;
  const featureOff = status === 404;

  function submitToken(e) {
    e.preventDefault();
    const t = tokenInput.trim();
    if (!t) return;
    setFounderToken(t);
    setTokenInput("");
    setHasToken(true);
    refetch();
  }

  function signOut() {
    clearFounderToken();
    setHasToken(false);
  }

  // Token gate: no token yet, or the server rejected it.
  if (!hasToken || unauthorized) {
    return (
      <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-4 px-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Recurso Cloud — Operator</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Enter your founder token to view signups and the charge dry-run.
          </p>
        </div>
        {unauthorized && (
          <p className="text-sm text-destructive">
            That token was rejected. Check the FOUNDER_TOKEN set on the API.
          </p>
        )}
        <form onSubmit={submitToken} className="flex flex-col gap-3">
          <Input
            type="password"
            autoComplete="off"
            placeholder="Founder token"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            aria-label="Founder token"
          />
          <Button type="submit">View dashboard</Button>
        </form>
      </div>
    );
  }

  if (featureOff) {
    return (
      <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center gap-3 px-6 text-center">
        <h1 className="text-xl font-semibold">Operator view is off</h1>
        <p className="text-sm text-muted-foreground">
          Set <code className="font-mono">FOUNDER_TOKEN</code> on the API to enable
          the cross-tenant operator view.
        </p>
        <Button variant="outline" onClick={signOut}>
          Use a different token
        </Button>
      </div>
    );
  }

  const charges = data?.cloud_charges ?? [];
  const chargeCurrency = data?.cloud_charge_currency || "USD";

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <div className="mb-6 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Recurso Cloud — Operator</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Your customers (the tenants on Recurso Cloud) and their charge dry-run.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? "Refreshing…" : "Refresh"}
          </Button>
          <Button variant="ghost" onClick={signOut}>
            Sign out
          </Button>
        </div>
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : (
        <>
          {/* Funnel */}
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <Stat label="Tenants" value={data?.total_tenants ?? 0} />
            <Stat label="Signups (30d)" value={data?.signups_last_30d ?? 0} />
            <Stat label="Activated" value={data?.activated_tenants ?? 0} />
            <Stat label="Trials ending (7d)" value={data?.trials_expiring_7d ?? 0} />
          </div>

          {/* Charge dry-run */}
          <Card className="mt-8">
            <CardHeader className="flex flex-row items-center justify-between gap-4">
              <div>
                <CardTitle>Charge dry-run — this month</CardTitle>
                <p className="mt-1 text-sm text-muted-foreground">
                  What each tenant <strong>would</strong> be charged. Preview only —
                  nothing is billed.
                </p>
              </div>
              <div className="text-right">
                <div className="text-xs uppercase tracking-wide text-muted-foreground">
                  Would bill total
                </div>
                <div className="text-xl font-semibold tabular-nums">
                  {formatCurrency(data?.cloud_charge_total_minor ?? 0, chargeCurrency)}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {charges.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No charge preview yet. It appears after the daily usage meter runs
                  (needs <code className="font-mono">PLATFORM_TENANT_ID</code> set on
                  the API).
                </p>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Tenant</TableHead>
                        <TableHead className="text-right">Tracked revenue</TableHead>
                        <TableHead className="text-right">Collected volume</TableHead>
                        <TableHead className="text-right">Would charge</TableHead>
                        <TableHead>Why</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {charges.map((c) => (
                        <TableRow key={c.tenant_id}>
                          <TableCell>
                            <div className="font-medium">{c.name || "—"}</div>
                            <div className="text-xs text-muted-foreground">{c.email}</div>
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            {formatCurrency(c.tracked_revenue_minor, chargeCurrency)}
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            {formatCurrency(c.collected_volume_minor, chargeCurrency)}
                          </TableCell>
                          <TableCell className="text-right font-semibold tabular-nums">
                            {formatCurrency(c.would_charge_minor, chargeCurrency)}
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {c.reason}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function Stat({ label, value }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
        <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
      </CardContent>
    </Card>
  );
}

import { useMemo, useState } from "react";

import { endpoints } from "../../lib/api";
import { formatCurrency } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// Read-only pricing simulator (A1.6). Given a proposed charge payload (the same
// ChargeInput[] a save would send), let the user enter sample usage per metric
// and preview what the plan would bill — pre-tax, with a balanced GL projection.
// Nothing is persisted; this calls POST /plans/:id/simulate-charges.
export default function PricingSimulator({ planId, currency, chargesPayload, metricName }) {
  const metricIds = useMemo(
    () => [...new Set((chargesPayload || []).map((c) => c.metric_id).filter(Boolean))],
    [chargesPayload],
  );
  const [qty, setQty] = useState({}); // metric_id -> string
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [running, setRunning] = useState(false);

  const run = async () => {
    setRunning(true);
    setError(null);
    try {
      const usage = metricIds
        .filter((id) => qty[id] !== "" && qty[id] != null)
        .map((id) => ({ metric_id: id, quantity: Math.round(Number(qty[id]) || 0) }));
      const res = await endpoints.simulateCharges(planId, {
        currency,
        charges: chargesPayload,
        usage,
      });
      setResult(res.data.data);
    } catch (e) {
      setError(e?.response?.data?.error?.message || "Simulation failed");
      setResult(null);
    } finally {
      setRunning(false);
    }
  };

  if (metricIds.length === 0) return null;

  return (
    <div className="mt-4 rounded-lg border border-border bg-muted/30 p-3">
      <p className="text-sm font-semibold text-foreground">Simulate pricing</p>
      <p className="mb-3 mt-0.5 text-xs text-muted-foreground">
        Enter sample usage to preview what this charge set would bill (pre-tax). Nothing is
        saved.
      </p>

      <div className="flex flex-col gap-2">
        {metricIds.map((id) => (
          <label key={id} className="flex items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate text-foreground">{metricName[id] || id}</span>
            <Input
              type="number"
              min="0"
              inputMode="numeric"
              className="h-8 w-36"
              placeholder="sample quantity"
              value={qty[id] ?? ""}
              onChange={(e) => setQty((p) => ({ ...p, [id]: e.target.value }))}
            />
          </label>
        ))}
      </div>

      <div className="mt-3">
        <Button size="sm" onClick={run} disabled={running}>
          {running ? "Simulating…" : "Run simulation"}
        </Button>
      </div>

      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}

      {result && (
        <div className="mt-3 flex flex-col gap-3">
          <div className="overflow-hidden rounded-md border border-border bg-background">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/50 text-xs text-muted-foreground">
                  <th className="px-3 py-1.5 text-left font-medium">Metric</th>
                  <th className="px-3 py-1.5 text-left font-medium">Model</th>
                  <th className="px-3 py-1.5 text-right font-medium">Quantity</th>
                  <th className="px-3 py-1.5 text-right font-medium">Amount</th>
                </tr>
              </thead>
              <tbody>
                {(result.charges || []).map((c) => (
                  <tr key={c.metric_id} className="border-b border-border last:border-0">
                    <td className="px-3 py-1.5 text-foreground">{c.metric_name || c.metric_code}</td>
                    <td className="px-3 py-1.5 font-mono text-xs text-muted-foreground">
                      {c.charge_model}
                    </td>
                    <td className="px-3 py-1.5 text-right tabular-nums">{c.quantity}</td>
                    <td className="px-3 py-1.5 text-right tabular-nums text-foreground">
                      {formatCurrency(c.amount, result.currency)}
                    </td>
                  </tr>
                ))}
                <tr className="bg-muted/40 font-medium">
                  <td className="px-3 py-1.5" colSpan={3}>
                    Subtotal (pre-tax)
                  </td>
                  <td className="px-3 py-1.5 text-right tabular-nums text-foreground">
                    {formatCurrency(result.subtotal, result.currency)}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          {result.gl_preview?.length > 0 && (
            <div>
              <div className="mb-1 flex items-center gap-2">
                <p className="text-xs font-medium text-muted-foreground">GL preview</p>
                {result.balanced && (
                  <Badge variant="success" className="text-[10px]">
                    balanced
                  </Badge>
                )}
              </div>
              <div className="overflow-hidden rounded-md border border-border bg-background">
                <table className="w-full text-xs">
                  <tbody>
                    {result.gl_preview.map((g, i) => (
                      <tr key={i} className="border-b border-border last:border-0">
                        <td className="px-3 py-1 text-muted-foreground">{g.account_name}</td>
                        <td className="px-3 py-1 text-right tabular-nums">
                          {g.debit ? formatCurrency(g.debit, result.currency) : ""}
                        </td>
                        <td className="px-3 py-1 text-right tabular-nums">
                          {g.credit ? formatCurrency(g.credit, result.currency) : ""}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {result.note && <p className="text-xs text-muted-foreground">{result.note}</p>}
        </div>
      )}
    </div>
  );
}

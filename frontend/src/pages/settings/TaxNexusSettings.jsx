import { useEffect, useState } from "react";
import { Plus, Save, X, MapPinned } from "lucide-react";

import { endpoints as api } from "../../lib/api";
import { toast } from "@/components/ui/sonner";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const NEXUS_TYPES = ["physical", "voluntary", "economic"];

// US sales-tax nexus: declare where you must collect, and watch economic
// thresholds per state (crossings auto-establish nexus server-side).
export default function TaxNexusSettings() {
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState(null);
  const [statusLoading, setStatusLoading] = useState(true);
  const [statusError, setStatusError] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const res = await api.getTaxNexus();
      setRows(
        (res.data.data || []).map((n) => ({
          state_code: n.state_code,
          nexus_type: n.nexus_type || "physical",
        }))
      );
    } catch {
      setRows([]);
    } finally {
      setLoading(false);
    }
    setStatusLoading(true);
    setStatusError(null);
    try {
      const res = await api.getTaxNexusStatus();
      setStatus(res.data.data);
    } catch (err) {
      setStatusError(
        err?.response?.status === 503
          ? "Economic-nexus tracking isn't available on this deployment."
          : err?.response?.data?.error?.message || "Failed to load nexus status"
      );
    } finally {
      setStatusLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const save = async () => {
    setSaving(true);
    try {
      await api.setTaxNexus(rows.filter((r) => r.state_code.trim()));
      toast.success("Nexus states saved.");
      load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to save nexus states");
    } finally {
      setSaving(false);
    }
  };

  const setRow = (i, patch) =>
    setRows((prev) => prev.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader
        title="US sales-tax nexus"
        description="Declare the states where you collect sales tax, and monitor economic-nexus thresholds."
        actions={
          <Button onClick={save} disabled={saving || loading}>
            <Save className="h-4 w-4" />
            {saving ? "Saving…" : "Save states"}
          </Button>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Declared nexus states</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : (
            <>
              {rows.length === 0 && (
                <p className="text-sm text-muted-foreground">
                  No states declared. Saving an empty list clears all declared nexus.
                </p>
              )}
              {rows.map((r, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    value={r.state_code}
                    onChange={(e) => setRow(i, { state_code: e.target.value.toUpperCase() })}
                    placeholder="CA"
                    maxLength={2}
                    className="w-20 font-mono uppercase"
                    aria-label={`State code ${i + 1}`}
                  />
                  <select
                    value={r.nexus_type}
                    onChange={(e) => setRow(i, { nexus_type: e.target.value })}
                    aria-label={`Nexus type ${i + 1}`}
                    className="h-9 rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {NEXUS_TYPES.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => setRows((prev) => prev.filter((_, idx) => idx !== i))}
                    aria-label={`Remove state ${i + 1}`}
                    className="text-stone-400 transition-colors hover:text-red-500"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setRows((prev) => [...prev, { state_code: "", nexus_type: "physical" }])}
              >
                <Plus className="h-4 w-4" />
                Add state
              </Button>
            </>
          )}
        </CardContent>
      </Card>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">
            Economic-nexus status {status?.year ? `(${status.year})` : ""}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {statusLoading ? (
            <p className="px-6 pb-6 text-sm text-muted-foreground">Loading…</p>
          ) : statusError ? (
            <p className="px-6 pb-6 text-sm text-muted-foreground">{statusError}</p>
          ) : !status?.states?.length ? (
            <div className="px-6 pb-6 pt-2 text-center text-sm text-muted-foreground">
              <MapPinned className="mx-auto mb-2 h-6 w-6 text-stone-300" />
              No state activity tracked yet this year.
            </div>
          ) : (
            <>
              {status.dataset_certified === false && (
                <p className="mx-6 mb-3 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-800">
                  Threshold dataset has not passed professional review — treat proximity
                  figures as indicative, not advice.
                </p>
              )}
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>State</TableHead>
                    <TableHead>Nexus</TableHead>
                    <TableHead>YTD taxable sales</TableHead>
                    <TableHead>YTD transactions</TableHead>
                    <TableHead className="text-right">Threshold proximity</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {status.states.map((s) => (
                    <TableRow key={s.state_code}>
                      <TableCell className="font-mono">{s.state_code}</TableCell>
                      <TableCell>
                        {s.has_nexus || s.nexus_type ? (
                          <Badge variant={s.crossed ? "destructive" : "success"}>
                            {s.nexus_type || "established"}
                          </Badge>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="tabular-nums">
                        {s.ytd_sales != null ? formatCurrency(s.ytd_sales, "USD") : "—"}
                      </TableCell>
                      <TableCell className="tabular-nums">{s.ytd_transactions ?? "—"}</TableCell>
                      <TableCell className="text-right tabular-nums">
                        {s.threshold_proximity != null
                          ? `${Math.round(s.threshold_proximity * 100)}%`
                          : "—"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

import { useState } from "react";
import { Download, FileSpreadsheet } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

const now = new Date();
// Default to the previous month — the period being filed.
const defaultPeriod = () => {
  const d = new Date(now.getFullYear(), now.getMonth() - 1, 1);
  return { month: d.getMonth() + 1, year: d.getFullYear() };
};

const downloadJSON = (obj, filename) => {
  const blob = new Blob([JSON.stringify(obj, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 10_000);
};

// One return's panel: fetch on demand, show the readable JSON, download either form.
function ReturnPanel({ kind, month, year }) {
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchReturn = async () => {
    setLoading(true);
    setResult(null);
    try {
      const res =
        kind === "gstr1" ? await api.getGSTR1(month, year) : await api.getGSTR3B(month, year);
      setResult(res.data);
    } catch (err) {
      toast.error(
        err?.response?.status === 503
          ? "GSTR export isn't configured on this deployment."
          : err?.response?.data?.error?.message || "Failed to build the return"
      );
    } finally {
      setLoading(false);
    }
  };

  const label = kind === "gstr1" ? "GSTR-1" : "GSTR-3B";
  const period = `${String(month).padStart(2, "0")}-${year}`;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base">
          {label} · {period}
        </CardTitle>
        <div className="flex gap-2">
          <Button size="sm" onClick={fetchReturn} disabled={loading}>
            {loading ? "Building…" : "Build return"}
          </Button>
          {result && (
            <>
              <Button
                size="sm"
                variant="outline"
                onClick={() => downloadJSON(result.gov_schema, `${kind}-${period}-gstn.json`)}
              >
                <Download className="h-4 w-4" />
                GSTN JSON
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => downloadJSON(result.data, `${kind}-${period}-readable.json`)}
              >
                <Download className="h-4 w-4" />
                Readable
              </Button>
            </>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {result ? (
          <pre className="max-h-96 overflow-auto rounded-md bg-muted p-4 font-mono text-xs text-foreground">
            {JSON.stringify(result.data, null, 2)}
          </pre>
        ) : (
          <p className="text-sm text-muted-foreground">
            Build the return to preview its sections and download the GSTN upload file.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

// India GST returns: GSTR-1 (outward supplies) and GSTR-3B (summary), with
// the exact JSON the GSTN portal accepts for upload.
const GSTReturns = () => {
  const [{ month, year }, setPeriod] = useState(defaultPeriod());

  return (
    <div>
      <PageHeader
        title="GST returns"
        description="Return-ready GSTR-1 and GSTR-3B for a tax period, exportable as GSTN upload JSON."
      />

      <div className="mb-6 flex items-end gap-3">
        <div>
          <Label>Month</Label>
          <Input
            type="number"
            min="1"
            max="12"
            value={month}
            onChange={(e) => setPeriod((p) => ({ ...p, month: Number(e.target.value) }))}
            className="w-24"
          />
        </div>
        <div>
          <Label>Year</Label>
          <Input
            type="number"
            min="2017"
            max="2100"
            value={year}
            onChange={(e) => setPeriod((p) => ({ ...p, year: Number(e.target.value) }))}
            className="w-28"
          />
        </div>
        <FileSpreadsheet className="mb-2 h-5 w-5 text-stone-300" />
      </div>

      <Tabs defaultValue="gstr1">
        <TabsList>
          <TabsTrigger value="gstr1">GSTR-1</TabsTrigger>
          <TabsTrigger value="gstr3b">GSTR-3B</TabsTrigger>
        </TabsList>
        <TabsContent value="gstr1" className="mt-6">
          <ReturnPanel kind="gstr1" month={month} year={year} />
        </TabsContent>
        <TabsContent value="gstr3b" className="mt-6">
          <ReturnPanel kind="gstr3b" month={month} year={year} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default GSTReturns;

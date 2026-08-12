import { useState } from "react";
import { Download, FileSpreadsheet } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Money } from "@/components/ui/money";
import { formatDate } from "@/lib/utils";
import { ReportScopeSelect } from "@/components/patterns/ReportScopeSelect";
import { SCOPE_ALL, scopeToParams } from "@/components/patterns/reportScope";

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

// GST is filed in rupees; every amount in the return payload is paise.
const Amount = ({ v }) => (
  <TableCell className="text-right">
    <Money amountMinor={v || 0} currency="INR" />
  </TableCell>
);

const TaxHeads = () => (
  <>
    <TableHead className="text-right">Taxable value</TableHead>
    <TableHead className="text-right">IGST</TableHead>
    <TableHead className="text-right">CGST</TableHead>
    <TableHead className="text-right">SGST</TableHead>
  </>
);

function Section({ title, hint, children }) {
  return (
    <section className="mt-6 first:mt-0">
      <h4 className="text-sm font-semibold text-foreground">{title}</h4>
      {hint && <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p>}
      <div className="mt-2">{children}</div>
    </section>
  );
}

const EmptySection = ({ what }) => (
  <p className="text-sm text-muted-foreground">No {what} in this period.</p>
);

// Long detail tables scroll inside the card instead of growing the page.
const Scroll = ({ children }) => (
  <div className="max-h-96 overflow-auto rounded-md border">{children}</div>
);

function Gstr1Summary({ data }) {
  const b2b = (data.b2b || []).flatMap((g) =>
    (g.invoices || []).map((inv) => ({ gstin: g.gstin, ...inv }))
  );
  const b2cs = data.b2cs || [];
  const cdnr = (data.cdnr || []).flatMap((g) =>
    (g.notes || []).map((n) => ({ gstin: g.gstin, ...n }))
  );
  const hsn = data.hsn_summary || [];

  return (
    <div>
      <Section title="Control totals">
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Section</TableHead>
                <TableHead className="text-right">Documents</TableHead>
                <TaxHeads />
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>Outward supplies</TableCell>
                <TableCell className="text-right">{data.invoice_count || 0}</TableCell>
                <Amount v={data.total_taxable_value} />
                <Amount v={data.total_igst} />
                <Amount v={data.total_cgst} />
                <Amount v={data.total_sgst} />
              </TableRow>
              <TableRow>
                <TableCell>Credit notes (CDNR)</TableCell>
                <TableCell className="text-right">{data.credit_note_count || 0}</TableCell>
                <Amount v={data.total_credit_taxable_value} />
                <Amount v={data.total_credit_igst} />
                <Amount v={data.total_credit_cgst} />
                <Amount v={data.total_credit_sgst} />
              </TableRow>
            </TableBody>
          </Table>
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground">
          GSTR-1 reports credit notes in their own section, so outward totals are gross,
          not net of notes.
        </p>
      </Section>

      <Section title="B2B — supplies to registered buyers" hint="Reported invoice by invoice under each buyer's GSTIN.">
        {b2b.length === 0 ? (
          <EmptySection what="B2B supplies" />
        ) : (
          <Scroll>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Buyer GSTIN</TableHead>
                  <TableHead>Invoice</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead>Place of supply</TableHead>
                  <TableHead className="text-right">Rate</TableHead>
                  <TaxHeads />
                </TableRow>
              </TableHeader>
              <TableBody>
                {b2b.map((r) => (
                  <TableRow key={`${r.gstin}-${r.invoice_number}`}>
                    <TableCell className="font-mono text-xs">{r.gstin}</TableCell>
                    <TableCell>{r.invoice_number}</TableCell>
                    <TableCell>{formatDate(r.date)}</TableCell>
                    <TableCell>{r.place_of_supply || "—"}</TableCell>
                    <TableCell className="text-right">{r.rate}%</TableCell>
                    <Amount v={r.taxable_value} />
                    <Amount v={r.igst} />
                    <Amount v={r.cgst} />
                    <Amount v={r.sgst} />
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Scroll>
        )}
      </Section>

      <Section title="B2CS — supplies to unregistered buyers" hint="Summarized per place of supply and rate, as GSTR-1 requires.">
        {b2cs.length === 0 ? (
          <EmptySection what="B2C supplies" />
        ) : (
          <Scroll>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Place of supply</TableHead>
                  <TableHead className="text-right">Rate</TableHead>
                  <TaxHeads />
                </TableRow>
              </TableHeader>
              <TableBody>
                {b2cs.map((r) => (
                  <TableRow key={`${r.place_of_supply}-${r.rate}`}>
                    <TableCell>{r.place_of_supply || "—"}</TableCell>
                    <TableCell className="text-right">{r.rate}%</TableCell>
                    <Amount v={r.taxable_value} />
                    <Amount v={r.igst} />
                    <Amount v={r.cgst} />
                    <Amount v={r.sgst} />
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Scroll>
        )}
      </Section>

      <Section title="CDNR — credit notes to registered buyers">
        {cdnr.length === 0 ? (
          <EmptySection what="credit notes" />
        ) : (
          <Scroll>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Buyer GSTIN</TableHead>
                  <TableHead>Note</TableHead>
                  <TableHead>Against invoice</TableHead>
                  <TableHead>Date</TableHead>
                  <TaxHeads />
                </TableRow>
              </TableHeader>
              <TableBody>
                {cdnr.map((r) => (
                  <TableRow key={`${r.gstin}-${r.note_number}`}>
                    <TableCell className="font-mono text-xs">{r.gstin}</TableCell>
                    <TableCell>{r.note_number}</TableCell>
                    <TableCell>{r.original_invoice_number || "—"}</TableCell>
                    <TableCell>{formatDate(r.date)}</TableCell>
                    <Amount v={r.taxable_value} />
                    <Amount v={r.igst} />
                    <Amount v={r.cgst} />
                    <Amount v={r.sgst} />
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Scroll>
        )}
      </Section>

      <Section title="HSN summary">
        {hsn.length === 0 ? (
          <EmptySection what="HSN lines" />
        ) : (
          <Scroll>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>HSN code</TableHead>
                  <TableHead className="text-right">Invoices</TableHead>
                  <TaxHeads />
                </TableRow>
              </TableHeader>
              <TableBody>
                {hsn.map((r) => (
                  <TableRow key={r.hsn_code}>
                    <TableCell className="font-mono text-xs">{r.hsn_code}</TableCell>
                    <TableCell className="text-right">{r.invoice_count}</TableCell>
                    <Amount v={r.taxable_value} />
                    <Amount v={r.igst} />
                    <Amount v={r.cgst} />
                    <Amount v={r.sgst} />
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Scroll>
        )}
      </Section>
    </div>
  );
}

const T31_ROWS = [
  { key: "outward_taxable", label: "3.1(a) Outward taxable supplies (net of credit notes)" },
  { key: "zero_rated", label: "3.1(b) Zero-rated supplies" },
  { key: "nil_exempt", label: "3.1(c) Nil-rated and exempt supplies" },
  { key: "inward_reverse_charge", label: "3.1(d) Inward supplies under reverse charge" },
  { key: "non_gst", label: "3.1(e) Non-GST outward supplies" },
];

function Gstr3bSummary({ data }) {
  const interState = data.inter_state_unregistered || [];

  return (
    <div>
      <Section title="Table 3.1 — outward supplies and reverse charge">
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nature of supply</TableHead>
                <TaxHeads />
              </TableRow>
            </TableHeader>
            <TableBody>
              {T31_ROWS.map(({ key, label }) => {
                const v = data[key] || {};
                return (
                  <TableRow key={key}>
                    <TableCell>{label}</TableCell>
                    <Amount v={v.taxable_value} />
                    <Amount v={v.igst} />
                    <Amount v={v.cgst} />
                    <Amount v={v.sgst} />
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground">
          Sections 3.1(b)–(e) are reported as zero: Recurso holds sales-side billing data
          only, and input tax credit (Table 4) is out of scope. Review purchase-side
          figures with your accountant before filing.
        </p>
      </Section>

      <Section
        title="Table 3.2 — inter-state supplies to unregistered persons"
        hint="A subset of 3.1(a), reported per place of supply. Only IGST applies inter-state."
      >
        {interState.length === 0 ? (
          <EmptySection what="inter-state unregistered supplies" />
        ) : (
          <Scroll>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Place of supply</TableHead>
                  <TableHead className="text-right">Taxable value</TableHead>
                  <TableHead className="text-right">IGST</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {interState.map((r) => (
                  <TableRow key={r.place_of_supply}>
                    <TableCell>{r.place_of_supply || "—"}</TableCell>
                    <Amount v={r.taxable_value} />
                    <Amount v={r.igst} />
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Scroll>
        )}
      </Section>

      <p className="mt-6 text-xs text-muted-foreground">
        Built from {data.invoice_count || 0} invoice{data.invoice_count === 1 ? "" : "s"} and{" "}
        {data.credit_note_count || 0} credit note{data.credit_note_count === 1 ? "" : "s"} in
        the period.
      </p>
    </div>
  );
}

// One return's panel: fetch on demand, render the sections a filer reviews,
// and download either the GSTN upload JSON or the readable form.
function ReturnPanel({ kind, month, year, scopeParams }) {
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchReturn = async () => {
    setLoading(true);
    setResult(null);
    setError(null);
    try {
      const res =
        kind === "gstr1"
          ? await api.getGSTR1(month, year, scopeParams)
          : await api.getGSTR3B(month, year, scopeParams);
      setResult(res.data);
    } catch (err) {
      setError(
        err?.response?.status === 503
          ? "GSTR export isn't configured on this deployment."
          : err?.response?.data?.error?.message || "We couldn't build the return. Try again."
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
          <Button size="sm" onClick={fetchReturn} loading={loading}>
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
        {error ? (
          <ErrorState title={`Couldn't build ${label}`} message={error} onRetry={fetchReturn} />
        ) : result ? (
          kind === "gstr1" ? (
            <Gstr1Summary data={result.data || {}} />
          ) : (
            <Gstr3bSummary data={result.data || {}} />
          )
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
  // GSTR is filed per GSTIN, so a multi-entity tenant picks which entity to file
  // for. Single-entity tenants see no selector and get the whole tenant (SCOPE_ALL).
  const [scope, setScope] = useState(SCOPE_ALL);
  const scopeParams = scopeToParams(scope);

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
        <div>
          <ReportScopeSelect value={scope} onChange={setScope} hideConsolidated />
        </div>
        <FileSpreadsheet className="mb-2 h-5 w-5 text-subtle/60" />
      </div>

      <Tabs defaultValue="gstr1">
        <TabsList>
          <TabsTrigger value="gstr1">GSTR-1</TabsTrigger>
          <TabsTrigger value="gstr3b">GSTR-3B</TabsTrigger>
        </TabsList>
        <TabsContent value="gstr1" className="mt-6">
          <ReturnPanel kind="gstr1" month={month} year={year} scopeParams={scopeParams} />
        </TabsContent>
        <TabsContent value="gstr3b" className="mt-6">
          <ReturnPanel kind="gstr3b" month={month} year={year} scopeParams={scopeParams} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default GSTReturns;

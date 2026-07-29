import { useMemo, useState } from "react";
import { Link } from "react-router";
import {
  ArrowDownToLine,
  UploadCloud,
  FileJson,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  ArrowLeft,
} from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

// Source registry: the wizard is source-agnostic — each entry supplies the
// endpoints, the id field the plan uses, and source-specific copy.
const SOURCES = {
  stripe: {
    label: "Stripe",
    preview: endpoints.stripeImportPreview,
    commit: endpoints.stripeImportCommit,
    idField: "stripe_id",
    blurb:
      "Export your Stripe data as JSON (customers, products, prices, subscriptions). Subscriptions import in their current billing state — no re-billing. Card payment methods can't be migrated from an export and are skipped.",
    placeholder: '{"customers":[…],"products":[…],"prices":[…]}',
  },
  chargebee: {
    label: "Chargebee",
    preview: endpoints.chargebeeImportPreview,
    commit: endpoints.chargebeeImportCommit,
    idField: "chargebee_id",
    blurb:
      "Export your Chargebee data as JSON (customers, plans, subscriptions). Subscriptions import in their current billing state — no re-billing.",
    placeholder: '{"customers":[…],"plans":[…],"subscriptions":[…]}',
  },
};

// Semantic colour per plan action (kept separate from the brand accent).
const ACTION_STYLE = {
  create: { label: "Create", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  link_existing: { label: "Link existing", cls: "bg-sky-50 text-sky-700 border-sky-200" },
  skip_already_imported: { label: "Already imported", cls: "bg-stone-100 text-stone-600 border-stone-200" },
  conflict: { label: "Conflict", cls: "bg-red-50 text-red-700 border-red-200" },
  unsupported: { label: "Skipped", cls: "bg-amber-50 text-amber-700 border-amber-200" },
};

function ActionBadge({ action }) {
  const s = ACTION_STYLE[action] || { label: action, cls: "bg-stone-100 text-stone-600 border-stone-200" };
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${s.cls}`}>
      {s.label}
    </span>
  );
}

function actionTotal(summary, action) {
  return Object.entries(summary || {}).reduce(
    (n, [key, count]) => (key.endsWith(`.${action}`) ? n + count : n),
    0
  );
}

export default function ImportData() {
  const [source, setSource] = useState(null); // null → source picker
  const [step, setStep] = useState("source"); // source | input | preview | done
  const [raw, setRaw] = useState("");
  const [fileName, setFileName] = useState("");
  const [exportData, setExportData] = useState(null);
  const [plan, setPlan] = useState(null);
  const [result, setResult] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(null);

  const cfg = source ? SOURCES[source] : null;

  const summaryStats = useMemo(() => {
    if (!plan) return [];
    return ["create", "link_existing", "skip_already_imported", "conflict", "unsupported"]
      .map((action) => ({ action, n: actionTotal(plan.summary, action) }))
      .filter((s) => s.n > 0);
  }, [plan]);

  const onFile = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = () => setRaw(String(reader.result || ""));
    reader.readAsText(file);
  };

  const pickSource = (key) => {
    setSource(key);
    setStep("input");
  };

  const runPreview = async () => {
    setError(null);
    let parsed;
    try {
      parsed = JSON.parse(raw);
    } catch {
      setError(`That doesn't look like valid JSON. Export your ${cfg.label} data as JSON and try again.`);
      return;
    }
    setBusy(true);
    try {
      const res = await cfg.preview(parsed);
      setExportData(parsed);
      setPlan(res.data);
      setStep("preview");
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Preview failed. Check the export and try again.");
    } finally {
      setBusy(false);
    }
  };

  const runCommit = async () => {
    setBusy(true);
    try {
      const res = await cfg.commit(exportData);
      setResult(res.data);
      setStep("done");
      toast.success("Import complete.");
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Import failed. Please try again.");
    } finally {
      setBusy(false);
    }
  };

  const reset = () => {
    setSource(null);
    setStep("source");
    setRaw("");
    setFileName("");
    setExportData(null);
    setPlan(null);
    setResult(null);
    setError(null);
  };

  const createTotal = plan ? actionTotal(plan.summary, "create") : 0;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Import data"
        description="Migrate your customers, plans, and subscriptions from another billing system. Preview everything before anything is written."
      />

      {/* Step 0 — source */}
      {step === "source" && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Where are you migrating from?</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2">
              {Object.entries(SOURCES).map(([key, s]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => pickSource(key)}
                  className="flex flex-col items-start gap-2 rounded-lg border border-border bg-white p-5 text-left transition-colors hover:border-primary hover:shadow-sm"
                >
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <ArrowDownToLine className="h-4 w-4" />
                  </span>
                  <span className="font-semibold text-foreground">{s.label}</span>
                  <span className="text-xs text-muted-foreground">
                    Customers, plans{key === "stripe" ? "/prices" : ""}, and subscriptions.
                  </span>
                </button>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 1 — upload */}
      {step === "input" && cfg && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ArrowDownToLine className="h-4 w-4 text-primary" />
              Upload your {cfg.label} export
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            <p className="text-sm text-muted-foreground">{cfg.blurb}</p>

            <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-stone-50 px-4 py-8 text-center transition-colors hover:border-stone-300">
              <UploadCloud className="h-7 w-7 text-stone-400" />
              <span className="text-sm font-medium text-foreground">{fileName || "Choose a JSON file"}</span>
              <span className="text-xs text-muted-foreground">or drag it onto this box</span>
              <input
                type="file"
                accept="application/json,.json"
                className="sr-only"
                onChange={onFile}
                aria-label={`${cfg.label} export JSON file`}
              />
            </label>

            <div>
              <label htmlFor="paste" className="mb-1.5 block text-xs font-medium text-muted-foreground">
                …or paste the JSON
              </label>
              <textarea
                id="paste"
                value={raw}
                onChange={(e) => setRaw(e.target.value)}
                rows={6}
                spellCheck={false}
                placeholder={cfg.placeholder}
                className="w-full rounded-md border border-border bg-white px-3 py-2 font-mono text-xs text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            {error && (
              <p className="flex items-start gap-2 text-sm font-medium text-red-600" role="alert">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                {error}
              </p>
            )}

            <div className="flex items-center justify-between">
              <Button variant="outline" onClick={reset}>
                <ArrowLeft className="h-4 w-4" />
                Change source
              </Button>
              <Button onClick={runPreview} disabled={busy || raw.trim() === ""}>
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <FileJson className="h-4 w-4" />}
                {busy ? "Previewing…" : "Preview import"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step 2 — preview */}
      {step === "preview" && plan && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Preview — nothing has been imported yet</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                {summaryStats.map((s) => (
                  <span key={s.action} className="inline-flex items-center gap-1.5 text-sm">
                    <ActionBadge action={s.action} />
                    <span className="font-semibold tabular-nums">{s.n}</span>
                  </span>
                ))}
              </div>

              {plan.warnings?.length > 0 && (
                <div className="space-y-1 rounded-md border border-amber-200 bg-amber-50 p-3">
                  {plan.warnings.map((w, i) => (
                    <p key={i} className="flex items-start gap-2 text-xs text-amber-900">
                      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                      {w}
                    </p>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-0">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Type</TableHead>
                      <TableHead>Item</TableHead>
                      <TableHead>Action</TableHead>
                      <TableHead>Detail</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {plan.items.map((it, i) => (
                      <TableRow key={`${it[cfg.idField] || it.stripe_id || it.chargebee_id}-${i}`}>
                        <TableCell className="capitalize text-muted-foreground">{it.kind.replace("_", " ")}</TableCell>
                        <TableCell className="font-medium">{it.label}</TableCell>
                        <TableCell><ActionBadge action={it.action} /></TableCell>
                        <TableCell className="text-xs text-muted-foreground">{it.detail}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>

          <div className="flex items-center justify-between">
            <Button variant="outline" onClick={() => setStep("input")}>
              <ArrowLeft className="h-4 w-4" />
              Back
            </Button>
            <Button onClick={runCommit} disabled={busy || createTotal === 0}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <ArrowDownToLine className="h-4 w-4" />}
              {busy
                ? "Importing…"
                : createTotal === 0
                  ? "Nothing to import"
                  : `Import ${createTotal} item${createTotal === 1 ? "" : "s"}`}
            </Button>
          </div>
        </div>
      )}

      {/* Step 3 — done */}
      {step === "done" && result && (
        <Card>
          <CardContent className="space-y-5 p-6">
            <div className="flex flex-col items-center text-center">
              <div className="mb-3 flex h-11 w-11 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
                <CheckCircle2 className="h-6 w-6" />
              </div>
              <p className="text-sm font-medium text-foreground">Import complete</p>
            </div>

            <div className="flex flex-wrap justify-center gap-4">
              {Object.entries(result.created || {}).map(([kind, n]) => (
                <div key={kind} className="rounded-lg border border-border px-4 py-3 text-center">
                  <div className="text-2xl font-semibold tabular-nums text-foreground">{n}</div>
                  <div className="text-xs capitalize text-muted-foreground">{kind.replace("_", " ")}s created</div>
                </div>
              ))}
              {(!result.created || Object.keys(result.created).length === 0) && (
                <p className="text-sm text-muted-foreground">Nothing new to create — everything was already imported.</p>
              )}
            </div>

            {result.failures?.length > 0 && (
              <div className="space-y-1 rounded-md border border-amber-200 bg-amber-50 p-3">
                <p className="text-xs font-semibold text-amber-900">{result.failures.length} item(s) couldn't be imported:</p>
                {result.failures.map((f, i) => (
                  <p key={i} className="text-xs text-amber-900">
                    <span className="font-medium capitalize">{f.kind}</span> {f.stripe_id}: {f.error}
                  </p>
                ))}
              </div>
            )}

            <div className="flex justify-center gap-3">
              <Button variant="outline" onClick={reset}>Import more</Button>
              <Button asChild>
                <Link to="/customers">View customers</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

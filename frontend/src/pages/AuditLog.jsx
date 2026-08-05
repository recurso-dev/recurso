import { useEffect, useState } from "react";
import { ShieldCheck, Copy } from "lucide-react";
import { toast } from "sonner";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

const PAGE_SIZE = 100;

// Actions are stored as "METHOD /v1/route/:id". Split so the verb can be a
// colored badge and the route stays scannable.
const splitAction = (action) => {
  const [method, ...rest] = String(action || "").split(" ");
  return { method, path: rest.join(" ") };
};

const METHOD_VARIANT = {
  POST: "success",
  PUT: "info",
  PATCH: "info",
  DELETE: "destructive",
};
const methodVariant = (m) => METHOD_VARIANT[m] || "neutral";

const fmtWhen = (x) =>
  new Date(x).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "medium" });

const copyText = async (v, label) => {
  try {
    await navigator.clipboard.writeText(v);
    toast.success(`${label} copied.`);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
};

// A copyable, monospace id — the workhorse of a "referenceable" audit trail.
const CopyableId = ({ value, label }) => (
  <button
    type="button"
    onClick={() => copyText(value, label)}
    className="group inline-flex max-w-full items-center gap-1.5 text-left"
    title={`Copy ${label.toLowerCase()}`}
  >
    <span className="truncate font-mono text-xs">{value}</span>
    <Copy className="h-3 w-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
  </button>
);

const Field = ({ label, children, span }) => (
  <div className={span ? "col-span-2" : undefined}>
    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
      {label}
    </dt>
    <dd className="mt-0.5 text-sm text-foreground">{children}</dd>
  </div>
);

// Pretty-print a stored request body if it's JSON; otherwise show it verbatim.
const prettyBody = (body) => {
  if (!body) return null;
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
};

// Append-only audit trail (Lago-parity C2): every successful config-grade
// mutation, immutable at the database level. Click a row to inspect the full
// actor, IDs, and the exact request payload that changed.
const AuditLog = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [entityFilter, setEntityFilter] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(0); // 0-based
  const [selected, setSelected] = useState(null);

  const fetchLogs = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = { limit: PAGE_SIZE, offset: page * PAGE_SIZE };
      if (entityFilter) params.entity_type = entityFilter;
      if (from) params.from = new Date(`${from}T00:00:00`).toISOString();
      if (to) params.to = new Date(`${to}T23:59:59`).toISOString();
      const res = await api.getAuditLogs(params);
      setLogs(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || err?.message || "Failed to load audit trail");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [entityFilter, from, to, page]);

  // Reset to the first page whenever a filter changes.
  useEffect(() => {
    setPage(0);
  }, [entityFilter, from, to]);

  const entityTypes = [...new Set(logs.map((l) => l.entity_type))].sort();

  const columns = [
    {
      key: "when",
      header: "When",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (l) => (
        <span className="whitespace-nowrap text-xs text-muted-foreground">{fmtWhen(l.created_at)}</span>
      ),
    },
    {
      key: "actor",
      header: "Actor",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (l) =>
        l.actor === "api_key" ? (
          <Badge variant="neutral">API key</Badge>
        ) : (
          <span className="font-mono text-xs">{String(l.actor).slice(0, 8)}…</span>
        ),
    },
    {
      // Flexible column: the verb + route absorbs the remaining width so the
      // table reads as a full record, not a few stranded cells.
      key: "action",
      header: "Action",
      className: "w-full",
      headerClassName: "w-full",
      cell: (l) => {
        const { method, path } = splitAction(l.action);
        return (
          <span className="flex items-center gap-2">
            <Badge variant={methodVariant(method)} className="font-mono text-[10px]">
              {method}
            </Badge>
            <span className="truncate font-mono text-xs text-foreground">{path}</span>
          </span>
        );
      },
    },
    {
      key: "entity",
      header: "Entity",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (l) => (
        <span className="text-xs text-muted-foreground">
          {l.entity_type}
          {l.entity_id ? ` · ${l.entity_id.slice(0, 8)}…` : ""}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (l) => <Badge variant={l.status < 300 ? "success" : "destructive"}>{l.status}</Badge>,
    },
  ];

  const filtered = Boolean(entityFilter || from || to);
  const body = selected ? prettyBody(selected.request_body) : null;

  return (
    <div>
      <PageHeader
        title="Audit Log"
        description="Every configuration change, immutably recorded. Updates and deletes are rejected at the database. Click a row to see who changed what."
      />

      <DataTable
        columns={columns}
        data={logs}
        loading={loading}
        error={error}
        onRetry={fetchLogs}
        onRowClick={setSelected}
        toolbar={
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="rounded-md border border-border bg-white px-3 py-1.5 text-sm"
              value={entityFilter}
              onChange={(e) => setEntityFilter(e.target.value)}
              aria-label="Entity type"
            >
              <option value="">All entities</option>
              {entityTypes.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
            <Input
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="w-40"
              aria-label="From date"
            />
            <span className="text-sm text-muted-foreground">to</span>
            <Input
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="w-40"
              aria-label="To date"
            />
          </div>
        }
        pagination={{
          page: page + 1,
          onPrev: () => setPage((p) => Math.max(0, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: logs.length === PAGE_SIZE,
        }}
        empty={{
          icon: ShieldCheck,
          title: page > 0 || filtered ? "No entries match these filters" : "No audit entries yet",
          description: "Config-grade mutations (plans, metrics, wallets, webhooks, team, ...) appear here.",
        }}
      />

      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent className="overflow-y-auto sm:max-w-md">
          {selected && (
            <>
              <SheetHeader>
                <SheetTitle className="flex items-center gap-2">
                  <Badge
                    variant={methodVariant(splitAction(selected.action).method)}
                    className="font-mono text-[10px]"
                  >
                    {splitAction(selected.action).method}
                  </Badge>
                  <span className="truncate font-mono text-sm">
                    {splitAction(selected.action).path}
                  </span>
                </SheetTitle>
              </SheetHeader>

              <div className="mt-4 space-y-5">
                <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-lg border border-border bg-muted/20 p-3">
                  <Field label="When" span>
                    {fmtWhen(selected.created_at)}
                  </Field>
                  <Field label="Status">
                    <Badge variant={selected.status < 300 ? "success" : "destructive"}>
                      {selected.status}
                    </Badge>
                  </Field>
                  <Field label="Actor">
                    {selected.actor === "api_key" ? (
                      <Badge variant="neutral">API key</Badge>
                    ) : (
                      <CopyableId value={selected.actor} label="Actor ID" />
                    )}
                  </Field>
                  <Field label="Entity">
                    <span className="capitalize">{selected.entity_type || "—"}</span>
                  </Field>
                  {selected.entity_id ? (
                    <Field label="Entity ID">
                      <CopyableId value={selected.entity_id} label="Entity ID" />
                    </Field>
                  ) : (
                    <Field label="Entity ID">
                      <span className="text-muted-foreground">—</span>
                    </Field>
                  )}
                  {selected.ip && (
                    <Field label="IP address" span>
                      <span className="font-mono text-xs">{selected.ip}</span>
                    </Field>
                  )}
                  <Field label="Audit ID" span>
                    <CopyableId value={selected.id} label="Audit ID" />
                  </Field>
                </dl>

                <div>
                  <div className="mb-1 flex items-center justify-between">
                    <span className="text-xs font-medium text-foreground">Request payload</span>
                    {body && (
                      <button
                        type="button"
                        onClick={() => copyText(body, "Payload")}
                        className="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
                      >
                        <Copy className="h-3 w-3" /> Copy
                      </button>
                    )}
                  </div>
                  {body ? (
                    <pre className="max-h-72 overflow-auto rounded-md border border-border bg-muted/40 p-3 text-xs">
                      {body}
                    </pre>
                  ) : (
                    <p className="rounded-md border border-dashed border-border p-3 text-xs text-muted-foreground">
                      No request payload was recorded for this action (e.g. a body-less
                      action such as send or convert).
                    </p>
                  )}
                </div>
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default AuditLog;

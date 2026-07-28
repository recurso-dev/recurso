import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";

const PAGE_SIZE = 100;

// Append-only audit trail (Lago-parity C2): every successful config-grade
// mutation, immutable at the database level.
const AuditLog = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [entityFilter, setEntityFilter] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(0); // 0-based

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
      cell: (l) => (
        <span className="whitespace-nowrap text-xs text-muted-foreground">
          {new Date(l.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: "actor",
      header: "Actor",
      cell: (l) => (
        <span className="font-mono text-xs">
          {l.actor === "api_key" ? <Badge variant="neutral">API key</Badge> : l.actor.slice(0, 8) + "…"}
        </span>
      ),
    },
    {
      key: "action",
      header: "Action",
      cell: (l) => <span className="font-mono text-xs text-foreground">{l.action}</span>,
    },
    {
      key: "entity",
      header: "Entity",
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
      cell: (l) => <Badge variant={l.status < 300 ? "success" : "destructive"}>{l.status}</Badge>,
    },
  ];

  const filtered = Boolean(entityFilter || from || to);

  return (
    <div>
      <PageHeader
        title="Audit Log"
        description="Every configuration change, immutably recorded. Updates and deletes are rejected at the database."
      />

      <DataTable
        columns={columns}
        data={logs}
        loading={loading}
        error={error}
        onRetry={fetchLogs}
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
    </div>
  );
};

export default AuditLog;

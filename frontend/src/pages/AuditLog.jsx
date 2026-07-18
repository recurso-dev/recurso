import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";

// Append-only audit trail (Lago-parity C2): every successful config-grade
// mutation, immutable at the database level.
const AuditLog = () => {
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [entityFilter, setEntityFilter] = useState("");

  const fetchLogs = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = { limit: 200 };
      if (entityFilter) params.entity_type = entityFilter;
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
  }, [entityFilter]);

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
          <select
            className="rounded-md border border-border bg-white px-3 py-1.5 text-sm"
            value={entityFilter}
            onChange={(e) => setEntityFilter(e.target.value)}
          >
            <option value="">All entities</option>
            {entityTypes.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        }
        empty={{
          icon: ShieldCheck,
          title: "No audit entries yet",
          description: "Config-grade mutations (plans, metrics, wallets, webhooks, team, ...) appear here.",
        }}
      />
    </div>
  );
};

export default AuditLog;

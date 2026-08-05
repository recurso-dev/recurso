import { useEffect, useState } from "react";
import { Webhook, Send } from "lucide-react";
import { toast } from "sonner";

import { endpoints as api } from "../lib/api";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Copy } from "lucide-react";

// "Aug 3, 2026, 2:17 PM" beats "03/08/2026, 14:17:34" for scanning.
const fmtWhen = (x) =>
  new Date(x).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });

const copyText = async (v, label) => {
  try {
    await navigator.clipboard.writeText(v);
    toast.success(`${label} copied.`);
  } catch {
    toast.error("Couldn't copy to clipboard.");
  }
};

const Field = ({ label, children }) => (
  <div>
    <dt className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</dt>
    <dd className="mt-0.5 text-sm text-foreground">{children}</dd>
  </div>
);

const PAGE_SIZE = 100;

// Color by event family so the stream scans at a glance: money events pop,
// lifecycle events stay quiet.
const FAMILY_VARIANT = {
  invoice: "success",
  payment: "info",
  subscription: "warning",
  customer: "neutral",
  usage: "info",
};
const familyVariant = (type) => FAMILY_VARIANT[(type || "").split(".")[0]] || "neutral";

// A one-line, human summary pulled from the event payload — so the stream is
// readable without opening every event. Money is shown in the currency's own
// units (990000 minor → $9,900.00), never the raw minor-unit integer.
const MONEY_KEYS = ["amount_paid", "amount_due", "amount_refunded", "amount", "total"];
const summarizeEvent = (ev) => {
  const d = ev.data || {};
  const parts = [];
  if (d.invoice_number) parts.push(String(d.invoice_number));
  const moneyKey = MONEY_KEYS.find((k) => typeof d[k] === "number");
  if (moneyKey) parts.push(formatCurrency(d[moneyKey], d.currency));
  if (!parts.length && d.status) parts.push(String(d.status));
  return parts.join(" · ");
};

// Webhook event inspector: recent outbound events, their payloads, and per-endpoint
// delivery attempts — with one-click redelivery. Backed by GET /events,
// GET /events/:id/deliveries, POST /events/:id/redeliver.
const Events = () => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(0);
  const [typeFilter, setTypeFilter] = useState("all");

  const [selected, setSelected] = useState(null); // event whose detail sheet is open
  const [deliveries, setDeliveries] = useState([]);
  const [delLoading, setDelLoading] = useState(false);
  const [redelivering, setRedelivering] = useState(false);

  const fetchEvents = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getEvents({ limit: PAGE_SIZE, offset: page * PAGE_SIZE });
      setEvents(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || err?.message || "Failed to load events");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEvents();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const loadDeliveries = async (eventId) => {
    setDelLoading(true);
    try {
      const res = await api.getEventDeliveries(eventId);
      setDeliveries(res.data.data || res.data || []);
    } catch {
      setDeliveries([]);
    } finally {
      setDelLoading(false);
    }
  };

  const openEvent = (ev) => {
    setSelected(ev);
    setDeliveries([]);
    loadDeliveries(ev.id);
  };

  const redeliver = async () => {
    if (!selected) return;
    setRedelivering(true);
    try {
      const res = await api.redeliverEvent(selected.id);
      const queued = res?.data?.queued ?? res?.data?.data?.queued;
      toast.success(
        queued != null ? `Re-queued delivery to ${queued} endpoint(s)` : "Event re-queued for delivery"
      );
      await loadDeliveries(selected.id);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Redelivery failed");
    } finally {
      setRedelivering(false);
    }
  };

  const columns = [
    {
      key: "when",
      header: "When",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (e) => (
        <span className="whitespace-nowrap text-xs text-muted-foreground">
          {fmtWhen(e.created_at)}
        </span>
      ),
    },
    {
      key: "type",
      header: "Type",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (e) => (
        <Badge variant={familyVariant(e.type)} className="font-mono text-[11px]">
          {e.type}
        </Badge>
      ),
    },
    {
      // Flexible column: absorbs the remaining width so the row isn't a few
      // short cells stranded across an empty table.
      key: "summary",
      header: "Summary",
      className: "w-full",
      headerClassName: "w-full",
      cell: (e) => {
        const s = summarizeEvent(e);
        return s ? (
          <span className="text-xs text-foreground">{s}</span>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        );
      },
    },
    {
      key: "object",
      header: "Object",
      headerClassName: "whitespace-nowrap",
      className: "whitespace-nowrap",
      cell: (e) => (
        <span className="text-xs text-muted-foreground">
          {e.object_type}
          {e.object_id ? ` · ${String(e.object_id).slice(0, 8)}…` : ""}
        </span>
      ),
    },
  ];

  // Distinct types on the loaded page — a cheap, zero-request way to narrow the
  // stream while hunting a specific event.
  const types = [...new Set(events.map((e) => e.type))].sort();
  const visible = typeFilter === "all" ? events : events.filter((e) => e.type === typeFilter);

  const deliveryBadge = (d) => {
    if (!d.status_code) return <Badge variant="neutral">pending</Badge>;
    const ok = d.status_code >= 200 && d.status_code < 300;
    return <Badge variant={ok ? "success" : "destructive"}>{d.status_code}</Badge>;
  };

  return (
    <div>
      <PageHeader
        title="Events"
        description="Outbound webhook events, their payloads, and delivery attempts. Click an event to inspect it and redeliver in one click."
        actions={
          types.length > 1 ? (
            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger className="w-[220px]" aria-label="Filter by event type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All types</SelectItem>
                {types.map((t) => (
                  <SelectItem key={t} value={t}>{t}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : undefined
        }
      />

      <DataTable
        columns={columns}
        data={visible}
        loading={loading}
        error={error}
        onRetry={fetchEvents}
        onRowClick={openEvent}
        pagination={{
          page: page + 1,
          onPrev: () => setPage((p) => Math.max(0, p - 1)),
          onNext: () => setPage((p) => p + 1),
          hasNext: events.length === PAGE_SIZE,
        }}
        empty={{
          icon: Webhook,
          title:
            typeFilter !== "all"
              ? "No events of this type on this page"
              : page > 0
                ? "No more events"
                : "No events yet",
          description:
            "Events appear here as your account creates customers, invoices, subscriptions, and more.",
        }}
      />

      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent className="overflow-y-auto sm:max-w-md">
          {selected && (
            <>
              <SheetHeader>
                <SheetTitle className="font-mono text-sm">{selected.type}</SheetTitle>
              </SheetHeader>

              <div className="mt-4 space-y-5">
                <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-lg border border-border bg-muted/20 p-3">
                  <Field label="Event ID">
                    <button
                      type="button"
                      onClick={() => copyText(selected.id, "Event ID")}
                      className="group inline-flex max-w-full items-center gap-1.5 text-left"
                      title="Copy event ID"
                    >
                      <span className="truncate font-mono text-xs">{selected.id}</span>
                      <Copy className="h-3 w-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                    </button>
                  </Field>
                  <Field label="Created">{fmtWhen(selected.created_at)}</Field>
                  <Field label="Object">
                    <span className="capitalize">{selected.object_type}</span>
                  </Field>
                  {selected.object_id && (
                    <Field label="Object ID">
                      <button
                        type="button"
                        onClick={() => copyText(selected.object_id, "Object ID")}
                        className="group inline-flex max-w-full items-center gap-1.5 text-left"
                        title="Copy object ID"
                      >
                        <span className="truncate font-mono text-xs">{selected.object_id}</span>
                        <Copy className="h-3 w-3 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
                      </button>
                    </Field>
                  )}
                </dl>

                <div>
                  <div className="mb-1 flex items-center justify-between">
                    <span className="text-xs font-medium text-foreground">Payload</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 px-2 text-xs"
                      onClick={() => copyText(JSON.stringify(selected.data, null, 2), "Payload")}
                    >
                      <Copy className="mr-1 h-3 w-3" /> Copy JSON
                    </Button>
                  </div>
                  <pre className="max-h-64 overflow-auto rounded-md border border-border bg-muted/40 p-3 text-xs">
                    {JSON.stringify(selected.data, null, 2)}
                  </pre>
                </div>

                <div>
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-medium text-foreground">Deliveries</span>
                    <Button size="sm" onClick={redeliver} disabled={redelivering}>
                      <Send className="mr-1.5 h-3.5 w-3.5" />
                      {redelivering ? "Re-queuing…" : "Redeliver"}
                    </Button>
                  </div>
                  {delLoading ? (
                    <p className="text-xs text-muted-foreground">Loading deliveries…</p>
                  ) : deliveries.length === 0 ? (
                    <p className="text-xs text-muted-foreground">
                      No delivery attempts yet — Redeliver queues one to every active endpoint.
                    </p>
                  ) : (
                    <ul className="space-y-2">
                      {deliveries.map((d) => (
                        <li key={d.id} className="rounded-md border border-border p-2 text-xs">
                          <div className="flex items-center justify-between">
                            {deliveryBadge(d)}
                            <span className="text-muted-foreground">attempt {d.attempt}</span>
                          </div>
                          {d.response_body ? (
                            <pre className="mt-1 max-h-20 overflow-auto text-[11px] text-muted-foreground">
                              {d.response_body}
                            </pre>
                          ) : null}
                        </li>
                      ))}
                    </ul>
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

export default Events;

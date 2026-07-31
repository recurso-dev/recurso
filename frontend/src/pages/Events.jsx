import { useEffect, useState } from "react";
import { Webhook, Send } from "lucide-react";
import { toast } from "sonner";

import { endpoints as api } from "../lib/api";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";

const PAGE_SIZE = 100;

// Webhook event inspector: recent outbound events, their payloads, and per-endpoint
// delivery attempts — with one-click redelivery. Backed by GET /events,
// GET /events/:id/deliveries, POST /events/:id/redeliver.
const Events = () => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(0);

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
      cell: (e) => (
        <span className="whitespace-nowrap text-xs text-muted-foreground">
          {new Date(e.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (e) => <span className="font-mono text-xs text-foreground">{e.type}</span>,
    },
    {
      key: "object",
      header: "Object",
      cell: (e) => (
        <span className="text-xs text-muted-foreground">
          {e.object_type}
          {e.object_id ? ` · ${String(e.object_id).slice(0, 8)}…` : ""}
        </span>
      ),
    },
  ];

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
      />

      <DataTable
        columns={columns}
        data={events}
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
          title: page > 0 ? "No more events" : "No events yet",
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
                <div className="space-y-0.5 text-xs text-muted-foreground">
                  <div>
                    ID: <span className="font-mono">{selected.id}</span>
                  </div>
                  <div>
                    Object: {selected.object_type}
                    {selected.object_id ? ` · ${selected.object_id}` : ""}
                  </div>
                  <div>Created: {new Date(selected.created_at).toLocaleString()}</div>
                </div>

                <div>
                  <div className="mb-1 text-xs font-medium text-foreground">Payload</div>
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

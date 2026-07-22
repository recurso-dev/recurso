import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Plus,
  Trash2,
  Webhook,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  Copy,
  AlertTriangle,
  CheckCircle2,
  Send,
  Clock,
  Inbox,
} from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { EmptyState } from "@/components/patterns/EmptyState";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";

// Map a derived delivery status to a Badge variant + label.
const DELIVERY_STATUS = {
  succeeded: { variant: "success", label: "Succeeded" },
  failed: { variant: "destructive", label: "Failed" },
  pending: { variant: "warning", label: "Pending" },
};

// Render a value or an em-dash when absent. Never invents data.
const dash = (v) => (v === null || v === undefined || v === "" ? "—" : v);
const fmtTime = (v) => (v ? new Date(v).toLocaleString() : "—");

function DeliveryStatusBadge({ status }) {
  const s = DELIVERY_STATUS[status] || { variant: "neutral", label: dash(status) };
  return <Badge variant={s.variant}>{s.label}</Badge>;
}

// Renders the per-event delivery attempts (loading / error / empty / rows).
function EventDeliveries({ state }) {
  if (!state || state.loading) {
    return (
      <p className="rounded-lg border border-border bg-background px-3 py-4 text-xs text-muted-foreground">
        Loading deliveries…
      </p>
    );
  }
  if (state.error) {
    return (
      <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-4 text-xs text-red-700">
        {state.error}
      </p>
    );
  }
  if (!state.data || state.data.length === 0) {
    return (
      <p className="rounded-lg border border-border bg-background px-3 py-4 text-xs text-muted-foreground">
        No delivery attempts recorded for this event.
      </p>
    );
  }
  return (
    <div className="flex flex-col gap-2">
      {state.data.map((d, i) => (
        <div
          key={d.id || d.endpoint_url || i}
          className="rounded-lg border border-border bg-background p-3"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <code className="break-all font-mono text-xs font-medium text-foreground">
              {dash(d.endpoint_url)}
            </code>
            <DeliveryStatusBadge status={d.status} />
          </div>
          <div className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground sm:grid-cols-4">
            <span>
              Attempts:{" "}
              <span className="text-foreground">{dash(d.attempts)}</span>
            </span>
            <span>
              Status code:{" "}
              <span className="text-foreground">{dash(d.last_status_code)}</span>
            </span>
            <span>
              Delivered:{" "}
              <span className="text-foreground">{fmtTime(d.delivered_at)}</span>
            </span>
            <span>
              Next retry:{" "}
              <span className="text-foreground">{fmtTime(d.next_retry_at)}</span>
            </span>
          </div>
          {d.last_error ? (
            <p className="mt-2 break-words rounded bg-red-50 px-2 py-1 font-mono text-xs text-red-700">
              {d.last_error}
            </p>
          ) : null}
        </div>
      ))}
    </div>
  );
}

export default function Developers() {
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState("keys"); // 'keys' | 'webhooks' | 'events'
  const [eventTypeFilter, setEventTypeFilter] = useState("all");
  const [expandedEventId, setExpandedEventId] = useState(null);
  const [generatedKey, setGeneratedKey] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isWebhookModalOpen, setIsWebhookModalOpen] = useState(false);
  const [newWebhook, setNewWebhook] = useState({ url: "", events: [] });
  const [createdWebhookSecret, setCreatedWebhookSecret] = useState(null);
  const [deleteWebhookTarget, setDeleteWebhookTarget] = useState(null);
  const [revokeKeyTarget, setRevokeKeyTarget] = useState(null);

  // Endpoint "View deliveries" sheet.
  const [deliveriesSheet, setDeliveriesSheet] = useState(null); // the webhook endpoint, or null
  const [endpointStatusFilter, setEndpointStatusFilter] = useState("all");

  // Reads (errors leave the list empty, matching the prior console-only behavior).
  const { data: keys = [], isLoading: loading } = useQuery({
    queryKey: ["api-keys"],
    queryFn: async () => (await endpoints.getAPIKeys()).data.data || [],
  });
  const { data: webhooks = [] } = useQuery({
    queryKey: ["webhooks"],
    queryFn: async () => (await endpoints.getWebhooks()).data.data || [],
  });
  const {
    data: events = [],
    isLoading: eventsLoading,
    refetch: refetchEvents,
  } = useQuery({
    queryKey: ["events", { limit: 50 }],
    queryFn: async () => (await endpoints.getEvents({ limit: 50 })).data.data || [],
  });
  const { data: eventTypes = [] } = useQuery({
    queryKey: ["event-types"],
    queryFn: async () => (await endpoints.getEventTypes()).data.data || [],
  });

  // Per-event deliveries: only the expanded row is fetched; results stay cached
  // in react-query so re-expanding a row doesn't refetch within staleTime.
  const {
    data: expandedDeliveries = [],
    isLoading: deliveriesLoading,
    isError: deliveriesIsError,
    error: deliveriesErr,
  } = useQuery({
    queryKey: ["event-deliveries", expandedEventId],
    queryFn: async () =>
      (await endpoints.getEventDeliveries(expandedEventId)).data.data || [],
    enabled: !!expandedEventId,
  });
  const expandedDeliveryState = expandedEventId
    ? {
        loading: deliveriesLoading,
        error: deliveriesIsError
          ? deliveriesErr?.response?.data?.error?.message ||
            deliveriesErr?.message ||
            "Failed to load deliveries"
          : null,
        data: expandedDeliveries,
      }
    : null;

  // Endpoint deliveries sheet: keyed on the endpoint + status filter, fetched
  // only while the sheet is open.
  const {
    data: endpointDeliveries = [],
    isLoading: endpointDeliveriesLoading,
    isError: endpointDeliveriesIsError,
    error: endpointDeliveriesErr,
    refetch: refetchEndpointDeliveries,
  } = useQuery({
    queryKey: ["webhook-deliveries", deliveriesSheet?.id, endpointStatusFilter],
    queryFn: async () => {
      const params = { limit: 50 };
      if (endpointStatusFilter && endpointStatusFilter !== "all")
        params.status = endpointStatusFilter;
      return (await endpoints.getWebhookDeliveries(deliveriesSheet.id, params)).data
        .data || [];
    },
    enabled: !!deliveriesSheet,
  });
  const endpointDeliveriesError = endpointDeliveriesIsError
    ? endpointDeliveriesErr?.response?.data?.error?.message ||
      endpointDeliveriesErr?.message ||
      "Failed to load deliveries"
    : null;

  const createKeyMutation = useMutation({
    mutationFn: () => endpoints.createKey({}),
    onSuccess: (response) => {
      // POST /developer/keys returns the APIKey object directly;
      // key_value is only present on creation.
      setGeneratedKey(response.data.data.key_value || response.data.data.key);
      setIsModalOpen(true);
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
    onError: (error) => console.error("Failed to create key:", error),
  });
  const handleCreateKey = () => createKeyMutation.mutate();

  const createWebhookMutation = useMutation({
    mutationFn: (payload) => endpoints.createWebhook(payload),
    onSuccess: (response) => {
      setCreatedWebhookSecret(response.data.data?.secret);
      setNewWebhook({ url: "", events: [] });
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
    },
    onError: (error) =>
      toast.error(
        "Failed to create webhook: " +
          (error.response?.data?.error?.message || error.message)
      ),
  });
  const handleCreateWebhook = () => {
    if (!newWebhook.url || newWebhook.events.length === 0) {
      toast.error("Enter a URL and select at least one event type.");
      return;
    }
    createWebhookMutation.mutate(newWebhook);
  };

  const revokeKeyMutation = useMutation({
    mutationFn: (id) => endpoints.revokeKey(id),
    onSuccess: () => {
      toast.success("API key revoked.");
      setRevokeKeyTarget(null);
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
    },
    onError: (error) =>
      toast.error(error.response?.data?.error?.message || "Failed to revoke key"),
  });
  const handleRevokeKey = () => {
    if (!revokeKeyTarget) return;
    revokeKeyMutation.mutate(revokeKeyTarget.id);
  };
  const revokingKey = revokeKeyMutation.isPending;

  const deleteWebhookMutation = useMutation({
    mutationFn: (id) => endpoints.deleteWebhook(id),
    onSuccess: () => {
      setDeleteWebhookTarget(null);
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
    },
    onError: (error) =>
      toast.error(error.response?.data?.error?.message || "Failed to delete webhook"),
  });
  const handleDeleteWebhook = () => {
    if (!deleteWebhookTarget) return;
    deleteWebhookMutation.mutate(deleteWebhookTarget);
  };

  // Expand/collapse an event row; the deliveries query loads lazily via `enabled`.
  const handleToggleEvent = (eventId) => {
    setExpandedEventId((cur) => (cur === eventId ? null : eventId));
  };

  const redeliverMutation = useMutation({
    mutationFn: (eventId) => endpoints.redeliverEvent(eventId),
    onSuccess: (response, eventId) => {
      const queued = response.data?.deliveries_queued ?? 0;
      toast.success(
        `Re-delivery queued for ${queued} ${queued === 1 ? "endpoint" : "endpoints"}.`
      );
      queryClient.invalidateQueries({ queryKey: ["event-deliveries", eventId] });
    },
    onError: (error) =>
      toast.error(
        error.response?.data?.error?.message ||
          error.message ||
          "Failed to queue re-delivery"
      ),
  });
  const handleRedeliver = (eventId) => redeliverMutation.mutate(eventId);
  const redeliveringId = redeliverMutation.isPending
    ? redeliverMutation.variables
    : null;

  const openEndpointDeliveries = (hook) => {
    setEndpointStatusFilter("all");
    setDeliveriesSheet(hook);
  };

  // Pause/resume deliveries for an endpoint; config and secret are kept.
  const toggleStatusMutation = useMutation({
    mutationFn: ({ id, status }) => endpoints.setWebhookStatus(id, status),
    onSuccess: (_data, { status }) => {
      toast.success(status === "active" ? "Endpoint resumed." : "Endpoint paused.");
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to update endpoint"),
  });
  const toggleWebhookStatus = (hook) => {
    const next = hook.status === "active" ? "inactive" : "active";
    toggleStatusMutation.mutate({ id: hook.id, status: next });
  };

  const toggleEventType = (eventType) => {
    setNewWebhook((prev) => {
      const events = prev.events.includes(eventType)
        ? prev.events.filter((e) => e !== eventType)
        : [...prev.events, eventType];
      return { ...prev, events };
    });
  };

  // GET /v1/events only supports limit/offset, so the type filter is applied
  // client-side over the fetched window. The API exposes no per-endpoint
  // delivery status for events, so none is rendered here.
  const filteredEvents =
    eventTypeFilter === "all"
      ? events
      : events.filter((e) => e.type === eventTypeFilter);

  const eventTypeOptions = [
    ...new Set([...eventTypes, ...events.map((e) => e.type)]),
  ].sort();

  const closeWebhookModal = () => {
    setIsWebhookModalOpen(false);
    setCreatedWebhookSecret(null);
  };

  const keyColumns = [
    {
      key: "key_prefix",
      header: "Key prefix",
      cell: (k) => (
        <code className="rounded bg-muted px-2 py-1 font-mono text-sm text-foreground">
          {k.key_prefix ? `${k.key_prefix}…` : "••••••••"}
        </code>
      ),
    },
    {
      key: "type",
      header: "Type",
      cell: (k) => (
        <span className="capitalize text-muted-foreground">{k.type || "secret"}</span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (k) =>
        k.is_active ? (
          <Badge variant="success">Active</Badge>
        ) : (
          <Badge variant="neutral">Inactive</Badge>
        ),
    },
    {
      key: "created_at",
      header: "Created",
      cell: (k) => (
        <span className="text-muted-foreground">
          {k.created_at ? new Date(k.created_at).toLocaleDateString() : "—"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "",
      align: "right",
      cell: (k) =>
        k.is_active && (
          <Button
            size="sm"
            variant="outline"
            onClick={(e) => {
              e.stopPropagation();
              setRevokeKeyTarget(k);
            }}
          >
            Revoke
          </Button>
        ),
    },
  ];

  const headerAction =
    activeTab === "keys" ? (
      <Button onClick={handleCreateKey}>
        <Plus className="h-4 w-4" />
        Create API key
      </Button>
    ) : activeTab === "webhooks" ? (
      <Button onClick={() => setIsWebhookModalOpen(true)}>
        <Plus className="h-4 w-4" />
        Add endpoint
      </Button>
    ) : null;

  return (
    <div>
      <PageHeader
        title="Developer settings"
        description="Manage your API keys, webhooks, and view event logs."
        actions={headerAction}
      />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="keys">API keys</TabsTrigger>
          <TabsTrigger value="webhooks">Webhooks</TabsTrigger>
          <TabsTrigger value="events">Event logs</TabsTrigger>
        </TabsList>

        {/* API Keys */}
        <TabsContent value="keys" className="mt-6">
          <DataTable
            columns={keyColumns}
            data={keys}
            loading={loading}
            empty={{
              title: "No API keys found",
              description: "Generate one to start authenticating your API requests.",
              action: (
                <Button onClick={handleCreateKey}>
                  <Plus className="h-4 w-4" />
                  Create API key
                </Button>
              ),
            }}
          />
        </TabsContent>

        {/* Webhooks */}
        <TabsContent value="webhooks" className="mt-6">
          {webhooks.length === 0 ? (
            <Card>
              <EmptyState
                icon={Webhook}
                title="No webhook endpoints configured"
                description="Add one to receive real-time events from Recurso."
                action={
                  <Button onClick={() => setIsWebhookModalOpen(true)}>
                    <Plus className="h-4 w-4" />
                    Add endpoint
                  </Button>
                }
              />
            </Card>
          ) : (
            <div className="flex flex-col gap-3">
              {webhooks.map((hook) => (
                <Card key={hook.id} className="p-4">
                  <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <Webhook className="h-4 w-4 shrink-0 text-stone-400" />
                        <code className="break-all font-mono text-sm font-semibold text-foreground">
                          {hook.url}
                        </code>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-1.5">
                        {hook.events?.map((e) => (
                          <Badge key={e} variant="neutral">
                            {e}
                          </Badge>
                        ))}
                      </div>
                      <div className="mt-3">
                        <p className="text-xs uppercase tracking-wide text-muted-foreground">
                          Signing secret
                        </p>
                        <code className="font-mono text-xs text-stone-400">
                          whsec_•••••••
                        </code>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge variant={hook.status === "active" ? "success" : "neutral"}>
                        {hook.status
                          ? hook.status.charAt(0).toUpperCase() + hook.status.slice(1)
                          : "—"}
                      </Badge>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => openEndpointDeliveries(hook)}
                      >
                        <Inbox className="h-4 w-4" />
                        View deliveries
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => toggleWebhookStatus(hook)}
                      >
                        {hook.status === "active" ? "Pause" : "Resume"}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setDeleteWebhookTarget(hook.id)}
                        className="text-stone-400 hover:text-red-600"
                        title="Delete endpoint"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>

        {/* Event Logs */}
        <TabsContent value="events" className="mt-6">
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm text-muted-foreground">
              History of events generated by your account. Click a row to inspect its
              payload.
            </p>
            <div className="flex items-center gap-2">
              <Select value={eventTypeFilter} onValueChange={setEventTypeFilter}>
                <SelectTrigger className="w-[200px]" aria-label="Filter by event type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All event types</SelectItem>
                  {eventTypeOptions.map((t) => (
                    <SelectItem key={t} value={t}>
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="icon"
                onClick={() => refetchEvents()}
                title="Refresh events"
              >
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <Card className="overflow-hidden">
            {eventsLoading ? (
              <div className="py-12 text-center text-sm text-muted-foreground">
                Loading events...
              </div>
            ) : filteredEvents.length === 0 ? (
              <EmptyState
                title={
                  eventTypeFilter === "all"
                    ? "No events yet"
                    : `No ${eventTypeFilter} events`
                }
                description={
                  eventTypeFilter === "all"
                    ? "Events will appear here when billing actions occur."
                    : "No matching events in the last 50."
                }
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/40 hover:bg-muted/40">
                    <TableHead>Event type</TableHead>
                    <TableHead>Object</TableHead>
                    <TableHead>Created at</TableHead>
                    <TableHead className="text-right">Payload</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredEvents.map((evt) => (
                    <React.Fragment key={evt.id}>
                      <TableRow
                        onClick={() => handleToggleEvent(evt.id)}
                        className="cursor-pointer"
                      >
                        <TableCell>
                          <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-semibold text-foreground">
                            {evt.type}
                          </code>
                        </TableCell>
                        <TableCell
                          className="font-mono text-xs text-muted-foreground"
                          title={evt.object_id}
                        >
                          {evt.object_type}:{evt.object_id?.substring(0, 8)}...
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {new Date(evt.created_at).toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right">
                          {expandedEventId === evt.id ? (
                            <ChevronUp className="ml-auto h-4 w-4 text-stone-400" />
                          ) : (
                            <ChevronDown className="ml-auto h-4 w-4 text-stone-400" />
                          )}
                        </TableCell>
                      </TableRow>
                      {expandedEventId === evt.id && (
                        <TableRow className="bg-muted/30 hover:bg-muted/30">
                          <TableCell colSpan={4}>
                            <div className="mb-4 flex items-center justify-between gap-3">
                              <p className="font-mono text-xs text-muted-foreground">
                                {evt.id}
                              </p>
                              <Button
                                variant="outline"
                                size="sm"
                                disabled={redeliveringId === evt.id}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleRedeliver(evt.id);
                                }}
                              >
                                <Send
                                  className={cn(
                                    "h-3.5 w-3.5",
                                    redeliveringId === evt.id && "animate-pulse"
                                  )}
                                />
                                {redeliveringId === evt.id ? "Queuing…" : "Redeliver"}
                              </Button>
                            </div>

                            {/* Delivery attempts */}
                            <div className="mb-4">
                              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                                Deliveries
                              </p>
                              <EventDeliveries state={expandedDeliveryState} />
                            </div>

                            {/* Raw payload */}
                            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                              Payload
                            </p>
                            <pre
                              data-testid={`event-payload-${evt.id}`}
                              className="max-h-80 overflow-auto rounded-lg bg-muted p-4 font-mono text-xs text-foreground"
                            >
                              {JSON.stringify(evt.data ?? {}, null, 2)}
                            </pre>
                          </TableCell>
                        </TableRow>
                      )}
                    </React.Fragment>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </TabsContent>
      </Tabs>

      {/* New API Key Dialog */}
      <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New API key generated</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-start gap-2 rounded-lg bg-amber-50 p-4 text-amber-800 ring-1 ring-inset ring-amber-200">
              <AlertTriangle className="h-5 w-5 shrink-0" />
              <p className="text-sm font-medium">
                Copy your secret API key and store it securely. You will not be able to
                see it again.
              </p>
            </div>
            <div className="flex gap-2">
              <Input readOnly value={generatedKey || ""} className="font-mono" />
              <Button
                variant="outline"
                size="icon"
                onClick={() => navigator.clipboard.writeText(generatedKey)}
                title="Copy to clipboard"
                aria-label="Copy API key to clipboard"
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setIsModalOpen(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Webhook slide-over */}
      <Sheet
        open={isWebhookModalOpen}
        onOpenChange={(open) => (!open ? closeWebhookModal() : setIsWebhookModalOpen(true))}
      >
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>
              {createdWebhookSecret ? "Webhook created" : "Add webhook endpoint"}
            </SheetTitle>
            <SheetDescription>
              {createdWebhookSecret
                ? "Store the signing secret before closing."
                : "We POST the selected events to this URL, signed with a per-endpoint secret."}
            </SheetDescription>
          </SheetHeader>

          {createdWebhookSecret ? (
            <div className="flex-1 space-y-4 overflow-y-auto px-6">
              <div className="flex items-start gap-2 rounded-lg bg-emerald-50 p-4 text-emerald-800 ring-1 ring-inset ring-emerald-200">
                <CheckCircle2 className="h-5 w-5 shrink-0" />
                <p className="text-sm font-medium">
                  Webhook endpoint created successfully.
                </p>
              </div>
              <div className="space-y-1.5">
                <Label>Signing secret</Label>
                <p className="text-xs text-muted-foreground">
                  Store this securely. You won't be able to see it again.
                </p>
                <div className="flex gap-2">
                  <Input readOnly value={createdWebhookSecret} className="font-mono" />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => navigator.clipboard.writeText(createdWebhookSecret)}
                    title="Copy to clipboard"
                    aria-label="Copy webhook signing secret to clipboard"
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <SheetFooter className="px-0">
                <Button onClick={closeWebhookModal}>Done</Button>
              </SheetFooter>
            </div>
          ) : (
            <div className="flex-1 space-y-4 overflow-y-auto px-6">
              <div className="space-y-1.5">
                <Label htmlFor="webhook-url">Endpoint URL</Label>
                <Input
                  id="webhook-url"
                  type="url"
                  value={newWebhook.url}
                  onChange={(e) =>
                    setNewWebhook((prev) => ({ ...prev, url: e.target.value }))
                  }
                  placeholder="https://example.com/webhooks/recurso"
                />
              </div>
              <div className="space-y-1.5">
                <Label>Events to receive</Label>
                <div className="grid max-h-48 grid-cols-1 gap-1 overflow-y-auto rounded-lg border border-border p-2 sm:grid-cols-2">
                  {eventTypes.map((eventType) => (
                    <label
                      key={eventType}
                      className="flex cursor-pointer items-center gap-2 rounded p-2 text-sm hover:bg-muted"
                    >
                      <input
                        type="checkbox"
                        checked={newWebhook.events.includes(eventType)}
                        onChange={() => toggleEventType(eventType)}
                        className="h-4 w-4 rounded border-input accent-emerald-600 focus:ring-ring"
                      />
                      <code className="text-xs text-muted-foreground">{eventType}</code>
                    </label>
                  ))}
                </div>
              </div>
              <SheetFooter className="px-0">
                <Button variant="outline" onClick={() => setIsWebhookModalOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreateWebhook}>Create endpoint</Button>
              </SheetFooter>
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* Endpoint deliveries slide-over */}
      <Sheet
        open={!!deliveriesSheet}
        onOpenChange={(open) => !open && setDeliveriesSheet(null)}
      >
        <SheetContent side="right" className="w-full sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>Recent deliveries</SheetTitle>
            <SheetDescription className="break-all font-mono">
              {deliveriesSheet?.url}
            </SheetDescription>
          </SheetHeader>

          <div className="flex items-center gap-2 border-b border-border px-6 py-3">
            <Select
              value={endpointStatusFilter}
              onValueChange={setEndpointStatusFilter}
            >
              <SelectTrigger className="w-[180px]" aria-label="Filter by delivery status">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="succeeded">Succeeded</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="icon"
              onClick={() => refetchEndpointDeliveries()}
              title="Refresh deliveries"
            >
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex-1 overflow-y-auto px-6 py-4">
            {endpointDeliveriesLoading ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                Loading deliveries…
              </p>
            ) : endpointDeliveriesError ? (
              <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-4 text-sm text-red-700">
                {endpointDeliveriesError}
              </p>
            ) : endpointDeliveries.length === 0 ? (
              <EmptyState
                icon={Inbox}
                title="No deliveries"
                description={
                  endpointStatusFilter === "all"
                    ? "This endpoint has not received any deliveries yet."
                    : `No ${endpointStatusFilter} deliveries in the recent window.`
                }
              />
            ) : (
              <div className="flex flex-col gap-3">
                {endpointDeliveries.map((d, i) => (
                  <Card key={d.id || i} className="p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-semibold text-foreground">
                        {dash(d.event_type || d.type)}
                      </code>
                      <DeliveryStatusBadge status={d.status} />
                    </div>
                    <div className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <span>
                        Attempts:{" "}
                        <span className="text-foreground">{dash(d.attempts)}</span>
                      </span>
                      <span>
                        Status code:{" "}
                        <span className="text-foreground">
                          {dash(d.last_status_code)}
                        </span>
                      </span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {fmtTime(d.created_at)}
                      </span>
                      <span>
                        Next retry:{" "}
                        <span className="text-foreground">
                          {fmtTime(d.next_retry_at)}
                        </span>
                      </span>
                    </div>
                    {d.last_error ? (
                      <p className="mt-2 break-words rounded bg-red-50 px-2 py-1 font-mono text-xs text-red-700">
                        {d.last_error}
                      </p>
                    ) : null}
                  </Card>
                ))}
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>
      <ConfirmDialog
        open={!!deleteWebhookTarget}
        onOpenChange={(open) => !open && setDeleteWebhookTarget(null)}
        title="Delete this webhook endpoint?"
        description="Event deliveries to this URL stop immediately. Past delivery history is kept."
        confirmLabel="Delete endpoint"
        destructive
        onConfirm={handleDeleteWebhook}
      />
      <ConfirmDialog
        open={!!revokeKeyTarget}
        onOpenChange={(open) => !open && setRevokeKeyTarget(null)}
        title={`Revoke ${revokeKeyTarget?.key_prefix ? `${revokeKeyTarget.key_prefix}…` : "this key"}?`}
        description="The key stops authenticating immediately and cannot be restored. Anything using it will start receiving 401s."
        confirmLabel="Revoke key"
        destructive
        busy={revokingKey}
        onConfirm={handleRevokeKey}
      />
    </div>
  );
}

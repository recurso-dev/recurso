import { useState } from "react";
import { useNavigate } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Megaphone } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { formatDate } from "@/lib/utils";
import { useTableSort, sortRows } from "@/lib/tableSort";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormSheet } from "@/components/patterns/FormSheet";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const TRIGGERS = [
  { value: "payment_failed", label: "Payment failed" },
  { value: "invoice_overdue", label: "Invoice overdue" },
];

const triggerLabel = (v) => TRIGGERS.find((t) => t.value === v)?.label || v;

const DunningCampaigns = () => {
  const navigate = useNavigate();
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", trigger_event: "payment_failed" });
  const queryClient = useQueryClient();

  const {
    data: campaigns = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["dunning-campaigns"],
    queryFn: async () => {
      const res = await api.getDunningCampaigns();
      return Array.isArray(res.data) ? res.data : res.data?.data || [];
    },
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load campaigns"
    : null;

  const createMutation = useMutation({
    mutationFn: (form) => api.createDunningCampaign(form),
    onSuccess: (res) => {
      setCreateOpen(false);
      setCreateForm({ name: "", trigger_event: "payment_failed" });
      queryClient.invalidateQueries({ queryKey: ["dunning-campaigns"] });
      // Jump straight into the new campaign to configure its steps.
      if (res.data?.id) navigate(`/dunning/campaigns/${res.data.id}`);
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to create campaign"),
  });
  const creating = createMutation.isPending;

  const submitCreate = () => {
    if (!createForm.name.trim()) return;
    createMutation.mutate(createForm);
  };

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New campaign
    </Button>
  );

  // Fully-loaded list (one fetch, no server paging), so sorting the complete
  // set client-side is honest; the sort persists in the URL (Batch F3).
  const { sort, onSortChange } = useTableSort();
  const columns = [
    {
      key: "name",
      header: "Campaign",
      sortable: true,
      sortValue: (c) => c.name,
      cell: (c) => <span className="font-medium text-foreground">{c.name}</span>,
    },
    {
      key: "trigger",
      header: "Trigger",
      cell: (c) => <span className="text-muted-foreground">{triggerLabel(c.trigger_event)}</span>,
    },
    {
      key: "status",
      header: "Status",
      sortable: true,
      sortValue: (c) => (c.is_active ? "active" : "inactive"),
      cell: (c) => <StatusBadge status={c.is_active ? "active" : "inactive"} />,
    },
    {
      key: "created",
      header: "Created",
      sortable: true,
      sortValue: (c) => (c.created_at ? Date.parse(c.created_at) || null : null),
      cell: (c) => <span className="text-muted-foreground">{formatDate(c.created_at)}</span>,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Dunning campaigns"
        description="Configure the sequence of reminders sent when a payment fails or an invoice goes overdue."
        actions={createButton}
      />

      <DataTable
        columns={columns}
        data={sortRows(campaigns, sort, columns)}
        sort={sort}
        onSortChange={onSortChange}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(c) => `/dunning/campaigns/${c.id}`}
        empty={{
          icon: Megaphone,
          title: "No campaigns yet",
          description:
            "Create a campaign to define how customers are reminded about failed payments.",
          action: createButton,
        }}
      />

      <FormSheet
        open={createOpen}
        onOpenChange={setCreateOpen}
        title="New dunning campaign"
        description="Steps run in order after the trigger until payment is recovered."
        onSubmit={submitCreate}
        submitLabel="Create campaign"
        busyLabel="Creating…"
        busy={creating}
        canSubmit={Boolean(createForm.name.trim())}
        dirty={Boolean(createForm.name)}
      >
        <div>
          <Label htmlFor="name">Name</Label>
          <Input id="name"
            value={createForm.name}
            onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
            placeholder="Failed payment recovery"
          />
        </div>
        <div>
          <Label htmlFor="trigger">Trigger</Label>
          <Select
            value={createForm.trigger_event}
            onValueChange={(v) => setCreateForm({ ...createForm, trigger_event: v })}
          >
            <SelectTrigger id="trigger">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TRIGGERS.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </FormSheet>
    </div>
  );
};

export default DunningCampaigns;

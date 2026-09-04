import { useState } from "react";
import { useNavigate } from "react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, HeartHandshake } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { useTableSort, sortRows } from "@/lib/tableSort";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormSheet } from "@/components/patterns/FormSheet";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const CancelFlows = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", is_default: false, cooldown_days: 30 });

  const {
    data: flows = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["cancel-flows"],
    queryFn: async () => {
      const res = await api.getCancelFlows();
      return Array.isArray(res.data) ? res.data : res.data?.data || [];
    },
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load cancel flows"
    : null;

  const createMutation = useMutation({
    mutationFn: (payload) => api.createCancelFlow(payload),
    onSuccess: (res) => {
      setCreateOpen(false);
      setCreateForm({ name: "", is_default: false, cooldown_days: 30 });
      queryClient.invalidateQueries({ queryKey: ["cancel-flows"] });
      // Jump straight into the new flow to configure its steps.
      if (res.data?.id) navigate(`/cancel-flows/${res.data.id}`);
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to create flow"),
  });
  const creating = createMutation.isPending;

  const submitCreate = () => {
    if (!createForm.name.trim()) return;
    createMutation.mutate({
      ...createForm,
      cooldown_days: Number(createForm.cooldown_days) || 30,
    });
  };

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New flow
    </Button>
  );

  // Fully-loaded list (one fetch, no server paging), so sorting the complete
  // set client-side is honest; the sort persists in the URL (Batch F3).
  const { sort, onSortChange } = useTableSort();
  const columns = [
    {
      key: "name",
      header: "Flow",
      sortable: true,
      sortValue: (f) => f.name,
      cell: (f) => (
        <span className="flex items-center gap-2">
          <span className="font-medium text-foreground">{f.name}</span>
          {f.is_default && <Badge variant="info">Default</Badge>}
        </span>
      ),
    },
    {
      key: "cooldown",
      header: "Cooldown",
      sortable: true,
      sortValue: (f) => Number(f.cooldown_days) || 0,
      cell: (f) => <span className="text-muted-foreground">{f.cooldown_days} days</span>,
    },
    {
      key: "status",
      header: "Status",
      sortable: true,
      sortValue: (f) => (f.is_active ? "active" : "inactive"),
      cell: (f) => <StatusBadge status={f.is_active ? "active" : "inactive"} />,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Cancel Flows"
        description="Design the survey, retention offers, and confirmation a customer sees when they try to cancel."
        actions={createButton}
      />

      <DataTable
        columns={columns}
        data={sortRows(flows, sort, columns)}
        sort={sort}
        onSortChange={onSortChange}
        loading={loading}
        error={error}
        onRetry={refetch}
        rowHref={(f) => `/cancel-flows/${f.id}`}
        empty={{
          icon: HeartHandshake,
          title: "No cancellation flows yet",
          description: "Create a flow to try to retain customers before they cancel.",
          action: createButton,
        }}
      />

      <FormSheet
        open={createOpen}
        onOpenChange={setCreateOpen}
        title="New cancellation flow"
        description="The retention steps a customer walks through before canceling."
        onSubmit={submitCreate}
        submitLabel="Create flow"
        busyLabel="Creating…"
        busy={creating}
        canSubmit={Boolean(createForm.name.trim())}
        dirty={Boolean(createForm.name)}
      >
        <div>
          <Label htmlFor="flow-name">Name</Label>
          <Input id="flow-name"
            value={createForm.name}
            onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
            placeholder="Standard retention flow"
          />
        </div>
        <div>
          <Label htmlFor="cooldown-days">Cooldown (days)</Label>
          <Input id="cooldown-days"
            type="number"
            min="0"
            value={createForm.cooldown_days}
            onChange={(e) => setCreateForm({ ...createForm, cooldown_days: e.target.value })}
          />
          <p className="mt-1 text-xs text-muted-foreground">
            Minimum days before the same customer sees this flow again.
          </p>
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            className="h-4 w-4 rounded border-input accent-primary"
            checked={createForm.is_default}
            onChange={(e) => setCreateForm({ ...createForm, is_default: e.target.checked })}
          />
          Use as the default flow
        </label>
      </FormSheet>
    </div>
  );
};

export default CancelFlows;

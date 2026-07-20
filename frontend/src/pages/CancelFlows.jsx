import { useEffect, useState } from "react";
import { Plus, HeartHandshake, Settings2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { EmptyState } from "@/components/patterns/EmptyState";
import { ErrorState } from "@/components/patterns/ErrorState";
import { CardGridSkeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import CancelFlowDetail from "@/components/slide-overs/CancelFlowDetail";

const CancelFlows = () => {
  const [flows, setFlows] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", is_default: false, cooldown_days: 30 });
  const [creating, setCreating] = useState(false);
  const [detailId, setDetailId] = useState(null);

  const fetchFlows = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getCancelFlows();
      setFlows(Array.isArray(res.data) ? res.data : res.data?.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load cancel flows");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchFlows();
  }, []);

  const submitCreate = async () => {
    if (!createForm.name.trim()) return;
    setCreating(true);
    try {
      const res = await api.createCancelFlow({
        ...createForm,
        cooldown_days: Number(createForm.cooldown_days) || 30,
      });
      setCreateOpen(false);
      setCreateForm({ name: "", is_default: false, cooldown_days: 30 });
      await fetchFlows();
      if (res.data?.id) setDetailId(res.data.id);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to create flow");
    } finally {
      setCreating(false);
    }
  };

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New flow
    </Button>
  );

  return (
    <div>
      <PageHeader
        title="Cancellation flows"
        description="Design the survey, retention offers, and confirmation a customer sees when they try to cancel."
        actions={createButton}
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={fetchFlows} />
      ) : flows.length === 0 ? (
        <EmptyState
          icon={HeartHandshake}
          title="No cancellation flows yet"
          description="Create a flow to try to retain customers before they cancel."
          action={createButton}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {flows.map((f) => (
            <Card key={f.id}>
              <CardContent className="flex flex-col gap-3 p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-semibold text-foreground">{f.name}</p>
                  <div className="flex shrink-0 gap-1">
                    {f.is_default && <Badge variant="info">Default</Badge>}
                    <Badge variant={f.is_active ? "success" : "neutral"}>
                      {f.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">Cooldown: {f.cooldown_days} days</p>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-1 self-start"
                  onClick={() => setDetailId(f.id)}
                >
                  <Settings2 className="h-4 w-4" />
                  Configure steps
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>New cancellation flow</SheetTitle>
            <SheetDescription>
              The retention steps a customer walks through before canceling.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Name</Label>
              <Input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="Standard retention flow"
              />
            </div>
            <div>
              <Label>Cooldown (days)</Label>
              <Input
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
                className="h-4 w-4 rounded border-input accent-emerald-600"
                checked={createForm.is_default}
                onChange={(e) => setCreateForm({ ...createForm, is_default: e.target.checked })}
              />
              Use as the default flow
            </label>
          </div>
          <SheetFooter>
            <Button onClick={submitCreate} disabled={creating || !createForm.name.trim()}>
              {creating ? "Creating…" : "Create flow"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <CancelFlowDetail
        flowId={detailId}
        isOpen={!!detailId}
        onClose={() => setDetailId(null)}
        onChanged={fetchFlows}
      />
    </div>
  );
};

export default CancelFlows;

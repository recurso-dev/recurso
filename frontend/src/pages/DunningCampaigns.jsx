import { useEffect, useState } from "react";
import { Plus, Megaphone, Settings2 } from "lucide-react";

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import DunningCampaignDetail from "@/components/slide-overs/DunningCampaignDetail";

const TRIGGERS = [
  { value: "payment_failed", label: "Payment failed" },
  { value: "invoice_overdue", label: "Invoice overdue" },
];

const triggerLabel = (v) => TRIGGERS.find((t) => t.value === v)?.label || v;

const DunningCampaigns = () => {
  const [campaigns, setCampaigns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", trigger_event: "payment_failed" });
  const [creating, setCreating] = useState(false);
  const [detailId, setDetailId] = useState(null);

  const fetchCampaigns = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getDunningCampaigns();
      setCampaigns(Array.isArray(res.data) ? res.data : res.data?.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load campaigns");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCampaigns();
  }, []);

  const submitCreate = async () => {
    if (!createForm.name.trim()) return;
    setCreating(true);
    try {
      const res = await api.createDunningCampaign(createForm);
      setCreateOpen(false);
      setCreateForm({ name: "", trigger_event: "payment_failed" });
      await fetchCampaigns();
      // Jump straight into configuring steps for the new campaign.
      if (res.data?.id) setDetailId(res.data.id);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to create campaign");
    } finally {
      setCreating(false);
    }
  };

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New campaign
    </Button>
  );

  return (
    <div>
      <PageHeader
        title="Dunning campaigns"
        description="Configure the sequence of reminders sent when a payment fails or an invoice goes overdue."
        actions={createButton}
      />

      {loading ? (
        <CardGridSkeleton count={3} />
      ) : error ? (
        <ErrorState message={error} onRetry={fetchCampaigns} />
      ) : campaigns.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="No campaigns yet"
          description="Create a campaign to define how customers are reminded about failed payments."
          action={createButton}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {campaigns.map((c) => (
            <Card key={c.id}>
              <CardContent className="flex flex-col gap-3 p-5">
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-semibold text-foreground">{c.name}</p>
                  <Badge variant={c.is_active ? "success" : "neutral"}>
                    {c.is_active ? "Active" : "Inactive"}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  Trigger: {triggerLabel(c.trigger_event)}
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-1 self-start"
                  onClick={() => setDetailId(c.id)}
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
            <SheetTitle>New dunning campaign</SheetTitle>
            <SheetDescription>
              Steps run in order after the trigger until payment is recovered.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
            <div>
              <Label>Name</Label>
              <Input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="Failed payment recovery"
              />
            </div>
            <div>
              <Label>Trigger</Label>
              <Select
                value={createForm.trigger_event}
                onValueChange={(v) => setCreateForm({ ...createForm, trigger_event: v })}
              >
                <SelectTrigger>
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
          </div>
          <SheetFooter>
            <Button onClick={submitCreate} disabled={creating || !createForm.name.trim()}>
              {creating ? "Creating…" : "Create campaign"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <DunningCampaignDetail
        campaignId={detailId}
        isOpen={!!detailId}
        onClose={() => setDetailId(null)}
        onChanged={fetchCampaigns}
      />
    </div>
  );
};

export default DunningCampaigns;

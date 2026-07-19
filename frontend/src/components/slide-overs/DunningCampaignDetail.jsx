import { useEffect, useState } from "react";
import { Plus, Trash2, Power, Mail, MessageSquare, Bell } from "lucide-react";

import { endpoints as api } from "../../lib/api";
import { toast } from "@/components/ui/sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";

const CHANNELS = [
  { value: "email", label: "Email", icon: Mail, variant: "info" },
  { value: "sms", label: "SMS", icon: MessageSquare, variant: "secondary" },
  { value: "in_app", label: "In-app", icon: Bell, variant: "neutral" },
];

const channelMeta = (channel) => CHANNELS.find((c) => c.value === channel) || CHANNELS[0];

const emptyStep = { step_order: 1, channel: "email", delay_hours: 0, subject: "", body: "", template_name: "", is_payment_wall: false };

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

export default function DunningCampaignDetail({ campaignId, isOpen, onClose, onChanged }) {
  const [campaign, setCampaign] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [form, setForm] = useState(emptyStep);
  const [editingId, setEditingId] = useState(null);
  const [showForm, setShowForm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteStep, setDeleteStep] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const loadCampaign = async () => {
    if (!campaignId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.getDunningCampaign(campaignId);
      setCampaign(res.data);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load campaign");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen && campaignId) {
      setShowForm(false);
      setEditingId(null);
      loadCampaign();
    }
    // loadCampaign is stable for a given campaignId; re-running on open is intended.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, campaignId]);

  const steps = [...(campaign?.steps || [])].sort((a, b) => a.step_order - b.step_order);

  const startAdd = () => {
    const nextOrder = steps.length ? Math.max(...steps.map((s) => s.step_order)) + 1 : 1;
    setForm({ ...emptyStep, step_order: nextOrder });
    setEditingId(null);
    setShowForm(true);
  };

  const startEdit = (step) => {
    setForm({
      step_order: step.step_order,
      channel: step.channel,
      delay_hours: step.delay_hours,
      subject: step.subject || "",
      body: step.body || "",
      template_name: step.template_name || "",
      is_payment_wall: step.is_payment_wall,
    });
    setEditingId(step.id);
    setShowForm(true);
  };

  const toggleActive = async () => {
    try {
      await api.updateDunningCampaign(campaign.id, { is_active: !campaign.is_active });
      await loadCampaign();
      onChanged?.();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to update campaign");
    }
  };

  const saveStep = async () => {
    setSaving(true);
    try {
      const payload = {
        step_order: Number(form.step_order),
        channel: form.channel,
        delay_hours: Number(form.delay_hours) || 0,
        subject: form.subject,
        body: form.body,
        template_name: form.template_name,
        is_payment_wall: form.is_payment_wall,
      };
      if (editingId) {
        await api.updateDunningStep(editingId, payload);
        toast.success("Step updated.");
      } else {
        await api.createDunningStep(campaign.id, payload);
        toast.success("Step added.");
      }
      setShowForm(false);
      setEditingId(null);
      await loadCampaign();
      onChanged?.();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to save step");
    } finally {
      setSaving(false);
    }
  };

  const confirmDelete = async () => {
    if (!deleteStep) return;
    setDeleting(true);
    try {
      await api.deleteDunningStep(deleteStep.id);
      toast.success("Step removed.");
      setDeleteStep(null);
      await loadCampaign();
      onChanged?.();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to delete step");
    } finally {
      setDeleting(false);
    }
  };

  return (
    <>
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{campaign?.name || "Campaign"}</SheetTitle>
          <SheetDescription>
            Steps run in order after the trigger, each waiting its delay before sending.
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 py-6">
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : error ? (
            <p className="text-sm text-red-600">{error}</p>
          ) : campaign ? (
            <>
              <div className="mb-6 flex items-center justify-between rounded-md border border-border bg-muted/30 px-4 py-3">
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {campaign.is_active ? "Active" : "Inactive"}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Trigger: {campaign.trigger_event}
                  </p>
                </div>
                <Button variant="outline" size="sm" onClick={toggleActive}>
                  <Power className="h-4 w-4" />
                  {campaign.is_active ? "Deactivate" : "Activate"}
                </Button>
              </div>

              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-foreground">Steps</h3>
                {!showForm && (
                  <Button size="sm" variant="outline" onClick={startAdd}>
                    <Plus className="h-4 w-4" />
                    Add step
                  </Button>
                )}
              </div>

              {steps.length === 0 && !showForm && (
                <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-sm text-muted-foreground">
                  No steps yet. Add the first outreach step.
                </p>
              )}

              <ol className="space-y-2">
                {steps.map((step) => {
                  const meta = channelMeta(step.channel);
                  const Icon = meta.icon;
                  return (
                    <li
                      key={step.id}
                      className="flex items-start justify-between gap-3 rounded-md border border-border px-4 py-3"
                    >
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-semibold text-muted-foreground">
                            #{step.step_order}
                          </span>
                          <Badge variant={meta.variant}>
                            <Icon className="h-3 w-3" />
                            {meta.label}
                          </Badge>
                          {step.is_payment_wall && <Badge variant="warning">Payment wall</Badge>}
                        </div>
                        <p className="mt-1 text-sm text-foreground">
                          {step.subject || step.template_name || "—"}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {step.delay_hours}h after {step.step_order === 1 ? "trigger" : "previous step"}
                        </p>
                      </div>
                      <div className="flex shrink-0 gap-1">
                        <Button size="sm" variant="ghost" onClick={() => startEdit(step)}>
                          Edit
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-red-600 hover:text-red-600"
                          onClick={() => setDeleteStep(step)}
                          aria-label="Delete step"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </li>
                  );
                })}
              </ol>

              {showForm && (
                <div className="mt-4 space-y-3 rounded-md border border-border bg-muted/30 p-4">
                  <h4 className="text-sm font-medium text-foreground">
                    {editingId ? "Edit step" : "New step"}
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <Label>Order</Label>
                      <Input
                        type="number"
                        min="1"
                        value={form.step_order}
                        onChange={(e) => setForm({ ...form, step_order: e.target.value })}
                      />
                    </div>
                    <div>
                      <Label>Delay (hours)</Label>
                      <Input
                        type="number"
                        min="0"
                        value={form.delay_hours}
                        onChange={(e) => setForm({ ...form, delay_hours: e.target.value })}
                      />
                    </div>
                  </div>
                  <div>
                    <Label>Channel</Label>
                    <Select
                      value={form.channel}
                      onValueChange={(v) => setForm({ ...form, channel: v })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {CHANNELS.map((c) => (
                          <SelectItem key={c.value} value={c.value}>
                            {c.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label>Subject</Label>
                    <Input
                      value={form.subject}
                      onChange={(e) => setForm({ ...form, subject: e.target.value })}
                      placeholder="Your payment is overdue"
                    />
                  </div>
                  <div>
                    <Label>Body</Label>
                    <textarea
                      className={textareaClass}
                      rows={3}
                      value={form.body}
                      onChange={(e) => setForm({ ...form, body: e.target.value })}
                      placeholder="Message body…"
                    />
                  </div>
                  <div>
                    <Label>Template name (optional)</Label>
                    <Input
                      value={form.template_name}
                      onChange={(e) => setForm({ ...form, template_name: e.target.value })}
                      placeholder="overdue_reminder"
                    />
                  </div>
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      className="h-4 w-4 rounded border-input accent-emerald-600"
                      checked={form.is_payment_wall}
                      onChange={(e) => setForm({ ...form, is_payment_wall: e.target.checked })}
                    />
                    Show a payment wall at this step
                  </label>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={saveStep} disabled={saving || !form.step_order}>
                      {saving ? "Saving…" : editingId ? "Save step" : "Add step"}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setShowForm(false);
                        setEditingId(null);
                      }}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              )}
            </>
          ) : null}
        </div>
      </SheetContent>
    </Sheet>

      <ConfirmDialog
        open={!!deleteStep}
        onOpenChange={(o) => !o && setDeleteStep(null)}
        title="Delete this step?"
        description="It is removed from the campaign immediately. In-flight executions are unaffected."
        confirmLabel="Delete step"
        destructive
        busy={deleting}
        onConfirm={confirmDelete}
      />
    </>
  );
}

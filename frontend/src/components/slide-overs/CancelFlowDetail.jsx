import { useEffect, useState } from "react";
import { Plus, Trash2, Power, ClipboardList, Gift, ShieldQuestion } from "lucide-react";

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
import { CancelFlowStepConfig } from "./CancelFlowStepConfig";
import { defaultConfigFor } from "./cancelFlowConfig";

const STEP_TYPES = [
  { value: "survey", label: "Survey", icon: ClipboardList, variant: "info" },
  { value: "offer", label: "Offer", icon: Gift, variant: "success" },
  { value: "confirmation", label: "Confirmation", icon: ShieldQuestion, variant: "warning" },
];

const stepMeta = (type) => STEP_TYPES.find((t) => t.value === type) || STEP_TYPES[0];

const stepSummary = (step) => {
  const cfg = step.config || {};
  if (step.step_type === "survey") return `${(cfg.questions || []).length} reasons`;
  if (step.step_type === "offer") return cfg.headline || `${(cfg.offers || []).length} offers`;
  if (step.step_type === "confirmation") return cfg.message || "Confirm cancellation";
  return "";
};

const pct = (v) => (v == null ? "—" : `${Math.round(v * 100)}%`);

export default function CancelFlowDetail({ flowId, isOpen, onClose, onChanged }) {
  const [flow, setFlow] = useState(null);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [form, setForm] = useState(null); // { step_order, step_type, config } or null
  const [editingId, setEditingId] = useState(null);
  const [saving, setSaving] = useState(false);
  const [deleteStep, setDeleteStep] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const load = async () => {
    if (!flowId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await api.getCancelFlow(flowId);
      setFlow(res.data);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load flow");
    } finally {
      setLoading(false);
    }
    try {
      const s = await api.getCancelFlowStats(flowId);
      setStats(s.data);
    } catch {
      setStats(null);
    }
  };

  useEffect(() => {
    if (isOpen && flowId) {
      setForm(null);
      setEditingId(null);
      load();
    }
    // load is stable for a given flowId; re-running on open is intended.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, flowId]);

  const steps = [...(flow?.steps || [])].sort((a, b) => a.step_order - b.step_order);

  const startAdd = () => {
    const nextOrder = steps.length ? Math.max(...steps.map((s) => s.step_order)) + 1 : 0;
    setForm({ step_order: nextOrder, step_type: "survey", config: defaultConfigFor("survey") });
    setEditingId(null);
  };

  const startEdit = (step) => {
    setForm({ step_order: step.step_order, step_type: step.step_type, config: step.config || {} });
    setEditingId(step.id);
  };

  const changeType = (type) =>
    setForm((f) => ({ ...f, step_type: type, config: defaultConfigFor(type) }));

  const updateFlow = async (patch) => {
    try {
      await api.updateCancelFlow(flow.id, patch);
      await load();
      onChanged?.();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to update flow");
    }
  };

  const saveStep = async () => {
    setSaving(true);
    try {
      const payload = {
        step_order: Number(form.step_order),
        step_type: form.step_type,
        config: form.config,
      };
      if (editingId) {
        await api.updateCancelFlowStep(editingId, payload);
        toast.success("Step updated.");
      } else {
        await api.createCancelFlowStep(flow.id, payload);
        toast.success("Step added.");
      }
      setForm(null);
      setEditingId(null);
      await load();
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
      await api.deleteCancelFlowStep(deleteStep.id);
      toast.success("Step removed.");
      setDeleteStep(null);
      await load();
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
            <SheetTitle>{flow?.name || "Cancel flow"}</SheetTitle>
            <SheetDescription>
              Steps are shown to a customer, in order, when they try to cancel.
            </SheetDescription>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto px-6 py-6">
            {loading ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : error ? (
              <p className="text-sm text-red-600">{error}</p>
            ) : flow ? (
              <>
                <div className="mb-4 flex flex-wrap items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => updateFlow({ is_active: !flow.is_active })}>
                    <Power className="h-4 w-4" />
                    {flow.is_active ? "Deactivate" : "Activate"}
                  </Button>
                  <Button
                    variant={flow.is_default ? "secondary" : "outline"}
                    size="sm"
                    onClick={() => updateFlow({ is_default: !flow.is_default })}
                  >
                    {flow.is_default ? "Default flow" : "Set as default"}
                  </Button>
                  <span className="text-xs text-muted-foreground">
                    Cooldown: {flow.cooldown_days}d
                  </span>
                </div>

                {stats && (
                  <div className="mb-6 grid grid-cols-3 gap-2 rounded-md border border-border bg-muted/30 p-3 text-center">
                    <div>
                      <p className="text-lg font-semibold tabular-nums text-foreground">
                        {stats.completed_count ?? 0}
                      </p>
                      <p className="text-xs text-muted-foreground">Completed</p>
                    </div>
                    <div>
                      <p className="text-lg font-semibold tabular-nums text-emerald-600">
                        {stats.saved_count ?? 0}
                      </p>
                      <p className="text-xs text-muted-foreground">Saved</p>
                    </div>
                    <div>
                      <p className="text-lg font-semibold tabular-nums text-foreground">
                        {pct(stats.save_rate)}
                      </p>
                      <p className="text-xs text-muted-foreground">Save rate</p>
                    </div>
                  </div>
                )}

                <div className="mb-3 flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-foreground">Steps</h3>
                  {!form && (
                    <Button size="sm" variant="outline" onClick={startAdd}>
                      <Plus className="h-4 w-4" />
                      Add step
                    </Button>
                  )}
                </div>

                {steps.length === 0 && !form && (
                  <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-sm text-muted-foreground">
                    No steps yet. Add a survey, an offer, or a confirmation.
                  </p>
                )}

                <ol className="space-y-2">
                  {steps.map((step) => {
                    const meta = stepMeta(step.step_type);
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
                          </div>
                          <p className="mt-1 truncate text-sm text-foreground">{stepSummary(step)}</p>
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

                {form && (
                  <div className="mt-4 space-y-3 rounded-md border border-border bg-muted/30 p-4">
                    <h4 className="text-sm font-medium text-foreground">
                      {editingId ? "Edit step" : "New step"}
                    </h4>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <Label>Order</Label>
                        <Input
                          type="number"
                          min="0"
                          value={form.step_order}
                          onChange={(e) => setForm({ ...form, step_order: e.target.value })}
                        />
                      </div>
                      <div>
                        <Label>Type</Label>
                        <Select value={form.step_type} onValueChange={changeType}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {STEP_TYPES.map((t) => (
                              <SelectItem key={t.value} value={t.value}>
                                {t.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    <CancelFlowStepConfig
                      stepType={form.step_type}
                      config={form.config}
                      onChange={(config) => setForm({ ...form, config })}
                    />

                    <div className="flex gap-2 pt-1">
                      <Button size="sm" onClick={saveStep} disabled={saving}>
                        {saving ? "Saving…" : editingId ? "Save step" : "Add step"}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setForm(null);
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
        description="It is removed from the flow immediately. In-flight sessions are unaffected."
        confirmLabel="Delete step"
        destructive
        busy={deleting}
        onConfirm={confirmDelete}
      />
    </>
  );
}

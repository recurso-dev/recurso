import { Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { usePlans } from "@/lib/useCustomers";
import { OFFER_TYPES } from "./cancelFlowConfig";

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

// Plan picker for a plan-switch offer — backed by the shared plans cache so the
// operator chooses by name rather than pasting a UUID.
function PlanSwitchField({ value, onChange }) {
  const { plans, isLoading } = usePlans();
  return (
    <div>
      <Label className="text-xs" htmlFor="switch-to-plan-id">Switch to plan</Label>
      <Select value={value || ""} onValueChange={onChange}>
        <SelectTrigger id="switch-to-plan-id">
          <SelectValue placeholder={isLoading ? "Loading plans…" : "Select a plan"} />
        </SelectTrigger>
        <SelectContent>
          {plans.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              {p.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

// Fields shown for a single retention offer, keyed by its type.
function OfferFields({ offer, onChange }) {
  const set = (patch) => onChange({ ...offer, ...patch });
  switch (offer.type) {
    case "discount":
      return (
        <div className="grid grid-cols-2 gap-2">
          <div>
            <Label className="text-xs" htmlFor="percent-off">Percent off</Label>
            <Input id="percent-off"
              type="number"
              min="1"
              max="100"
              value={offer.discount_percent ?? ""}
              onChange={(e) => set({ discount_percent: Number(e.target.value) })}
            />
          </div>
          <div>
            <Label className="text-xs" htmlFor="for-months">For (months)</Label>
            <Input id="for-months"
              type="number"
              min="1"
              value={offer.discount_duration_months ?? ""}
              onChange={(e) => set({ discount_duration_months: Number(e.target.value) })}
            />
          </div>
        </div>
      );
    case "pause":
      return (
        <div>
          <Label className="text-xs" htmlFor="pause-months">Pause (months)</Label>
          <Input id="pause-months"
            type="number"
            min="1"
            value={offer.pause_months ?? ""}
            onChange={(e) => set({ pause_months: Number(e.target.value) })}
          />
        </div>
      );
    case "trial_extension":
      return (
        <div>
          <Label className="text-xs" htmlFor="extend-days">Extend (days)</Label>
          <Input id="extend-days"
            type="number"
            min="1"
            value={offer.extension_days ?? ""}
            onChange={(e) => set({ extension_days: Number(e.target.value) })}
          />
        </div>
      );
    case "plan_switch":
      return (
        <PlanSwitchField
          value={offer.switch_to_plan_id}
          onChange={(v) => set({ switch_to_plan_id: v })}
        />
      );
    default:
      return null;
  }
}

// Controlled editor for a step's `config` object. `config` is a plain object;
// `onChange` receives the next object. Shape depends on `stepType`.
export function CancelFlowStepConfig({ stepType, config, onChange }) {
  const set = (patch) => onChange({ ...config, ...patch });

  if (stepType === "survey") {
    return (
      <div className="space-y-3">
        <div>
          <Label htmlFor="reasons-one-per-line">Reasons (one per line)</Label>
          <Textarea id="reasons-one-per-line"
            className={textareaClass}
            rows={4}
            value={(config.questions || []).join("\n")}
            onChange={(e) =>
              set({ questions: e.target.value.split("\n").map((q) => q.trim()).filter(Boolean) })
            }
            placeholder={"Too expensive\nMissing features\nOther"}
          />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            className="h-4 w-4 rounded border-input accent-primary"
            checked={!!config.allow_feedback}
            onChange={(e) => set({ allow_feedback: e.target.checked })}
          />
          Allow free-text feedback
        </label>
      </div>
    );
  }

  if (stepType === "offer") {
    const offers = config.offers || [];
    const updateOffer = (i, next) =>
      set({ offers: offers.map((o, idx) => (idx === i ? next : o)) });
    const removeOffer = (i) => set({ offers: offers.filter((_, idx) => idx !== i) });
    const addOffer = () => set({ offers: [...offers, { type: "discount", discount_percent: 10 }] });

    return (
      <div className="space-y-3">
        <div>
          <Label htmlFor="headline">Headline</Label>
          <Input id="headline"
            value={config.headline || ""}
            onChange={(e) => set({ headline: e.target.value })}
            placeholder="Before you go…"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="offers">Offers</Label>
          {offers.map((offer, i) => (
            <div key={i} className="space-y-2 rounded-md border border-border p-3">
              <div className="flex items-center gap-2">
                <Select value={offer.type} onValueChange={(v) => updateOffer(i, { type: v })}>
                  <SelectTrigger id="offers" className="h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {OFFER_TYPES.map((t) => (
                      <SelectItem key={t.value} value={t.value}>
                        {t.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="text-destructive hover:text-destructive"
                  onClick={() => removeOffer(i)}
                  aria-label="Remove offer"
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
              <OfferFields offer={offer} onChange={(next) => updateOffer(i, next)} />
            </div>
          ))}
          <Button type="button" size="sm" variant="outline" onClick={addOffer}>
            <Plus className="h-4 w-4" />
            Add offer
          </Button>
        </div>
      </div>
    );
  }

  // confirmation
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="message">Message</Label>
        <Input id="message"
          value={config.message || ""}
          onChange={(e) => set({ message: e.target.value })}
          placeholder="Are you sure you want to cancel?"
        />
      </div>
      <div>
        <Label htmlFor="confirm-button">Confirm button</Label>
        <Input id="confirm-button"
          value={config.confirm_button || ""}
          onChange={(e) => set({ confirm_button: e.target.value })}
          placeholder="Yes, cancel"
        />
      </div>
      <div>
        <Label htmlFor="cancel-button">Cancel button</Label>
        <Input id="cancel-button"
          value={config.cancel_button || ""}
          onChange={(e) => set({ cancel_button: e.target.value })}
          placeholder="No, keep my subscription"
        />
      </div>
    </div>
  );
}

export default CancelFlowStepConfig;

import { Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { OFFER_TYPES } from "./cancelFlowConfig";

const textareaClass =
  "flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

// Fields shown for a single retention offer, keyed by its type.
function OfferFields({ offer, onChange }) {
  const set = (patch) => onChange({ ...offer, ...patch });
  switch (offer.type) {
    case "discount":
      return (
        <div className="grid grid-cols-2 gap-2">
          <div>
            <Label className="text-xs">Percent off</Label>
            <Input
              type="number"
              min="1"
              max="100"
              value={offer.discount_percent ?? ""}
              onChange={(e) => set({ discount_percent: Number(e.target.value) })}
            />
          </div>
          <div>
            <Label className="text-xs">For (months)</Label>
            <Input
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
          <Label className="text-xs">Pause (months)</Label>
          <Input
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
          <Label className="text-xs">Extend (days)</Label>
          <Input
            type="number"
            min="1"
            value={offer.extension_days ?? ""}
            onChange={(e) => set({ extension_days: Number(e.target.value) })}
          />
        </div>
      );
    case "plan_switch":
      return (
        <div>
          <Label className="text-xs">Switch to plan ID</Label>
          <Input
            value={offer.switch_to_plan_id ?? ""}
            onChange={(e) => set({ switch_to_plan_id: e.target.value })}
            placeholder="plan uuid"
          />
        </div>
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
          <Label>Reasons (one per line)</Label>
          <textarea
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
            className="h-4 w-4 rounded border-input accent-emerald-600"
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
          <Label>Headline</Label>
          <Input
            value={config.headline || ""}
            onChange={(e) => set({ headline: e.target.value })}
            placeholder="Before you go…"
          />
        </div>
        <div className="space-y-2">
          <Label>Offers</Label>
          {offers.map((offer, i) => (
            <div key={i} className="space-y-2 rounded-md border border-border p-3">
              <div className="flex items-center gap-2">
                <Select value={offer.type} onValueChange={(v) => updateOffer(i, { type: v })}>
                  <SelectTrigger className="h-8">
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
                  className="text-red-600 hover:text-red-600"
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
        <Label>Message</Label>
        <Input
          value={config.message || ""}
          onChange={(e) => set({ message: e.target.value })}
          placeholder="Are you sure you want to cancel?"
        />
      </div>
      <div>
        <Label>Confirm button</Label>
        <Input
          value={config.confirm_button || ""}
          onChange={(e) => set({ confirm_button: e.target.value })}
          placeholder="Yes, cancel"
        />
      </div>
      <div>
        <Label>Cancel button</Label>
        <Input
          value={config.cancel_button || ""}
          onChange={(e) => set({ cancel_button: e.target.value })}
          placeholder="No, keep my subscription"
        />
      </div>
    </div>
  );
}

export default CancelFlowStepConfig;

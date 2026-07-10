import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { endpoints } from "../lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// BuyGiftModal creates a gift subscription code on behalf of a buyer customer.
const BuyGiftModal = ({ isOpen, onClose, plans, onSuccess }) => {
  const [buyerCustomerId, setBuyerCustomerId] = useState("");
  const [planId, setPlanId] = useState(plans[0]?.id || "");
  const [durationMonths, setDurationMonths] = useState("12");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [giftCode, setGiftCode] = useState(null);
  const [copied, setCopied] = useState(false);

  const reset = () => {
    setError(null);
    setGiftCode(null);
    setCopied(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const response = await endpoints.purchaseGift({
        buyer_customer_id: buyerCustomerId,
        plan_id: planId,
        duration_months: parseInt(durationMonths, 10),
      });
      setGiftCode(response.data.code);
      if (onSuccess) onSuccess();
    } catch (err) {
      setError(err.response?.data?.error?.message || err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    if (!giftCode) return;
    navigator.clipboard.writeText(giftCode);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          reset();
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create a gift subscription</DialogTitle>
          <DialogDescription>
            Generates a redeemable gift code paid for by an existing customer.
          </DialogDescription>
        </DialogHeader>

        {giftCode ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-border bg-muted/40 p-4 text-center">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">
                Gift code
              </p>
              <p className="mt-1 font-mono text-lg font-semibold text-foreground">
                {giftCode}
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" size="sm" onClick={handleCopy}>
                {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                {copied ? "Copied" : "Copy code"}
              </Button>
              <Button size="sm" onClick={() => { reset(); onClose(); }}>
                Done
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="gift-buyer">Buyer customer ID</Label>
              <Input
                id="gift-buyer"
                required
                value={buyerCustomerId}
                onChange={(e) => setBuyerCustomerId(e.target.value)}
                placeholder="Customer UUID paying for the gift"
              />
            </div>
            <div className="space-y-1.5">
              <Label>Plan</Label>
              <Select value={planId} onValueChange={setPlanId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a plan" />
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
            <div className="space-y-1.5">
              <Label htmlFor="gift-months">Duration (months)</Label>
              <Input
                id="gift-months"
                type="number"
                min="1"
                max="36"
                required
                value={durationMonths}
                onChange={(e) => setDurationMonths(e.target.value)}
              />
            </div>
            {error && (
              <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={loading}>
                Cancel
              </Button>
              <Button type="submit" size="sm" disabled={loading}>
                {loading ? "Creating…" : "Create gift"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default BuyGiftModal;

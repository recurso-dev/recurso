import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { endpoints } from "../lib/api";
import { useCustomers } from "../lib/useCustomers";
import { CustomerSelect } from "@/components/patterns/CustomerSelect";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Overline } from "@/components/ui/overline";
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
  // Shared react-query customer cache (ADR-005) — never ask for a raw UUID.
  const { customers } = useCustomers();
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
              <Overline as="p">Gift code</Overline>
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
              <Label htmlFor="gift-buyer">Paying customer</Label>
              <CustomerSelect
                id="gift-buyer"
                value={buyerCustomerId}
                onChange={setBuyerCustomerId}
                customers={customers}
                placeholder="Select the customer paying for the gift"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="plan">Plan</Label>
              <Select value={planId} onValueChange={setPlanId}>
                <SelectTrigger id="plan">
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
              <p className="rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                {error}
              </p>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" size="sm" onClick={onClose} disabled={loading}>
                Cancel
              </Button>
              <Button type="submit" size="sm" disabled={loading || !buyerCustomerId}>
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

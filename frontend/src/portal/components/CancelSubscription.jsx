import { useState } from "react";
import { AlertTriangle, Check } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const cancellationReasons = [
  { id: "too_expensive", label: "Too expensive" },
  { id: "not_using", label: "Not using it enough" },
  { id: "missing_features", label: "Missing features I need" },
  { id: "switching_competitor", label: "Switching to a competitor" },
  { id: "temporary", label: "Just need a break" },
  { id: "other", label: "Other reason" },
];

const formatDate = (dateString) =>
  new Date(dateString).toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

const CancelSubscription = ({ subscription, onCancel, onClose }) => {
  const [step, setStep] = useState("confirm"); // confirm | reason | success
  const [reason, setReason] = useState("");
  const [otherReason, setOtherReason] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleCancel = async () => {
    setIsLoading(true);
    try {
      await onCancel({
        reason: reason === "other" ? otherReason : reason,
        subscriptionId: subscription.id,
      });
      setStep("success");
    } catch (error) {
      console.error("Cancellation failed:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const accessUntil = formatDate(subscription?.current_period_end);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {step === "confirm" && (
          <>
            <DialogHeader className="items-center text-center">
              <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-amber-50 text-amber-600">
                <AlertTriangle className="h-8 w-8" />
              </div>
              <DialogTitle className="mt-4">Cancel Subscription?</DialogTitle>
              <DialogDescription>
                You&apos;re about to cancel your {subscription?.plan_name} subscription.
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-1 rounded-lg bg-muted p-4 text-sm">
              <div className="flex items-center justify-between py-1">
                <span className="text-muted-foreground">Current Plan</span>
                <span className="font-medium text-foreground">
                  {subscription?.plan_name}
                </span>
              </div>
              <div className="flex items-center justify-between py-1">
                <span className="text-muted-foreground">Access Until</span>
                <span className="font-medium text-foreground">{accessUntil}</span>
              </div>
            </div>

            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm">
              <p className="font-semibold text-amber-800">After cancellation:</p>
              <ul className="mt-2 list-disc space-y-1 pl-5 text-amber-700">
                <li>You&apos;ll have access until {accessUntil}</li>
                <li>You won&apos;t be charged again</li>
                <li>You can reactivate anytime before the period ends</li>
              </ul>
            </div>

            <DialogFooter className="sm:justify-between">
              <Button variant="secondary" className="flex-1" onClick={onClose}>
                Keep Subscription
              </Button>
              <Button
                variant="destructive"
                className="flex-1"
                onClick={() => setStep("reason")}
              >
                Continue to Cancel
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "reason" && (
          <>
            <DialogHeader>
              <DialogTitle>Help us improve</DialogTitle>
              <DialogDescription>Why are you cancelling? (Optional)</DialogDescription>
            </DialogHeader>

            <div className="flex flex-col gap-2">
              {cancellationReasons.map((r) => (
                <label
                  key={r.id}
                  className={cn(
                    "flex cursor-pointer items-center gap-3 rounded-lg border px-4 py-3 text-sm text-foreground transition-colors",
                    reason === r.id
                      ? "border-foreground bg-muted"
                      : "border-border hover:border-input hover:bg-muted/50"
                  )}
                >
                  <input
                    type="radio"
                    name="reason"
                    value={r.id}
                    checked={reason === r.id}
                    onChange={(e) => setReason(e.target.value)}
                    className="text-primary focus:ring-ring"
                  />
                  <span>{r.label}</span>
                </label>
              ))}
            </div>

            {reason === "other" && (
              <textarea
                className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                placeholder="Please tell us more..."
                value={otherReason}
                onChange={(e) => setOtherReason(e.target.value)}
                rows={3}
              />
            )}

            <DialogFooter className="sm:justify-between">
              <Button
                variant="secondary"
                className="flex-1"
                onClick={() => setStep("confirm")}
              >
                Back
              </Button>
              <Button
                variant="destructive"
                className="flex-1"
                onClick={handleCancel}
                disabled={isLoading}
              >
                {isLoading ? "Cancelling..." : "Confirm Cancellation"}
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "success" && (
          <>
            <DialogHeader className="items-center text-center">
              <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
                <Check className="h-8 w-8" />
              </div>
              <DialogTitle className="mt-4">Subscription Cancelled</DialogTitle>
              <DialogDescription>We&apos;re sorry to see you go!</DialogDescription>
            </DialogHeader>

            <div className="rounded-lg bg-muted p-4 text-sm">
              <div className="flex items-center justify-between py-1">
                <span className="text-muted-foreground">Access Until</span>
                <span className="font-medium text-foreground">{accessUntil}</span>
              </div>
            </div>

            <p className="text-center text-sm leading-relaxed text-muted-foreground">
              You&apos;ll receive a confirmation email shortly. You can reactivate your
              subscription anytime from this portal.
            </p>

            <DialogFooter className="sm:justify-center">
              <Button onClick={onClose}>Done</Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default CancelSubscription;

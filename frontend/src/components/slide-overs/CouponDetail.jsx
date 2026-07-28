import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Power, PowerOff } from "lucide-react";

import { endpoints } from "../../lib/api";
import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

const Field = ({ label, children }) => (
  <div className="flex flex-col gap-1">
    <dt className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
      {label}
    </dt>
    <dd className="text-sm font-medium text-foreground">{children}</dd>
  </div>
);

// Build a readable discount from the API's discount_type + discount_value.
const discountLabel = (coupon) => {
  const { discount_type, discount_value, currency } = coupon;
  if (discount_type === "percent") return `${discount_value}% off`;
  if (discount_type === "fixed" || discount_type === "amount")
    return `${formatCurrency(discount_value, currency)} off`;
  return coupon.discount || "—";
};

const durationLabel = (coupon) =>
  coupon.duration === "repeating" && coupon.duration_in_months
    ? `For ${coupon.duration_in_months} months`
    : coupon.duration || "—";

const CouponDetail = ({ coupon, isOpen, onClose }) => {
  const queryClient = useQueryClient();
  // Optimistic local active state so the badge and button flip in place without
  // reopening the sheet. Reset whenever a different coupon is opened.
  const [activeOverride, setActiveOverride] = useState(null);
  useEffect(() => setActiveOverride(null), [coupon?.id]);

  const toggleMutation = useMutation({
    mutationFn: (next) => endpoints.setCouponActive(coupon.id, next),
    onSuccess: (_data, next) => {
      setActiveOverride(next);
      queryClient.invalidateQueries({ queryKey: ["coupons"] });
      toast.success(next ? "Coupon reactivated." : "Coupon deactivated.");
    },
    onError: (err) => {
      toast.error(err?.response?.data?.error?.message || "Failed to update coupon.");
    },
  });

  if (!coupon) return null;

  const isActive = activeOverride ?? coupon.active;
  const hasCap = coupon.max_redemptions != null && coupon.max_redemptions > 0;
  const progress = hasCap
    ? Math.round(((coupon.redemptions || 0) / coupon.max_redemptions) * 100)
    : 0;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-lg">{coupon.code}</span>
            <Badge variant="neutral">{discountLabel(coupon)}</Badge>
            <Badge variant={isActive ? "success" : "neutral"} className="capitalize">
              {isActive ? "active" : "inactive"}
            </Badge>
          </SheetTitle>
        </SheetHeader>

        <div className="space-y-6 px-6 py-6">
          {/* Redemptions progress — only when the coupon has a redemption cap */}
          {hasCap && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-foreground">Redemptions</p>
                <p className="text-sm text-muted-foreground tabular-nums">
                  {progress}%
                </p>
              </div>
              <div className="h-2 w-full rounded-full bg-stone-200">
                <div
                  className="h-2 rounded-full bg-primary transition-all duration-500"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                {coupon.redemptions || 0} of {coupon.max_redemptions} used
              </p>
              <Separator />
            </div>
          )}

          <dl className="grid grid-cols-2 gap-x-4 gap-y-5">
            <Field label="Discount">{discountLabel(coupon)}</Field>
            <Field label="Duration">
              <span className="capitalize">{durationLabel(coupon)}</span>
            </Field>
            <Field label="Created">
              {coupon.created_at ? formatDate(coupon.created_at) : "—"}
            </Field>
            {coupon.redemptions != null && !hasCap && (
              <Field label="Times redeemed">{coupon.redemptions}</Field>
            )}
          </dl>

          <Separator />

          <div>
            {isActive ? (
              <Button
                variant="outline"
                size="sm"
                className="text-red-600 hover:bg-red-50 hover:text-red-700"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(false)}
              >
                <PowerOff className="mr-2 h-4 w-4" />
                Deactivate coupon
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                disabled={toggleMutation.isPending}
                onClick={() => toggleMutation.mutate(true)}
              >
                <Power className="mr-2 h-4 w-4" />
                Reactivate coupon
              </Button>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CouponDetail;

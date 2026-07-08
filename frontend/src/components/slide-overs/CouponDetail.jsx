import { formatCurrency, formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
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
  if (!coupon) return null;

  const hasCap = coupon.max_redemptions != null && coupon.max_redemptions > 0;
  const progress = hasCap
    ? Math.round(((coupon.redemptions || 0) / coupon.max_redemptions) * 100)
    : 0;

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-3">
            <span className="font-mono text-lg">{coupon.code}</span>
            <Badge variant="success">{discountLabel(coupon)}</Badge>
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
              <div className="h-2 w-full rounded-full bg-zinc-200">
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
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default CouponDetail;

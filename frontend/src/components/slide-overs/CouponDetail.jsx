import { formatDate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet";

const statusVariant = (status) =>
  ({ active: "success", expired: "destructive" })[status] || "neutral";

const CouponDetail = ({ coupon, isOpen, onClose }) => {
  if (!coupon) return null;

  const progress = coupon.max_redemptions
    ? Math.round((coupon.redemptions / coupon.max_redemptions) * 100)
    : 0;

  const details = [
    { label: "Discount", value: coupon.discount },
    {
      label: "Duration",
      value:
        coupon.duration === "repeating"
          ? `For ${coupon.duration_in_months} months`
          : coupon.duration,
      capitalize: true,
    },
    { label: "Created date", value: formatDate(coupon.created_at) },
    { label: "Applies to", value: "All products" },
    { label: "Customer limit", value: "One per customer" },
  ];

  return (
    <Sheet open={isOpen} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="font-mono text-xl">{coupon.code}</SheetTitle>
          <div>
            <Badge variant={statusVariant(coupon.status)} className="capitalize">
              {coupon.status}
            </Badge>
          </div>
        </SheetHeader>

        <div className="flex-1 space-y-6 overflow-y-auto px-6 py-6">
          {/* Redemptions progress */}
          {coupon.max_redemptions ? (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-foreground">Redemptions</p>
                <p className="text-sm text-muted-foreground tabular-nums">{progress}%</p>
              </div>
              <div className="h-2 w-full rounded-full bg-zinc-200">
                <div
                  className="h-2 rounded-full bg-primary transition-all duration-500"
                  style={{ width: `${progress}%` }}
                />
              </div>
              <p className="text-sm text-muted-foreground">
                {coupon.redemptions} of {coupon.max_redemptions} used
              </p>
              <Separator />
            </div>
          ) : null}

          {/* Details */}
          <dl className="grid grid-cols-1 gap-x-4 gap-y-5 sm:grid-cols-2">
            {details.map((d) => (
              <div key={d.label} className="flex flex-col gap-1">
                <dt className="text-sm text-muted-foreground">{d.label}</dt>
                <dd
                  className={`text-sm font-medium text-foreground${d.capitalize ? " capitalize" : ""}`}
                >
                  {d.value}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        <SheetFooter>
          <Button variant="outline">Deactivate</Button>
          <Button>Edit coupon</Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
};

export default CouponDetail;

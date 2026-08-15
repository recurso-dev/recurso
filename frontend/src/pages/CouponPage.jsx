import { useState } from "react";
import { useParams } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Power, PowerOff } from "lucide-react";

import { endpoints } from "../lib/api";
import { useObjectQuery } from "@/lib/useObjectQuery";
import { formatDateTime, formatCurrency } from "@/lib/utils";
import { useSubscriptions, useCustomers } from "@/lib/useCustomers";
import { CustomerName } from "@/components/patterns/CustomerSelect";
import {
  ObjectHeader,
  ObjectPageLayout,
  ObjectSection,
  AttributeList,
  RelatedRow,
  RelatedEmpty,
  ObjectPageSkeleton,
  ObjectNotFound,
  ObjectPageError,
} from "@/components/patterns/ObjectPage";
import { StatusBadge } from "@/components/ui/status-badge";
import { Badge } from "@/components/ui/badge";
import { CopyableId } from "@/components/ui/copyable-id";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { toast } from "@/components/ui/sonner";

// Human-readable discount from the API's discount_type + discount_value.
const discountLabel = (c) =>
  c.discount_type === "percent"
    ? `${c.discount_value}% off`
    : `${formatCurrency(c.discount_value, c.currency)} off`;

// A plain-language sentence for what the coupon actually does over time.
const durationSentence = (c) => {
  const d = discountLabel(c);
  if (c.duration === "forever") return `${d}, applied to every invoice for the life of the subscription.`;
  if (c.duration === "once") return `${d}, applied once — to the first invoice only.`;
  if (c.duration === "repeating")
    return `${d}, applied for the first ${c.duration_months || "—"} month${c.duration_months === 1 ? "" : "s"}.`;
  return d;
};

/**
 * CouponPage — one discount code as a first-class object at /coupons/:id.
 * Replaces the detail slide-over (which referenced redemption fields the API
 * never returns). Shows only the real coupon fields, a plain-language summary
 * of what it does over time, the activate/deactivate gate, and — the genuine
 * depth — the subscriptions actually redeeming it (a real reverse lookup, not a
 * fabricated redemption counter).
 */
export default function CouponPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const { names } = useCustomers();
  const subscriptions = useSubscriptions();

  const [confirmDeactivate, setConfirmDeactivate] = useState(false);

  const {
    object: coupon,
    loading,
    notFound,
    isError,
    error: couponError,
    refetch,
  } = useObjectQuery(
    ["coupon", id],
    async () => (await endpoints.getCoupon(id)).data.data,
    { enabled: Boolean(id) }
  );

  const toggleMutation = useMutation({
    mutationFn: (next) => endpoints.setCouponActive(id, next),
    onSuccess: (_data, next) => {
      toast.success(next ? "Coupon reactivated." : "Coupon deactivated.");
      setConfirmDeactivate(false);
      queryClient.invalidateQueries({ queryKey: ["coupon", id] });
      queryClient.invalidateQueries({ queryKey: ["coupons"] });
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Failed to update coupon"),
  });

  if (loading) return <ObjectPageSkeleton />;
  if (notFound) {
    return (
      <ObjectNotFound
        objectLabel="coupon"
        identifier={id ? String(id).slice(0, 8) : undefined}
        backTo="/coupons"
        backLabel="Coupons"
      />
    );
  }
  if (isError) {
    return <ObjectPageError objectLabel="coupon" error={couponError} onRetry={refetch} backTo="/coupons" backLabel="Coupons" />;
  }

  const isActive = coupon.active;
  // Real redemptions: subscriptions currently carrying this coupon.
  const redeemers = subscriptions.filter((s) => s.coupon_id === coupon.id);

  return (
    <div>
      <ObjectHeader
        backTo="/coupons"
        backLabel="Coupons"
        kicker="Coupon"
        title={<span className="font-mono">{coupon.code}</span>}
        badge={<StatusBadge status={isActive ? "active" : "inactive"} />}
        meta={
          <>
            <Badge variant="neutral">{discountLabel(coupon)}</Badge>
            <span className="capitalize">{coupon.duration}</span>
            <CopyableId value={coupon.id} />
          </>
        }
        actions={
          isActive ? (
            <Button
              variant="outline"
              className="text-destructive hover:text-destructive"
              disabled={toggleMutation.isPending}
              onClick={() => setConfirmDeactivate(true)}
            >
              <PowerOff className="h-4 w-4" />
              Deactivate
            </Button>
          ) : (
            <Button
              variant="outline"
              disabled={toggleMutation.isPending}
              onClick={() => toggleMutation.mutate(true)}
            >
              <Power className="h-4 w-4" />
              Reactivate
            </Button>
          )
        }
      />

      <ObjectPageLayout
        rail={
          <ObjectSection title="Details">
            <AttributeList
              columns={1}
              items={[
                { label: "Coupon ID", value: <CopyableId value={coupon.id} /> },
                { label: "Code", value: <span className="font-mono text-sm">{coupon.code}</span> },
                { label: "Discount", value: discountLabel(coupon) },
                {
                  label: "Duration",
                  value: (
                    <span className="capitalize">
                      {coupon.duration === "repeating"
                        ? `Repeating · ${coupon.duration_months || "—"} months`
                        : coupon.duration}
                    </span>
                  ),
                },
                { label: "Status", value: <StatusBadge status={isActive ? "active" : "inactive"} /> },
                { label: "Created", value: formatDateTime(coupon.created_at) },
              ]}
            />
          </ObjectSection>
        }
      >
        <ObjectSection title="What it does">
          <p className="text-sm text-foreground">{durationSentence(coupon)}</p>
          {!isActive && (
            <p className="mt-2 text-sm text-muted-foreground">
              Deactivated — new subscriptions can no longer redeem this code. Subscriptions already
              using it keep their discount.
            </p>
          )}
        </ObjectSection>

        <ObjectSection title={`Redeemed by${redeemers.length ? ` (${redeemers.length})` : ""}`} flush>
          {redeemers.length === 0 ? (
            <RelatedEmpty>
              No active subscriptions are using this coupon yet.
            </RelatedEmpty>
          ) : (
            redeemers.map((s) => (
              <RelatedRow key={s.id} to={`/subscriptions/${s.id}`}>
                <span className="min-w-0 truncate text-foreground">
                  <CustomerName id={s.customer_id} names={names} link={false} />
                </span>
                <span className="flex shrink-0 items-center gap-3">
                  <StatusBadge status={s.status} />
                  <span className="text-xs text-muted-foreground">View →</span>
                </span>
              </RelatedRow>
            ))
          )}
        </ObjectSection>
      </ObjectPageLayout>

      <ConfirmDialog
        open={confirmDeactivate}
        onOpenChange={(o) => !o && setConfirmDeactivate(false)}
        title={`Deactivate ${coupon.code}?`}
        description="New subscriptions can no longer redeem this code. Customers already using it keep their discount. You can reactivate it later."
        confirmLabel="Deactivate coupon"
        destructive
        busy={toggleMutation.isPending}
        onConfirm={() => toggleMutation.mutate(false)}
      />
    </div>
  );
}

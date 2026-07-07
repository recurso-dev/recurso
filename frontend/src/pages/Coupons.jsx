import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, BadgePercent } from "lucide-react";

import { endpoints as api } from "../lib/api";
import CouponDetail from "../components/slide-overs/CouponDetail";
import { cn } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

const STATUS_FILTERS = ["all", "active", "expired"];

const statusVariant = (status) =>
  ({ active: "success", expired: "destructive" })[status] || "neutral";

const Coupons = () => {
  const navigate = useNavigate();
  const [coupons, setCoupons] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");

  const [selectedCoupon, setSelectedCoupon] = useState(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const fetchCoupons = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await api.getCoupons();
      // Map backend fields to frontend expectations (unchanged logic).
      const mappedCoupons = (response.data.data || []).map((c) => ({
        ...c,
        status: "active", // Backend doesn't return status yet; default to active.
        redemptions: 0,
        max_redemptions: null,
        discount:
          c.discount_type === "percent"
            ? `${c.discount_value}%`
            : `$${(c.discount_value / 100).toFixed(2)}`,
        duration_in_months: c.duration_months,
      }));
      setCoupons(mappedCoupons);
    } catch (err) {
      console.error("Failed to fetch coupons:", err);
      setError(
        err?.response?.data?.error?.message || err?.message || "Failed to load coupons"
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCoupons();
  }, []);

  const handleRowClick = (coupon) => {
    setSelectedCoupon(coupon);
    setIsDetailOpen(true);
  };

  const closeDetail = () => {
    setIsDetailOpen(false);
    setTimeout(() => setSelectedCoupon(null), 300);
  };

  const filteredCoupons = useMemo(() => {
    const q = search.trim().toLowerCase();
    return coupons.filter((c) => {
      if (statusFilter !== "all" && c.status !== statusFilter) return false;
      if (q && !(c.code || "").toLowerCase().includes(q)) return false;
      return true;
    });
  }, [coupons, statusFilter, search]);

  const columns = [
    {
      key: "code",
      header: "Coupon Code",
      cell: (c) => <span className="font-mono text-sm font-medium text-foreground">{c.code}</span>,
    },
    {
      key: "discount",
      header: "Discount",
      cell: (c) => <span className="text-muted-foreground">{c.discount}</span>,
    },
    {
      key: "duration",
      header: "Duration",
      cell: (c) => (
        <span className="capitalize text-muted-foreground">
          {c.duration === "repeating" ? `For ${c.duration_in_months} months` : c.duration}
        </span>
      ),
    },
    {
      key: "redemptions",
      header: "Redemptions",
      cell: (c) => (
        <span className="tabular-nums text-muted-foreground">
          {c.redemptions} {c.max_redemptions ? `/ ${c.max_redemptions}` : ""}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (c) => (
        <Badge variant={statusVariant(c.status)} className="capitalize">
          {c.status}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Coupons"
        description="Create and manage discount codes for your customers."
        actions={
          <Button onClick={() => navigate("/coupons/new")}>
            <Plus className="h-4 w-4" />
            Create coupon
          </Button>
        }
      />

      <DataTable
        columns={columns}
        data={filteredCoupons}
        loading={loading}
        error={error}
        onRetry={fetchCoupons}
        onRowClick={handleRowClick}
        search={{
          value: search,
          onChange: setSearch,
          placeholder: "Search coupons...",
        }}
        toolbar={
          <div className="flex items-center gap-1 rounded-lg border border-border bg-white p-0.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f}
                onClick={() => setStatusFilter(f)}
                className={cn(
                  "rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors",
                  statusFilter === f
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-zinc-500 hover:text-zinc-900"
                )}
              >
                {f}
              </button>
            ))}
          </div>
        }
        empty={{
          icon: BadgePercent,
          title:
            search || statusFilter !== "all" ? "No matching coupons" : "No coupons yet",
          description:
            search || statusFilter !== "all"
              ? "Try adjusting your search or filters."
              : "Create your first discount code to get started.",
          action:
            !search && statusFilter === "all" ? (
              <Button onClick={() => navigate("/coupons/new")}>
                <Plus className="h-4 w-4" />
                Create coupon
              </Button>
            ) : null,
        }}
      />

      <CouponDetail coupon={selectedCoupon} isOpen={isDetailOpen} onClose={closeDetail} />
    </div>
  );
};

export default Coupons;

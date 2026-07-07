import { useEffect, useState } from "react";
import { Plus, Gift, CheckCircle2, Clock } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { StatCard } from "@/components/patterns/StatCard";
import { DataTable } from "@/components/patterns/DataTable";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const statusVariant = (status) =>
  ({ redeemed: "success", purchased: "warning" })[status] || "neutral";

function Gifts() {
  const [gifts, setGifts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [plans, setPlans] = useState([]);
  const [customers, setCustomers] = useState([]);
  const [form, setForm] = useState({
    buyer_customer_id: "",
    plan_id: "",
    duration_months: 12,
  });

  useEffect(() => {
    fetchGifts();
    fetchPlans();
    fetchCustomers();
  }, []);

  const fetchGifts = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await endpoints.getGifts();
      setGifts(Array.isArray(response.data?.data) ? response.data.data : []);
    } catch (err) {
      console.error("Error fetching gifts:", err);
      setError(err?.response?.data?.error?.message || err?.message || "Failed to load gifts");
    } finally {
      setLoading(false);
    }
  };

  const fetchPlans = async () => {
    try {
      const response = await endpoints.getPlans();
      setPlans(response.data?.data || []);
    } catch (error) {
      console.error("Error fetching plans:", error);
    }
  };

  const fetchCustomers = async () => {
    try {
      const response = await endpoints.getCustomers();
      setCustomers(response.data?.data || []);
    } catch (error) {
      console.error("Error fetching customers:", error);
    }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!form.buyer_customer_id || !form.plan_id) return;
    try {
      setCreating(true);
      await endpoints.purchaseGift({
        buyer_customer_id: form.buyer_customer_id,
        plan_id: form.plan_id,
        duration_months: parseInt(form.duration_months),
      });
      setShowCreate(false);
      setForm({ buyer_customer_id: "", plan_id: "", duration_months: 12 });
      fetchGifts();
    } catch (error) {
      console.error("Error creating gift:", error);
    } finally {
      setCreating(false);
    }
  };

  const redeemedCount = gifts.filter((g) => g.status === "redeemed").length;
  const pendingCount = gifts.filter((g) => g.status === "purchased").length;

  const columns = [
    {
      key: "code",
      header: "Gift Code",
      cell: (g) => (
        <span className="rounded-md bg-muted px-2 py-1 font-mono text-sm text-foreground">
          {g.code}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (g) => (
        <Badge variant={statusVariant(g.status)} className="capitalize">
          {g.status}
        </Badge>
      ),
    },
    {
      key: "duration",
      header: "Duration",
      cell: (g) => <span className="text-muted-foreground">{g.duration_months} Months</span>,
    },
    {
      key: "recipient",
      header: "Recipient",
      cell: (g) => <span className="text-muted-foreground">{g.recipient_email || "—"}</span>,
    },
    {
      key: "purchased",
      header: "Purchased",
      cell: (g) => <span className="text-muted-foreground">{formatDate(g.created_at)}</span>,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Gift Subscriptions"
        description="Manage purchased gift subscriptions and track redemptions."
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4" />
            Create gift
          </Button>
        }
      />

      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Total Gifts Sold" value={gifts.length.toLocaleString()} icon={Gift} />
        <StatCard label="Redeemed" value={redeemedCount.toLocaleString()} icon={CheckCircle2} />
        <StatCard label="Pending" value={pendingCount.toLocaleString()} icon={Clock} />
      </div>

      <DataTable
        columns={columns}
        data={gifts}
        loading={loading}
        error={error}
        onRetry={fetchGifts}
        empty={{
          icon: Gift,
          title: "No gifts yet",
          description: "Create your first gift subscription for a customer.",
          action: (
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="h-4 w-4" />
              Create gift
            </Button>
          ),
        }}
      />

      <Sheet open={showCreate} onOpenChange={setShowCreate}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>Create gift subscription</SheetTitle>
            <SheetDescription>Purchase a gift subscription on behalf of a customer.</SheetDescription>
          </SheetHeader>

          <form
            id="create-gift-form"
            onSubmit={handleCreate}
            className="flex-1 space-y-6 overflow-y-auto px-6 py-6"
          >
            <FormField label="Buyer customer" htmlFor="buyer_customer_id" required>
              <Select
                value={form.buyer_customer_id}
                onValueChange={(v) => setForm({ ...form, buyer_customer_id: v })}
              >
                <SelectTrigger id="buyer_customer_id">
                  <SelectValue placeholder="Select customer..." />
                </SelectTrigger>
                <SelectContent>
                  {customers.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name} ({c.email})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField label="Plan" htmlFor="plan_id" required>
              <Select value={form.plan_id} onValueChange={(v) => setForm({ ...form, plan_id: v })}>
                <SelectTrigger id="plan_id">
                  <SelectValue placeholder="Select plan..." />
                </SelectTrigger>
                <SelectContent>
                  {plans.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField label="Duration (months)" htmlFor="duration_months" required>
              <Input
                id="duration_months"
                type="number"
                min="1"
                max="36"
                required
                value={form.duration_months}
                onChange={(e) => setForm({ ...form, duration_months: e.target.value })}
              />
            </FormField>
          </form>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button type="submit" form="create-gift-form" disabled={creating}>
              {creating ? "Creating..." : "Create gift"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

export default Gifts;

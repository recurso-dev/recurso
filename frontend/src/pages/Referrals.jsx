import { useEffect, useState } from "react";
import { Plus, Users, DollarSign, Clock, Share2 } from "lucide-react";

import { endpoints } from "../lib/api";
import { formatCurrency, formatDate } from "@/lib/utils";
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
  ({ rewarded: "success", qualified: "info" })[status] || "warning";

function Referrals() {
  const [referrals, setReferrals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [customers, setCustomers] = useState([]);
  const [form, setForm] = useState({
    referrer_id: "",
    referred_id: "",
    reward_amount: 500,
    currency: "USD",
  });

  useEffect(() => {
    fetchReferrals();
    fetchCustomers();
  }, []);

  const fetchReferrals = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await endpoints.getReferrals();
      setReferrals(response.data?.data || response.data || []);
    } catch (err) {
      console.error("Error fetching referrals:", err);
      setError(
        err?.response?.data?.error?.message || err?.message || "Failed to load referrals"
      );
    } finally {
      setLoading(false);
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
    if (!form.referrer_id || !form.referred_id) return;
    try {
      setCreating(true);
      await endpoints.createReferral({
        referrer_id: form.referrer_id,
        referred_id: form.referred_id,
        reward_amount: parseInt(form.reward_amount),
        currency: form.currency,
      });
      setShowCreate(false);
      setForm({ referrer_id: "", referred_id: "", reward_amount: 500, currency: "USD" });
      fetchReferrals();
    } catch (error) {
      console.error("Error creating referral:", error);
    } finally {
      setCreating(false);
    }
  };

  const totalRewards = referrals
    .filter((r) => r.status === "rewarded")
    .reduce((acc, curr) => acc + (curr.reward_amount || 0), 0);
  const pendingCount = referrals.filter((r) => r.status === "pending").length;

  const columns = [
    {
      key: "code",
      header: "Code",
      cell: (r) => (
        <span className="rounded-md bg-muted px-2 py-1 font-mono text-sm text-foreground">
          {r.code}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      cell: (r) => (
        <Badge variant={statusVariant(r.status)} className="capitalize">
          {r.status}
        </Badge>
      ),
    },
    {
      key: "reward",
      header: "Reward",
      cell: (r) => (
        <span className="tabular-nums text-foreground">
          {formatCurrency(r.reward_amount, r.currency || "USD")}
        </span>
      ),
    },
    {
      key: "created",
      header: "Created",
      cell: (r) => <span className="text-muted-foreground">{formatDate(r.created_at)}</span>,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Referral Program"
        description="Manage your customer referral program and track rewards."
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4" />
            Create referral
          </Button>
        }
      />

      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Total Referrals" value={referrals.length.toLocaleString()} icon={Users} />
        <StatCard label="Total Rewards Paid" value={formatCurrency(totalRewards)} icon={DollarSign} />
        <StatCard label="Pending" value={pendingCount.toLocaleString()} icon={Clock} />
      </div>

      <DataTable
        columns={columns}
        data={referrals}
        loading={loading}
        error={error}
        onRetry={fetchReferrals}
        empty={{
          icon: Share2,
          title: "No referrals yet",
          description: "Create your first referral to start tracking rewards.",
          action: (
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="h-4 w-4" />
              Create referral
            </Button>
          ),
        }}
      />

      <Sheet open={showCreate} onOpenChange={setShowCreate}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>Create referral</SheetTitle>
            <SheetDescription>Link a referrer to a new customer and set the reward.</SheetDescription>
          </SheetHeader>

          <form
            id="create-referral-form"
            onSubmit={handleCreate}
            className="flex-1 space-y-6 overflow-y-auto px-6 py-6"
          >
            <FormField label="Referrer (who referred)" htmlFor="referrer_id" required>
              <Select
                value={form.referrer_id}
                onValueChange={(v) => setForm({ ...form, referrer_id: v })}
              >
                <SelectTrigger id="referrer_id">
                  <SelectValue placeholder="Select referrer..." />
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

            <FormField label="Referred (new customer)" htmlFor="referred_id" required>
              <Select
                value={form.referred_id}
                onValueChange={(v) => setForm({ ...form, referred_id: v })}
              >
                <SelectTrigger id="referred_id">
                  <SelectValue placeholder="Select referred customer..." />
                </SelectTrigger>
                <SelectContent>
                  {customers
                    .filter((c) => c.id !== form.referrer_id)
                    .map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.name} ({c.email})
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField
              label="Reward amount (cents)"
              htmlFor="reward_amount"
              required
              description="500 = $5.00"
            >
              <Input
                id="reward_amount"
                type="number"
                min="0"
                required
                value={form.reward_amount}
                onChange={(e) => setForm({ ...form, reward_amount: e.target.value })}
              />
            </FormField>
          </form>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
              Cancel
            </Button>
            <Button type="submit" form="create-referral-form" disabled={creating}>
              {creating ? "Creating..." : "Create referral"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

export default Referrals;

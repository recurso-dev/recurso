import { useEffect, useState } from "react";
import { Plus, Building2, Trash2 } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { formatCurrency } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";

const shortId = (id) => (id ? String(id).slice(0, 8) : "—");

// Multi-tenant admin: group tenants under an organization and see
// consolidated MRR across them.
const Organizations = () => {
  const [orgs, setOrgs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ name: "", owner_email: "" });
  const [creating, setCreating] = useState(false);

  // Detail sheet state
  const [selected, setSelected] = useState(null);
  const [tenants, setTenants] = useState([]);
  const [mrr, setMRR] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [addTenantId, setAddTenantId] = useState("");
  const [addingTenant, setAddingTenant] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const fetchOrgs = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.getOrganizations();
      setOrgs(res.data.data || []);
    } catch (err) {
      setError(err?.response?.data?.error?.message || "Failed to load organizations");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrgs();
  }, []);

  const openDetail = async (org) => {
    setSelected(org);
    setTenants([]);
    setMRR(null);
    setDetailLoading(true);
    try {
      const [t, m] = await Promise.allSettled([
        api.getOrgTenants(org.id),
        api.getOrgMRR(org.id),
      ]);
      if (t.status === "fulfilled") setTenants(t.value.data.data || []);
      if (m.status === "fulfilled") setMRR(m.value.data.data);
    } finally {
      setDetailLoading(false);
    }
  };

  const submitCreate = async () => {
    setCreating(true);
    try {
      await api.createOrganization(createForm);
      toast.success("Organization created.");
      setCreateOpen(false);
      setCreateForm({ name: "", owner_email: "" });
      fetchOrgs();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to create organization");
    } finally {
      setCreating(false);
    }
  };

  const submitAddTenant = async () => {
    if (!selected || !addTenantId.trim()) return;
    setAddingTenant(true);
    try {
      await api.addOrgTenant(selected.id, addTenantId.trim());
      toast.success("Tenant added.");
      setAddTenantId("");
      openDetail(selected);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to add tenant");
    } finally {
      setAddingTenant(false);
    }
  };

  const removeTenant = async (tenantId) => {
    try {
      await api.removeOrgTenant(selected.id, tenantId);
      toast.success("Tenant removed.");
      openDetail(selected);
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to remove tenant");
    }
  };

  const confirmDelete = async () => {
    setDeleting(true);
    try {
      await api.deleteOrganization(selected.id);
      toast.success("Organization deleted.");
      setDeleteOpen(false);
      setSelected(null);
      fetchOrgs();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to delete organization");
    } finally {
      setDeleting(false);
    }
  };

  const columns = [
    {
      key: "name",
      header: "Organization",
      cell: (o) => <span className="text-sm font-medium text-foreground">{o.name}</span>,
    },
    {
      key: "owner",
      header: "Owner",
      cell: (o) => <span className="text-sm text-muted-foreground">{o.owner_email || "—"}</span>,
    },
    {
      key: "id",
      header: "ID",
      align: "right",
      cell: (o) => <span className="font-mono text-xs text-muted-foreground">{shortId(o.id)}</span>,
    },
  ];

  const createButton = (
    <Button onClick={() => setCreateOpen(true)}>
      <Plus className="h-4 w-4" />
      New organization
    </Button>
  );

  return (
    <div>
      <PageHeader
        title="Organizations"
        description="Group tenants under one umbrella and see consolidated MRR."
        actions={createButton}
      />

      <DataTable
        columns={columns}
        data={orgs}
        loading={loading}
        error={error}
        onRetry={fetchOrgs}
        onRowClick={openDetail}
        empty={{
          icon: Building2,
          title: "No organizations yet",
          description: "Create one to group related tenants together.",
          action: createButton,
        }}
      />

      {/* Create */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New organization</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>Name</Label>
              <Input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="Acme Group"
              />
            </div>
            <div>
              <Label>Owner email</Label>
              <Input
                type="email"
                value={createForm.owner_email}
                onChange={(e) => setCreateForm({ ...createForm, owner_email: e.target.value })}
                placeholder="owner@example.com"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={submitCreate}
              disabled={creating || !createForm.name.trim() || !createForm.owner_email.trim()}
            >
              {creating ? "Creating…" : "Create organization"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Detail */}
      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{selected?.name}</SheetTitle>
            <SheetDescription>{selected?.owner_email}</SheetDescription>
          </SheetHeader>

          <div className="flex-1 space-y-6 overflow-y-auto px-6 py-6">
            {mrr != null && (
              <div className="rounded-md border border-border bg-muted/30 p-4">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  Consolidated MRR
                </p>
                <p className="mt-1 text-2xl font-bold tabular-nums text-foreground">
                  {typeof mrr?.mrr === "number"
                    ? formatCurrency(mrr.mrr, mrr.currency || "USD")
                    : "—"}
                </p>
              </div>
            )}

            <div>
              <h3 className="mb-3 text-sm font-semibold text-foreground">Tenants</h3>
              {detailLoading ? (
                <p className="text-sm text-muted-foreground">Loading…</p>
              ) : tenants.length === 0 ? (
                <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-sm text-muted-foreground">
                  No tenants attached yet.
                </p>
              ) : (
                <ul className="space-y-2">
                  {tenants.map((t) => (
                    <li
                      key={t.id}
                      className="flex items-center justify-between gap-3 rounded-md border border-border px-4 py-3"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-foreground">
                          {t.name || shortId(t.id)}
                        </p>
                        <p className="font-mono text-xs text-muted-foreground">{t.id}</p>
                      </div>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-red-600 hover:text-red-600"
                        onClick={() => removeTenant(t.id)}
                        aria-label={`Remove tenant ${t.name || t.id}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </li>
                  ))}
                </ul>
              )}

              <div className="mt-3 flex gap-2">
                <Input
                  value={addTenantId}
                  onChange={(e) => setAddTenantId(e.target.value)}
                  placeholder="tenant uuid"
                  className="font-mono"
                />
                <Button
                  size="sm"
                  onClick={submitAddTenant}
                  disabled={addingTenant || !addTenantId.trim()}
                >
                  {addingTenant ? "Adding…" : "Add"}
                </Button>
              </div>
            </div>

            <div className="border-t border-border pt-4">
              <Button variant="outline" size="sm" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="h-4 w-4" />
                Delete organization
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${selected?.name}?`}
        description="The grouping is removed. Tenants themselves are not deleted."
        confirmLabel="Delete organization"
        destructive
        busy={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
};

export default Organizations;

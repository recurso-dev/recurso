import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Building2, Trash2, Pencil } from "lucide-react";

import { endpoints as api } from "../lib/api";
import { toast } from "@/components/ui/sonner";
import { formatCurrency, shortId } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { DataTable } from "@/components/patterns/DataTable";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";


// Multi-tenant admin: group tenants under an organization and see
// consolidated MRR across them.
const Organizations = () => {
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameValue, setRenameValue] = useState("");
  const [createForm, setCreateForm] = useState({ name: "", owner_email: "" });

  // Detail sheet state
  const [selected, setSelected] = useState(null);
  const [addTenantId, setAddTenantId] = useState("");
  const [deleteOpen, setDeleteOpen] = useState(false);

  const {
    data: orgs = [],
    isLoading: loading,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: ["organizations"],
    queryFn: async () => (await api.getOrganizations()).data.data || [],
  });
  const error = queryError
    ? queryError?.response?.data?.error?.message || "Failed to load organizations"
    : null;

  // Tenants + consolidated MRR for the open organization, fetched together
  // (allSettled: a failed MRR read still shows the tenant list).
  const { data: detail, isLoading: detailLoading } = useQuery({
    queryKey: ["org-detail", selected?.id],
    queryFn: async () => {
      const [t, m] = await Promise.allSettled([
        api.getOrgTenants(selected.id),
        api.getOrgMRR(selected.id),
      ]);
      return {
        tenants: t.status === "fulfilled" ? t.value.data.data || [] : [],
        mrr: m.status === "fulfilled" ? m.value.data.data : null,
      };
    },
    enabled: !!selected,
  });
  const tenants = detail?.tenants ?? [];
  const mrr = detail?.mrr ?? null;

  const invalidateOrgs = () => queryClient.invalidateQueries({ queryKey: ["organizations"] });
  const invalidateDetail = () =>
    queryClient.invalidateQueries({ queryKey: ["org-detail", selected?.id] });

  const openDetail = (org) => setSelected(org);

  const createMutation = useMutation({
    mutationFn: () => api.createOrganization(createForm),
    onSuccess: () => {
      toast.success("Organization created.");
      setCreateOpen(false);
      setCreateForm({ name: "", owner_email: "" });
      invalidateOrgs();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to create organization"),
  });
  const creating = createMutation.isPending;
  const submitCreate = () => createMutation.mutate();

  const addTenantMutation = useMutation({
    mutationFn: () => api.addOrgTenant(selected.id, addTenantId.trim()),
    onSuccess: () => {
      toast.success("Tenant added.");
      setAddTenantId("");
      invalidateDetail();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to add tenant"),
  });
  const addingTenant = addTenantMutation.isPending;
  const submitAddTenant = () => {
    if (!selected || !addTenantId.trim()) return;
    addTenantMutation.mutate();
  };

  const removeTenantMutation = useMutation({
    mutationFn: (tenantId) => api.removeOrgTenant(selected.id, tenantId),
    onSuccess: () => {
      toast.success("Tenant removed.");
      invalidateDetail();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to remove tenant"),
  });
  const removeTenant = (tenantId) => removeTenantMutation.mutate(tenantId);

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteOrganization(selected.id),
    onSuccess: () => {
      toast.success("Organization deleted.");
      setDeleteOpen(false);
      setSelected(null);
      invalidateOrgs();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to delete organization"),
  });
  const deleting = deleteMutation.isPending;
  const confirmDelete = () => deleteMutation.mutate();

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

  const renameMutation = useMutation({
    mutationFn: () =>
      api.updateOrganization(selected.id, {
        name: renameValue.trim(),
        owner_email: selected.owner_email,
      }),
    onSuccess: () => {
      toast.success("Organization renamed.");
      setSelected((prev) => ({ ...prev, name: renameValue.trim() }));
      setRenaming(false);
      invalidateOrgs();
    },
    onError: (err) => toast.error(err?.response?.data?.error?.message || "Failed to rename"),
  });
  const savingRename = renameMutation.isPending;
  const submitRename = () => {
    if (!selected || !renameValue.trim()) return;
    renameMutation.mutate();
  };

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
        onRetry={refetch}
        onRowClick={openDetail}
        empty={{
          icon: Building2,
          title: "No organizations yet",
          description: "Create one to group related tenants together.",
          action: createButton,
        }}
      />

      {/* Create */}
      <Sheet open={createOpen} onOpenChange={setCreateOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>New organization</SheetTitle>
            <SheetDescription>
              Group related tenants under one umbrella with consolidated MRR.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 overflow-y-auto px-6">
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
          <SheetFooter>
            <Button
              onClick={submitCreate}
              disabled={creating || !createForm.name.trim() || !createForm.owner_email.trim()}
            >
              {creating ? "Creating…" : "Create organization"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      {/* Detail */}
      <Sheet open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <SheetContent side="right" className="w-full sm:max-w-lg">
          <SheetHeader>
            {renaming ? (
              <div className="flex items-center gap-2 pr-8">
                <Input
                  value={renameValue}
                  onChange={(e) => setRenameValue(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && submitRename()}
                  autoFocus
                />
                <Button size="sm" onClick={submitRename} disabled={savingRename || !renameValue.trim()}>
                  {savingRename ? "Saving…" : "Save"}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setRenaming(false)}>
                  Cancel
                </Button>
              </div>
            ) : (
              <SheetTitle className="flex items-center gap-2">
                {selected?.name}
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7 text-muted-foreground"
                  title="Rename organization"
                  aria-label="Rename organization"
                  onClick={() => {
                    setRenameValue(selected?.name || "");
                    setRenaming(true);
                  }}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
              </SheetTitle>
            )}
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

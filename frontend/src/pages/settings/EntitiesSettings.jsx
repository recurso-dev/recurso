import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Building2, Plus, Trash2, Pencil } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { ErrorState } from "@/components/patterns/ErrorState";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { FormField } from "@/components/patterns/FormField";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet";

const empty = { name: "", legal_name: "", invoice_prefix: "", country_code: "" };

export default function EntitiesSettings() {
  const queryClient = useQueryClient();
  const [sheet, setSheet] = useState(null); // { mode: 'create'|'edit', form, id }
  const [removeTarget, setRemoveTarget] = useState(null);

  const {
    data: entities = [],
    isLoading: loading,
    isError: loadError,
    refetch,
  } = useQuery({
    queryKey: ["entities"],
    queryFn: async () => (await endpoints.getEntities()).data?.data || [],
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["entities"] });

  const saveMutation = useMutation({
    mutationFn: ({ mode, id, form }) =>
      mode === "edit" ? endpoints.updateEntity(id, form) : endpoints.createEntity(form),
    onSuccess: (_res, vars) => {
      toast.success(vars.mode === "edit" ? "Entity updated." : "Entity created.");
      setSheet(null);
      invalidate();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Couldn't save the entity."),
  });

  const removeMutation = useMutation({
    mutationFn: (id) => endpoints.deleteEntity(id),
    onSuccess: () => {
      toast.success("Entity deleted.");
      setRemoveTarget(null);
      invalidate();
    },
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Couldn't delete the entity."),
  });

  const openCreate = () => setSheet({ mode: "create", id: null, form: { ...empty } });
  const openEdit = (e) =>
    setSheet({
      mode: "edit",
      id: e.id,
      form: {
        name: e.name || "",
        legal_name: e.legal_name || "",
        invoice_prefix: e.invoice_prefix || "",
        country_code: e.country_code || "",
      },
    });

  const setField = (k, v) => setSheet((s) => ({ ...s, form: { ...s.form, [k]: v } }));
  const submit = (e) => {
    e.preventDefault();
    saveMutation.mutate(sheet);
  };

  return (
    <div className="mx-auto max-w-4xl">
      <PageHeader
        title="Legal entities"
        description="Bill under multiple legal entities, each with its own books, tax identity, and invoice series."
        actions={
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4" />
            Add entity
          </Button>
        }
      />

      {loading ? (
        <Skeleton className="h-48 w-full rounded-xl" />
      ) : loadError ? (
        <ErrorState
          title="Couldn't load entities"
          message="We couldn't reach the settings service. Please try again."
          onRetry={refetch}
        />
      ) : (
        <Card className="mt-6">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Invoice prefix</TableHead>
                <TableHead>Country</TableHead>
                <TableHead>Ledger</TableHead>
                <TableHead className="w-20" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {entities.map((e) => (
                <TableRow key={e.id}>
                  <TableCell className="font-medium text-foreground">
                    <span className="flex items-center gap-2">
                      <Building2 className="h-4 w-4 shrink-0 text-muted-foreground" />
                      {e.name}
                      {e.is_primary && <Badge variant="success">Primary</Badge>}
                    </span>
                    {e.legal_name && (
                      <span className="ml-6 block text-xs text-muted-foreground">
                        {e.legal_name}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                      {e.invoice_prefix}
                    </code>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {e.country_code || "—"}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    #{e.tb_ledger_id}
                  </TableCell>
                  <TableCell>
                    <div className="flex justify-end gap-1">
                      <button
                        type="button"
                        onClick={() => openEdit(e)}
                        className="text-muted-foreground hover:text-foreground"
                        aria-label="Edit entity"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                      {!e.is_primary && (
                        <button
                          type="button"
                          onClick={() => setRemoveTarget(e)}
                          className="text-muted-foreground hover:text-red-600"
                          aria-label="Delete entity"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <Sheet open={!!sheet} onOpenChange={(open) => !open && setSheet(null)}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{sheet?.mode === "edit" ? "Edit entity" : "Add entity"}</SheetTitle>
            <SheetDescription>
              An entity issues invoices under its own number series and tax identity.
            </SheetDescription>
          </SheetHeader>
          {sheet && (
            <form onSubmit={submit} className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
              <FormField label="Name" htmlFor="e-name" required>
                <Input
                  id="e-name"
                  required
                  value={sheet.form.name}
                  onChange={(ev) => setField("name", ev.target.value)}
                  placeholder="ACME India"
                />
              </FormField>
              <FormField label="Legal name" htmlFor="e-legal">
                <Input
                  id="e-legal"
                  value={sheet.form.legal_name}
                  onChange={(ev) => setField("legal_name", ev.target.value)}
                  placeholder="ACME India Private Limited"
                />
              </FormField>
              <FormField
                label="Invoice prefix"
                htmlFor="e-prefix"
                description="Prefix for this entity's invoice series. Defaults to a slug of the name."
              >
                <Input
                  id="e-prefix"
                  value={sheet.form.invoice_prefix}
                  onChange={(ev) => setField("invoice_prefix", ev.target.value)}
                  placeholder="ACME-IN"
                />
              </FormField>
              <FormField label="Country" htmlFor="e-country" description="ISO 3166-1 alpha-2 (e.g. IN).">
                <Input
                  id="e-country"
                  maxLength={2}
                  value={sheet.form.country_code}
                  onChange={(ev) => setField("country_code", ev.target.value.toUpperCase())}
                  placeholder="IN"
                  className="max-w-[120px]"
                />
              </FormField>
              <SheetFooter className="px-0">
                <Button type="button" variant="outline" onClick={() => setSheet(null)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={saveMutation.isPending}>
                  {saveMutation.isPending ? "Saving…" : "Save"}
                </Button>
              </SheetFooter>
            </form>
          )}
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!removeTarget}
        onOpenChange={(open) => !open && setRemoveTarget(null)}
        title={`Delete ${removeTarget?.name || "this entity"}?`}
        description="This can't be undone. An entity that has issued invoices or holds ledger balances should not be deleted."
        confirmLabel="Delete entity"
        destructive
        busy={removeMutation.isPending}
        onConfirm={() => removeMutation.mutate(removeTarget.id)}
      />
    </div>
  );
}

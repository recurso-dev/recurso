import { useEffect, useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { UserPlus, Trash2 } from "lucide-react";

import { endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { formatDate } from "@/lib/utils";
import { PageHeader } from "@/components/patterns/PageHeader";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { FormField } from "@/components/patterns/FormField";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";

const ROLES = ["owner", "admin", "member"];
const roleVariant = (r) =>
  r === "owner" ? "success" : r === "admin" ? "info" : "neutral";

export default function Team() {
  const { user } = useAuth();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [inviteOpen, setInviteOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({ name: "", email: "", role: "member" });

  const canManage = user?.role === "owner" || user?.role === "admin";

  const load = () =>
    endpoints
      .getUsers()
      .then((res) => setUsers(res.data?.data || []))
      .catch(() => toast.error("Failed to load team"))
      .finally(() => setLoading(false));

  useEffect(() => {
    load();
  }, []);

  const invite = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      await endpoints.inviteUser(form);
      toast.success("Invitation sent — they'll get an email to set their password");
      setInviteOpen(false);
      setForm({ name: "", email: "", role: "member" });
      await load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to send invitation");
    } finally {
      setBusy(false);
    }
  };

  const changeRole = async (id, role) => {
    try {
      await endpoints.updateUserRole(id, role);
      await load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to update role");
    }
  };

  const [removeTarget, setRemoveTarget] = useState(null);

  const remove = async () => {
    if (!removeTarget) return;
    try {
      await endpoints.deleteUser(removeTarget);
      setRemoveTarget(null);
      await load();
    } catch (err) {
      toast.error(err?.response?.data?.error?.message || "Failed to remove");
    }
  };

  return (
    <div>
      <PageHeader
        title="Team"
        description="Manage who can access this workspace and their roles."
        actions={
          canManage && (
            <Button onClick={() => setInviteOpen(true)}>
              <UserPlus className="h-4 w-4" />
              Add member
            </Button>
          )
        }
      />

      <Card className="mt-6">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Joined</TableHead>
              <TableHead className="w-10" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Loading…
                </TableCell>
              </TableRow>
            ) : users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  No team members yet.
                </TableCell>
              </TableRow>
            ) : (
              users.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-medium text-foreground">
                    {u.name}
                    {u.id === user?.id && (
                      <span className="ml-2 text-xs text-muted-foreground">(you)</span>
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">{u.email}</TableCell>
                  <TableCell>
                    {canManage && u.id !== user?.id ? (
                      <Select value={u.role} onValueChange={(r) => changeRole(u.id, r)}>
                        <SelectTrigger className="h-8 w-28">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {ROLES.map((r) => (
                            <SelectItem key={r} value={r} className="capitalize">
                              {r}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    ) : (
                      <Badge variant={roleVariant(u.role)} className="capitalize">
                        {u.role}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(u.created_at)}
                  </TableCell>
                  <TableCell>
                    {canManage && u.id !== user?.id && (
                      <button
                        type="button"
                        onClick={() => setRemoveTarget(u.id)}
                        className="text-muted-foreground hover:text-red-600"
                        aria-label="Remove member"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      <Sheet open={inviteOpen} onOpenChange={setInviteOpen}>
        <SheetContent side="right" className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>Add a team member</SheetTitle>
            <SheetDescription>
              We'll email them a link to set their own password and sign in.
            </SheetDescription>
          </SheetHeader>
          <form onSubmit={invite} className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
            <FormField label="Name" htmlFor="t-name" required>
              <Input
                id="t-name"
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </FormField>
            <FormField label="Email" htmlFor="t-email" required>
              <Input
                id="t-email"
                type="email"
                required
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </FormField>
            <FormField label="Role" htmlFor="t-role">
              <Select value={form.role} onValueChange={(r) => setForm({ ...form, role: r })}>
                <SelectTrigger id="t-role">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ROLES.map((r) => (
                    <SelectItem key={r} value={r} className="capitalize">
                      {r}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <Button type="submit" className="w-full" disabled={busy}>
              {busy ? "Sending…" : "Send invitation"}
            </Button>
          </form>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={!!removeTarget}
        onOpenChange={(open) => !open && setRemoveTarget(null)}
        title="Remove this teammate?"
        description="They lose dashboard access immediately. You can invite them again later."
        confirmLabel="Remove teammate"
        destructive
        onConfirm={remove}
      />
    </div>
  );
}

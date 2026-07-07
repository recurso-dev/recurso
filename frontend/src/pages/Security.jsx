import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import {
  ShieldCheck,
  ShieldOff,
  Smartphone,
  Monitor,
  Copy,
  Check,
  KeyRound,
  Loader2,
  LogOut,
  Trash2,
  AlertTriangle,
} from "lucide-react";

import { endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { toast } from "@/components/ui/sonner";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { formatDate } from "@/lib/utils";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function CopyButton({ value, label = "Copy" }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error("Couldn't copy to clipboard.");
    }
  };
  return (
    <Button variant="outline" size="sm" onClick={copy} type="button">
      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
      {copied ? "Copied" : label}
    </Button>
  );
}

function MfaSection() {
  const { user } = useAuth();
  // We can't always tell MFA state from /auth/me, so we track it locally and
  // seed from the user object when the field is present.
  const [enabled, setEnabled] = useState(!!user?.mfa_enabled);

  const [setup, setSetup] = useState(null); // { secret, otpauth_url }
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [backupCodes, setBackupCodes] = useState(null); // shown once
  const [disabling, setDisabling] = useState(false);
  const [disableCode, setDisableCode] = useState("");

  const startSetup = async () => {
    setBusy(true);
    try {
      const res = await endpoints.mfaSetup();
      setSetup(res.data);
      setCode("");
    } catch {
      toast.error("Could not start MFA setup.");
    } finally {
      setBusy(false);
    }
  };

  const verify = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await endpoints.mfaVerify(code);
      setBackupCodes(res.data?.backup_codes || []);
      setEnabled(true);
      setSetup(null);
      setCode("");
      toast.success("Two-factor authentication enabled.");
    } catch {
      toast.error("That code is incorrect. Try again.");
    } finally {
      setBusy(false);
    }
  };

  const disable = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      await endpoints.mfaDisable(disableCode);
      setEnabled(false);
      setDisabling(false);
      setDisableCode("");
      setBackupCodes(null);
      toast.success("Two-factor authentication disabled.");
    } catch {
      toast.error("That code is incorrect. Try again.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ShieldCheck className="h-4 w-4 text-emerald-600" />
          Two-factor authentication
          {enabled && (
            <Badge variant="success" className="ml-1">
              Enabled
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* One-time backup codes after enabling */}
        {backupCodes ? (
          <div className="space-y-4">
            <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <p>
                Save these backup codes now. Each can be used once if you lose
                your authenticator. They won't be shown again.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 rounded-md border border-border bg-zinc-50 p-4 font-mono text-sm">
              {backupCodes.map((c) => (
                <span key={c} className="text-foreground">
                  {c}
                </span>
              ))}
            </div>
            <div className="flex gap-2">
              <CopyButton
                value={backupCodes.join("\n")}
                label="Copy codes"
              />
              <Button
                variant="ghost"
                size="sm"
                type="button"
                onClick={() => setBackupCodes(null)}
              >
                <Check className="h-4 w-4" />
                I've saved them
              </Button>
            </div>
          </div>
        ) : setup ? (
          /* Setup flow: QR + secret + verify */
          <form onSubmit={verify} className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Scan this QR code with your authenticator app (Google
              Authenticator, 1Password, Authy…), then enter the 6-digit code to
              confirm.
            </p>
            <div className="flex flex-col items-start gap-4 sm:flex-row sm:items-center">
              <div className="rounded-lg border border-border bg-white p-3">
                <QRCodeSVG value={setup.otpauth_url} size={160} />
              </div>
              <div className="space-y-2">
                <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  Or enter this key manually
                </p>
                <div className="flex items-center gap-2">
                  <code className="rounded bg-zinc-100 px-2 py-1 font-mono text-sm break-all">
                    {setup.secret}
                  </code>
                  <CopyButton value={setup.secret} />
                </div>
              </div>
            </div>
            <FormField label="Verification code" htmlFor="mfa-verify" required>
              <Input
                id="mfa-verify"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={8}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, ""))}
                placeholder="123456"
                className="max-w-[200px] text-center font-mono tracking-[0.3em]"
              />
            </FormField>
            <div className="flex gap-2">
              <Button type="submit" disabled={busy || code.length < 6}>
                {busy ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ShieldCheck className="h-4 w-4" />
                )}
                Verify & enable
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setSetup(null);
                  setCode("");
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        ) : disabling ? (
          /* Disable flow: confirm with a current code */
          <form onSubmit={disable} className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Enter a current authentication code to turn off two-factor
              authentication.
            </p>
            <FormField label="Authentication code" htmlFor="mfa-disable" required>
              <Input
                id="mfa-disable"
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={8}
                value={disableCode}
                onChange={(e) =>
                  setDisableCode(e.target.value.replace(/[^0-9]/g, ""))
                }
                placeholder="123456"
                className="max-w-[200px] text-center font-mono tracking-[0.3em]"
              />
            </FormField>
            <div className="flex gap-2">
              <Button
                type="submit"
                variant="destructive"
                disabled={busy || disableCode.length < 6}
              >
                {busy ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ShieldOff className="h-4 w-4" />
                )}
                Disable
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setDisabling(false);
                  setDisableCode("");
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        ) : (
          /* Idle */
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Add a second step to your login using a time-based code from an
              authenticator app. This keeps your account safe even if your
              password is compromised.
            </p>
            {enabled ? (
              <Button
                variant="outline"
                type="button"
                onClick={() => setDisabling(true)}
              >
                <ShieldOff className="h-4 w-4" />
                Disable two-factor authentication
              </Button>
            ) : (
              <Button type="button" onClick={startSetup} disabled={busy}>
                {busy ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ShieldCheck className="h-4 w-4" />
                )}
                Enable two-factor authentication
              </Button>
            )}
            {!user?.mfa_enabled && !enabled && (
              <p className="text-xs text-muted-foreground">
                Already enabled on another device? You can still start setup —
                verifying re-confirms it. To turn it off, use the disable option
                after enabling here.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function SessionsSection() {
  const [sessions, setSessions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [working, setWorking] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(false);
    try {
      const res = await endpoints.getSessions();
      setSessions(res.data?.data || []);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const revoke = async (id) => {
    setWorking(true);
    try {
      await endpoints.revokeSession(id);
      toast.success("Session revoked.");
      await load();
    } catch {
      toast.error("Couldn't revoke that session.");
    } finally {
      setWorking(false);
    }
  };

  const revokeOthers = async () => {
    setWorking(true);
    try {
      await endpoints.revokeOtherSessions();
      toast.success("Signed out of all other sessions.");
      await load();
    } catch {
      toast.error("Couldn't sign out other sessions.");
    } finally {
      setWorking(false);
    }
  };

  const hasOthers = sessions.some((s) => !s.current);

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardTitle className="flex items-center gap-2 text-base">
          <Monitor className="h-4 w-4 text-muted-foreground" />
          Active sessions
        </CardTitle>
        <Button
          variant="outline"
          size="sm"
          type="button"
          onClick={revokeOthers}
          disabled={working || !hasOthers}
        >
          <LogOut className="h-4 w-4" />
          Log out all other sessions
        </Button>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading sessions…
          </div>
        ) : error ? (
          <p className="py-6 text-sm text-red-600">
            Couldn't load your sessions.{" "}
            <button
              type="button"
              onClick={load}
              className="font-medium underline"
            >
              Retry
            </button>
          </p>
        ) : sessions.length === 0 ? (
          <p className="py-6 text-sm text-muted-foreground">
            No active sessions found.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Device</TableHead>
                <TableHead>Started</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sessions.map((s) => (
                <TableRow key={s.id}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Smartphone className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <span className="max-w-[320px] truncate text-sm text-foreground">
                        {s.user_agent || "Unknown device"}
                      </span>
                      {s.current && (
                        <Badge variant="success">This device</Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(s.created_at)}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(s.expires_at)}
                  </TableCell>
                  <TableCell className="text-right">
                    {s.current ? (
                      <span className="text-xs text-muted-foreground">
                        Current
                      </span>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        type="button"
                        onClick={() => revoke(s.id)}
                        disabled={working}
                      >
                        <Trash2 className="h-4 w-4" />
                        Revoke
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

export default function Security() {
  return (
    <div>
      <PageHeader
        title="Security"
        description="Manage two-factor authentication and your active sessions."
      />
      <div className="max-w-3xl space-y-6">
        <MfaSection />
        <SessionsSection />
      </div>
    </div>
  );
}

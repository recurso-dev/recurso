import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Gift, Loader2 } from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";

const PortalRedeem = () => {
  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState({ type: "", message: "" }); // type: 'success' | 'error'
  const navigate = useNavigate();

  const sessionToken = localStorage.getItem("portal_session");

  useEffect(() => {
    if (!sessionToken) {
      navigate("/portal/login");
      return;
    }
  }, [sessionToken, navigate]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setStatus({ type: "", message: "" });

    try {
      const response = await fetch(`${API_BASE}/portal/api/redeem`, {
        method: "POST",
        headers: {
          "X-Portal-Session": sessionToken,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ code }),
      });

      const data = await response.json();

      if (!response.ok) {
        if (response.status === 401) {
          localStorage.removeItem("portal_session");
          navigate("/portal/login");
          return;
        }
        throw new Error(data.error?.message || "Failed to redeem gift");
      }

      setStatus({
        type: "success",
        message: "Gift redeemed successfully! Redirecting...",
      });
      setTimeout(() => navigate("/portal/dashboard"), 2000);
    } catch (err) {
      setStatus({ type: "error", message: err.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4">
      <div className="w-full max-w-md">
        <Card>
          <CardContent className="p-8">
            <div className="mb-8 text-center">
              <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
                <Gift className="h-6 w-6" />
              </div>
              <h1 className="text-xl font-semibold text-foreground">
                Redeem gift
              </h1>
              <p className="mt-2 text-sm text-muted-foreground">
                Enter your gift code to claim your subscription.
              </p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {status.message && (
                <div
                  className={
                    status.type === "success"
                      ? "rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700"
                      : "rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
                  }
                >
                  {status.message}
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="code">Gift code</Label>
                <Input
                  id="code"
                  type="text"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  placeholder="GIFT-XXXXXXXX"
                  required
                  className="font-mono"
                />
              </div>

              <Button type="submit" disabled={loading} className="w-full">
                {loading ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Redeeming...
                  </>
                ) : (
                  "Redeem gift"
                )}
              </Button>
            </form>

            <div className="mt-6 text-center">
              <button
                type="button"
                onClick={() => navigate("/portal/dashboard")}
                className="inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                <ArrowLeft className="h-3.5 w-3.5" />
                Back to dashboard
              </button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default PortalRedeem;

import React, { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Loader2 } from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";

const PortalVerify = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get("token");

  useEffect(() => {
    if (!token) {
      navigate("/portal/login");
      return;
    }

    const verifyToken = async () => {
      try {
        const response = await fetch(
          `${API_BASE}/portal/auth/verify?token=${token}`
        );
        const data = await response.json();

        if (response.ok && data.session_token) {
          localStorage.setItem("portal_session", data.session_token);
          navigate("/portal/dashboard");
        } else {
          navigate("/portal/login?error=invalid");
        }
      } catch (err) {
        navigate("/portal/login?error=network");
      }
    };

    verifyToken();
  }, [token, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50">
      <div className="text-center">
        <Loader2 className="mx-auto mb-4 h-8 w-8 animate-spin text-primary" />
        <p className="text-sm text-muted-foreground">Verifying your login...</p>
      </div>
    </div>
  );
};

export default PortalVerify;

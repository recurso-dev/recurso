import { useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Loader2 } from "lucide-react";

import { API_ROOT as API_BASE } from "../../lib/api";

const PortalVerify = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get("token");
  // The magic-link token is single-use, and React StrictMode double-invokes
  // effects in dev — the second verify would consume an already-used token and
  // bounce a successfully logged-in customer to the error page.
  const verifyStarted = useRef(false);

  useEffect(() => {
    if (!token) {
      navigate("/portal/login");
      return;
    }
    if (verifyStarted.current) return;
    verifyStarted.current = true;

    const verifyToken = async () => {
      try {
        // credentials: "include" lets the browser store the httpOnly session
        // cookie the server sets on success — the session is never exposed to JS.
        const response = await fetch(
          `${API_BASE}/portal/auth/verify?token=${token}`,
          { credentials: "include" }
        );

        if (response.ok) {
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
    <div className="flex min-h-screen items-center justify-center bg-stone-50">
      <div className="text-center">
        <Loader2 className="mx-auto mb-4 h-8 w-8 animate-spin text-primary" />
        <p className="text-sm text-muted-foreground">Verifying your login...</p>
      </div>
    </div>
  );
};

export default PortalVerify;

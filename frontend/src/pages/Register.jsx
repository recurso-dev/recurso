import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Layers } from "lucide-react";

import { endpoints } from "@/lib/api";
import { useAuth } from "@/auth/AuthProvider";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

export default function Register() {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [formData, setFormData] = useState({
    orgName: "",
    email: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      // Call API to register — payload unchanged.
      const response = await endpoints.register({
        name: formData.orgName,
        email: formData.email,
      });

      const { api_key } = response.data;

      // Auto login with the new key.
      if (api_key) {
        login(api_key);
        navigate("/");
      }
    } catch (err) {
      console.error("Registration failed:", err);
      setError(
        err.response?.data?.error?.message || "Registration failed. Please try again."
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-zinc-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Create your workspace
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Get started with your isolated tenant environment.
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            <form onSubmit={handleSubmit} className="space-y-5">
              {error && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
                  {error}
                </div>
              )}

              <FormField label="Organization name" htmlFor="orgName" required>
                <Input
                  id="orgName"
                  name="orgName"
                  type="text"
                  required
                  value={formData.orgName}
                  onChange={handleChange}
                  placeholder="Acme Corp"
                />
              </FormField>

              <FormField label="Work email" htmlFor="email" required>
                <Input
                  id="email"
                  name="email"
                  type="email"
                  required
                  value={formData.email}
                  onChange={handleChange}
                  placeholder="name@company.com"
                />
              </FormField>

              {/* No password field: registration is API-key based —
                  the backend only needs a workspace name and email. */}

              <div className="flex items-start gap-3">
                <input
                  id="terms"
                  name="terms"
                  type="checkbox"
                  required
                  className="mt-0.5 h-4 w-4 rounded border-input text-primary accent-emerald-600 focus:ring-ring"
                />
                <label htmlFor="terms" className="text-sm text-muted-foreground">
                  I agree to the{" "}
                  <a href="#" className="font-medium text-primary hover:text-primary/80">
                    Terms
                  </a>{" "}
                  and{" "}
                  <a href="#" className="font-medium text-primary hover:text-primary/80">
                    Privacy Policy
                  </a>
                  .
                </label>
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "Creating..." : "Create workspace"}
              </Button>
            </form>
          </CardContent>
        </Card>

        <div className="mt-6 text-center">
          <p className="text-sm text-muted-foreground">
            Already have an account?{" "}
            <Link to="/login" className="font-semibold text-primary hover:text-primary/80">
              Log in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}

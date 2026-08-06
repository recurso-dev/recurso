import { useState } from "react";
import { Link, useNavigate } from "react-router";
import { Layers } from "lucide-react";

import { useAuth } from "@/auth/AuthProvider";
import { COUNTRIES } from "@/lib/countries";
import { FormField } from "@/components/patterns/FormField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export default function Register() {
  const navigate = useNavigate();
  const { registerAccount } = useAuth();
  const [formData, setFormData] = useState({
    company_name: "",
    name: "",
    email: "",
    password: "",
    country: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Functional update so batched multi-field changes (browser autofill /
  // password managers) don't clobber each other via a stale `formData` closure.
  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (formData.password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      // Country is optional — omit it entirely when unset.
      const { country, ...rest } = formData;
      await registerAccount(country ? { ...rest, country } : rest);
      navigate("/");
    } catch (err) {
      setError(
        err.response?.data?.error?.message || "Registration failed. Please try again."
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-stone-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
            <Layers className="h-6 w-6" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Create your workspace
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            You'll be the owner of a new, isolated tenant.
          </p>
        </div>

        <Card>
          <CardContent className="p-6">
            <form onSubmit={handleSubmit} className="space-y-5">
              {error && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">
                  {error}
                </div>
              )}

              <FormField label="Company name" htmlFor="company_name" required>
                <Input
                  id="company_name"
                  name="company_name"
                  required
                  value={formData.company_name}
                  onChange={handleChange}
                  placeholder="Acme Corp"
                />
              </FormField>

              <FormField label="Your name" htmlFor="name" required>
                <Input
                  id="name"
                  name="name"
                  required
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Jane Doe"
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

              <FormField
                label="Business country"
                htmlFor="country"
                description="Sets your tax jurisdiction (US sales tax, India GST, EU VAT). You can change it later in Settings."
              >
                <Select
                  value={formData.country}
                  onValueChange={(v) => setFormData({ ...formData, country: v })}
                >
                  <SelectTrigger id="country">
                    <SelectValue placeholder="Select country (optional)" />
                  </SelectTrigger>
                  <SelectContent>
                    {COUNTRIES.map((c) => (
                      <SelectItem key={c.code} value={c.code}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField label="Password" htmlFor="password" required>
                <PasswordInput
                  id="password"
                  name="password"
                  autoComplete="new-password"
                  required
                  value={formData.password}
                  onChange={handleChange}
                  placeholder="At least 8 characters"
                />
              </FormField>

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
                  <a href="#" className="font-medium text-primary hover:text-primary/80">Terms</a>{" "}
                  and{" "}
                  <a href="#" className="font-medium text-primary hover:text-primary/80">Privacy Policy</a>.
                </label>
              </div>

              <Button type="submit" className="w-full" disabled={loading}>
                {loading ? "Creating…" : "Create workspace"}
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

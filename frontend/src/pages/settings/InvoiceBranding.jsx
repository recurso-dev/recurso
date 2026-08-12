import { useEffect, useRef, useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Save, Upload, X } from "lucide-react";

import { endpoints } from "@/lib/api";
import { toast } from "@/components/ui/sonner";
import { Textarea } from "@/components/ui/textarea";
import { PageHeader } from "@/components/patterns/PageHeader";
import { FormField } from "@/components/patterns/FormField";
import { Skeleton } from "@/components/patterns/LoadingSkeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";

const EMPTY = {
  company_name: "",
  logo_data_url: "",
  signature_data_url: "",
  signatory_name: "",
  bank_details: "",
  terms: "",
};

const MAX_IMAGE_BYTES = 300 * 1024;

// File → validated data URL. PNG/JPEG only (matches the API's allow-list) and
// size-checked client-side so users get an instant error, not a 400.
function readImageFile(file) {
  return new Promise((resolve, reject) => {
    if (!["image/png", "image/jpeg"].includes(file.type)) {
      reject(new Error("Use a PNG or JPEG image."));
      return;
    }
    if (file.size > MAX_IMAGE_BYTES) {
      reject(new Error("Image is too large — keep it under 300KB."));
      return;
    }
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.onerror = () => reject(new Error("Couldn't read the file."));
    reader.readAsDataURL(file);
  });
}

// One upload slot: preview + replace/remove. Signature and logo share it.
function ImageField({ id, label, description, value, onChange, previewClass }) {
  const inputRef = useRef(null);
  const pick = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = ""; // same-file re-select still fires change
    if (!file) return;
    try {
      onChange(await readImageFile(file));
    } catch (err) {
      toast.error(err.message);
    }
  };
  return (
    <FormField label={label} htmlFor={id} description={description}>
      <input
        ref={inputRef}
        id={id}
        type="file"
        accept="image/png,image/jpeg"
        className="hidden"
        onChange={pick}
      />
      {value ? (
        <div className="flex items-center gap-3">
          <div className="rounded-md border border-border bg-muted/30 p-2">
            <img src={value} alt="" className={previewClass} />
          </div>
          <Button type="button" variant="outline" size="sm" onClick={() => inputRef.current?.click()}>
            <Upload className="h-3.5 w-3.5" />
            Replace
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="text-muted-foreground"
            onClick={() => onChange("")}
          >
            <X className="h-3.5 w-3.5" />
            Remove
          </Button>
        </div>
      ) : (
        <Button type="button" variant="outline" size="sm" onClick={() => inputRef.current?.click()}>
          <Upload className="h-3.5 w-3.5" />
          Upload image
        </Button>
      )}
    </FormField>
  );
}

// Invoice branding: the presentation layer of the invoice document — display
// name, logo, signature, bank details, terms. Statutory identity (GST / W-9)
// stays in its own settings and wins on tax invoices.
export default function InvoiceBranding() {
  const [config, setConfig] = useState(EMPTY);

  const { data, isLoading: loading } = useQuery({
    queryKey: ["invoice-branding"],
    queryFn: async () => (await endpoints.getInvoiceBranding()).data?.data || null,
  });
  useEffect(() => {
    if (data) setConfig((prev) => ({ ...prev, ...data }));
  }, [data]);

  const set = (patch) => setConfig((prev) => ({ ...prev, ...patch }));

  const saveMutation = useMutation({
    mutationFn: (cfg) => endpoints.updateInvoiceBranding(cfg),
    onSuccess: () => toast.success("Invoice branding saved."),
    onError: (err) =>
      toast.error(err?.response?.data?.error?.message || "Failed to save branding"),
  });
  const saving = saveMutation.isPending;

  const handleSave = (e) => {
    e.preventDefault();
    saveMutation.mutate(config);
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="Invoice branding"
        description="How your invoices look: company name, logo, signature, bank details and terms. Legal identity from your GST or W-9 settings still takes precedence on tax invoices."
      />

      {loading ? (
        <Skeleton className="h-96 w-full rounded-xl" />
      ) : (
        <form onSubmit={handleSave}>
          <Card>
            <CardContent className="space-y-6 pt-6">
              <FormField
                label="Company name"
                htmlFor="company_name"
                description="Shown at the top of every invoice. Leave blank to use your account name."
              >
                <Input
                  id="company_name"
                  value={config.company_name}
                  onChange={(e) => set({ company_name: e.target.value })}
                  placeholder="Acme Labs"
                  maxLength={200}
                />
              </FormField>

              <ImageField
                id="logo"
                label="Logo"
                description="PNG or JPEG, up to 300KB. Rendered above the company name."
                value={config.logo_data_url}
                onChange={(v) => set({ logo_data_url: v })}
                previewClass="max-h-12 max-w-44"
              />

              <ImageField
                id="signature"
                label="Signature"
                description="PNG or JPEG, up to 300KB. Shown in the authorized-signatory block."
                value={config.signature_data_url}
                onChange={(v) => set({ signature_data_url: v })}
                previewClass="max-h-10 max-w-36"
              />

              <FormField
                label="Signatory name"
                htmlFor="signatory_name"
                description="Printed under the signature line."
              >
                <Input
                  id="signatory_name"
                  value={config.signatory_name}
                  onChange={(e) => set({ signatory_name: e.target.value })}
                  placeholder="Jordan Doe, Director"
                  maxLength={200}
                />
              </FormField>

              <FormField
                label="Bank details"
                htmlFor="bank_details"
                description="Remittance details shown in the invoice footer — bank, account, IFSC/routing."
              >
                <Textarea
                  id="bank_details"
                  value={config.bank_details}
                  onChange={(e) => set({ bank_details: e.target.value })}
                  placeholder={"Bank: HDFC Bank\nAccount: 1234567890\nIFSC: HDFC0000001"}
                  rows={3}
                  maxLength={4000}
                  className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
                />
              </FormField>

              <FormField
                label="Terms & conditions"
                htmlFor="terms"
                description="Shown at the bottom of every invoice."
              >
                <Textarea
                  id="terms"
                  value={config.terms}
                  onChange={(e) => set({ terms: e.target.value })}
                  placeholder="Payment due within 30 days of the invoice date."
                  rows={3}
                  maxLength={4000}
                  className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
                />
              </FormField>

              <div className="flex justify-end">
                <Button type="submit" disabled={saving}>
                  <Save className="h-4 w-4" />
                  {saving ? "Saving…" : "Save branding"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </form>
      )}
    </div>
  );
}

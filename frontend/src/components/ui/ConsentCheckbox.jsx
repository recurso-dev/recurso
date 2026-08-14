import { useState } from "react";

import { cn } from "@/lib/utils";
import { Checkbox } from "@/components/ui/checkbox";

/**
 * ConsentCheckbox — a consent card for a money/legal action (RBI e-mandate
 * authorization, terms, marketing opt-in). Uses the design-system tokens and
 * the shared Checkbox; the consent contract (onConsentChange payload, the
 * stored consent text, the version) is unchanged.
 */
const ConsentCheckbox = ({
  id,
  type = "recurring_billing",
  planName = "subscription",
  amount = "",
  billingInterval = "month",
  onConsentChange,
  required = true,
}) => {
  const [consented, setConsented] = useState(false);

  // The full legal text stored with the consent record (unchanged).
  const consentTexts = {
    recurring_billing: `I authorize recurring charges of ${amount} per ${billingInterval} to my payment method for the ${planName} plan. I understand that:
• I will be charged automatically on each billing cycle
• I will receive a reminder email 24 hours before each charge
• I can cancel my subscription at any time from my account dashboard
• Refunds are processed according to the refund policy`,
    terms_of_service: `I have read and agree to the Terms of Service and Privacy Policy.`,
    email_marketing: `I agree to receive product updates and promotional emails. I can unsubscribe at any time.`,
  };

  const handleChange = (e) => {
    const checked = e.target.checked;
    setConsented(checked);
    if (onConsentChange) {
      onConsentChange({
        type,
        granted: checked,
        consentText: consentTexts[type],
        version: "2024.01.1",
      });
    }
  };

  const linkClass =
    "font-medium text-primary underline underline-offset-2 hover:text-primary/80";

  return (
    <label
      className={cn(
        "my-4 flex cursor-pointer gap-3 rounded-xl border p-4 transition-colors duration-fast ease-standard",
        consented
          ? "border-primary bg-primary/5"
          : "border-border bg-muted/40 hover:border-input hover:bg-muted",
      )}
    >
      <Checkbox
        id={id}
        checked={consented}
        onChange={handleChange}
        required={required}
        className="mt-0.5"
      />
      <div className="flex-1 text-sm leading-relaxed text-muted-foreground">
        {type === "recurring_billing" && (
          <>
            <strong className="mb-1 block text-sm font-semibold text-foreground">
              Authorize Recurring Payments
            </strong>
            <p className="mb-2">
              I authorize recurring charges of{" "}
              <strong className="font-semibold text-foreground">{amount}</strong> per{" "}
              {billingInterval} to my payment method. I will receive a reminder 24
              hours before each charge and can cancel anytime.
            </p>
            <a href="/terms" target="_blank" rel="noopener noreferrer" className={cn(linkClass, "text-xs")}>
              View full authorization terms
            </a>
          </>
        )}
        {type === "terms_of_service" && (
          <>
            I agree to the{" "}
            <a href="/terms" target="_blank" rel="noopener noreferrer" className={linkClass}>
              Terms of Service
            </a>{" "}
            and{" "}
            <a href="/privacy" target="_blank" rel="noopener noreferrer" className={linkClass}>
              Privacy Policy
            </a>
          </>
        )}
        {type === "email_marketing" && (
          <>
            I agree to receive product updates and promotional emails.
            <span className="ml-2 inline-block rounded bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
              Optional
            </span>
          </>
        )}
      </div>
    </label>
  );
};

export default ConsentCheckbox;

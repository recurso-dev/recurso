import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { cn, shortId } from "@/lib/utils";

/**
 * CopyableId — the canonical technical-identifier affordance. The audit found
 * copy-to-clipboard reimplemented ≥6 times (AuditLog, Events, Integrations,
 * CreditNoteDetail, SubscriptionDetail, PaymentGateways) with divergent
 * feedback. One treatment: shortened mono text, a labelled copy button, and a
 * visible + announced "copied" confirmation.
 *
 * Props:
 *  - value:  the full identifier (what gets copied)
 *  - label:  accessible name context, e.g. "invoice ID" → aria-label
 *            "Copy invoice ID" (default "ID")
 *  - full:   render the full value instead of shortId(value)
 */
export function CopyableId({ value, label = "ID", full = false, className }) {
  const [copied, setCopied] = useState(false);
  if (!value) return <span className="text-xs text-muted-foreground">—</span>;

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(String(value));
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable — leave the text selectable */
    }
  };

  return (
    <span className={cn("inline-flex max-w-full min-w-0 items-center gap-1.5", className)}>
      <code className="truncate font-mono text-xs text-muted-foreground">
        {full ? value : shortId(value)}
      </code>
      <button
        type="button"
        onClick={copy}
        aria-label={copied ? "Copied" : `Copy ${label}`}
        className="rounded p-0.5 text-subtle transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {copied ? (
          <Check className="h-3.5 w-3.5 text-success" aria-hidden="true" />
        ) : (
          <Copy className="h-3.5 w-3.5" aria-hidden="true" />
        )}
      </button>
      <span aria-live="polite" className="sr-only">
        {copied ? "Copied to clipboard" : ""}
      </span>
    </span>
  );
}

export default CopyableId;

import { useState } from "react";
import { Check, Copy, Eye, EyeOff } from "lucide-react";

import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

// CopyableSecret — a read-only secret (API key, signing secret) with a copy
// button that CONFIRMS the copy, plus an optional reveal toggle for masked
// values. Consolidates the several hand-rolled "readonly input + copy icon"
// blocks, none of which gave any copied feedback.
export function CopyableSecret({
  value,
  ariaLabel = "Secret value",
  mask = false,
  className,
}) {
  const [copied, setCopied] = useState(false);
  const [revealed, setRevealed] = useState(!mask);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value ?? "");
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard unavailable (insecure context / denied) — the field is
      // focus-selected so it can still be copied by hand.
    }
  };

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Input
        readOnly
        value={value ?? ""}
        aria-label={ariaLabel}
        type={mask && !revealed ? "password" : "text"}
        className="font-mono"
        onFocus={(e) => e.target.select()}
      />
      {mask && (
        <Button
          type="button"
          variant="outline"
          size="icon"
          onClick={() => setRevealed((r) => !r)}
          aria-pressed={revealed}
          aria-label={revealed ? "Hide secret" : "Reveal secret"}
          title={revealed ? "Hide" : "Reveal"}
        >
          {revealed ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </Button>
      )}
      <Button
        type="button"
        variant="outline"
        size="icon"
        onClick={copy}
        aria-label={copied ? "Copied to clipboard" : "Copy to clipboard"}
        title={copied ? "Copied" : "Copy"}
      >
        {copied ? <Check className="h-4 w-4 text-primary" /> : <Copy className="h-4 w-4" />}
      </Button>
    </div>
  );
}

import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { cn } from "@/lib/utils";

// CodeSample — a read-only code block with optional language tabs and a
// one-click copy button. Follows the app's light code-block convention
// (bg-muted, JetBrains Mono via font-mono), not a dark terminal, to stay on the
// warm-stone identity.
//
// Usage:
//   <CodeSample tabs={[{ label: "cURL", code: "..." }, { label: "Node", code: "..." }]} />
//   <CodeSample label="cURL" code="curl ..." />
export function CodeSample({ tabs, code, label, caption, className }) {
  const items = tabs && tabs.length ? tabs : [{ label: label || "Code", code: code || "" }];
  const [active, setActive] = useState(0);
  const [copied, setCopied] = useState(false);
  const current = items[Math.min(active, items.length - 1)];

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(current.code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard unavailable (insecure context / denied) — fail quietly; the
      // code is still selectable by hand.
    }
  };

  return (
    <figure className={cn("overflow-hidden rounded-lg border border-border", className)}>
      <div className="flex items-center justify-between gap-2 border-b border-border bg-muted/50 px-2 py-1">
        <div className="flex items-center gap-0.5" role="tablist" aria-label="Code language">
          {items.map((t, i) => (
            <button
              key={t.label}
              type="button"
              role="tab"
              aria-selected={i === active}
              onClick={() => setActive(i)}
              className={cn(
                "rounded px-2 py-1 text-xs font-medium transition-colors",
                i === active
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
        <button
          type="button"
          onClick={copy}
          className="inline-flex items-center gap-1.5 rounded px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={copied ? "Copied to clipboard" : "Copy code"}
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-primary" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto bg-muted px-4 py-3 font-mono text-xs leading-relaxed text-foreground">
        <code>{current.code}</code>
      </pre>
      {caption && (
        <figcaption className="border-t border-border bg-muted/30 px-4 py-2 text-xs text-muted-foreground">
          {caption}
        </figcaption>
      )}
    </figure>
  );
}

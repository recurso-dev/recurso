import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { ArrowRight, Plus, Search, BookOpen, Library, Code2, ExternalLink } from "lucide-react";

import { cn } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { ALL_DESTINATIONS } from "@/lib/navigation";
import { DOCS_HOME, DOCS_GUIDES, DOCS_API_REFERENCE } from "@/lib/docsLinks";

// Derived from the canonical navigation definition (lib/navigation.js) —
// the palette can no longer drift from the sidebar (audit IA finding #13:
// the old hand-coded list was 18 destinations stale and duplicated
// Products/Plans).
const DESTINATIONS = [
  ...ALL_DESTINATIONS.map((d) => ({
    group: "Go to",
    label: d.label,
    to: d.to,
    icon: d.icon || ArrowRight,
  })),
  { group: "Create", label: "New customer", to: "/customers/new", icon: Plus },
  { group: "Create", label: "New plan", to: "/plans/new", icon: Plus },
  { group: "Create", label: "New subscription", to: "/subscriptions/new", icon: Plus },
  { group: "Create", label: "New coupon", to: "/coupons/new", icon: Plus },
  { group: "Create", label: "New quote", to: "/quotes/new", icon: Plus },
  { group: "Create", label: "New credit note", to: "/credit-notes/new", icon: Plus },
  { group: "Help", label: "Documentation", href: DOCS_HOME, icon: BookOpen },
  { group: "Help", label: "Dashboard guides", href: DOCS_GUIDES, icon: Library },
  { group: "Help", label: "API reference", href: DOCS_API_REFERENCE, icon: Code2 },
];

// CommandPalette is the dashboard's keyboard-first navigator (⌘K / Ctrl-K).
export function CommandPalette({ open, onOpenChange }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef(null);

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return DESTINATIONS;
    return DESTINATIONS.filter((d) => d.label.toLowerCase().includes(q));
  }, [query]);

  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
    }
  }, [open]);

  useEffect(() => setActive(0), [query]);

  // Keep the highlighted option scrolled into view while arrowing.
  useEffect(() => {
    document
      .getElementById(`palette-option-${active}`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const go = (item) => {
    onOpenChange(false);
    if (item.href) {
      window.open(item.href, "_blank", "noopener,noreferrer");
      return;
    }
    navigate(item.to);
  };

  const onKeyDown = (e) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => Math.min(a + 1, results.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, 0));
    } else if (e.key === "Enter" && results[active]) {
      e.preventDefault();
      go(results[active]);
    }
  };

  let lastGroup = null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="top-[20%] translate-y-0 gap-0 overflow-hidden p-0 sm:max-w-lg">
        <DialogTitle className="sr-only">Search the dashboard</DialogTitle>
        <div className="flex items-center gap-2 border-b border-border px-3">
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
          <input
            ref={inputRef}
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Where to?"
            // Combobox + activedescendant: DOM focus stays here while arrow
            // keys move the highlighted option — the standard listbox pattern.
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-listbox"
            aria-activedescendant={results[active] ? `palette-option-${active}` : undefined}
            aria-autocomplete="list"
            aria-label="Search the dashboard"
            className="h-11 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          <kbd>esc</kbd>
        </div>
        {/* Result count for screen readers — filtering is otherwise silent. */}
        <p aria-live="polite" className="sr-only">
          {results.length} result{results.length === 1 ? "" : "s"}
        </p>
        <div id="palette-listbox" role="listbox" aria-label="Destinations" className="max-h-72 overflow-y-auto p-1.5">
          {results.length === 0 && (
            <p className="px-3 py-6 text-center text-sm text-muted-foreground">
              Nothing matches "{query}".
            </p>
          )}
          {results.map((item, i) => {
            const showGroup = item.group !== lastGroup;
            lastGroup = item.group;
            const Icon = item.icon;
            return (
              <div key={item.group + item.label}>
                {showGroup && (
                  <p className="px-3 pb-1 pt-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                    {item.group}
                  </p>
                )}
                <button
                  type="button"
                  role="option"
                  id={`palette-option-${i}`}
                  aria-selected={i === active}
                  onClick={() => go(item)}
                  onMouseEnter={() => setActive(i)}
                  className={cn(
                    "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm",
                    i === active
                      ? "bg-accent text-foreground ring-2 ring-inset ring-ring"
                      : "text-muted-foreground"
                  )}
                >
                  <Icon className="h-4 w-4 text-subtle" />
                  <span className="flex-1">{item.label}</span>
                  {item.href && <ExternalLink className="h-3.5 w-3.5 text-subtle/60" />}
                </button>
              </div>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default CommandPalette;

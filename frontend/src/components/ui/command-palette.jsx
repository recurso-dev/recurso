import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  Home, Users, Package, Layers, Repeat, Receipt, ScrollText, FileMinus,
  Ticket, Megaphone, Gift, Brain, Landmark, Scale, BarChart3, Code2,
  Settings, ShieldCheck, UserCog, Plus, Search,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

const DESTINATIONS = [
  { group: "Go to", label: "Home", to: "/", icon: Home },
  { group: "Go to", label: "Customers", to: "/customers", icon: Users },
  { group: "Go to", label: "Products", to: "/products", icon: Package },
  { group: "Go to", label: "Plans", to: "/plans", icon: Layers },
  { group: "Go to", label: "Subscriptions", to: "/subscriptions", icon: Repeat },
  { group: "Go to", label: "Invoices", to: "/invoices", icon: Receipt },
  { group: "Go to", label: "Quotes", to: "/quotes", icon: ScrollText },
  { group: "Go to", label: "Credit Notes", to: "/credit-notes", icon: FileMinus },
  { group: "Go to", label: "Coupons", to: "/coupons", icon: Ticket },
  { group: "Go to", label: "Referrals", to: "/referrals", icon: Megaphone },
  { group: "Go to", label: "Gifts", to: "/gifts", icon: Gift },
  { group: "Go to", label: "Dunning", to: "/dunning", icon: Brain },
  { group: "Go to", label: "Ledger", to: "/ledger", icon: Landmark },
  { group: "Go to", label: "Reconciliation", to: "/finance/reconciliation", icon: Scale },
  { group: "Go to", label: "Usage", to: "/usage", icon: BarChart3 },
  { group: "Go to", label: "Developers", to: "/developers", icon: Code2 },
  { group: "Go to", label: "Settings", to: "/settings", icon: Settings },
  { group: "Go to", label: "Security", to: "/security", icon: ShieldCheck },
  { group: "Go to", label: "Team", to: "/team", icon: UserCog },
  { group: "Create", label: "New customer", to: "/customers/new", icon: Plus },
  { group: "Create", label: "New plan", to: "/plans/new", icon: Plus },
  { group: "Create", label: "New subscription", to: "/subscriptions/new", icon: Plus },
  { group: "Create", label: "New coupon", to: "/coupons/new", icon: Plus },
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

  const go = (item) => {
    onOpenChange(false);
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
            className="h-11 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          <kbd>esc</kbd>
        </div>
        <div className="max-h-72 overflow-y-auto p-1.5">
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
                  onClick={() => go(item)}
                  onMouseEnter={() => setActive(i)}
                  className={cn(
                    "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm",
                    i === active
                      ? "bg-stone-100 text-foreground"
                      : "text-stone-600"
                  )}
                >
                  <Icon className="h-4 w-4 text-stone-400" />
                  {item.label}
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

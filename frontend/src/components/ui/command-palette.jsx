import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight, Plus, Search, BookOpen, Library, Code2, ExternalLink,
  Users, Package, Repeat, Loader2, AlertTriangle,
} from "lucide-react";

import { cn, shortId, formatCurrency } from "@/lib/utils";
import { Overline } from "@/components/ui/overline";
import { endpoints } from "@/lib/api";
import { useDebounce } from "@/hooks/useDebounce";
import { rankResults } from "@/lib/paletteSearch";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { StatusBadge } from "@/components/ui/status-badge";
import { ALL_DESTINATIONS } from "@/lib/navigation";
import { DOCS_HOME, DOCS_GUIDES, DOCS_API_REFERENCE } from "@/lib/docsLinks";

// Route / create / help destinations — derived from the canonical navigation
// (lib/navigation.js) so the palette can't drift from the sidebar. Unchanged.
const DESTINATIONS = [
  ...ALL_DESTINATIONS.map((d) => ({
    kind: "nav", group: "Go to", label: d.label, to: d.to, icon: d.icon || ArrowRight,
  })),
  { kind: "nav", group: "Create", label: "New customer", to: "/customers/new", icon: Plus },
  { kind: "nav", group: "Create", label: "New plan", to: "/plans/new", icon: Plus },
  { kind: "nav", group: "Create", label: "New subscription", to: "/subscriptions/new", icon: Plus },
  { kind: "nav", group: "Create", label: "New coupon", to: "/coupons/new", icon: Plus },
  { kind: "nav", group: "Create", label: "New quote", to: "/quotes/new", icon: Plus },
  { kind: "nav", group: "Create", label: "New credit note", to: "/credit-notes/new", icon: Plus },
  { kind: "nav", group: "Help", label: "Documentation", href: DOCS_HOME, icon: BookOpen },
  { kind: "nav", group: "Help", label: "Dashboard guides", href: DOCS_GUIDES, icon: Library },
  { kind: "nav", group: "Help", label: "API reference", href: DOCS_API_REFERENCE, icon: Code2 },
];

const MIN_QUERY = 2;
const LIMIT = 6;
const DEBOUNCE_MS = 200;

const intervalAbbr = (u) => (u === "year" ? "yr" : u === "month" ? "mo" : u || "");

// One object search. Keyed by `q` so a stale response lands in a DIFFERENT cache
// entry and can never overwrite the current query; the AbortSignal lets
// react-query cancel the network. staleTime makes a re-typed query instant.
function useObjectSearch(type, q, enabled, getter) {
  return useQuery({
    queryKey: ["palette", type, q],
    queryFn: async ({ signal }) => (await getter({ q, limit: LIMIT }, { signal }))?.data?.data || [],
    enabled,
    staleTime: 30_000,
    retry: false,
  });
}

export function CommandPalette({ open, onOpenChange }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef(null);

  const debounced = useDebounce(query, DEBOUNCE_MS);
  const q = debounced.trim();
  const searchEnabled = open && q.length >= MIN_QUERY;

  // Parallel per-object fan-out (Customers / Plans / Subscriptions).
  const customersQ = useObjectSearch("customers", q, searchEnabled, endpoints.getCustomers);
  const plansQ = useObjectSearch("plans", q, searchEnabled, endpoints.getPlans);
  const subsQ = useObjectSearch("subscriptions", q, searchEnabled, endpoints.getSubscriptions);

  // Read the shared id→name caches WITHOUT triggering their large fetch —
  // getQueryData is a synchronous snapshot (a list page may have loaded it, else
  // undefined → we degrade to a short id). Never downloaded just for search;
  // re-read when subscription results arrive (the cache is usually warm by then).
  const custName = useMemo(() => {
    const arr = queryClient.getQueryData(["customers", "all"]) || [];
    return Object.fromEntries(arr.map((c) => [c.id, c.name]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryClient, subsQ.data]);
  const planName = useMemo(() => {
    const arr = queryClient.getQueryData(["plans", "all"]) || [];
    return Object.fromEntries(arr.map((p) => [p.id, p.name]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryClient, subsQ.data]);

  const sections = useMemo(() => {
    const out = [];

    if (searchEnabled) {
      // Customers — identity is the name, context is the email.
      out.push(buildSection("Customers", Users, customersQ, "/customers", q, {
        map: (c) => ({ to: `/customers/${c.id}`, primary: c.name || shortId(c.id), secondary: c.email }),
        rank: { id: (c) => c.id, name: (c) => c.name, secondary: (c) => c.email },
      }));
      // Plans — identity is the name, context is code + price.
      out.push(buildSection("Plans", Package, plansQ, "/plans", q, {
        map: (p) => {
          const price = p.prices?.[0];
          const sub = price
            ? `${p.code} · ${formatCurrency(price.amount, price.currency)}/${intervalAbbr(p.interval_unit)}`
            : p.code;
          return { to: `/plans/${p.id}`, primary: p.name || shortId(p.id), secondary: sub };
        },
        rank: { id: (p) => p.id, code: (p) => p.code, name: (p) => p.name, secondary: (p) => p.code },
      }));
      // Subscriptions — identity is the plan, context is the customer, plus state.
      out.push(buildSection("Subscriptions", Repeat, subsQ, "/subscriptions", q, {
        map: (s) => ({
          to: `/subscriptions/${s.id}`,
          primary: planName[s.plan_id] || "Subscription",
          secondary: custName[s.customer_id] || shortId(s.customer_id),
          status: s.status,
        }),
        rank: { id: (s) => s.id, name: (s) => custName[s.customer_id], secondary: (s) => s.status },
      }));
    }

    // Route / create / help — instant local filter (never blocks on the network).
    const navFiltered = q
      ? DESTINATIONS.filter((d) => d.label.toLowerCase().includes(q.toLowerCase()))
      : DESTINATIONS;
    for (const group of ["Go to", "Create", "Help"]) {
      const items = navFiltered.filter((d) => d.group === group);
      if (items.length) out.push({ group, items, loading: false, error: false, searchAll: null });
    }

    // Object groups only appear when they're loading, errored, or have results.
    return out.filter((s) => s.loading || s.error || s.items.length > 0);
  }, [searchEnabled, q, customersQ, plansQ, subsQ, custName, planName]);

  // Flat, navigable option list (object results + "search all" + nav items;
  // NOT loading/error rows) — the keyboard index space.
  const options = useMemo(() => {
    const flat = [];
    sections.forEach((s) => {
      flat.push(...s.items);
      if (s.searchAll) flat.push(s.searchAll);
    });
    return flat;
  }, [sections]);

  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
    }
  }, [open]);
  useEffect(() => setActive(0), [q, options.length]);
  useEffect(() => {
    document.getElementById(`palette-option-${active}`)?.scrollIntoView?.({ block: "nearest" });
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
      setActive((a) => Math.min(a + 1, options.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(a - 1, 0));
    } else if (e.key === "Enter" && options[active]) {
      e.preventDefault();
      go(options[active]);
    }
  };

  const anyLoading = searchEnabled && (customersQ.isLoading || plansQ.isLoading || subsQ.isLoading);
  const noResults = q.length >= MIN_QUERY && !anyLoading && options.length === 0;

  let optionIndex = -1; // running index aligned with `options`

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="top-[20%] translate-y-0 gap-0 overflow-hidden p-0 sm:max-w-lg">
        <DialogTitle className="sr-only">Search Recurso</DialogTitle>
        <div className="flex items-center gap-2 border-b border-border px-3">
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
          <input
            ref={inputRef}
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={onKeyDown}
            placeholder="Search Recurso…"
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-listbox"
            aria-activedescendant={options[active] ? `palette-option-${active}` : undefined}
            aria-autocomplete="list"
            aria-label="Search Recurso"
            className="h-11 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          {anyLoading && <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-subtle" aria-hidden="true" />}
          <kbd>esc</kbd>
        </div>
        <p aria-live="polite" className="sr-only">
          {anyLoading ? "Searching…" : `${options.length} result${options.length === 1 ? "" : "s"}`}
        </p>
        <div id="palette-listbox" role="listbox" aria-label="Search results" className="max-h-80 overflow-y-auto p-1.5">
          {noResults && (
            <p className="px-3 py-6 text-center text-sm text-muted-foreground">
              Nothing matches &quot;{q}&quot;.
            </p>
          )}
          {sections.map((s) => (
            <div key={s.group}>
              <Overline as="p" className="px-3 pb-1 pt-2">
                {s.group}
              </Overline>
              {s.loading && s.items.length === 0 && (
                <p className="flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" /> Searching…
                </p>
              )}
              {s.error && (
                <p className="flex items-center gap-2 px-3 py-2 text-sm text-warning" role="status">
                  <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
                  Couldn&apos;t search {s.group.toLowerCase()}.
                </p>
              )}
              {s.items.map((item) => {
                optionIndex += 1;
                return <Option key={item.to || item.href} item={item} index={optionIndex} active={active} onGo={go} onHover={setActive} />;
              })}
              {s.searchAll &&
                (() => {
                  optionIndex += 1;
                  return <Option key={s.searchAll.to} item={s.searchAll} index={optionIndex} active={active} onGo={go} onHover={setActive} />;
                })()}
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

// buildSection turns one object list-query into a render-ready section:
// ranked+mapped items, a loading/error flag, and a "search all" deep-link when
// the result set was capped at LIMIT (there may be more on the list page).
function buildSection(group, icon, listQuery, listPath, q, { map, rank }) {
  const rows = listQuery.data || [];
  const ranked = rankResults(rows, q, rank);
  const items = ranked.map((row) => ({ kind: "object", group, icon, ...map(row) }));
  const capped = rows.length >= LIMIT;
  return {
    group,
    icon,
    loading: listQuery.isLoading,
    error: listQuery.isError,
    items,
    searchAll: capped
      ? {
          kind: "searchAll",
          group,
          to: `${listPath}?q=${encodeURIComponent(q)}`,
          searchAllLabel: `Search all ${group.toLowerCase()} for “${q}”`,
        }
      : null,
  };
}

function Option({ item, index, active, onGo, onHover }) {
  const isActive = index === active;
  const Icon = item.icon;
  if (item.kind === "searchAll") {
    return (
      <button
        type="button"
        role="option"
        id={`palette-option-${index}`}
        aria-selected={isActive}
        onClick={() => onGo(item)}
        onMouseEnter={() => onHover(index)}
        className={cn(
          "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm",
          isActive ? "bg-accent text-foreground ring-2 ring-inset ring-ring" : "text-muted-foreground"
        )}
      >
        <Search className="h-4 w-4 text-subtle" />
        <span className="flex-1">{item.searchAllLabel}</span>
        <ArrowRight className="h-3.5 w-3.5 text-subtle/60" />
      </button>
    );
  }
  return (
    <button
      type="button"
      role="option"
      id={`palette-option-${index}`}
      aria-selected={isActive}
      onClick={() => onGo(item)}
      onMouseEnter={() => onHover(index)}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-sm",
        isActive ? "bg-accent ring-2 ring-inset ring-ring" : ""
      )}
    >
      <Icon className={cn("h-4 w-4 shrink-0", isActive ? "text-foreground" : "text-subtle")} />
      {item.kind === "object" ? (
        <>
          <span className="min-w-0 flex-1">
            <span className={cn("block truncate", isActive ? "text-foreground" : "text-foreground")}>
              {item.primary}
            </span>
            {item.secondary && (
              <span className="block truncate text-xs text-muted-foreground">{item.secondary}</span>
            )}
          </span>
          {item.status && <StatusBadge status={item.status} />}
        </>
      ) : (
        <span className={cn("flex-1", isActive ? "text-foreground" : "text-muted-foreground")}>
          {item.label}
          {item.href && <ExternalLink className="ml-1 inline h-3.5 w-3.5 text-subtle/60" />}
        </span>
      )}
    </button>
  );
}

export default CommandPalette;

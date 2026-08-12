import { NavLink } from "react-router";
import { Home, LayoutDashboard, Users, Layers, Repeat, Receipt, ScrollText, FileMinus, Ticket, Megaphone, Gift, Brain, Landmark, Scale, BookOpenCheck, Waves, CalendarClock, TrendingUp, FileClock, Gauge, PieChart, Globe, BarChart3, Code2, Settings, UserCog, ShieldCheck, Wallet2, Plug, MailWarning, HeartHandshake, TrendingDown, Repeat2, Banknote, FileQuestion, Building2, Sparkles, FileSpreadsheet, ClipboardCheck, Inbox, BookOpen, ArrowUpRight, Webhook } from "lucide-react";

import { cn } from "@/lib/utils";
import { DOCS_HOME } from "@/lib/docsLinks";

// Grouped navigation. Each item: { to, label, icon, end? }.
// `end` forces exact matching (used for Home so it isn't active everywhere).
const NAV_GROUPS = [
  {
    label: "",
    items: [
      { to: "/", label: "Home", icon: Home, end: true },
      { to: "/ask", label: "Ask AI", icon: Sparkles },
    ],
  },
  {
    label: "Billing",
    items: [
      { to: "/customers", label: "Customers", icon: Users },
      { to: "/subscriptions", label: "Subscriptions", icon: Repeat },
      { to: "/plans", label: "Plans", icon: Layers },
      { to: "/invoices", label: "Invoices", icon: Receipt },
      { to: "/quotes", label: "Quotes", icon: ScrollText },
      { to: "/credit-notes", label: "Credit Notes", icon: FileMinus },
      { to: "/coupons", label: "Coupons", icon: Ticket },
      { to: "/gifts", label: "Gifts", icon: Gift },
    ],
  },
  {
    label: "Usage",
    items: [
      { to: "/metering", label: "Metering", icon: Gauge },
      { to: "/usage", label: "Usage Explorer", icon: BarChart3 },
      { to: "/wallets", label: "Wallets", icon: Wallet2 },
    ],
  },
  {
    label: "Revenue Recovery",
    items: [
      { to: "/collections", label: "Collections", icon: Inbox },
      { to: "/dunning", label: "Dunning", icon: Brain },
      { to: "/dunning/campaigns", label: "Dunning Campaigns", icon: MailWarning },
      { to: "/disputes", label: "Disputes", icon: FileQuestion },
      { to: "/cancel-flows", label: "Cancel Flows", icon: HeartHandshake },
      { to: "/churn", label: "Churn Risk", icon: TrendingDown },
    ],
  },
  {
    label: "Payments",
    items: [
      { to: "/mandates", label: "Mandates", icon: Repeat2 },
      { to: "/payments/offline", label: "Offline Payments", icon: Banknote },
      { to: "/referrals", label: "Referrals", icon: Megaphone },
    ],
  },
  {
    // Books-of-record: the auditor/controller surface. Reconciliation leads —
    // the self-verifying ledger is the product's identity, not a sub-feature.
    label: "Books",
    items: [
      { to: "/finance/reconciliation", label: "Reconciliation", icon: Scale },
      { to: "/ledger", label: "Ledger", icon: Landmark },
      { to: "/finance/trial-balance", label: "Trial Balance", icon: BookOpenCheck },
      { to: "/finance/close", label: "Month-End Close", icon: ClipboardCheck },
      { to: "/finance/revenue-recognition", label: "Revenue Recognition", icon: CalendarClock },
      { to: "/finance/entities", label: "Entities", icon: Building2 },
      { to: "/finance/gst-returns", label: "GST Returns", icon: FileSpreadsheet },
      { to: "/audit-log", label: "Audit Log", icon: ShieldCheck },
    ],
  },
  {
    // Analytics, cleanly split from books-of-record: the CFO's morning tab.
    label: "Reports",
    items: [
      { to: "/overview", label: "Executive Summary", icon: LayoutDashboard },
      { to: "/finance/revenue-waterfall", label: "Revenue Waterfall", icon: Waves },
      { to: "/finance/mrr-waterfall", label: "MRR Waterfall", icon: TrendingUp },
      { to: "/finance/invoice-aging", label: "Invoice Aging", icon: FileClock },
      { to: "/finance/unit-economics", label: "Unit Economics", icon: Gauge },
      { to: "/finance/revenue-by-plan", label: "Revenue by Plan", icon: PieChart },
      { to: "/finance/revenue-by-geography", label: "Revenue by Geography", icon: Globe },
    ],
  },
  {
    label: "System",
    items: [
      { to: "/developers", label: "Developers", icon: Code2 },
      { to: "/events", label: "Events", icon: Webhook },
      { to: "/integrations", label: "Integrations", icon: Plug },
      { to: "/settings", label: "Settings", icon: Settings },
      { to: "/security", label: "Security", icon: ShieldCheck },
      { to: "/team", label: "Team", icon: UserCog },
      { to: "/organizations", label: "Organizations", icon: Building2 },
    ],
  },
];

function SidebarItem({ to, label, icon: Icon, end }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          "group flex items-center gap-2.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
          isActive
            ? "bg-primary/10 text-primary"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        )
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            className={cn(
              "h-4 w-4 shrink-0",
              isActive ? "text-primary" : "text-subtle group-hover:text-foreground"
            )}
          />
          <span className="truncate">{label}</span>
        </>
      )}
    </NavLink>
  );
}

export function Sidebar() {
  return (
    <aside className="flex h-full w-60 flex-col border-r border-border bg-background">
      {/* Brand */}
      <div className="flex h-16 items-center gap-2.5 border-b border-border px-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
          <Layers className="h-4 w-4" />
        </div>
        <span className="text-sm font-semibold tracking-tight text-foreground">
          Recurso
        </span>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {NAV_GROUPS.map((group, i) => (
          <div key={group.label || i} className="mb-5 last:mb-0">
            {group.label && (
              <p className="mb-1.5 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {group.label}
              </p>
            )}
            <div className="space-y-0.5">
              {group.items.map((item) => (
                <SidebarItem key={item.to} {...item} />
              ))}
            </div>
          </div>
        ))}
      </nav>

      {/* Docs — a persistent link to the guides, always in reach. */}
      <div className="shrink-0 border-t border-border p-3">
        <a
          href={DOCS_HOME}
          target="_blank"
          rel="noopener noreferrer"
          className="group flex items-center gap-2.5 rounded-md px-3 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <BookOpen className="h-4 w-4 shrink-0 text-subtle group-hover:text-foreground" />
          <span className="flex-1 truncate">Documentation</span>
          <ArrowUpRight className="h-3.5 w-3.5 text-subtle/60 group-hover:text-subtle" />
        </a>
      </div>
    </aside>
  );
}

export default Sidebar;

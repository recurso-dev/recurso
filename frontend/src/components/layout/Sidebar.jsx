import { NavLink } from "react-router-dom";
import { Home, LayoutDashboard, Users, Package, Layers, Repeat, Receipt, ScrollText, FileMinus, Ticket, Megaphone, Gift, Brain, Landmark, Scale, BookOpenCheck, Waves, CalendarClock, TrendingUp, FileClock, Gauge, PieChart, Globe, BarChart3, Code2, Settings, UserCog, ShieldCheck, Wallet2, Plug, MailWarning, HeartHandshake, TrendingDown } from "lucide-react";

import { cn } from "@/lib/utils";

// Grouped navigation. Each item: { to, label, icon, end? }.
// `end` forces exact matching (used for Home so it isn't active everywhere).
const NAV_GROUPS = [
  {
    label: "Core",
    items: [
      { to: "/", label: "Home", icon: Home, end: true },
      { to: "/overview", label: "Overview", icon: LayoutDashboard },
      { to: "/customers", label: "Customers", icon: Users },
      { to: "/products", label: "Products", icon: Package },
      { to: "/plans", label: "Plans", icon: Layers },
      { to: "/subscriptions", label: "Subscriptions", icon: Repeat },
      { to: "/invoices", label: "Invoices", icon: Receipt },
      { to: "/quotes", label: "Quotes", icon: ScrollText },
      { to: "/credit-notes", label: "Credit Notes", icon: FileMinus },
    ],
  },
  {
    label: "Growth",
    items: [
      { to: "/metering", label: "Metering", icon: Gauge },
      { to: "/wallets", label: "Wallets", icon: Wallet2 },
      { to: "/coupons", label: "Coupons", icon: Ticket },
      { to: "/referrals", label: "Referrals", icon: Megaphone },
      { to: "/gifts", label: "Gifts", icon: Gift },
      { to: "/dunning", label: "Dunning", icon: Brain },
      { to: "/dunning/campaigns", label: "Dunning Campaigns", icon: MailWarning },
      { to: "/cancel-flows", label: "Cancel Flows", icon: HeartHandshake },
      { to: "/churn", label: "Churn Risk", icon: TrendingDown },
    ],
  },
  {
    label: "Finance",
    items: [
      { to: "/ledger", label: "Ledger", icon: Landmark },
      { to: "/audit-log", label: "Audit Log", icon: ShieldCheck },
      { to: "/finance/trial-balance", label: "Trial Balance", icon: BookOpenCheck },
      { to: "/finance/reconciliation", label: "Reconciliation", icon: Scale },
      { to: "/finance/revenue-recognition", label: "Revenue Recognition", icon: CalendarClock },
      { to: "/finance/revenue-waterfall", label: "Revenue Waterfall", icon: Waves },
      { to: "/finance/mrr-waterfall", label: "MRR Waterfall", icon: TrendingUp },
      { to: "/finance/invoice-aging", label: "Invoice Aging", icon: FileClock },
      { to: "/finance/unit-economics", label: "Unit Economics", icon: Gauge },
      { to: "/finance/revenue-by-plan", label: "Revenue by Plan", icon: PieChart },
      { to: "/finance/revenue-by-geography", label: "Revenue by Geography", icon: Globe },
      { to: "/usage", label: "Usage", icon: BarChart3 },
    ],
  },
  {
    label: "System",
    items: [
      { to: "/developers", label: "Developers", icon: Code2 },
      { to: "/integrations", label: "Integrations", icon: Plug },
      { to: "/settings", label: "Settings", icon: Settings },
      { to: "/security", label: "Security", icon: ShieldCheck },
      { to: "/team", label: "Team", icon: UserCog },
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
            ? "bg-emerald-50 text-emerald-700"
            : "text-stone-600 hover:bg-stone-100 hover:text-stone-900"
        )
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            className={cn(
              "h-4 w-4 shrink-0",
              isActive ? "text-emerald-600" : "text-stone-400 group-hover:text-stone-600"
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
    <aside className="flex h-full w-60 flex-col border-r border-border bg-white">
      {/* Brand */}
      <div className="flex h-16 items-center gap-2.5 border-b border-border px-5">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-emerald-500 text-white">
          <Layers className="h-4 w-4" />
        </div>
        <span className="text-[15px] font-semibold tracking-tight text-foreground">
          Recurso
        </span>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {NAV_GROUPS.map((group) => (
          <div key={group.label} className="mb-5 last:mb-0">
            <p className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-stone-400">
              {group.label}
            </p>
            <div className="space-y-0.5">
              {group.items.map((item) => (
                <SidebarItem key={item.to} {...item} />
              ))}
            </div>
          </div>
        ))}
      </nav>
    </aside>
  );
}

export default Sidebar;

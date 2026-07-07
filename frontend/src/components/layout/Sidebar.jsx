import { NavLink } from "react-router-dom";
import {
  Home,
  Users,
  Package,
  Layers,
  Repeat,
  Receipt,
  ScrollText,
  FileMinus,
  Ticket,
  Megaphone,
  Gift,
  Brain,
  Landmark,
  Scale,
  BarChart3,
  Code2,
  Settings,
  UserCog,
  ShieldCheck,
} from "lucide-react";

import { cn } from "@/lib/utils";

// Grouped navigation. Each item: { to, label, icon, end? }.
// `end` forces exact matching (used for Home so it isn't active everywhere).
const NAV_GROUPS = [
  {
    label: "Core",
    items: [
      { to: "/", label: "Home", icon: Home, end: true },
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
      { to: "/coupons", label: "Coupons", icon: Ticket },
      { to: "/referrals", label: "Referrals", icon: Megaphone },
      { to: "/gifts", label: "Gifts", icon: Gift },
      { to: "/dunning", label: "Dunning", icon: Brain },
    ],
  },
  {
    label: "Finance",
    items: [
      { to: "/ledger", label: "Ledger", icon: Landmark },
      { to: "/finance/reconciliation", label: "Reconciliation", icon: Scale },
      { to: "/usage", label: "Usage", icon: BarChart3 },
    ],
  },
  {
    label: "System",
    items: [
      { to: "/developers", label: "Developers", icon: Code2 },
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
            : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
        )
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            className={cn(
              "h-4 w-4 shrink-0",
              isActive ? "text-emerald-600" : "text-zinc-400 group-hover:text-zinc-600"
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
            <p className="mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
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

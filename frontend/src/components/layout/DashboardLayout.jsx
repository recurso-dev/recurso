import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate, Link } from "react-router-dom";
import { Search, Bell, LogOut, User, ChevronDown, FlaskConical } from "lucide-react";

import { useAuth } from "../../auth/AuthProvider";
import { API_ROOT } from "../../lib/api";
import Sidebar from "./Sidebar";
import { CommandPalette } from "@/components/ui/command-palette";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";

// Human-readable page titles keyed by the first path segment.
const TITLES = {
  "": "Home",
  customers: "Customers",
  products: "Products",
  plans: "Plans",
  subscriptions: "Subscriptions",
  invoices: "Invoices",
  quotes: "Quotes",
  "credit-notes": "Credit Notes",
  coupons: "Coupons",
  referrals: "Referrals",
  gifts: "Gifts",
  dunning: "Dunning",
  ledger: "Ledger",
  finance: "Reconciliation",
  usage: "Usage",
  developers: "Developers",
  settings: "Settings",
  notifications: "Notifications",
  profile: "Profile",
};

// Full-path titles for nested routes where the first segment isn't enough
// (e.g. both pages under /finance/*). Checked before the first-segment map.
const PATH_TITLES = {
  "/finance/reconciliation": "Reconciliation",
  "/finance/revenue-recognition": "Revenue Recognition",
  "/finance/mrr-waterfall": "MRR Waterfall",
  "/finance/invoice-aging": "Invoice Aging",
  "/finance/unit-economics": "Unit Economics",
  "/overview": "Executive Summary",
};

function usePageTitle() {
  const { pathname } = useLocation();
  const segment = pathname.split("/").filter(Boolean)[0] || "";
  return PATH_TITLES[pathname] ?? TITLES[segment] ?? "Recurso";
}

export function DashboardLayout() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const title = usePageTitle();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [gatewayMode, setGatewayMode] = useState(null);

  // ⌘K / Ctrl-K opens the command palette from anywhere in the dashboard.
  useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen((o) => !o);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // The Test-mode chip mirrors Stripe's: shown whenever the backend runs on
  // test gateway keys, so nobody mistakes sandbox money for real money.
  useEffect(() => {
    fetch(`${API_ROOT}/version`)
      .then((r) => r.json())
      .then((d) => setGatewayMode(d.gateway_mode || null))
      .catch(() => {});
  }, []);

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <div className="flex h-screen w-full overflow-hidden bg-stone-50 font-sans text-foreground">
      <Sidebar />

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border bg-white/80 px-6 backdrop-blur">
          <h1 className="text-sm font-semibold text-foreground">{title}</h1>

          <div className="flex flex-1 items-center justify-end gap-3">
            {gatewayMode === "test" && (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700">
                <FlaskConical className="h-3 w-3" />
                Test mode
              </span>
            )}

            <button
              type="button"
              onClick={() => setPaletteOpen(true)}
              className="hidden h-9 w-full max-w-xs items-center gap-2 rounded-md border border-border bg-stone-50 px-3 text-sm text-stone-400 transition-colors hover:border-stone-300 hover:text-stone-500 md:flex"
            >
              <Search className="h-4 w-4" />
              <span className="flex-1 text-left">Search…</span>
              <kbd>⌘K</kbd>
            </button>

            <Link
              to="/notifications"
              className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-white text-stone-500 transition-colors hover:bg-stone-50 hover:text-stone-900"
              aria-label="Notifications"
            >
              <Bell className="h-4 w-4" />
            </Link>

            <DropdownMenu>
              <DropdownMenuTrigger className="flex items-center gap-1.5 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>AD</AvatarFallback>
                </Avatar>
                <ChevronDown className="h-4 w-4 text-stone-400" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => navigate("/profile")}>
                  <User className="text-stone-500" />
                  Profile
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="text-red-600 focus:bg-red-50 focus:text-red-700"
                >
                  <LogOut />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />

        {/* Page content */}
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-[1400px] px-6 py-6 lg:px-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}

export default DashboardLayout;

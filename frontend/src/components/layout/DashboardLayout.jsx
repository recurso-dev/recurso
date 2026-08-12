import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate, Link } from "react-router";
import { Search, Bell, LogOut, User, ChevronDown, FlaskConical } from "lucide-react";

import { useAuth } from "../../auth/AuthProvider";
import { API_ROOT } from "../../lib/api";
import Sidebar from "./Sidebar";
import DocsHelpMenu from "./DocsHelpMenu";
import VerifyEmailBanner from "./VerifyEmailBanner";
import TrialBanner from "./TrialBanner";
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
  finance: "Finance",
  usage: "Usage",
  developers: "Developers",
  settings: "Settings",
  notifications: "Notifications",
  profile: "Profile",
  ask: "Ask AI",
  metering: "Metering",
  wallets: "Wallets",
  "cancel-flows": "Cancel Flows",
  churn: "Churn Risk",
  mandates: "Mandates",
  disputes: "Disputes",
  integrations: "Integrations",
  security: "Security",
  team: "Team",
  organizations: "Organizations",
  "audit-log": "Audit Log",
  events: "Events",
  overview: "Executive Summary",
};

// Full-path titles for nested routes where the first segment isn't enough
// (e.g. both pages under /finance/*). Checked before the first-segment map.
const PATH_TITLES = {
  "/finance/reconciliation": "Reconciliation",
  "/finance/trial-balance": "Trial Balance",
  "/finance/revenue-recognition": "Revenue Recognition",
  "/finance/revenue-waterfall": "Revenue Waterfall",
  "/finance/mrr-waterfall": "MRR Waterfall",
  "/finance/invoice-aging": "Invoice Aging",
  "/finance/unit-economics": "Unit Economics",
  "/finance/revenue-by-plan": "Revenue by Plan",
  "/finance/revenue-by-geography": "Revenue by Geography",
  "/finance/gst-returns": "GST Returns",
  "/dunning/campaigns": "Dunning Campaigns",
  "/payments/offline": "Offline Payments",
};

// Initials for the account menu — derived from the signed-in user (the
// fallback was hardcoded "AD" for everyone).
function initialsOf(user) {
  const src = user?.name || user?.email || "";
  const parts = src.replace(/@.*/, "").split(/[\s._-]+/).filter(Boolean);
  const chars = (parts.length >= 2 ? parts[0][0] + parts[1][0] : src.slice(0, 2)) || "?";
  return chars.toUpperCase();
}

function usePageTitle() {
  const { pathname } = useLocation();
  const segment = pathname.split("/").filter(Boolean)[0] || "";
  return PATH_TITLES[pathname] ?? TITLES[segment] ?? "Recurso";
}

export function DashboardLayout() {
  const { logout, user } = useAuth();
  const isDemo = user?.email === "demo@demo.recurso.dev";
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
    <div className="flex h-screen w-full overflow-hidden bg-canvas font-sans text-foreground">
      <Sidebar />

      <div className="flex min-w-0 flex-1 flex-col">
        {isDemo && (
          <div className="flex shrink-0 items-center justify-center gap-2 border-b border-warning/20 bg-warning/5 px-4 py-1.5 text-xs text-warning">
            <span className="font-semibold">Demo environment</span>
            <span className="hidden sm:inline">
              — data is public and resets every hour. API key for curl:
            </span>
            <code className="hidden rounded bg-warning/10 px-1.5 py-0.5 font-mono sm:inline">sk_test_12345</code>
          </div>
        )}
        <VerifyEmailBanner />
        <TrialBanner />
        {/* Top bar */}
        <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border bg-background/80 px-6 backdrop-blur">
          <h1 className="text-sm font-semibold text-foreground">{title}</h1>

          <div className="flex flex-1 items-center justify-end gap-3">
            {gatewayMode === "test" && (
              <span className="inline-flex items-center gap-1.5 rounded-full border border-warning/20 bg-warning/5 px-2.5 py-1 text-xs font-medium text-warning">
                <FlaskConical className="h-3 w-3" />
                Test mode
              </span>
            )}

            <button
              type="button"
              onClick={() => setPaletteOpen(true)}
              className="hidden h-9 w-full max-w-xs items-center gap-2 rounded-md border border-border bg-muted px-3 text-sm text-muted-foreground transition-colors hover:border-input hover:text-foreground md:flex"
            >
              <Search className="h-4 w-4" />
              <span className="flex-1 text-left">Search…</span>
              <kbd>⌘K</kbd>
            </button>

            <DocsHelpMenu pageTitle={title} />

            <Link
              to="/notifications"
              className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background text-subtle transition-colors hover:bg-muted hover:text-foreground"
              aria-label="Notifications"
            >
              <Bell className="h-4 w-4" />
            </Link>

            <DropdownMenu>
              <DropdownMenuTrigger className="flex items-center gap-1.5 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>{initialsOf(user)}</AvatarFallback>
                </Avatar>
                <ChevronDown className="h-4 w-4 text-subtle" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => navigate("/profile")}>
                  <User className="text-subtle" />
                  Profile
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={handleLogout}
                  className="text-destructive focus:bg-destructive/10 focus:text-destructive"
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

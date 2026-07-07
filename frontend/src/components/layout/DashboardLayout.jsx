import { Outlet, useLocation, useNavigate, Link } from "react-router-dom";
import { Search, Bell, LogOut, User, ChevronDown } from "lucide-react";

import { useAuth } from "../../auth/AuthProvider";
import Sidebar from "./Sidebar";
import { Input } from "@/components/ui/input";
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

function usePageTitle() {
  const { pathname } = useLocation();
  const segment = pathname.split("/").filter(Boolean)[0] || "";
  return TITLES[segment] ?? "Recurso";
}

export function DashboardLayout() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const title = usePageTitle();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <div className="flex h-screen w-full overflow-hidden bg-zinc-50 font-sans text-foreground">
      <Sidebar />

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-border bg-white/80 px-6 backdrop-blur">
          <h1 className="text-sm font-semibold text-foreground">{title}</h1>

          <div className="flex flex-1 items-center justify-end gap-3">
            <div className="relative hidden w-full max-w-xs md:block">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <Input
                type="search"
                placeholder="Search..."
                className="h-9 bg-zinc-50 pl-9"
              />
            </div>

            <Link
              to="/notifications"
              className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-white text-zinc-500 transition-colors hover:bg-zinc-50 hover:text-zinc-900"
              aria-label="Notifications"
            >
              <Bell className="h-4 w-4" />
            </Link>

            <DropdownMenu>
              <DropdownMenuTrigger className="flex items-center gap-1.5 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>AD</AvatarFallback>
                </Avatar>
                <ChevronDown className="h-4 w-4 text-zinc-400" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => navigate("/profile")}>
                  <User className="text-zinc-500" />
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

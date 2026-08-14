import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate, Link } from "react-router";
import { Search, Bell, LogOut, User, ChevronDown, FlaskConical, Menu } from "lucide-react";

import { useAuth } from "../../auth/AuthProvider";
import { API_ROOT } from "../../lib/api";
import Sidebar, { NavList, Brand, DocsFooterLink } from "./Sidebar";
import DocsHelpMenu from "./DocsHelpMenu";
import VerifyEmailBanner from "./VerifyEmailBanner";
import TrialBanner from "./TrialBanner";
import { labelForPath } from "@/lib/navigation";
import { MotionReveal } from "@/components/patterns";
import { CommandPalette } from "@/components/ui/command-palette";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";

// Initials for the account menu — derived from the signed-in user (the
// fallback was hardcoded "AD" for everyone).
function initialsOf(user) {
  const src = user?.name || user?.email || "";
  const parts = src.replace(/@.*/, "").split(/[\s._-]+/).filter(Boolean);
  const chars = (parts.length >= 2 ? parts[0][0] + parts[1][0] : src.slice(0, 2)) || "?";
  return chars.toUpperCase();
}

export function DashboardLayout() {
  const { logout, user } = useAuth();
  const isDemo = user?.email === "demo@demo.recurso.dev";
  const navigate = useNavigate();
  const { pathname } = useLocation();
  // Current-page label from the canonical nav definition (lib/navigation.js)
  // — the old TITLES/PATH_TITLES maps drifted from the sidebar and missed
  // pages (Collections rendered as "Recurso").
  const title = labelForPath(pathname);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
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
    <div className="flex h-dvh w-full overflow-hidden bg-canvas font-sans text-foreground">
      {/* Skip link — the sidebar is ~60 tab stops; keyboard users go straight
          to the content (WCAG 2.4.1). Visible only while focused. */}
      <a
        href="#main-content"
        className="sr-only z-50 rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground focus:not-sr-only focus:absolute focus:left-4 focus:top-4"
      >
        Skip to content
      </a>

      <Sidebar />

      {/* Mobile drawer — same NavList as the desktop rail. */}
      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent side="left" className="flex w-full flex-col gap-0 p-0 sm:max-w-xs">
          <SheetTitle className="sr-only">Navigation</SheetTitle>
          <Brand />
          <NavList onNavigate={() => setMobileNavOpen(false)} />
          <DocsFooterLink />
        </SheetContent>
      </Sheet>

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
        <header className="flex h-16 shrink-0 items-center justify-between gap-3 border-b border-border bg-background/80 px-4 backdrop-blur sm:px-6">
          <div className="flex min-w-0 items-center gap-2">
            <button
              type="button"
              onClick={() => setMobileNavOpen(true)}
              aria-label="Open navigation"
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-background text-subtle transition-colors hover:bg-muted hover:text-foreground lg:hidden"
            >
              <Menu className="h-4 w-4" />
            </button>
            {/* Context label, not a heading — each page owns its single h1
                via PageHeader (the duplicate topbar h1 inverted the outline). */}
            <span className="truncate text-sm font-semibold text-foreground">{title}</span>
          </div>

          <div className="flex flex-1 items-center justify-end gap-3">
            {gatewayMode === "test" && (
              <span
                className="inline-flex items-center gap-1.5 rounded-full border border-warning/20 bg-warning/5 px-2.5 py-1 text-xs font-medium text-warning"
                title="Test mode — no real charges"
              >
                <FlaskConical className="h-3 w-3" />
                {/* Icon-only below sm: the chip text starved the page-context
                    label to a sliver at 375px (visual QA). */}
                <span className="hidden sm:inline">Test mode</span>
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
            {/* Touch has no ⌘K — give small screens a search button. */}
            <button
              type="button"
              onClick={() => setPaletteOpen(true)}
              aria-label="Search"
              className="flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background text-subtle transition-colors hover:bg-muted hover:text-foreground md:hidden"
            >
              <Search className="h-4 w-4" />
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

        {/* Page content — a subtle fade + rise on each route change so
            navigation feels connected without a cinematic transition. Keyed on
            pathname so it replays only when the destination actually changes;
            reduced motion renders it instantly (MotionReveal). */}
        <main id="main-content" tabIndex={-1} className="flex-1 overflow-y-auto outline-none">
          <div className="mx-auto max-w-[1400px] px-4 py-6 sm:px-6 lg:px-8">
            <MotionReveal key={pathname}>
              <Outlet />
            </MotionReveal>
          </div>
        </main>
      </div>
    </div>
  );
}

export default DashboardLayout;

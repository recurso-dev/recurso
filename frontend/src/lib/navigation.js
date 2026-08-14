import {
  Home, LayoutDashboard, Users, Layers, Repeat, Receipt, ScrollText, FileMinus,
  Ticket, Megaphone, Gift, Brain, Landmark, Scale, BookOpenCheck, Waves,
  CalendarClock, TrendingUp, FileClock, Gauge, PieChart, Globe, BarChart3,
  Code2, Settings, Wallet2, Plug, MailWarning, HeartHandshake, TrendingDown,
  Repeat2, Banknote, FileQuestion, Building2, Sparkles, FileSpreadsheet,
  ClipboardCheck, Inbox, ShieldCheck, Webhook, CreditCard,
} from "lucide-react";

/**
 * The canonical navigation definition — the ONE source of truth for
 * destinations and their names (DASHBOARD_REDESIGN.md Stage 3: "Do not
 * maintain separate title maps that drift"). The sidebar renders it, the
 * command palette derives from it, and the top bar labels the current page
 * from it. A destination's `label` is its name everywhere: sidebar, top bar,
 * page header, palette.
 *
 * Grouping follows the operator's mental model (audit IA findings): Disputes
 * live with the invoice lifecycle, Gifts and Referrals are one growth pair,
 * the Audit Log is platform/System (a config change log, not a book of
 * record). Security and Team live in the Settings sub-nav, not here.
 */
export const NAV_GROUPS = [
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
      { to: "/disputes", label: "Disputes", icon: FileQuestion },
      { to: "/coupons", label: "Coupons", icon: Ticket },
    ],
  },
  {
    label: "Growth",
    items: [
      { to: "/gifts", label: "Gifts", icon: Gift },
      { to: "/referrals", label: "Referrals", icon: Megaphone },
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
      { to: "/dunning", label: "Dunning", icon: Brain, end: true },
      { to: "/dunning/campaigns", label: "Dunning Campaigns", icon: MailWarning },
      { to: "/cancel-flows", label: "Cancel Flows", icon: HeartHandshake },
      { to: "/churn", label: "Churn Risk", icon: TrendingDown },
    ],
  },
  {
    label: "Payments",
    items: [
      { to: "/payments", label: "Payments Log", icon: CreditCard, end: true },
      { to: "/mandates", label: "Mandates", icon: Repeat2 },
      { to: "/payments/offline", label: "Offline Payments", icon: Banknote },
    ],
  },
  {
    // Books-of-record: the auditor/controller surface. Reconciliation leads —
    // the self-verifying ledger is the product's identity.
    label: "Books",
    items: [
      { to: "/finance/reconciliation", label: "Reconciliation", icon: Scale },
      { to: "/ledger", label: "Ledger", icon: Landmark },
      { to: "/finance/trial-balance", label: "Trial Balance", icon: BookOpenCheck },
      { to: "/finance/close", label: "Month-End Close", icon: ClipboardCheck },
      { to: "/finance/revenue-recognition", label: "Revenue Recognition", icon: CalendarClock },
      { to: "/finance/entities", label: "Entities", icon: Building2 },
      { to: "/finance/gst-returns", label: "GST Returns", icon: FileSpreadsheet },
    ],
  },
  {
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
      { to: "/audit-log", label: "Audit Log", icon: ShieldCheck },
      { to: "/organizations", label: "Organizations", icon: Building2 },
      { to: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

// Destinations reachable outside the sidebar (header icons / menus) — still
// part of the canon so the palette and top bar can name them.
export const AUX_DESTINATIONS = [
  { to: "/notifications", label: "Notifications" },
  { to: "/profile", label: "Profile" },
  { to: "/security", label: "Security" },
  { to: "/team", label: "Team" },
  { to: "/settings/invoice-branding", label: "Invoice Branding" },
  { to: "/settings/entities", label: "Legal Entities" },
  { to: "/settings/gst", label: "GST Configuration" },
  { to: "/settings/irp", label: "E-Invoicing (IRP)" },
  { to: "/settings/eu-einvoice", label: "EU E-Invoicing" },
  { to: "/settings/tax-nexus", label: "US Sales-Tax Nexus" },
  { to: "/settings/tax-us", label: "US Tax Identity" },
  { to: "/settings/mcp", label: "MCP Server" },
  { to: "/settings/import", label: "Import Data" },
  { to: "/settings/billing", label: "Billing & Plan" },
];

export const ALL_DESTINATIONS = [
  ...NAV_GROUPS.flatMap((g) => g.items),
  ...AUX_DESTINATIONS,
];

// Longest-prefix match so /dunning/campaigns names itself, not "Dunning".
const BY_PATH = [...ALL_DESTINATIONS].sort((a, b) => b.to.length - a.to.length);

export function labelForPath(pathname) {
  const exact = ALL_DESTINATIONS.find((d) => d.to === pathname);
  if (exact) return exact.label;
  const prefix = BY_PATH.find(
    (d) => d.to !== "/" && pathname.startsWith(d.to + "/")
  );
  return prefix ? prefix.label : "Recurso";
}

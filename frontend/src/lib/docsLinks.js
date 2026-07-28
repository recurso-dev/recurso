// Maps dashboard routes to their guide on docs.recurso.dev, so the in-app
// Help menu can always deep-link to the doc for the page you're on.
//
// Resolution mirrors DashboardLayout's title logic: a full-path map is checked
// first (for nested /finance/* and /dunning/* routes that share a first
// segment), then a first-segment map, then a sensible default.

export const DOCS_BASE = "https://docs.recurso.dev";

export const DOCS_HOME = DOCS_BASE;
export const DOCS_GUIDES = `${DOCS_BASE}/dashboard/overview`;
export const DOCS_API_REFERENCE = `${DOCS_BASE}/api-reference/introduction`;

// Full-path → doc slug. Checked before the segment map.
const PATH_DOCS = {
  "/finance/entities": "dashboard/entities",
  "/finance/trial-balance": "dashboard/finance-reports",
  "/finance/reconciliation": "dashboard/finance-reports",
  "/finance/close": "dashboard/workflows/monthly-close",
  "/finance/revenue-recognition": "advanced/revenue-recognition",
  "/finance/revenue-waterfall": "dashboard/finance-reports",
  "/finance/mrr-waterfall": "dashboard/finance-reports",
  "/finance/invoice-aging": "dashboard/finance-reports",
  "/finance/unit-economics": "dashboard/finance-reports",
  "/finance/revenue-by-plan": "dashboard/finance-reports",
  "/finance/revenue-by-geography": "dashboard/finance-reports",
  "/finance/gst-returns": "compliance/gst-returns",
  "/dunning/campaigns": "dashboard/dunning",
  "/payments/offline": "dashboard/payments",
};

// First path segment → doc slug.
const SEGMENT_DOCS = {
  "": "dashboard/overview",
  overview: "dashboard/overview",
  ask: "dashboard/ask-ai",
  customers: "dashboard/customers",
  plans: "dashboard/plans",
  subscriptions: "dashboard/subscriptions",
  invoices: "dashboard/invoices",
  quotes: "dashboard/quotes",
  "credit-notes": "dashboard/credit-notes",
  metering: "dashboard/metering-usage",
  usage: "dashboard/metering-usage",
  wallets: "dashboard/wallets",
  coupons: "dashboard/coupons",
  referrals: "dashboard/gifts-referrals",
  gifts: "dashboard/gifts-referrals",
  dunning: "dashboard/dunning",
  "cancel-flows": "dashboard/cancel-flows",
  churn: "dashboard/churn",
  collections: "dashboard/collections",
  mandates: "dashboard/payments",
  payments: "dashboard/payments",
  disputes: "dashboard/disputes",
  ledger: "dashboard/ledger-audit",
  "audit-log": "dashboard/ledger-audit",
  developers: "dashboard/developers",
  integrations: "dashboard/integrations",
  settings: "dashboard/settings",
  profile: "dashboard/settings",
  security: "dashboard/team-security",
  team: "dashboard/team-security",
  organizations: "dashboard/organizations",
  notifications: "dashboard/notifications",
};

/** Doc slug for a route (relative to DOCS_BASE), or null if none is mapped. */
export function docsSlugFor(pathname) {
  if (PATH_DOCS[pathname]) return PATH_DOCS[pathname];
  const segment = pathname.split("/").filter(Boolean)[0] ?? "";
  return SEGMENT_DOCS[segment] ?? null;
}

/** Absolute docs URL for a route. Falls back to the docs home. */
export function docsUrlFor(pathname) {
  const slug = docsSlugFor(pathname);
  return slug ? `${DOCS_BASE}/${slug}` : DOCS_HOME;
}

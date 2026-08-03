import { render, screen, cleanup } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";

import PortalDashboard from "../portal/PortalDashboard";

// The portal talks raw fetch (cookie-authed), not the api client — stub fetch
// per-URL. jsdom lacks localStorage in this config; some transitive imports
// touch it.
const store = {};
vi.stubGlobal("localStorage", {
  getItem: (k) => (k in store ? store[k] : null),
  setItem: (k, v) => {
    store[k] = String(v);
  },
  removeItem: (k) => {
    delete store[k];
  },
  clear: () => {
    for (const k in store) delete store[k];
  },
});

const json = (data) =>
  Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(data) });

// A mixed-currency customer: a paid INR invoice (118000 paise = ₹1,180.00), a
// paid JPY invoice (5000 minor = ¥5,000 — zero-decimal), and an open USD
// invoice with 2500 outstanding ($25.00).
const invoices = [
  { id: "i1", status: "paid", total: 118000, amount_paid: 118000, currency: "INR", invoice_number: "INV-1", created_at: "2026-08-01T00:00:00Z" },
  { id: "i2", status: "paid", total: 5000, amount_paid: 5000, currency: "JPY", invoice_number: "INV-2", created_at: "2026-08-01T00:00:00Z" },
  { id: "i3", status: "open", total: 4000, amount_paid: 1500, currency: "USD", invoice_number: "INV-3", created_at: "2026-08-01T00:00:00Z" },
];

beforeEach(() => {
  global.fetch = vi.fn((url) => {
    const u = String(url);
    if (u.includes("/portal/api/profile")) return json({ name: "Test Cust", email: "t@t.com" });
    if (u.includes("/portal/api/invoices")) return json({ data: invoices });
    if (u.includes("/portal/api/disputes")) return json({ data: [] });
    return json({});
  });
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// Regression lock for the portal money bug: Total Paid / Outstanding used to
// sum minor units ACROSS currencies and format the result as USD — an INR
// customer saw "$118,000.00" and JPY was divided by 100 by the USD exponent.
// The cards must show each currency's own total, each formatted with its own
// symbol and exponent.
describe("PortalDashboard money totals", () => {
  it("shows per-currency totals, never a cross-currency USD sum", async () => {
    render(
      <MemoryRouter>
        <PortalDashboard />
      </MemoryRouter>
    );

    // Paid: ₹1,180.00 (exp 2) and ¥5,000 (exp 0) — both present, joined.
    const paid = await screen.findByText((t) => t.includes("₹1,180.00") && t.includes("¥5,000"));
    expect(paid).toBeTruthy();

    // Outstanding: only the USD invoice's 2500 minor = $25.00.
    expect(await screen.findByText("$25.00")).toBeTruthy();

    // The old bug's outputs must not appear: the cross-currency sum
    // (118000+5000 minor as USD) or the JPY value mangled by /100.
    expect(screen.queryByText(/\$1,230\.00/)).toBeNull();
    expect(screen.queryByText(/\$50\.00/)).toBeNull();
  });
});

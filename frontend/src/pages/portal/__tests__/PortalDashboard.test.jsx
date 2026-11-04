import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import PortalDashboard from "../PortalDashboard";

const invoice = (status, invoice_number) => ({
  id: `id-${status}`,
  invoice_number,
  status,
  total: 9900,
  currency: "USD",
  created_at: "2026-11-04T00:00:00Z",
});

const json = (body) => Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(body) });

beforeEach(() => {
  global.fetch = vi.fn((url) => {
    if (url.includes("/portal/api/profile")) return json({ data: { name: "Buyer", email: "b@example.com" } });
    if (url.includes("/portal/api/invoices")) {
      return json({
        data: [
          invoice("open", "INV-OPEN"),
          invoice("past_due", "INV-LATE"),
          invoice("paid", "INV-PAID"),
          invoice("void", "INV-VOID"),
        ],
      });
    }
    return json({ data: [] });
  });
});

const payLink = async (number) => {
  const row = (await screen.findByText(number)).closest("tr");
  return within(row).queryByRole("link", { name: /pay/i });
};

describe("PortalDashboard invoice payment", () => {
  it("links unpaid invoices to hosted checkout", async () => {
    render(<MemoryRouter><PortalDashboard /></MemoryRouter>);
    expect(await payLink("INV-OPEN")).toHaveAttribute("href", "/checkout/id-open");
    expect(await payLink("INV-LATE")).toHaveAttribute("href", "/checkout/id-past_due");
  });

  it("offers no payment on paid or void invoices", async () => {
    render(<MemoryRouter><PortalDashboard /></MemoryRouter>);
    expect(await payLink("INV-PAID")).toBeNull();
    expect(await payLink("INV-VOID")).toBeNull();
  });
});

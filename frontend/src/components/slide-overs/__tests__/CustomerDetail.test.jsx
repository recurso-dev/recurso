import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CustomerDetail from "../CustomerDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getCustomerConsents: vi.fn(),
    getCustomerChurn: vi.fn(),
    getCreditStatement: vi.fn(),
    getCustomerEntitlements: vi.fn(),
    updateCustomer: vi.fn(),
    revokeConsent: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const customer = {
  id: "cus_1",
  name: "Acme Corp",
  email: "billing@acme.com",
  active: true,
  card_brand: "visa",
  card_last4: "4242",
  card_exp_month: 12,
  card_exp_year: 2028,
  created_at: "2026-01-02T00:00:00Z",
};

const renderDetail = (overrides = {}) =>
  render(
    <CustomerDetail customer={{ ...customer, ...overrides }} isOpen onClose={() => {}} />
  );

describe("CustomerDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCustomerConsents.mockResolvedValue({ data: { data: [] } });
    endpoints.getCreditStatement.mockResolvedValue({ data: { data: null } });
    endpoints.getCustomerEntitlements.mockResolvedValue({ data: { data: [] } });
    endpoints.getCustomerChurn.mockResolvedValue({ data: { data: null } });
    endpoints.updateCustomer.mockResolvedValue({ data: { data: customer } });
  });

  it("renders the customer name and email", async () => {
    renderDetail();
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getAllByText("billing@acme.com").length).toBeGreaterThanOrEqual(1);
  });

  it("shows the payment method on file", () => {
    renderDetail();
    expect(screen.getByText("Payment method")).toBeInTheDocument();
    // brand + last4 + expiry
    expect(screen.getByText(/4242/)).toBeInTheDocument();
    expect(screen.getByText(/exp 12\/28/)).toBeInTheDocument();
  });

  it("omits the payment method when no card is on file", () => {
    renderDetail({ card_last4: undefined, card_brand: undefined });
    expect(screen.queryByText("Payment method")).not.toBeInTheDocument();
  });

  it("renders the churn drill-in when a score is available", async () => {
    endpoints.getCustomerChurn.mockResolvedValue({
      data: {
        data: {
          score: 72,
          risk_level: "high",
          features: {
            failed_invoices_90d: 3,
            payment_failure_rate: 0.25,
            avg_days_to_pay: 12,
            plan_downgrades: 1,
            months_active: 8,
            usage_trend: -0.1,
          },
        },
      },
    });
    renderDetail();
    await waitFor(() => expect(screen.getByText("Failed invoices (90d)")).toBeInTheDocument());
    expect(screen.getByText("72 · high")).toBeInTheDocument();
    expect(screen.getByText("25%")).toBeInTheDocument(); // payment failure rate
  });

  it("enters edit mode and saves via updateCustomer", async () => {
    renderDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));
    fireEvent.click(screen.getByRole("button", { name: /save customer/i }));
    await waitFor(() => expect(endpoints.updateCustomer).toHaveBeenCalledWith("cus_1", expect.any(Object)));
  });
  it("submits the edit form on Enter (real <form> semantics)", async () => {
    renderDetail();
    fireEvent.click(await screen.findByRole("button", { name: /edit customer/i }));
    const form = screen.getByRole("button", { name: /save customer/i }).closest("form");
    expect(form).not.toBeNull();
    fireEvent.submit(form);
    await waitFor(() =>
      expect(endpoints.updateCustomer).toHaveBeenCalledWith(
        "cus_1",
        expect.objectContaining({ email: "billing@acme.com" })
      )
    );
  });
});
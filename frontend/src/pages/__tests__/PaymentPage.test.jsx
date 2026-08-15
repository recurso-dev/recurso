import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import PaymentPage from "../PaymentPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getPayment: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));

const payment = (over = {}) => ({
  id: "pa_123",
  invoice_id: "inv_9",
  invoice_number: "INV-009",
  currency: "USD",
  customer_id: "cus_1",
  subscription_id: "sub_1",
  gateway: "stripe",
  method: "card",
  gateway_payment_intent_id: "pi_abc",
  status: "failed",
  failure_code: "card_declined",
  amount: 5000,
  created_at: "2026-08-14T10:00:00Z",
  updated_at: "2026-08-14T10:05:00Z",
  ...over,
});

function renderPage(id = "pa_123") {
  return render(
    <MemoryRouter initialEntries={[`/payments/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/payments/:id" element={<PaymentPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("PaymentPage", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders identity, amount, status and a humanized failure", async () => {
    endpoints.getPayment.mockResolvedValue({ data: { data: payment() } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Payment · INV-009")).toBeInTheDocument());
    // Human-readable outcome, not the raw code, is the primary explanation.
    expect(screen.getByText(/Failed — Card declined/)).toBeInTheDocument();
    // Raw code kept as technical detail.
    expect(screen.getByText(/gateway code: card_declined/)).toBeInTheDocument();
    expect(screen.getByLabelText("Needs attention")).toBeInTheDocument();
  });

  it("links to the related invoice, customer and subscription", async () => {
    endpoints.getPayment.mockResolvedValue({ data: { data: payment() } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Related")).toBeInTheDocument());
    const hrefs = Array.from(document.querySelectorAll("a")).map((a) => a.getAttribute("href"));
    expect(hrefs).toContain("/invoices/inv_9");
    expect(hrefs).toContain("/customers/cus_1");
    expect(hrefs).toContain("/subscriptions/sub_1");
  });

  it("stays calm (no attention) for a succeeded payment", async () => {
    endpoints.getPayment.mockResolvedValue({
      data: { data: payment({ status: "succeeded", failure_code: "", settled_at: "2026-08-14T10:05:00Z" }) },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText(/Settled successfully/)).toBeInTheDocument());
    expect(screen.queryByLabelText("Needs attention")).not.toBeInTheDocument();
  });

  it("shows a 404 state for a missing payment", async () => {
    endpoints.getPayment.mockRejectedValue({ response: { status: 404 } });
    renderPage();
    await waitFor(() => expect(screen.getByText("Payment not found")).toBeInTheDocument());
  });

  it("shows a retryable error on API failure", async () => {
    endpoints.getPayment.mockRejectedValue({ response: { status: 500 } });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Couldn’t load this payment")).toBeInTheDocument()
    );
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });
});

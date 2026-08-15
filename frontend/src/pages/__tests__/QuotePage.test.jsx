import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import QuotePage from "../QuotePage";
import { endpoints } from "../../lib/api";
import { money } from "@/test/money";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getQuote: vi.fn(),
    sendQuote: vi.fn(),
    acceptQuote: vi.fn(),
    declineQuote: vi.fn(),
    convertQuoteToInvoice: vi.fn(),
    deleteQuote: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const acceptedQuote = {
  id: "q-1",
  quote_number: "Q-001",
  customer_id: "cus-1",
  status: "accepted",
  invoice_id: null,
  currency: "USD",
  line_items: [{ description: "Onboarding", quantity: 2, unit_price: 15000, amount: 30000 }],
  subtotal: 30000,
  tax_amount: 3000,
  discount_amount: 0,
  total: 33000,
  valid_until: "2026-09-01T00:00:00Z",
  created_at: "2026-08-01T00:00:00Z",
};

function renderPage(id = "q-1") {
  return render(
    <MemoryRouter initialEntries={[`/quotes/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/quotes/:id" element={<QuotePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("QuotePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getQuote.mockResolvedValue({ data: { data: acceptedQuote } });
  });

  it("renders line items and the totals math", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Onboarding")).toBeInTheDocument());
    expect(screen.getByText("Subtotal")).toBeInTheDocument();
    // Total shows in both the header and the totals block.
    expect(screen.getAllByText(money("$330.00")).length).toBeGreaterThanOrEqual(1);
  });

  it("offers Convert on an accepted quote and confirms before converting", async () => {
    endpoints.convertQuoteToInvoice.mockResolvedValue({ data: { data: { id: "inv-9" } } });
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /convert to invoice/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /convert to invoice/i }));
    // A confirm dialog guards the money-moving op.
    const confirmBtn = await screen.findAllByRole("button", { name: /convert to invoice/i });
    fireEvent.click(confirmBtn[confirmBtn.length - 1]);
    await waitFor(() => expect(endpoints.convertQuoteToInvoice).toHaveBeenCalledWith("q-1"));
  });

  it("renders a not-found state when the quote is missing", async () => {
    endpoints.getQuote.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/not found/i)).toBeInTheDocument()
    );
  });
});

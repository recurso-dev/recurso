import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import QuoteDetail from "../QuoteDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    sendQuote: vi.fn(),
    acceptQuote: vi.fn(),
    declineQuote: vi.fn(),
    convertQuoteToInvoice: vi.fn(),
    deleteQuote: vi.fn(),
  },
}));

const baseQuote = {
  id: "q-1",
  quote_number: "Q-DEMO-0006",
  currency: "usd",
  subtotal: 75014,
  tax_amount: 13502,
  total: 88516,
  customer_id: "cust-1",
  line_items: [{ description: "Enterprise onboarding", quantity: 1, amount: 75014, unit_price: 75014 }],
};

const renderQuote = (quote, onClose = () => {}) => {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <QuoteDetail quote={quote} isOpen={true} onClose={onClose} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("QuoteDetail actions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.convertQuoteToInvoice.mockResolvedValue({ data: {} });
    endpoints.acceptQuote.mockResolvedValue({ data: {} });
    endpoints.declineQuote.mockResolvedValue({ data: {} });
    endpoints.sendQuote.mockResolvedValue({ data: {} });
  });

  it("shows Convert to invoice on an accepted quote and calls the endpoint", async () => {
    const onClose = vi.fn();
    renderQuote({ ...baseQuote, status: "accepted" }, onClose);

    const convert = screen.getByRole("button", { name: /convert to invoice/i });
    expect(convert).toBeInTheDocument();
    // An accepted quote is not decided again.
    expect(screen.queryByRole("button", { name: /^accept$/i })).not.toBeInTheDocument();

    fireEvent.click(convert);
    await waitFor(() => expect(endpoints.convertQuoteToInvoice).toHaveBeenCalledWith("q-1"));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("shows Accept / Decline / Resend on a sent quote, not Convert", () => {
    renderQuote({ ...baseQuote, status: "sent" });
    expect(screen.getByRole("button", { name: /^accept$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /decline/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /resend/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /convert to invoice/i })).not.toBeInTheDocument();
  });

  it("reveals Convert in place after accepting a sent quote", async () => {
    renderQuote({ ...baseQuote, status: "sent" });
    fireEvent.click(screen.getByRole("button", { name: /^accept$/i }));
    await waitFor(() => expect(endpoints.acceptQuote).toHaveBeenCalledWith("q-1"));
    // Optimistic status override flips the sheet to accepted → Convert appears.
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /convert to invoice/i })).toBeInTheDocument(),
    );
  });

  it("shows Send / Edit / Delete on a draft quote, not Accept or Convert", () => {
    renderQuote({ ...baseQuote, status: "draft" });
    expect(screen.getByRole("button", { name: /^send$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^edit$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^delete$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^accept$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /convert to invoice/i })).not.toBeInTheDocument();
  });
});

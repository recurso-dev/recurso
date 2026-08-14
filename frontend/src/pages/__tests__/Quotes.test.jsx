import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Quotes from "../Quotes";
import { endpoints } from "../../lib/api";
import { money } from "@/test/money";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getQuotes: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    sendQuote: vi.fn(),
    convertQuoteToInvoice: vi.fn(),
  },
}));
const toastMock = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@/components/ui/sonner", () => ({ toast: toastMock }));

const quotes = [
  { id: "q1", quote_number: "Q-001", status: "draft", total: 50000, currency: "USD", customer_id: "cus_1" },
  { id: "q2", quote_number: "Q-002", status: "accepted", total: 90000, currency: "USD", customer_id: "cus_2" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Quotes page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getQuotes.mockResolvedValue({ data: { data: quotes } });
  });

  it("renders quotes with number and status", async () => {
    render(<Quotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("Q-001")).toBeInTheDocument());
    expect(screen.getByText("Q-002")).toBeInTheDocument();
    expect(screen.getByText("Draft")).toBeInTheDocument();
    // Amounts render in the tabular-mono <Money> component.
    expect(screen.getByText(money("$500.00"))).toBeInTheDocument();
    expect(screen.getByText(money("$900.00"))).toBeInTheDocument();
  });

  it("shows the empty state with no quotes", async () => {
    endpoints.getQuotes.mockResolvedValue({ data: { data: [] } });
    render(<Quotes />, { wrapper });
    await waitFor(() => expect(screen.getByText(/no quotes/i)).toBeInTheDocument());
  });

  it("toasts (never silently console-logs) when convert-to-invoice fails", async () => {
    endpoints.convertQuoteToInvoice.mockRejectedValue({
      response: { data: { error: { message: "quote already converted" } } },
    });
    render(<Quotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("Q-002")).toBeInTheDocument());

    // Q-002 is accepted → the "Convert to invoice" action is available.
    fireEvent.click(screen.getByTitle("Convert to invoice"));
    // The one-click money op now confirms first (audit §7 guard).
    const confirmBtn = await screen.findByRole("button", { name: /convert to invoice/i });
    fireEvent.click(confirmBtn);

    await waitFor(() =>
      expect(toastMock.error).toHaveBeenCalledWith("quote already converted")
    );
  });

  it("confirms success when a quote converts to an invoice", async () => {
    endpoints.convertQuoteToInvoice.mockResolvedValue({ data: { data: { id: "inv_1" } } });
    render(<Quotes />, { wrapper });
    await waitFor(() => expect(screen.getByText("Q-002")).toBeInTheDocument());

    fireEvent.click(screen.getByTitle("Convert to invoice"));
    // The one-click money op now confirms first (audit §7 guard).
    const confirmBtn = await screen.findByRole("button", { name: /convert to invoice/i });
    fireEvent.click(confirmBtn);

    await waitFor(() =>
      expect(toastMock.success).toHaveBeenCalledWith("Quote converted to an invoice.")
    );
  });
});

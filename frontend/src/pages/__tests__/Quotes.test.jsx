import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Quotes from "../Quotes";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getQuotes: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("../../components/slide-overs/QuoteDetail", () => ({ default: () => <div /> }));

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
    expect(screen.getByText("draft")).toBeInTheDocument();
  });

  it("shows the empty state with no quotes", async () => {
    endpoints.getQuotes.mockResolvedValue({ data: { data: [] } });
    render(<Quotes />, { wrapper });
    await waitFor(() => expect(screen.getByText(/no quotes/i)).toBeInTheDocument());
  });
});

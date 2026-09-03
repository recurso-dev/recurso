import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RevenueRecognition from "../RevenueRecognition";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getRevenueRecognition: vi.fn() },
}));

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

const now = new Date();

describe("RevenueRecognition page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows recognized / deferred totals and the release schedule", async () => {
    endpoints.getRevenueRecognition.mockResolvedValue({
      data: {
        data: {
          recognized_amount: 125000,
          deferred_balance: 875000,
          by_currency: [{ currency: "USD", deferred: 875000 }],
          upcoming: [
            { month: 10, year: 2026, amount: 125000 },
            { month: 11, year: 2026, amount: 125000 },
          ],
        },
      },
    });
    render(<RevenueRecognition />, { wrapper });
    expect(screen.getByText("Revenue Recognition")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Deferred balance")).toBeInTheDocument());
    // Requested for the current month/year by default.
    expect(endpoints.getRevenueRecognition).toHaveBeenCalledWith(now.getMonth() + 1, now.getFullYear());
    // Minor units rendered with the currency exponent: 875000 → $8,750.00.
    expect(screen.getAllByText("$8,750.00").length).toBeGreaterThan(0);
    expect(screen.getByText("October 2026")).toBeInTheDocument();
    expect(screen.getByText("November 2026")).toBeInTheDocument();
    expect(screen.getByText("Release schedule")).toBeInTheDocument();
  });

  it("refuses to sum across currencies in the headline totals", async () => {
    endpoints.getRevenueRecognition.mockResolvedValue({
      data: {
        data: {
          recognized_amount: 1000,
          deferred_balance: 5000,
          by_currency: [
            { currency: "USD", deferred: 3000 },
            { currency: "INR", deferred: 2000 },
          ],
          upcoming: [{ month: 10, year: 2026, amount: 1000 }],
        },
      },
    });
    render(<RevenueRecognition />, { wrapper });
    await waitFor(() => expect(screen.getAllByText("Multiple currencies").length).toBeGreaterThan(0));
    // Per-currency rows still carry real, exponent-correct numbers.
    expect(screen.getByText("$30.00")).toBeInTheDocument();
    expect(screen.getByText("₹20.00")).toBeInTheDocument();
  });

  it("shows the empty state when nothing is deferred", async () => {
    endpoints.getRevenueRecognition.mockResolvedValue({
      data: { data: { recognized_amount: 0, deferred_balance: 0, by_currency: [], upcoming: [] } },
    });
    render(<RevenueRecognition />, { wrapper });
    await waitFor(() =>
      expect(screen.getByText("No recognition schedules yet")).toBeInTheDocument()
    );
  });

  it("shows an error state and retries the same period", async () => {
    endpoints.getRevenueRecognition.mockRejectedValueOnce({
      response: { data: { error: { message: "report unavailable" } } },
    });
    endpoints.getRevenueRecognition.mockResolvedValue({
      data: { data: { recognized_amount: 0, deferred_balance: 0, by_currency: [], upcoming: [] } },
    });
    render(<RevenueRecognition />, { wrapper });
    expect(await screen.findByText("report unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(endpoints.getRevenueRecognition).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("No recognition schedules yet")).toBeInTheDocument();
  });
});

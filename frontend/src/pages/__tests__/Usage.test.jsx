import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Usage from "../Usage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getUsageStats: vi.fn(),
    getUsageEvents: vi.fn(),
    getBillableMetrics: vi.fn(),
  },
}));
vi.mock("@/lib/useCustomers", () => ({ useCustomers: () => ({ names: {} }) }));

// jsdom has no ResizeObserver; recharts' ResponsiveContainer needs it.
global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

describe("Usage — top-level fetch error state (#7a)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getBillableMetrics.mockResolvedValue({ data: { data: [] } });
  });

  // Before the fix the stats fetch only console.error'd on failure, so the
  // page showed empty StatCards — a user read "no usage" instead of "couldn't
  // load". It must now render an explicit, retryable error state.
  it("shows an error state (not a blank page) when the stats fetch fails", async () => {
    endpoints.getUsageStats.mockRejectedValue({ message: "boom" });
    endpoints.getUsageEvents.mockResolvedValue({ data: { data: [] } });

    render(<Usage />, { wrapper });

    await waitFor(() =>
      expect(screen.getByText(/Unable to load usage metering/i)).toBeInTheDocument()
    );
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("renders metering data on success", async () => {
    endpoints.getUsageStats.mockResolvedValue({
      data: { data: [{ customer_id: "c1", plan_id: "p1", dimension: "api_calls", total_quantity: 100 }], customers_metered: 1 },
    });
    endpoints.getUsageEvents.mockResolvedValue({ data: { data: [] } });

    render(<Usage />, { wrapper });

    await waitFor(() =>
      expect(screen.getByText(/Usage Explorer/i)).toBeInTheDocument()
    );
    expect(screen.queryByText(/Unable to load usage metering/i)).toBeNull();
  });

  it("names the meter and its aggregation for a raw event's dimension", async () => {
    endpoints.getUsageStats.mockResolvedValue({ data: { data: [], customers_metered: 0 } });
    endpoints.getUsageEvents.mockResolvedValue({
      data: {
        data: [
          {
            id: "ev1",
            timestamp: "2026-08-01T00:00:00Z",
            customer_id: "c1",
            dimension: "api_calls",
            quantity: 42,
            transaction_id: "tx-1",
          },
        ],
      },
    });
    endpoints.getBillableMetrics.mockResolvedValue({
      data: { data: [{ id: "m1", code: "api_calls", name: "API calls", aggregation_type: "sum" }] },
    });

    render(<Usage />, { wrapper });

    // The event's dimension resolves to its meter (name) and how it aggregates.
    await waitFor(() => expect(screen.getByText(/API calls · aggregates by/)).toBeInTheDocument());
    expect(screen.getByText("sum")).toBeInTheDocument();
  });
});

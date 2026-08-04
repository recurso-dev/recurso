import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Usage from "../Usage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getUsageStats: vi.fn(),
    getUsageEvents: vi.fn(),
  },
}));
vi.mock("@/lib/useCustomers", () => ({ useCustomers: () => ({ names: {} }) }));

// jsdom has no ResizeObserver; recharts' ResponsiveContainer needs it.
global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };

const wrapper = ({ children }) => <MemoryRouter>{children}</MemoryRouter>;

describe("Usage — top-level fetch error state (#7a)", () => {
  beforeEach(() => vi.clearAllMocks());

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
      expect(screen.getByText(/Usage Metering/i)).toBeInTheDocument()
    );
    expect(screen.queryByText(/Unable to load usage metering/i)).toBeNull();
  });
});

import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ExecutiveSummary from "../ExecutiveSummary";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getMRR: vi.fn(),
    getMRRWaterfall: vi.fn(),
    getUnitEconomics: vi.fn(),
    getInvoiceAging: vi.fn(),
    getRevenueRecognition: vi.fn(),
    getMRRByEntity: vi.fn(),
  },
}));

// Tremor charts need layout jsdom doesn't provide.
vi.mock("@tremor/react", () => {
  const Stub = ({ children }) => <div>{children}</div>;
  return { __esModule: true, AreaChart: Stub, BarChart: Stub };
});

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

const ok = (data) => Promise.resolve({ data: { data } });

describe("ExecutiveSummary page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getMRR.mockImplementation(() =>
      ok({ normalized_mrr: 1234500, arr: 14814000, reporting_currency: "USD" })
    );
    endpoints.getMRRWaterfall.mockImplementation(() =>
      ok({
        starting_mrr: 1000000,
        ending_mrr: 1234500,
        new: 300000,
        expansion: 50000,
        contraction: 15500,
        churned: 100000,
        has_start_history: true,
        net_dollar_retention: 93.5,
        reporting_currency: "USD",
      })
    );
    endpoints.getUnitEconomics.mockImplementation(() =>
      ok({ arpa: 50000, arpu: 25000, has_ltv: false, active_customers: 24, active_subscriptions: 48 })
    );
    endpoints.getInvoiceAging.mockImplementation(() =>
      ok({
        total_outstanding: 200000,
        total_count: 5,
        buckets: [{ label: "current", amount: 120000 }, { label: "30", amount: 80000 }],
      })
    );
    endpoints.getRevenueRecognition.mockImplementation(() =>
      ok({ deferred_balance: 900000, recognized_amount: 75000 })
    );
    endpoints.getMRRByEntity.mockImplementation(() => ok({ entities: [] }));
  });

  it("renders the headline tiles from minor-unit figures", async () => {
    render(<ExecutiveSummary />, { wrapper });
    expect(screen.getByText("Executive Summary")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("MRR")).toBeInTheDocument());
    expect(screen.getByText("$12,345")).toBeInTheDocument(); // MRR headline
    expect(screen.getByText("$148,140")).toBeInTheDocument(); // ARR
    expect(screen.getByText("+$2,345")).toBeInTheDocument(); // net change, signed
    expect(screen.getByText("93.5%")).toBeInTheDocument(); // NDR
    // Overdue = outstanding − current bucket = $800.00 → headline keeps no cents.
    expect(screen.getByText("$800")).toBeInTheDocument();
    expect(screen.getByText("5 open invoices")).toBeInTheDocument();
    // LTV needs history; must say so rather than showing a number.
    expect(screen.getByText("Needs history")).toBeInTheDocument();
    // Single-entity tenant: no per-entity breakdown.
    expect(screen.queryByText("MRR by entity")).toBeNull();
  });

  it("re-keys the movement window when the period changes", async () => {
    render(<ExecutiveSummary />, { wrapper });
    await waitFor(() => expect(screen.getByText("Net change (30d)")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "90d" }));
    await waitFor(() => expect(screen.getByText("Net change (90d)")).toBeInTheDocument());
    // 6 trend windows + the initial 30d + the 90d movement window.
    await waitFor(() => expect(endpoints.getMRRWaterfall.mock.calls.length).toBeGreaterThanOrEqual(8));
  });

  it("shows the per-entity breakdown only for multi-entity tenants", async () => {
    endpoints.getMRRByEntity.mockImplementation(() =>
      ok({
        reporting_currency: "USD",
        entities: [
          { entity_id: "e1", entity_name: "Acme US", normalized_mrr: 1000000, subscriptions: 30, is_primary: true },
          { entity_id: "e2", entity_name: "Acme India", normalized_mrr: 234500, subscriptions: 18 },
        ],
      })
    );
    render(<ExecutiveSummary />, { wrapper });
    await waitFor(() => expect(screen.getByText("MRR by entity")).toBeInTheDocument());
    expect(screen.getByText("Acme India")).toBeInTheDocument();
    expect(screen.getByText("$2,345.00")).toBeInTheDocument();
  });

  it("degrades a single failed tile, but errors when every metric fails", async () => {
    endpoints.getUnitEconomics.mockRejectedValue(new Error("ue down"));
    const { unmount } = render(<ExecutiveSummary />, { wrapper });
    await waitFor(() => expect(screen.getByText("$12,345")).toBeInTheDocument());
    // ARPA tile falls back to a dash instead of taking the page down.
    expect(screen.getByText("ARPA")).toBeInTheDocument();
    unmount();

    for (const fn of [
      endpoints.getMRR,
      endpoints.getMRRWaterfall,
      endpoints.getInvoiceAging,
      endpoints.getRevenueRecognition,
    ]) {
      fn.mockRejectedValue(new Error("down"));
    }
    render(<ExecutiveSummary />, { wrapper });
    expect(await screen.findByText(/Could not load metrics/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });
});

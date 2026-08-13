import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import PlanPage from "../PlanPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getPlan: vi.fn(),
    getPlanEntitlements: vi.fn(),
    getPlanCharges: vi.fn(),
    getBillableMetrics: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn(),
    getAuditLogs: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
// The edit sheet (radix portal) isn't under test here.
vi.mock("../../components/slide-overs/PlanDetail", () => ({
  default: () => <div data-testid="plan-edit-sheet" />,
}));

const plan = {
  id: "plan-123",
  name: "Pro Tier",
  code: "pro-monthly",
  active: true,
  interval_unit: "month",
  interval_count: 1,
  created_at: "2026-01-01T00:00:00Z",
  prices: [{ amount: 9900, currency: "usd", type: "recurring" }],
};

function renderPage(id = "plan-123") {
  return render(
    <MemoryRouter initialEntries={[`/plans/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/plans/:id" element={<PlanPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("PlanPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getPlan.mockResolvedValue({ data: { data: plan } });
    endpoints.getPlanEntitlements.mockResolvedValue({
      data: { data: [{ feature_key: "seats", kind: "limit", limit_value: 10 }] },
    });
    endpoints.getPlanCharges.mockResolvedValue({ data: { data: [] } });
    endpoints.getSubscriptions.mockResolvedValue({
      data: { data: [{ id: "sub-aaaa1111", status: "active" }] },
    });
  });

  it("renders the plan identity, pricing, and entitlements", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "Pro Tier" })).toBeInTheDocument()
    );
    // Fetched by its route id, as a full page (not the list).
    expect(endpoints.getPlan).toHaveBeenCalledWith("plan-123");
    expect(screen.getByText("Pricing")).toBeInTheDocument();
    expect(screen.getByText("seats")).toBeInTheDocument();
    expect(screen.getByText("limit: 10")).toBeInTheDocument();
  });

  it("shows the subscriptions on this plan (reverse lookup) with drill-through", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Subscriptions (1)")).toBeInTheDocument());
    expect(endpoints.getSubscriptions).toHaveBeenCalledWith(
      expect.objectContaining({ plan_id: "plan-123" })
    );
    expect(screen.getByText("sub-aaaa…").closest("a")).toHaveAttribute(
      "href",
      "/subscriptions/sub-aaaa1111"
    );
  });

  it("shows a not-found state on 404", async () => {
    endpoints.getPlan.mockRejectedValue({ response: { status: 404 } });
    renderPage("plan-missing");
    await waitFor(() => expect(screen.getByText("Plan not found")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

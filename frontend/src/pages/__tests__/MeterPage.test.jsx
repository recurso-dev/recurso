import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import MeterPage from "../MeterPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getBillableMetric: vi.fn(),
    getMetricCharges: vi.fn(),
    getUsageEvents: vi.fn(),
    getAuditLogs: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));

const metric = {
  id: "m-1",
  name: "API calls",
  code: "api_calls",
  aggregation_type: "sum",
  field_name: "count",
};

function renderPage(id = "m-1") {
  return render(
    <MemoryRouter initialEntries={[`/billable-metrics/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/billable-metrics/:id" element={<MeterPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("MeterPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getBillableMetric.mockResolvedValue({ data: { data: metric } });
    endpoints.getMetricCharges.mockResolvedValue({
      data: {
        data: [
          { charge_id: "ch-1", plan_id: "plan-9", plan_name: "Pro", plan_code: "pro", plan_active: true, charge_model: "per_unit", pay_in_advance: false },
        ],
      },
    });
    endpoints.getUsageEvents.mockResolvedValue({
      data: { data: [{ id: "ev1", timestamp: "2026-01-01T00:00:00Z", quantity: 42 }] },
    });
  });

  it("shows the definition, aggregation, and the plans pricing on this meter", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("heading", { name: "API calls" })).toBeInTheDocument());
    expect(endpoints.getBillableMetric).toHaveBeenCalledWith("m-1");
    expect(screen.getByText("Definition")).toBeInTheDocument();
    // Reverse lookup: the plan pricing on this meter, drilling to the plan.
    expect(endpoints.getMetricCharges).toHaveBeenCalledWith("m-1");
    expect(screen.getByText("Pro").closest("a")).toHaveAttribute("href", "/plans/plan-9");
    expect(screen.getByText("per_unit")).toBeInTheDocument();
  });

  it("shows the recent events feeding it, filtered to the metric's dimension", async () => {
    renderPage();
    await waitFor(() =>
      expect(endpoints.getUsageEvents).toHaveBeenCalledWith(
        expect.objectContaining({ dimension: "api_calls" })
      )
    );
    expect(screen.getByText(/Recent events/)).toBeInTheDocument();
  });

  it("shows a not-found state on 404", async () => {
    endpoints.getBillableMetric.mockRejectedValue({ response: { status: 404 } });
    renderPage("m-missing");
    await waitFor(() => expect(screen.getByText("Meter not found")).toBeInTheDocument());
  });
});

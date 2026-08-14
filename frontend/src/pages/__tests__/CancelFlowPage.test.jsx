import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CancelFlowPage from "../CancelFlowPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCancelFlow: vi.fn(),
    getCancelFlowStats: vi.fn(),
    updateCancelFlow: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
// The step editor slide-over is reused as the edit sheet; stub it out.
vi.mock("@/components/slide-overs/CancelFlowDetail", () => ({ default: () => <div /> }));

const flow = {
  id: "cf-1",
  name: "Standard retention flow",
  is_active: true,
  is_default: true,
  cooldown_days: 30,
  created_at: "2026-08-01T00:00:00Z",
  steps: [
    { id: "s-1", step_order: 1, step_type: "survey", config: { questions: ["a", "b", "c"] } },
    { id: "s-2", step_order: 2, step_type: "offer", config: { headline: "Stay for 20% off" } },
  ],
};

const stats = {
  total_sessions: 40,
  completed_count: 32,
  saved_count: 12,
  save_rate: 0.375,
  offer_accept_rate: 0.5,
  reason_breakdown: { too_expensive: 18, missing_features: 6 },
};

function renderPage(id = "cf-1") {
  return render(
    <MemoryRouter initialEntries={[`/cancel-flows/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/cancel-flows/:id" element={<CancelFlowPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("CancelFlowPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Both endpoints return the payload directly (not wrapped in { data }).
    endpoints.getCancelFlow.mockResolvedValue({ data: flow });
    endpoints.getCancelFlowStats.mockResolvedValue({ data: stats });
  });

  it("shows the retention effectiveness stats and cancel-reason breakdown", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Effectiveness")).toBeInTheDocument());
    expect(screen.getByText("Save rate")).toBeInTheDocument();
    expect(screen.getByText("38%")).toBeInTheDocument(); // 0.375 → 38%
    // Reason breakdown humanizes and ranks reasons.
    expect(screen.getByText("too expensive")).toBeInTheDocument();
  });

  it("renders the ordered steps with type-appropriate summaries", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("3 cancellation reasons")).toBeInTheDocument());
    expect(screen.getByText("Stay for 20% off")).toBeInTheDocument();
  });

  it("falls back gracefully when no sessions have run yet", async () => {
    endpoints.getCancelFlowStats.mockResolvedValue({ data: { total_sessions: 0 } });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/No cancellation sessions have run/i)).toBeInTheDocument()
    );
  });

  it("renders a not-found state when the flow is missing", async () => {
    endpoints.getCancelFlow.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/Couldn't load this flow/i)).toBeInTheDocument()
    );
  });
});

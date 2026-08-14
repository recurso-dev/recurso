import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import DunningCampaignPage from "../DunningCampaignPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getDunningCampaign: vi.fn(),
    updateDunningCampaign: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
// The editor slide-over is reused as the edit sheet; stub it out here.
vi.mock("@/components/slide-overs/DunningCampaignDetail", () => ({ default: () => <div /> }));

const campaign = {
  id: "dc-1",
  name: "Failed payment recovery",
  is_active: true,
  trigger_event: "payment_failed",
  created_at: "2026-08-01T00:00:00Z",
  steps: [
    { id: "s-1", step_order: 1, channel: "email", delay_hours: 0, subject: "Your payment failed", is_payment_wall: false },
    { id: "s-2", step_order: 2, channel: "sms", delay_hours: 72, subject: "Reminder", is_payment_wall: true },
  ],
};

function renderPage(id = "dc-1") {
  return render(
    <MemoryRouter initialEntries={[`/dunning/campaigns/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/dunning/campaigns/:id" element={<DunningCampaignPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("DunningCampaignPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // This endpoint returns the campaign directly (not wrapped in { data }).
    endpoints.getDunningCampaign.mockResolvedValue({ data: campaign });
  });

  it("renders the ordered step cadence with delays and payment-wall marker", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Step cadence (2)")).toBeInTheDocument());
    expect(screen.getByText("Your payment failed")).toBeInTheDocument();
    // 0h → "immediately after the trigger"; 72h → "after 3 days after the previous step".
    expect(screen.getByText(/immediately after the trigger/i)).toBeInTheDocument();
    expect(screen.getByText(/after 3 days after the previous step/i)).toBeInTheDocument();
    // The payment-wall step is flagged.
    expect(screen.getByText("Payment wall")).toBeInTheDocument();
  });

  it("renders a not-found state when the campaign is missing", async () => {
    endpoints.getDunningCampaign.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/Couldn't load this campaign/i)).toBeInTheDocument()
    );
  });
});

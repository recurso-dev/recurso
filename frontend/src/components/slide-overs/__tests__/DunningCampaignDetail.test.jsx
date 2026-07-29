import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import DunningCampaignDetail from "../DunningCampaignDetail";
import { endpoints as api } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getDunningCampaign: vi.fn(),
    updateDunningCampaign: vi.fn(),
    createDunningStep: vi.fn(),
    deleteDunningStep: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const campaign = {
  id: "camp_1",
  name: "Standard recovery",
  is_active: true,
  steps: [
    { id: "s1", step_order: 1, channel: "email", delay_hours: 0, subject: "Payment failed" },
    { id: "s2", step_order: 2, channel: "sms", delay_hours: 72 },
  ],
};

const renderDetail = () =>
  render(<DunningCampaignDetail campaignId="camp_1" isOpen onClose={() => {}} />);

describe("DunningCampaignDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getDunningCampaign.mockResolvedValue({ data: campaign });
    api.updateDunningCampaign.mockResolvedValue({ data: {} });
  });

  it("loads and renders the campaign with its steps", async () => {
    renderDetail();
    await waitFor(() => expect(screen.getByText("Standard recovery")).toBeInTheDocument());
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Payment failed")).toBeInTheDocument();
  });

  it("deactivates an active campaign", async () => {
    renderDetail();
    await waitFor(() => expect(screen.getByText("Standard recovery")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /deactivate/i }));
    await waitFor(() =>
      expect(api.updateDunningCampaign).toHaveBeenCalledWith("camp_1", { is_active: false })
    );
  });
});

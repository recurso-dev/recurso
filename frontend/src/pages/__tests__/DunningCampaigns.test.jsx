import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import DunningCampaigns from "../DunningCampaigns";
import { endpoints as api } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getDunningCampaigns: vi.fn(),
    createDunningCampaign: vi.fn(),
  },
}));
vi.mock("@/components/slide-overs/DunningCampaignDetail", () => ({ default: () => <div /> }));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("DunningCampaigns page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders campaigns with active state", async () => {
    api.getDunningCampaigns.mockResolvedValue({
      data: { data: [{ id: "c1", name: "Standard recovery", is_active: true }] },
    });
    render(<DunningCampaigns />, { wrapper });
    await waitFor(() => expect(screen.getByText("Standard recovery")).toBeInTheDocument());
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows the empty state with no campaigns", async () => {
    api.getDunningCampaigns.mockResolvedValue({ data: { data: [] } });
    render(<DunningCampaigns />, { wrapper });
    await waitFor(() => expect(screen.getByText("No campaigns yet")).toBeInTheDocument());
  });
});

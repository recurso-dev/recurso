import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Referrals from "../Referrals";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getReferrals: vi.fn(),
    qualifyReferral: vi.fn(),
    createReferral: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const referrals = [
  { id: "r1", code: "REF1", status: "pending", reward_amount: 500, currency: "USD" },
  { id: "r2", code: "REF2", status: "rewarded", reward_amount: 500, currency: "USD" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Referrals page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getReferrals.mockResolvedValue({ data: { data: referrals } });
    endpoints.qualifyReferral.mockResolvedValue({ data: {} });
  });

  it("renders referral codes", async () => {
    render(<Referrals />, { wrapper });
    await waitFor(() => expect(screen.getByText("REF1")).toBeInTheDocument());
    expect(screen.getByText("REF2")).toBeInTheDocument();
  });

  it("qualifies a pending referral", async () => {
    render(<Referrals />, { wrapper });
    await waitFor(() => expect(screen.getByText("REF1")).toBeInTheDocument());
    const qualify = screen.getByRole("button", { name: /^qualify$/i });
    fireEvent.click(qualify);
    await waitFor(() => expect(endpoints.qualifyReferral).toHaveBeenCalledWith("r1"));
  });

  it("shows the empty state with no referrals", async () => {
    endpoints.getReferrals.mockResolvedValue({ data: { data: [] } });
    render(<Referrals />, { wrapper });
    await waitFor(() => expect(screen.getByText("No referrals yet")).toBeInTheDocument());
  });
});

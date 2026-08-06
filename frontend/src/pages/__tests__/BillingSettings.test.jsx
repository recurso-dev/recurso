import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import BillingSettings from "../settings/BillingSettings";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: {
    getBillingStatus: vi.fn(),
    getBillingPlans: vi.fn(),
  },
}));

const PLANS = {
  data: {
    plans: [
      { tier: "self_hosted", name: "Self-Hosted", price: "Free", period: "forever", free_note: "", features: ["MIT"], cta: "GitHub", recommended: false },
      { tier: "cloud", name: "Cloud", price: "0.4% of volume", period: "usage-based", free_note: "Free to $10k/mo", features: ["Managed hosting"], cta: "Start free", recommended: true },
      { tier: "enterprise", name: "Enterprise", price: "Custom", period: "", free_note: "", features: ["SOC 2"], cta: "Talk to us", recommended: false },
    ],
  },
};

describe("BillingSettings", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows the trial status and the plan catalog", async () => {
    endpoints.getBillingStatus.mockResolvedValue({ data: { billing_status: "trialing", trial_days_left: 7, trial_expired: false } });
    endpoints.getBillingPlans.mockResolvedValue(PLANS);

    render(<BillingSettings />);

    // Trial status surfaces...
    expect(await screen.findByText(/7 days left/i)).toBeInTheDocument();
    expect(screen.getByText(/free trial/i)).toBeInTheDocument();
    // ...and the three plans render with the popular one flagged.
    expect(screen.getByText("Cloud")).toBeInTheDocument();
    expect(screen.getByText("Enterprise")).toBeInTheDocument();
    expect(screen.getByText(/popular/i)).toBeInTheDocument();
    expect(screen.getByText(/0.4% of volume/i)).toBeInTheDocument();
  });

  it("renders the catalog even if status fails (self-host / error)", async () => {
    endpoints.getBillingStatus.mockRejectedValue(new Error("no billing"));
    endpoints.getBillingPlans.mockResolvedValue(PLANS);

    render(<BillingSettings />);
    expect(await screen.findByText("Cloud")).toBeInTheDocument();
  });

  it("shows a retryable error when the plan catalog fails to load", async () => {
    endpoints.getBillingStatus.mockResolvedValue({ data: null });
    endpoints.getBillingPlans.mockRejectedValue(new Error("boom"));

    render(<BillingSettings />);
    expect(await screen.findByText(/couldn't load the available plans/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });
});

import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import TrialBanner from "../TrialBanner";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: { getBillingStatus: vi.fn() },
}));

const renderWith = (data) => {
  endpoints.getBillingStatus.mockResolvedValue({ data });
  return render(<TrialBanner />);
};

describe("TrialBanner", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders nothing for a non-trialing (active) tenant", async () => {
    const { container } = renderWith({ billing_status: "active", trial_days_left: 0 });
    // Give the effect a tick; then assert nothing rendered.
    await waitFor(() => expect(endpoints.getBillingStatus).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("shows days left and a choose-a-plan CTA while trialing", async () => {
    renderWith({ billing_status: "trialing", trial_days_left: 9, trial_expired: false });
    expect(await screen.findByText(/9 days left in your free trial/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /choose a plan/i })).toHaveAttribute("href", "https://recurso.dev/pricing");
  });

  it("shows an ended state (no dismiss) when the trial has expired", async () => {
    renderWith({ billing_status: "trialing", trial_days_left: 0, trial_expired: true });
    expect(await screen.findByText(/your free trial has ended/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /choose a plan to continue/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /dismiss/i })).not.toBeInTheDocument();
  });

  it("can be dismissed while still trialing", async () => {
    renderWith({ billing_status: "trialing", trial_days_left: 5, trial_expired: false });
    expect(await screen.findByText(/5 days left/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(screen.queryByText(/5 days left/i)).not.toBeInTheDocument();
  });

  it("renders nothing if the status call fails (self-host / error)", async () => {
    endpoints.getBillingStatus.mockRejectedValue(new Error("boom"));
    const { container } = render(<TrialBanner />);
    await waitFor(() => expect(endpoints.getBillingStatus).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });
});

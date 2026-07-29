import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import SubscriptionDetail from "../SubscriptionDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getSubscriptionUsage: vi.fn().mockResolvedValue({ data: { data: null } }),
    getSubscriptionAddons: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getUsageAmount: vi.fn().mockResolvedValue({ data: { data: null } }),
    getSubscriptionUsageAmount: vi.fn().mockResolvedValue({ data: { data: null } }),
    getSubscriptionCharges: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getCancellationReasons: vi.fn(),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    previewPlanChange: vi.fn().mockResolvedValue({ data: { data: null } }),
    cancelSubscription: vi.fn(),
    pauseSubscription: vi.fn(),
    resumeSubscription: vi.fn(),
    reactivateSubscription: vi.fn(),
    updateSubscription: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const subscription = { id: "sub_abc12345", status: "active", plan_id: "pl_pro", customer_id: "cus_1" };
const plan = { id: "pl_pro", name: "Pro", currency: "USD" };
const customer = { id: "cus_1", name: "Acme" };

const renderDetail = (overrides = {}) =>
  render(
    <SubscriptionDetail
      subscription={{ ...subscription, ...overrides }}
      plan={plan}
      customer={customer}
      isOpen
      onClose={() => {}}
      onRefresh={() => {}}
    />
  );

describe("SubscriptionDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCancellationReasons.mockResolvedValue({
      data: { data: [{ id: "too_expensive", label: "Too expensive", allows_feedback: false }] },
    });
  });

  it("renders the subscription id and active state", () => {
    renderDetail();
    expect(screen.getByText("sub_abc12345")).toBeInTheDocument();
  });

  it("opens the cancel dialog and loads the reason catalog", async () => {
    renderDetail();
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    await waitFor(() => expect(endpoints.getCancellationReasons).toHaveBeenCalled());
    // The dialog is up (its destructive confirm button exists).
    expect(screen.getByRole("button", { name: /cancel subscription/i })).toBeInTheDocument();
  });

  it("keeps the confirm disabled until a reason is chosen (the #290 guard)", async () => {
    renderDetail();
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    const confirm = await screen.findByRole("button", { name: /cancel subscription/i });
    // Disabled with no reason selected, so cancelSubscription never fires.
    expect(confirm).toBeDisabled();
    expect(endpoints.cancelSubscription).not.toHaveBeenCalled();
  });
});

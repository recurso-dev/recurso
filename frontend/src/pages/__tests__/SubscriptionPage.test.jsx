import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import SubscriptionPage from "../SubscriptionPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getSubscription: vi.fn(),
    getCustomer: vi.fn(),
    getPlans: vi.fn(),
    getSubscriptionAddons: vi.fn(),
    getUsageAmount: vi.fn(),
    getSubscriptionUsage: vi.fn(),
    getSubscriptionCharges: vi.fn(),
    getInvoices: vi.fn(),
    getAuditLogs: vi.fn(),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getCancellationReasons: vi.fn(),
    previewPlanChange: vi.fn(),
    cancelSubscription: vi.fn(),
    pauseSubscription: vi.fn(),
    resumeSubscription: vi.fn(),
    reactivateSubscription: vi.fn(),
    updateSubscription: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

const subscription = {
  id: "sub_abc12345",
  status: "active",
  plan_id: "pl_pro",
  customer_id: "cus_1",
  created_at: "2026-01-01T00:00:00Z",
  current_period_start: "2026-08-01T00:00:00Z",
  current_period_end: "2026-09-01T00:00:00Z",
};

function renderPage(id = "sub_abc12345") {
  return render(
    <MemoryRouter initialEntries={[`/subscriptions/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/subscriptions/:id" element={<SubscriptionPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("SubscriptionPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getSubscription.mockResolvedValue({ data: { data: subscription } });
    endpoints.getCustomer.mockResolvedValue({
      data: { data: { id: "cus_1", name: "Acme Inc", email: "billing@acme.com" } },
    });
    endpoints.getPlans.mockResolvedValue({
      data: {
        data: [
          {
            id: "pl_pro",
            name: "Pro Plan",
            interval_unit: "month",
            prices: [{ amount: 4900, currency: "USD" }],
          },
        ],
      },
    });
    endpoints.getSubscriptionAddons.mockResolvedValue({ data: { data: [] } });
    endpoints.getUsageAmount.mockResolvedValue({ data: { data: null } });
    endpoints.getSubscriptionUsage.mockResolvedValue({ data: { data: null } });
    endpoints.getSubscriptionCharges.mockResolvedValue({ data: { data: [] } });
    endpoints.getInvoices.mockResolvedValue({
      data: {
        data: [
          {
            id: "inv_1",
            invoice_number: "INV-001",
            status: "paid",
            total: 4900,
            currency: "USD",
            created_at: "2026-08-01T00:00:00Z",
          },
        ],
        pagination: { page: 1, per_page: 5, total: 8 },
      },
    });
    endpoints.getAuditLogs.mockResolvedValue({ data: { data: [] } });
    endpoints.getCancellationReasons.mockResolvedValue({
      data: {
        data: [{ id: "too_expensive", label: "Too expensive", allows_feedback: false }],
      },
    });
    endpoints.pauseSubscription.mockResolvedValue({ data: { data: {} } });
  });

  it("renders the identity header with customer, plan, and status", async () => {
    renderPage();
    await waitFor(() =>
      expect(
        screen.getByRole("heading", { name: "Acme Inc — Pro Plan" })
      ).toBeInTheDocument()
    );
    expect(screen.getAllByText("Active").length).toBeGreaterThan(0);
    expect(endpoints.getSubscription).toHaveBeenCalledWith("sub_abc12345");
    // Overview links to the customer's and plan's own pages.
    expect(screen.getByRole("link", { name: "Acme Inc" })).toHaveAttribute(
      "href",
      "/customers/cus_1"
    );
    expect(screen.getByRole("link", { name: "Pro Plan" })).toHaveAttribute(
      "href",
      "/plans/pl_pro"
    );
  });

  it("shows subscription-scoped invoices with links and the true total", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("INV-001")).toBeInTheDocument());
    expect(endpoints.getInvoices).toHaveBeenCalledWith(
      expect.objectContaining({ subscription_id: "sub_abc12345" })
    );
    expect(screen.getByText("INV-001").closest("a")).toHaveAttribute("href", "/invoices/inv_1");
    expect(screen.getByText("Invoices (8)")).toBeInTheDocument();
    expect(screen.getByText("View all")).toBeInTheDocument();
  });

  it("pauses via the confirm dialog and refreshes the subscription", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /pause/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /pause/i }));
    const confirm = await screen.findByRole("button", { name: /pause subscription/i });
    fireEvent.click(confirm);
    await waitFor(() =>
      expect(endpoints.pauseSubscription).toHaveBeenCalledWith("sub_abc12345")
    );
    // The page refetches its own object after the action.
    await waitFor(() =>
      expect(endpoints.getSubscription.mock.calls.length).toBeGreaterThan(1)
    );
  });

  it("keeps cancel confirm disabled until a reason is chosen (the #290 guard)", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^cancel$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    const confirm = await screen.findByRole("button", { name: /cancel subscription/i });
    expect(confirm).toBeDisabled();
    expect(endpoints.cancelSubscription).not.toHaveBeenCalled();
    await waitFor(() => expect(endpoints.getCancellationReasons).toHaveBeenCalled());
  });

  it("fetches the subscription's audit trail", async () => {
    renderPage();
    await waitFor(() =>
      expect(endpoints.getAuditLogs).toHaveBeenCalledWith(
        expect.objectContaining({
          entity_type: "subscriptions",
          entity_id: "sub_abc12345",
        })
      )
    );
  });

  it("shows a not-found state on 404", async () => {
    endpoints.getSubscription.mockRejectedValue({ response: { status: 404 } });
    renderPage("sub_missing");
    await waitFor(() =>
      expect(screen.getByText("Subscription not found")).toBeInTheDocument()
    );
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

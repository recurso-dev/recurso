import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import SubscriptionPage from "../SubscriptionPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getSubscription: vi.fn(),
    getSubscriptionHistory: vi.fn().mockResolvedValue({ data: { data: { history: [] } } }),
    getCustomer: vi.fn(),
    getPlans: vi.fn(),
    getSubscriptionAddons: vi.fn(),
    getUsageAmount: vi.fn(),
    getSubscriptionUsage: vi.fn(),
    getSubscriptionCharges: vi.fn(),
    getInvoices: vi.fn(),
    getSubscriptionFinancialSummary: vi.fn(),
    getSubscriptionCancelPreview: vi.fn(),
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
    endpoints.getSubscriptionFinancialSummary.mockResolvedValue({
      data: {
        data: {
          subscription_id: "sub_abc12345",
          status: "active",
          currency: "USD",
          mrr: 4900,
          recurring_amount: 4900,
          interval_unit: "month",
          interval_count: 1,
          current_period_start: "2026-08-01T00:00:00Z",
          current_period_end: "2026-09-01T00:00:00Z",
          next_invoice_date: "2026-09-01T00:00:00Z",
          next_invoice_base_amount: 4900,
          outstanding: [],
        },
      },
    });
    endpoints.getSubscriptionCancelPreview.mockResolvedValue({
      data: {
        data: {
          subscription_id: "sub_abc12345",
          immediately: true,
          effective_date: "2026-08-15T00:00:00Z",
          resulting_status: "canceled",
          cancel_at_period_end: false,
          currency: "USD",
          deferred_revenue_forfeited: 2450,
          recognized_as_breakage: 2450,
          avoided_future_recurring: 4900,
          flat_fee_refund: 0,
        },
      },
    });
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

  it("gates the cancel flow on a reason and never executes from the reason step (the #290 guard)", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^cancel$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    // Step 1's primary action advances to the preview — disabled until a reason
    // is chosen, and it must never call the mutation directly.
    const review = await screen.findByRole("button", { name: /review impact/i });
    expect(review).toBeDisabled();
    expect(endpoints.cancelSubscription).not.toHaveBeenCalled();
    await waitFor(() => expect(endpoints.getCancellationReasons).toHaveBeenCalled());
  });

  it("previews the financial impact before executing an immediate cancel", async () => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^cancel$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));

    // Pick a reason, uncheck "at period end" to preview an IMMEDIATE cancel.
    await user.click(document.getElementById("reason-for-cancellation"));
    await user.click(await screen.findByRole("option", { name: "Too expensive" }));
    fireEvent.click(document.querySelector('input[type="checkbox"]'));

    // Review impact → the preview is fetched and shown; nothing canceled yet.
    await user.click(screen.getByRole("button", { name: /review impact/i }));
    await waitFor(() =>
      expect(endpoints.getSubscriptionCancelPreview).toHaveBeenCalledWith(
        "sub_abc12345",
        expect.objectContaining({ immediately: true })
      )
    );
    // Amount-anchored: the forfeited breakage ($24.50) and avoided recurring
    // ($49.00) from the backend appear (Money splits the symbol into its own
    // span, so match on the element's full text content).
    const money = (amt) => (_, el) =>
      el?.classList?.contains("money") && el.textContent === amt;
    await waitFor(() => expect(screen.getAllByText(money("$24.50")).length).toBeGreaterThan(0));
    expect(screen.getAllByText(money("$49.00")).length).toBeGreaterThan(0);
    expect(endpoints.cancelSubscription).not.toHaveBeenCalled();

    // Confirm executes only from the preview step.
    await user.click(screen.getByRole("button", { name: /confirm cancellation/i }));
    await waitFor(() =>
      expect(endpoints.cancelSubscription).toHaveBeenCalledWith(
        "sub_abc12345",
        expect.objectContaining({ immediately: true, reason: "too_expensive" })
      )
    );
  });

  it("does not imply a trustworthy preview when the preview endpoint fails", async () => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    endpoints.getSubscriptionCancelPreview.mockRejectedValue(new Error("down"));
    const user = userEvent.setup();
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^cancel$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    await user.click(document.getElementById("reason-for-cancellation"));
    await user.click(await screen.findByRole("option", { name: "Too expensive" }));
    await user.click(screen.getByRole("button", { name: /review impact/i }));

    // The UI states the preview is unavailable, and the confirm is relabeled so
    // it can't be mistaken for a previewed, trustworthy action.
    expect(await screen.findByText(/unavailable/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /cancel without preview/i })).toBeInTheDocument();
  });

  it("shows MRR distinct from the recurring billing amount and next invoice", async () => {
    renderPage();
    // MRR label present with the backend's canonical figure ($49.00), kept
    // separate from "Billed each period" and "Next invoice".
    await waitFor(() => expect(screen.getByText("Financial summary")).toBeInTheDocument());
    expect(screen.getByText("MRR")).toBeInTheDocument();
    expect(screen.getByText("Billed each period")).toBeInTheDocument();
    expect(screen.getByText("Next invoice")).toBeInTheDocument();
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

  it("stays calm (no attention banner) for a healthy active subscription", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Overview")).toBeInTheDocument());
    expect(screen.queryByLabelText("Needs attention")).not.toBeInTheDocument();
  });

  it("explains a past_due state with the decline reason and links to the invoice", async () => {
    endpoints.getSubscription.mockResolvedValue({
      data: { data: { ...subscription, status: "past_due" } },
    });
    endpoints.getInvoices.mockResolvedValue({
      data: {
        data: [
          {
            id: "inv_pd",
            invoice_number: "INV-009",
            status: "past_due",
            total: 4900,
            currency: "USD",
            last_payment_error: "card_declined",
            next_retry_at: "2026-09-03T00:00:00Z",
            created_at: "2026-08-20T00:00:00Z",
          },
        ],
        pagination: { page: 1, per_page: 5, total: 1 },
      },
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByLabelText("Needs attention")).toBeInTheDocument()
    );
    // The cause (not just "failed") and the link to the fix.
    expect(screen.getByText(/card_declined/)).toBeInTheDocument();
    expect(screen.getByText(/Renewal payment failed/).closest("a")).toHaveAttribute(
      "href",
      "/invoices/inv_pd"
    );
  });

  it("warns when a subscription is scheduled to cancel at period end", async () => {
    endpoints.getSubscription.mockResolvedValue({
      data: { data: { ...subscription, cancel_at_period_end: true } },
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/Scheduled to cancel at period end/)).toBeInTheDocument()
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

  it("renders the lifecycle timeline (status + plan changes)", async () => {
    endpoints.getSubscriptionHistory.mockResolvedValue({
      data: {
        data: {
          subscription_id: "sub_1",
          history: [
            { id: "l1", change_type: "status", from_value: null, to_value: "trialing", changed_at: "2026-01-01T00:00:00Z" },
            { id: "l2", change_type: "status", from_value: "trialing", to_value: "active", changed_at: "2026-01-08T00:00:00Z" },
            { id: "l3", change_type: "plan", from_value: "plan_a", to_value: "plan_b", changed_at: "2026-02-01T00:00:00Z" },
          ],
        },
      },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("Lifecycle")).toBeInTheDocument());
    // The creation row + a status transition (StatusBadge humanizes trialing).
    expect(screen.getByText("Created as")).toBeInTheDocument();
    expect(screen.getAllByText("Active").length).toBeGreaterThanOrEqual(1);
    // The plan switch resolves each plan id (short id when not in the catalog),
    // unique to the lifecycle row.
    expect(screen.getByText("plan_a")).toBeInTheDocument();
    expect(screen.getByText("plan_b")).toBeInTheDocument();
    // Regression: the connector line joins consecutive events, so N events draw
    // N-1 connectors — the last item has none. The old code compared the index
    // against `lifecycle.length` (an actions object, .length undefined → NaN),
    // so `last` was never true and every item drew a connector.
    expect(screen.getAllByTestId("lifecycle-connector")).toHaveLength(2);
  });
});

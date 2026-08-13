import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CustomerPage from "../CustomerPage";
import { money } from "@/test/money";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCustomer: vi.fn(),
    getSubscriptions: vi.fn(),
    getInvoices: vi.fn(),
    getCreditNotes: vi.fn(),
    getCustomerWallets: vi.fn(),
    getCustomerFinancialSummary: vi.fn(),
    getAuditLogs: vi.fn(),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn(),
  },
}));
// The edit sheet (radix portal) is not under test here.
vi.mock("../../components/slide-overs/CustomerDetail", () => ({
  default: () => <div data-testid="customer-edit-sheet" />,
}));

const customer = {
  id: "cus_1",
  name: "Acme Inc",
  email: "billing@acme.com",
  phone: "+1 555 0100",
  active: true,
  risk_score: 12,
  created_at: "2026-01-05T00:00:00Z",
  ledger_account_id: "led_1",
  billing_address: { line1: "1 Main St", city: "Austin", state: "TX", zip: "78701", country: "US" },
};

function renderPage(id = "cus_1") {
  return render(
    <MemoryRouter initialEntries={[`/customers/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/customers/:id" element={<CustomerPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("CustomerPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCustomer.mockResolvedValue({ data: { data: customer } });
    endpoints.getSubscriptions.mockResolvedValue({
      data: {
        data: [
          {
            id: "sub_1",
            plan_id: "pl_pro",
            status: "active",
            current_period_end: "2026-09-01T00:00:00Z",
          },
        ],
      },
    });
    endpoints.getInvoices.mockResolvedValue({
      data: {
        data: [
          {
            id: "inv_1",
            invoice_number: "INV-001",
            status: "paid",
            total: 5000,
            currency: "USD",
            created_at: "2026-08-01T00:00:00Z",
          },
        ],
        pagination: { page: 1, per_page: 5, total: 12 },
      },
    });
    endpoints.getCreditNotes.mockResolvedValue({ data: { data: [] } });
    endpoints.getCustomerWallets.mockResolvedValue({ data: { data: [] } });
    endpoints.getCustomerFinancialSummary.mockResolvedValue({
      data: {
        data: {
          customer_id: "cus_1",
          currencies: [
            {
              currency: "USD",
              outstanding: 15000,
              past_due: 9000,
              past_due_count: 2,
              billed: 100000,
              paid: 85000,
            },
          ],
        },
      },
    });
    endpoints.getAuditLogs.mockResolvedValue({
      data: {
        data: [
          {
            id: "al_1",
            action: "PUT /v1/customers/:id",
            actor: "user-1",
            created_at: "2026-08-10T09:00:00Z",
          },
        ],
      },
    });
    endpoints.getPlans.mockResolvedValue({
      data: { data: [{ id: "pl_pro", name: "Pro Plan" }] },
    });
  });

  it("renders identity header, attributes, and metadata", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "Acme Inc" })).toBeInTheDocument()
    );
    expect(screen.getAllByText("Active").length).toBeGreaterThan(0);
    expect(screen.getByText("+1 555 0100")).toBeInTheDocument();
    expect(screen.getByText("1 Main St, Austin, TX, 78701, US")).toBeInTheDocument();
    // Fetches the object by its route id.
    expect(endpoints.getCustomer).toHaveBeenCalledWith("cus_1");
  });

  it("links related subscriptions and invoices to their object routes", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Pro Plan")).toBeInTheDocument());
    expect(screen.getByText("Pro Plan").closest("a")).toHaveAttribute(
      "href",
      "/subscriptions/sub_1"
    );
    expect(screen.getByText("INV-001").closest("a")).toHaveAttribute(
      "href",
      "/invoices/inv_1"
    );
    expect(screen.getByText(money("$50.00"))).toBeInTheDocument();
    // Related lists are customer-scoped server-side.
    expect(endpoints.getSubscriptions).toHaveBeenCalledWith(
      expect.objectContaining({ customer_id: "cus_1" })
    );
    expect(endpoints.getInvoices).toHaveBeenCalledWith(
      expect.objectContaining({ customer_id: "cus_1" })
    );
    // 12 total but only 5 fetched → the section offers the full list.
    expect(screen.getByText("Invoices (12)")).toBeInTheDocument();
    expect(screen.getByText("View all")).toBeInTheDocument();
  });

  it("renders the object's audit trail", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("/v1/customers/:id")).toBeInTheDocument()
    );
    expect(endpoints.getAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({ entity_type: "customers", entity_id: "cus_1" })
    );
  });

  it("fetches the object's timeline (events scoped by object_id)", async () => {
    renderPage();
    await waitFor(() =>
      expect(endpoints.getEvents).toHaveBeenCalledWith(
        expect.objectContaining({ object_id: "cus_1" })
      )
    );
  });

  it("leads with the financial summary and surfaces past-due as an exception", async () => {
    renderPage();
    // The page body (which gates on the customer load) renders the section.
    await waitFor(() => expect(screen.getByText("Financial summary")).toBeInTheDocument());
    // Financial summary is fetched per-customer.
    expect(endpoints.getCustomerFinancialSummary).toHaveBeenCalledWith("cus_1");
    // Outstanding value ($150.00 from 15000 minor units).
    expect(screen.getByText(money("$150.00"))).toBeInTheDocument();
    // Exceptions-first: the past-due invoices surface in the attention banner.
    expect(screen.getByLabelText("Needs attention")).toBeInTheDocument();
    expect(screen.getByText(/2 past-due invoices/)).toBeInTheDocument();
  });

  it("stays calm (no attention banner) when nothing is past due", async () => {
    endpoints.getCustomerFinancialSummary.mockResolvedValue({
      data: {
        data: {
          customer_id: "cus_1",
          currencies: [
            { currency: "USD", outstanding: 0, past_due: 0, past_due_count: 0, billed: 100000, paid: 100000 },
          ],
        },
      },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("Financial summary")).toBeInTheDocument());
    expect(screen.queryByLabelText("Needs attention")).not.toBeInTheDocument();
  });

  it("shows a not-found state on 404", async () => {
    endpoints.getCustomer.mockRejectedValue({ response: { status: 404 } });
    renderPage("cus_missing");
    await waitFor(() =>
      expect(screen.getByText("Customer not found")).toBeInTheDocument()
    );
    // A missing object offers no retry — retrying can't make it exist.
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

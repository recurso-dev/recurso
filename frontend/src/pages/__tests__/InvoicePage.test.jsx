import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import InvoicePage from "../InvoicePage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getInvoice: vi.fn(),
    getInvoiceJournalEntries: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEUEInvoice: vi.fn(),
    retryEUEInvoice: vi.fn(),
    retryEInvoice: vi.fn(),
    cancelEInvoice: vi.fn(),
    getInvoicePdf: vi.fn(),
    getInvoicePreview: vi.fn(),
    sendInvoice: vi.fn(),
    getAuditLogs: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getEvents: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));

// jsdom lacks these; the PDF download path uses them.
vi.stubGlobal("URL", { createObjectURL: vi.fn(() => "blob:x"), revokeObjectURL: vi.fn() });
window.open = vi.fn();

const baseInvoice = {
  id: "inv-1",
  invoice_number: "INV-1",
  status: "paid",
  customer_id: "cus_1",
  subscription_id: "sub_1",
  subtotal: 100000,
  total: 108750,
  amount_paid: 108750,
  created_at: "2026-01-01T00:00:00Z",
  due_date: "2026-01-31T00:00:00Z",
};

function renderPage(id = "inv-1") {
  return render(
    <MemoryRouter initialEntries={[`/invoices/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/invoices/:id" element={<InvoicePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("InvoicePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEUEInvoice.mockResolvedValue({ data: { data: null } });
    endpoints.getInvoice.mockResolvedValue({
      data: { data: { ...baseInvoice, currency: "usd" } },
    });
    endpoints.getInvoiceJournalEntries.mockResolvedValue({
      data: {
        data: {
          invoice_id: "inv-1",
          entries: [
            {
              transaction_id: "tx1",
              code: 1,
              description: "Invoice raised",
              debit_account_code: 1100,
              debit_account_name: "Accounts Receivable",
              credit_account_code: 2100,
              credit_account_name: "Deferred Revenue",
              amount: 108750,
              timestamp: "2026-01-01T00:00:00Z",
            },
            {
              transaction_id: "tx2",
              code: 3,
              description: "Payment received",
              debit_account_code: 1000,
              debit_account_name: "Cash",
              credit_account_code: 1100,
              credit_account_name: "Accounts Receivable",
              amount: 108750,
              timestamp: "2026-01-02T00:00:00Z",
            },
          ],
        },
      },
    });
    endpoints.getInvoicePdf.mockResolvedValue({ data: new Blob(["pdf"]) });
    endpoints.sendInvoice.mockResolvedValue({ data: { message: "sent" } });
    endpoints.getInvoicePreview.mockResolvedValue({ data: "<html>invoice</html>" });
  });

  it("renders the header and links to the customer and subscription", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("heading", { name: "INV-1" })).toBeInTheDocument()
    );
    expect(endpoints.getInvoice).toHaveBeenCalledWith("inv-1");
    expect(screen.getByText("Paid")).toBeInTheDocument();
    // Rail links to the customer's and subscription's own pages.
    const links = screen.getAllByRole("link").map((a) => a.getAttribute("href"));
    expect(links).toContain("/customers/cus_1");
    expect(links).toContain("/subscriptions/sub_1");
  });

  // Migrated from the deleted InvoiceDetail sheet suite: the regime
  // presentation rules must not regress in the page port.
  it("hides GST artifacts and labels tax as Sales Tax for a US invoice", async () => {
    endpoints.getInvoice.mockResolvedValue({
      data: {
        data: {
          ...baseInvoice,
          currency: "usd",
          tax_regime: "sales_tax",
          tax_amount: 8750,
          igst_amount: 0,
          cgst_amount: 0,
          sgst_amount: 0,
          line_items: [
            {
              id: "li-1",
              description: "Pro plan",
              quantity: 1,
              amount: 100000,
              hsn_code: "998314",
              tax_rate: 8.75,
            },
          ],
        },
      },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("Line items")).toBeInTheDocument());
    expect(screen.getByText("Sales Tax")).toBeInTheDocument();
    expect(screen.queryByText(/GST/)).not.toBeInTheDocument();
    expect(screen.queryByText(/HSN/)).not.toBeInTheDocument();
  });

  it("shows GST artifacts for an India GST invoice", async () => {
    endpoints.getInvoice.mockResolvedValue({
      data: {
        data: {
          ...baseInvoice,
          currency: "inr",
          tax_regime: "gst",
          tax_amount: 18000,
          igst_amount: 0,
          cgst_amount: 9000,
          sgst_amount: 9000,
          line_items: [
            {
              id: "li-1",
              description: "Pro plan",
              quantity: 1,
              amount: 100000,
              hsn_code: "998314",
              tax_rate: 18,
            },
          ],
        },
      },
    });
    renderPage();
    await waitFor(() => expect(screen.getByText("Line items")).toBeInTheDocument());
    expect(screen.getByText("CGST")).toBeInTheDocument();
    expect(screen.getByText("SGST")).toBeInTheDocument();
    expect(screen.getByText(/HSN 998314/)).toBeInTheDocument();
  });

  it("downloads the PDF for the invoice", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /download pdf/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /download pdf/i }));
    await waitFor(() => expect(endpoints.getInvoicePdf).toHaveBeenCalledWith("inv-1"));
    expect(window.open).toHaveBeenCalled();
  });

  it("sends the invoice to the customer", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^send$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    // Sending emails the customer — a confirm step now guards it (audit §7).
    const confirmBtn = await screen.findByRole("button", { name: /send invoice/i });
    fireEvent.click(confirmBtn);
    await waitFor(() => expect(endpoints.sendInvoice).toHaveBeenCalledWith("inv-1"));
  });

  it("loads the printable preview", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^preview$/i })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: /^preview$/i }));
    await waitFor(() => expect(endpoints.getInvoicePreview).toHaveBeenCalledWith("inv-1"));
  });

  it("fetches the invoice's audit trail", async () => {
    renderPage();
    await waitFor(() =>
      expect(endpoints.getAuditLogs).toHaveBeenCalledWith(
        expect.objectContaining({ entity_type: "invoices", entity_id: "inv-1" })
      )
    );
  });

  it("shows the finance-accounting side: the invoice's journal entries", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Journal entries")).toBeInTheDocument());
    expect(endpoints.getInvoiceJournalEntries).toHaveBeenCalledWith("inv-1");
    // The real transfer postings — Code 1 issuance + Code 3 payment — with accounts.
    // (Text is split across nested code/name spans, so match by substring.)
    expect(screen.getByText(/Invoice raised/)).toBeInTheDocument();
    expect(screen.getByText(/Payment received/)).toBeInTheDocument();
    expect(screen.getAllByText(/Accounts Receivable/).length).toBeGreaterThan(0);
    expect(screen.getByText("Debits = Credits")).toBeInTheDocument();
  });

  it("explains a past_due invoice with its decline reason", async () => {
    endpoints.getInvoice.mockResolvedValue({
      data: {
        data: {
          ...baseInvoice,
          currency: "usd",
          status: "past_due",
          amount_paid: 0,
          amount_due: 108750,
          last_payment_error: "insufficient_funds",
          next_retry_at: "2026-02-03T00:00:00Z",
        },
      },
    });
    renderPage();
    await waitFor(() =>
      expect(screen.getByLabelText("Needs attention")).toBeInTheDocument()
    );
    expect(screen.getByText(/insufficient_funds/)).toBeInTheDocument();
  });

  it("shows a not-found state on 404", async () => {
    endpoints.getInvoice.mockRejectedValue({ response: { status: 404 } });
    renderPage("inv_missing");
    await waitFor(() =>
      expect(screen.getByText("Invoice not found")).toBeInTheDocument()
    );
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });
});

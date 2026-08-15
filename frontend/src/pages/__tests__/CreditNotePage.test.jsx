import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreditNotePage from "../CreditNotePage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCreditNote: vi.fn(),
    getCreditNoteJournalEntries: vi.fn(),
    getCreditNotePdf: vi.fn(),
    approveCreditNote: vi.fn(),
    rejectCreditNote: vi.fn(),
    voidCreditNote: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/auth/AuthProvider", () => ({
  useAuth: () => ({ user: { role: "admin" } }),
}));

const cn = {
  id: "cn-1",
  reference: "CN-001",
  customer_id: "cus-1",
  customer: { name: "Acme" },
  invoice_id: "inv-9",
  type: "adjustment",
  reason: "goodwill",
  status: "issued",
  amount: 5000,
  balance: 4000,
  currency: "USD",
  subtotal: 4237,
  igst_amount: 763,
  hsn_code: "9983",
  created_at: "2026-08-01T10:00:00Z",
};

const journal = [
  {
    transaction_id: "tx-1",
    code: 26,
    debit_account_code: 4000,
    debit_account_name: "Revenue",
    credit_account_code: 2200,
    credit_account_name: "Customer Credit",
    amount: 5000,
    timestamp: "2026-08-01T10:00:00Z",
  },
];

function renderPage(id = "cn-1") {
  return render(
    <MemoryRouter initialEntries={[`/credit-notes/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/credit-notes/:id" element={<CreditNotePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("CreditNotePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCreditNote.mockResolvedValue({ data: { data: cn } });
    endpoints.getCreditNoteJournalEntries.mockResolvedValue({
      data: { data: { credit_note_id: "cn-1", entries: journal } },
    });
  });

  it("shows the amounts, tax reversal, and the ledger postings", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Credit amount")).toBeInTheDocument());
    expect(screen.getAllByText("CN-001").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Balance remaining")).toBeInTheDocument();
    // Statutory tax reversal is broken out.
    expect(screen.getByText("IGST reversed")).toBeInTheDocument();
    // The journal drill renders the posting's accounts.
    expect(screen.getByText("Customer Credit")).toBeInTheDocument();
  });

  it("links to the customer and the offset invoice", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByRole("link", { name: /view invoice/i })).toBeInTheDocument()
    );
    expect(screen.getByRole("link", { name: /view invoice/i })).toHaveAttribute(
      "href",
      "/invoices/inv-9"
    );
  });

  it("renders a not-found state when the note is missing", async () => {
    endpoints.getCreditNote.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/not found/i)).toBeInTheDocument()
    );
  });
});

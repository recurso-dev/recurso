import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
const wrapper = ({ children }) => <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
import { describe, it, expect, vi, beforeEach } from "vitest";
import InvoiceDetail from "../InvoiceDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getEUEInvoice: vi.fn(),
    retryEUEInvoice: vi.fn(),
    retryEInvoice: vi.fn(),
    cancelEInvoice: vi.fn(),
    getInvoicePdf: vi.fn(),
    getInvoicePreview: vi.fn(),
    sendInvoice: vi.fn(),
  },
}));

const baseInvoice = {
  id: "inv-1",
  invoice_number: "INV-1",
  status: "paid",
  subtotal: 100000,
  total: 108750,
  amount_paid: 108750,
  created_at: "2026-01-01T00:00:00Z",
  due_date: "2026-01-31T00:00:00Z",
};

const renderDetail = (invoice) =>
  render(<InvoiceDetail invoice={invoice} isOpen={true} onClose={() => {}} />, { wrapper });

describe("InvoiceDetail tax regime presentation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEUEInvoice.mockResolvedValue({ data: { data: null } });
  });

  it("hides GST artifacts and labels tax as Sales Tax for a US invoice", async () => {
    renderDetail({
      ...baseInvoice,
      currency: "usd",
      tax_regime: "sales_tax",
      tax_amount: 8750,
      igst_amount: 0,
      cgst_amount: 0,
      sgst_amount: 0,
      line_items: [
        { id: "li-1", description: "Pro plan", quantity: 1, amount: 100000, hsn_code: "998314", tax_rate: 8.75 },
      ],
    });

    await waitFor(() => expect(screen.getByText("Line items")).toBeInTheDocument());

    // The single tax line reads "Sales Tax", never "GST".
    expect(screen.getByText("Sales Tax")).toBeInTheDocument();
    expect(screen.queryByText(/GST/)).not.toBeInTheDocument();
    // The India-only HSN prefix must not leak onto a US line item.
    expect(screen.queryByText(/HSN/)).not.toBeInTheDocument();
  });

  it("shows GST artifacts for an India GST invoice", async () => {
    renderDetail({
      ...baseInvoice,
      currency: "inr",
      tax_regime: "gst",
      tax_amount: 18000,
      igst_amount: 0,
      cgst_amount: 9000,
      sgst_amount: 9000,
      line_items: [
        { id: "li-1", description: "Pro plan", quantity: 1, amount: 100000, hsn_code: "998314", tax_rate: 18 },
      ],
    });

    await waitFor(() => expect(screen.getByText("Line items")).toBeInTheDocument());

    // CGST/SGST breakdown rows and the HSN line render for a GST invoice.
    expect(screen.getByText("CGST")).toBeInTheDocument();
    expect(screen.getByText("SGST")).toBeInTheDocument();
    expect(screen.getByText(/HSN 998314/)).toBeInTheDocument();
  });
});

// jsdom lacks these; the PDF download path uses them.
vi.stubGlobal("URL", { createObjectURL: vi.fn(() => "blob:x"), revokeObjectURL: vi.fn() });
window.open = vi.fn();

describe("InvoiceDetail money-path actions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEUEInvoice.mockResolvedValue({ data: { data: null } });
    endpoints.getInvoicePdf.mockResolvedValue({ data: new Blob(["pdf"]) });
    endpoints.sendInvoice.mockResolvedValue({ data: { message: "sent" } });
    endpoints.getInvoicePreview.mockResolvedValue({ data: "<html>invoice</html>" });
  });

  it("downloads the PDF for the invoice", async () => {
    renderDetail({ ...baseInvoice, currency: "usd" });
    fireEvent.click(screen.getByRole("button", { name: /download pdf/i }));
    await waitFor(() => expect(endpoints.getInvoicePdf).toHaveBeenCalledWith("inv-1"));
    expect(window.open).toHaveBeenCalled();
  });

  it("sends the invoice to the customer", async () => {
    renderDetail({ ...baseInvoice, currency: "usd" });
    fireEvent.click(screen.getByRole("button", { name: /send invoice to customer/i }));
    await waitFor(() => expect(endpoints.sendInvoice).toHaveBeenCalledWith("inv-1"));
  });

  it("loads the printable preview", async () => {
    renderDetail({ ...baseInvoice, currency: "usd" });
    fireEvent.click(screen.getByRole("button", { name: /^preview$/i }));
    await waitFor(() => expect(endpoints.getInvoicePreview).toHaveBeenCalledWith("inv-1"));
  });
});

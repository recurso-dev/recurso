import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import DisputePage from "../DisputePage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getDispute: vi.fn(),
    getInvoice: vi.fn(),
    resolveDispute: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const openDispute = {
  id: "d-1",
  invoice_id: "inv-1",
  customer_id: "cus-1",
  reason: "Double charged for the same seat",
  status: "open",
  note: null,
  created_at: "2026-02-01T00:00:00Z",
  resolved_at: null,
};

const invoice = { id: "inv-1", invoice_number: "INV-1001", total: 12000, currency: "USD", status: "open" };

function renderPage(id = "d-1") {
  return render(
    <MemoryRouter initialEntries={[`/disputes/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/disputes/:id" element={<DisputePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("DisputePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getDispute.mockResolvedValue({ data: { data: openDispute } });
    endpoints.getInvoice.mockResolvedValue({ data: { data: invoice } });
    endpoints.resolveDispute.mockResolvedValue({ data: { status: "resolved" } });
  });

  it("shows the reason and links the contested invoice by its number", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Double charged for the same seat")).toBeInTheDocument()
    );
    await waitFor(() =>
      expect(screen.getByRole("link", { name: /INV-1001/i })).toHaveAttribute(
        "href",
        "/invoices/inv-1"
      )
    );
  });

  it("resolves via the Review dialog", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByRole("button", { name: /review/i })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /review/i }));
    await waitFor(() => expect(screen.getByText("Review dispute")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /^accept$/i }));
    await waitFor(() =>
      expect(endpoints.resolveDispute).toHaveBeenCalledWith("d-1", { outcome: "accept", note: "" })
    );
  });

  it("renders a not-found state when the dispute is missing", async () => {
    endpoints.getDispute.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/Couldn't load this dispute/i)).toBeInTheDocument()
    );
  });
});

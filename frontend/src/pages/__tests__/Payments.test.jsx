import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Payments from "../Payments";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getPaymentAttempts: vi.fn(),
  },
}));

const page = {
  data: [
    {
      id: "pa-1",
      invoice_id: "inv-1",
      invoice_number: "INV-1001",
      currency: "USD",
      gateway: "stripe",
      method: "card",
      status: "failed",
      failure_code: "card_declined",
      amount: 4200,
      created_at: "2026-08-14T10:00:00Z",
    },
    {
      id: "pa-2",
      invoice_id: "inv-2",
      invoice_number: "INV-1002",
      currency: "USD",
      gateway: "stripe",
      method: "ach",
      status: "succeeded",
      failure_code: "",
      amount: 9900,
      created_at: "2026-08-14T09:00:00Z",
    },
  ],
  pagination: { page: 1, per_page: 50, total: 2, total_pages: 1 },
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/payments"]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Payments />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("Payments log", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getPaymentAttempts.mockResolvedValue({ data: page });
  });

  it("lists attempts with their invoice, status, and failure reason", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("INV-1001")).toBeInTheDocument());
    // The failed attempt surfaces its declining reason.
    expect(screen.getByText("card_declined")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("succeeded")).toBeInTheDocument();
    // Invoice numbers link to the addressable invoice page.
    expect(screen.getByRole("link", { name: "INV-1001" })).toHaveAttribute(
      "href",
      "/invoices/inv-1"
    );
  });

  it("requests the tenant-wide log with pagination", async () => {
    renderPage();
    await waitFor(() => expect(endpoints.getPaymentAttempts).toHaveBeenCalled());
    expect(endpoints.getPaymentAttempts).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, per_page: 50 })
    );
  });
});

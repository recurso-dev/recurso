import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import OfflinePayments from "../OfflinePayments";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getOfflinePayments: vi.fn(),
    getVirtualAccounts: vi.fn().mockResolvedValue({ data: { data: [] } }),
    recordOfflinePayment: vi.fn(),
    createVirtualAccount: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getInvoices: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("OfflinePayments page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders a recorded offline payment amount", async () => {
    endpoints.getOfflinePayments.mockResolvedValue({
      data: { data: [{ id: "op1", customer_id: "cus_1", amount: 250000, currency: "USD", method: "neft" }] },
    });
    render(<OfflinePayments />, { wrapper });
    // $2,500.00
    await waitFor(() => expect(screen.getByText(/2,500/)).toBeInTheDocument());
  });

  it("shows the empty state with no offline payments", async () => {
    endpoints.getOfflinePayments.mockResolvedValue({ data: { data: [] } });
    render(<OfflinePayments />, { wrapper });
    await waitFor(() => expect(screen.getByText("No offline payments recorded")).toBeInTheDocument());
  });
});

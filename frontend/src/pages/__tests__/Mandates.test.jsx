import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Mandates from "../Mandates";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getMandates: vi.fn(),
    revokeMandate: vi.fn(),
    createMandate: vi.fn(),
    // used by the shared useCustomers/usePlans/useSubscriptions hooks
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const mandates = [
  {
    id: "mnd_1",
    customer_id: "cus_1",
    vpa: "user@upi",
    max_amount: 500000,
    currency: "INR",
    frequency: "monthly",
    status: "active",
  },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Mandates page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getMandates.mockResolvedValue({ data: { data: mandates } });
    endpoints.revokeMandate.mockResolvedValue({ data: {} });
  });

  it("renders a mandate with its max/cycle amount and method", async () => {
    render(<Mandates />, { wrapper });
    await waitFor(() => expect(screen.getByText("user@upi")).toBeInTheDocument());
    // 500000 paise = ₹5,000.00
    expect(screen.getByText(/5,000/)).toBeInTheDocument();
  });

  it("revokes only after confirmation", async () => {
    render(<Mandates />, { wrapper });
    await waitFor(() => expect(screen.getByText("user@upi")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /^revoke$/i }));
    // Confirm dialog is up; nothing sent yet.
    expect(endpoints.revokeMandate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /revoke mandate/i }));
    await waitFor(() => expect(endpoints.revokeMandate).toHaveBeenCalledWith("mnd_1"));
  });

  it("shows the empty state with no mandates", async () => {
    endpoints.getMandates.mockResolvedValue({ data: { data: [] } });
    render(<Mandates />, { wrapper });
    await waitFor(() => expect(screen.getByText(/no mandates yet/i)).toBeInTheDocument());
  });
});

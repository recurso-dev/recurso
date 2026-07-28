import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Coupons from "../Coupons";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCoupons: vi.fn(),
    setCouponActive: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("../../components/slide-overs/CouponDetail", () => ({ default: () => <div /> }));

const coupons = [
  { id: "c1", code: "SAVE20", discount_type: "percent", discount_value: 20, active: true, redemptions: 3 },
  { id: "c2", code: "OLD10", discount_type: "amount", discount_value: 1000, currency: "USD", active: false, redemptions: 0 },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Coupons page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCoupons.mockResolvedValue({ data: { data: coupons } });
    endpoints.setCouponActive.mockResolvedValue({ data: {} });
  });

  it("renders active and inactive coupons", async () => {
    render(<Coupons />, { wrapper });
    await waitFor(() => expect(screen.getByText("SAVE20")).toBeInTheDocument());
    expect(screen.getByText("OLD10")).toBeInTheDocument();
    expect(screen.getByText("20%")).toBeInTheDocument();
  });

  it("filters by status", async () => {
    render(<Coupons />, { wrapper });
    await waitFor(() => expect(screen.getByText("SAVE20")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: /^inactive$/i }));
    expect(screen.queryByText("SAVE20")).not.toBeInTheDocument();
    expect(screen.getByText("OLD10")).toBeInTheDocument();
  });

  it("reactivates an inactive coupon directly", async () => {
    render(<Coupons />, { wrapper });
    await waitFor(() => expect(screen.getByText("OLD10")).toBeInTheDocument());
    // Filter to inactive so only the reactivatable coupon's row is shown.
    await userEvent.click(screen.getByRole("button", { name: /^inactive$/i }));
    const reactivate = await screen.findByRole("button", { name: /^reactivate$/i });
    await userEvent.click(reactivate);
    await waitFor(() =>
      expect(endpoints.setCouponActive).toHaveBeenCalledWith("c2", true)
    );
  });

  it("confirms before deactivating an active coupon", async () => {
    render(<Coupons />, { wrapper });
    await waitFor(() => expect(screen.getByText("SAVE20")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /^deactivate$/i }));
    // A confirm dialog appears; setCouponActive is not called until confirmed.
    expect(endpoints.setCouponActive).not.toHaveBeenCalled();
  });

  it("shows the empty state when there are no coupons", async () => {
    endpoints.getCoupons.mockResolvedValue({ data: { data: [] } });
    render(<Coupons />, { wrapper });
    await waitFor(() =>
      expect(screen.getByText(/no coupons/i)).toBeInTheDocument()
    );
  });
});

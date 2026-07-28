import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CouponDetail from "../CouponDetail";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: { setCouponActive: vi.fn() },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const base = {
  id: "cpn_1",
  code: "SAVE20",
  discount_type: "percent",
  discount_value: 20,
  duration: "once",
  active: true,
  created_at: "2026-01-02T00:00:00Z",
};

const renderCoupon = (coupon) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <CouponDetail coupon={coupon} isOpen onClose={() => {}} />
    </QueryClientProvider>
  );

describe("CouponDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.setCouponActive.mockResolvedValue({ data: {} });
  });

  it("shows the code and an active status badge", () => {
    renderCoupon(base);
    expect(screen.getByText("SAVE20")).toBeInTheDocument();
    expect(screen.getByText("active")).toBeInTheDocument();
  });

  it("deactivates an active coupon", async () => {
    renderCoupon(base);
    fireEvent.click(screen.getByRole("button", { name: /deactivate coupon/i }));
    await waitFor(() =>
      expect(endpoints.setCouponActive).toHaveBeenCalledWith("cpn_1", false)
    );
  });

  it("reactivates an inactive coupon", async () => {
    renderCoupon({ ...base, active: false });
    expect(screen.getByText("inactive")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /reactivate coupon/i }));
    await waitFor(() =>
      expect(endpoints.setCouponActive).toHaveBeenCalledWith("cpn_1", true)
    );
  });

  it("renders a redemption progress bar when capped", () => {
    renderCoupon({ ...base, max_redemptions: 100, redemptions: 25 });
    expect(screen.getByText(/25 of 100 used/i)).toBeInTheDocument();
  });
});

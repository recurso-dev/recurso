import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CouponPage from "../CouponPage";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCoupon: vi.fn(),
    setCouponActive: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getSubscriptions: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const coupon = {
  id: "c-1",
  code: "SAVE20",
  discount_type: "percent",
  discount_value: 20,
  duration: "repeating",
  duration_months: 3,
  active: true,
  created_at: "2026-08-01T00:00:00Z",
};

// One subscription redeems this coupon; another doesn't.
const subs = [
  { id: "s-1", customer_id: "cus-1", coupon_id: "c-1", status: "active" },
  { id: "s-2", customer_id: "cus-2", coupon_id: null, status: "active" },
];

function renderPage(id = "c-1") {
  return render(
    <MemoryRouter initialEntries={[`/coupons/${id}`]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
      >
        <Routes>
          <Route path="/coupons/:id" element={<CouponPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("CouponPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCoupon.mockResolvedValue({ data: { data: coupon } });
    endpoints.getSubscriptions.mockResolvedValue({ data: { data: subs } });
  });

  it("summarizes what the coupon does over time", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("What it does")).toBeInTheDocument());
    // Repeating coupon → applied for the first N months.
    expect(screen.getByText(/applied for the first 3 months/i)).toBeInTheDocument();
  });

  it("lists the subscriptions actually redeeming it (real reverse lookup)", async () => {
    renderPage();
    // Only the subscription whose coupon_id matches is linked.
    await waitFor(() =>
      expect(screen.getByRole("link", { name: /View/i })).toHaveAttribute(
        "href",
        "/subscriptions/s-1"
      )
    );
    expect(screen.getByText(/Redeemed by \(1\)/)).toBeInTheDocument();
  });

  it("renders a not-found state when the coupon is missing", async () => {
    endpoints.getCoupon.mockRejectedValue({ response: { status: 404 } });
    renderPage("missing");
    await waitFor(() =>
      expect(screen.getByText(/not found/i)).toBeInTheDocument()
    );
  });
});

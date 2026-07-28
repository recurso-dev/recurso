import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Gifts from "../Gifts";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getGifts: vi.fn(),
    cancelGift: vi.fn(),
    purchaseGift: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const gifts = [
  { id: "g1", code: "GIFT100", status: "purchased", amount: 10000, currency: "USD" },
  { id: "g2", code: "GIFT200", status: "redeemed", amount: 20000, currency: "USD" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Gifts page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getGifts.mockResolvedValue({ data: { data: gifts } });
    endpoints.cancelGift.mockResolvedValue({ data: {} });
  });

  it("renders gift codes", async () => {
    render(<Gifts />, { wrapper });
    await waitFor(() => expect(screen.getByText("GIFT100")).toBeInTheDocument());
    expect(screen.getByText("GIFT200")).toBeInTheDocument();
  });

  it("offers Cancel only for a purchased (unredeemed) gift", async () => {
    render(<Gifts />, { wrapper });
    await waitFor(() => expect(screen.getByText("GIFT100")).toBeInTheDocument());
    // Exactly one Cancel action — the purchased gift, not the redeemed one.
    expect(screen.getAllByRole("button", { name: /^cancel$/i })).toHaveLength(1);
  });

  it("cancels a gift only after confirmation", async () => {
    render(<Gifts />, { wrapper });
    await waitFor(() => expect(screen.getByText("GIFT100")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));
    expect(endpoints.cancelGift).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /cancel gift/i }));
    await waitFor(() => expect(endpoints.cancelGift).toHaveBeenCalledWith("g1"));
  });

  it("shows the empty state with no gifts", async () => {
    endpoints.getGifts.mockResolvedValue({ data: { data: [] } });
    render(<Gifts />, { wrapper });
    await waitFor(() => expect(screen.getByText("No gifts yet")).toBeInTheDocument());
  });
});

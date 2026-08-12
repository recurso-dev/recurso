import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import CreateCoupon from "../CreateCoupon";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    createCoupon: vi.fn(),
    getPlans: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <MemoryRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </MemoryRouter>
);

const plansOf = (...currencies) => ({
  data: { data: currencies.map((c, i) => ({ id: `p${i}`, name: `Plan ${i}`, currency: c })) },
});

// Switch the discount type to "Amount off" through the Radix select.
const pickAmountOff = async (user) => {
  if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
  if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
  await user.click(screen.getByRole("combobox", { name: /discount type/i }));
  await user.click(await screen.findByRole("option", { name: /amount off/i }));
};

describe("CreateCoupon — currency-aware amount-off", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.createCoupon.mockResolvedValue({ data: { data: { id: "c1" } } });
  });

  it("converts amount-off with the catalog's dominant currency exponent (JPY: no ×100)", async () => {
    endpoints.getPlans.mockResolvedValue(plansOf("JPY", "JPY", "USD"));
    const user = userEvent.setup();
    render(<CreateCoupon />, { wrapper });
    await pickAmountOff(user);

    // Dominant plan currency (JPY) is the default; its symbol is the prefix.
    await waitFor(() => expect(screen.getByText("¥")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText(/coupon code/i), { target: { value: "YEN500" } });
    fireEvent.change(screen.getByLabelText(/discount value/i), { target: { value: "500" } });
    fireEvent.click(screen.getByRole("button", { name: /create coupon/i }));

    // JPY has no minor unit: ¥500 off must send 500, not 50000.
    await waitFor(() =>
      expect(endpoints.createCoupon).toHaveBeenCalledWith(
        expect.objectContaining({ discount_type: "amount", discount_value: 500 })
      )
    );
  });

  it("converts a USD amount to cents", async () => {
    endpoints.getPlans.mockResolvedValue(plansOf("USD"));
    const user = userEvent.setup();
    render(<CreateCoupon />, { wrapper });
    await pickAmountOff(user);
    await waitFor(() => expect(screen.getByText("$")).toBeInTheDocument());

    fireEvent.change(screen.getByLabelText(/coupon code/i), { target: { value: "USD25" } });
    fireEvent.change(screen.getByLabelText(/discount value/i), { target: { value: "25" } });
    fireEvent.click(screen.getByRole("button", { name: /create coupon/i }));

    await waitFor(() =>
      expect(endpoints.createCoupon).toHaveBeenCalledWith(
        expect.objectContaining({ discount_value: 2500 })
      )
    );
  });

  it("keeps percent as a plain integer with no currency field", async () => {
    endpoints.getPlans.mockResolvedValue(plansOf("USD"));
    render(<CreateCoupon />, { wrapper });

    expect(screen.queryByLabelText(/^currency$/i)).toBeNull();
    fireEvent.change(screen.getByLabelText(/coupon code/i), { target: { value: "TEN" } });
    fireEvent.change(screen.getByLabelText(/discount value/i), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: /create coupon/i }));

    await waitFor(() =>
      expect(endpoints.createCoupon).toHaveBeenCalledWith(
        expect.objectContaining({ discount_type: "percent", discount_value: 10 })
      )
    );
  });
});

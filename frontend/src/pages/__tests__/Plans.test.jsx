import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Plans from "../Plans";
import { endpoints } from "../../lib/api";
import { money } from "@/test/money";

vi.mock("../../lib/api", () => ({
  endpoints: { getPlans: vi.fn() },
}));
vi.mock("../../components/BuyGiftModal", () => ({ default: () => <div /> }));
vi.mock("../../components/slide-overs/PlanDetail", () => ({ default: () => <div /> }));

const plans = [
  {
    id: "pl_pro",
    name: "Pro",
    interval_unit: "month",
    prices: [{ amount: 4900, currency: "USD" }],
  },
  {
    id: "pl_ent",
    name: "Enterprise",
    interval_unit: "year",
    prices: [{ amount: 990000, currency: "USD" }],
  },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Plans page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getPlans.mockResolvedValue({ data: { data: plans } });
  });

  it("renders plans with formatted first-price money", async () => {
    render(<Plans />, { wrapper });
    await waitFor(() => expect(screen.getByText("Pro")).toBeInTheDocument());
    expect(screen.getByText("Enterprise")).toBeInTheDocument();
    expect(screen.getByText(money("$49.00"))).toBeInTheDocument();
    expect(screen.getByText(money("$9,900.00"))).toBeInTheDocument();
  });

  it("shows the empty state when there are no plans", async () => {
    endpoints.getPlans.mockResolvedValue({ data: { data: [] } });
    render(<Plans />, { wrapper });
    await waitFor(() => expect(screen.getByText("No plans yet")).toBeInTheDocument());
  });

  // The currency filter must reach the server — filtering only the fetched
  // page silently hid matching plans on other pages.
  it("sends the currency filter to the server", async () => {
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    const { default: userEvent } = await import("@testing-library/user-event");
    const user = userEvent.setup();

    render(<Plans />, { wrapper });
    await waitFor(() => expect(screen.getByText("Pro")).toBeInTheDocument());

    // Toolbar selects in order: currency, interval.
    await user.click(screen.getAllByRole("combobox")[0]);
    await user.click(await screen.findByRole("option", { name: "Currency: INR" }));

    await waitFor(() =>
      expect(endpoints.getPlans).toHaveBeenCalledWith(
        expect.objectContaining({ currency: "INR", page: 1 })
      )
    );
  });
});

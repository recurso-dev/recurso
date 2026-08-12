import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Subscriptions from "../Subscriptions";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getSubscriptions: vi.fn(),
    getCustomers: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getPlans: vi.fn().mockResolvedValue({ data: { data: [] } }),
  },
}));
vi.mock("../../components/slide-overs/SubscriptionDetail", () => ({
  default: ({ subscription }) =>
    subscription ? <div data-testid="sub-detail">{subscription.id}</div> : null,
}));

const subs = [
  { id: "sub_1", customer_id: "cus_1", plan_id: "pl_pro", status: "active" },
  { id: "sub_2", customer_id: "cus_2", plan_id: "pl_free", status: "canceled" },
];

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Subscriptions page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getSubscriptions.mockResolvedValue({ data: { data: subs } });
  });

  it("renders subscription rows with status badges", async () => {
    render(<Subscriptions />, { wrapper });
    await waitFor(() => expect(screen.getByText("Active")).toBeInTheDocument());
    expect(screen.getByText("Canceled")).toBeInTheDocument();
  });

  it("opens the detail sheet on row click", async () => {
    render(<Subscriptions />, { wrapper });
    await waitFor(() => expect(screen.getByText("Active")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Active"));
    await waitFor(() => expect(screen.getByTestId("sub-detail")).toHaveTextContent("sub_1"));
  });

  it("shows the empty state with no subscriptions", async () => {
    endpoints.getSubscriptions.mockResolvedValue({ data: { data: [] } });
    render(<Subscriptions />, { wrapper });
    await waitFor(() => expect(screen.getByText("No subscriptions yet")).toBeInTheDocument());
  });

  // The plan filter must reach the server — filtering only the fetched page
  // silently hid matching subscriptions on other pages.
  it("sends the plan filter to the server", async () => {
    endpoints.getCustomers.mockResolvedValue({ data: { data: [] } });
    endpoints.getPlans.mockResolvedValue({
      data: { data: [{ id: "pl_pro", name: "Pro" }] },
    });
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    const user = userEvent.setup();

    render(<Subscriptions />, { wrapper });
    await waitFor(() => expect(screen.getByText("Active")).toBeInTheDocument());

    // Toolbar selects in order: status, plan, date.
    await user.click(screen.getAllByRole("combobox")[1]);
    await user.click(await screen.findByRole("option", { name: "Pro" }));

    await waitFor(() =>
      expect(endpoints.getSubscriptions).toHaveBeenCalledWith(
        expect.objectContaining({ plan_id: "pl_pro", page: 1 })
      )
    );
  });
});

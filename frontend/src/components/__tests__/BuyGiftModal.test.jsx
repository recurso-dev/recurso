import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import BuyGiftModal from "../BuyGiftModal";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getCustomers: vi.fn(),
    purchaseGift: vi.fn(),
  },
}));

// jsdom lacks these; Radix Select touches them.
beforeEach(() => {
  if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
  if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
});

const CUSTOMER = { id: "c0000000-0000-0000-0000-000000000001", name: "Acme Corp", email: "ops@acme.com" };
const PLANS = [{ id: "p0000000-0000-0000-0000-000000000001", name: "Pro" }];

const renderModal = (props = {}) =>
  render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      <BuyGiftModal isOpen onClose={() => {}} plans={PLANS} {...props} />
    </QueryClientProvider>
  );

describe("BuyGiftModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getCustomers.mockResolvedValue({ data: { data: [CUSTOMER] } });
    endpoints.purchaseGift.mockResolvedValue({ data: { code: "GIFT-ABCD" } });
  });

  it("offers a customer picker instead of a raw UUID input", async () => {
    renderModal();
    expect(screen.getByText("Create a gift subscription")).toBeInTheDocument();
    // No free-text ID field — the buyer is chosen from the shared customer list.
    expect(screen.queryByPlaceholderText(/uuid/i)).toBeNull();
    expect(document.getElementById("gift-buyer")).toBeTruthy();
    // Submit stays disabled until a buyer is picked.
    expect(screen.getByRole("button", { name: /create gift/i })).toBeDisabled();
    await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalledWith({ limit: 1000 }));
  });

  it("submits the picked customer's id and shows the gift code", async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    renderModal({ onSuccess });
    await waitFor(() => expect(endpoints.getCustomers).toHaveBeenCalled());

    await user.click(document.getElementById("gift-buyer"));
    await user.click(await screen.findByRole("option", { name: /Acme Corp/ }));
    await user.click(screen.getByRole("button", { name: /create gift/i }));

    await waitFor(() =>
      expect(endpoints.purchaseGift).toHaveBeenCalledWith({
        buyer_customer_id: CUSTOMER.id,
        plan_id: PLANS[0].id,
        duration_months: 12,
      })
    );
    await waitFor(() => expect(screen.getByText("GIFT-ABCD")).toBeInTheDocument());
    expect(onSuccess).toHaveBeenCalled();
  });
});

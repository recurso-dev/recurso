import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ImportData from "../ImportData";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: {
    stripeImportPreview: vi.fn(),
    stripeImportCommit: vi.fn(),
    chargebeeImportPreview: vi.fn(),
    chargebeeImportCommit: vi.fn(),
    revenuecatImportPreview: vi.fn(),
    revenuecatImportCommit: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const renderPage = () =>
  render(
    <MemoryRouter>
      <ImportData />
    </MemoryRouter>
  );

const stripeExport = JSON.stringify({
  customers: [{ id: "cus_1", email: "a@b.com" }],
  products: [{ id: "prod_1", name: "Pro" }],
  prices: [{ id: "price_1", product: "prod_1", unit_amount: 4900, currency: "usd", recurring: { interval: "month" } }],
});

const chargebeeExport = JSON.stringify({
  customers: [{ id: "cb_1", email: "a@b.com" }],
  plans: [{ id: "pro", name: "Pro", price: 4900, period: 1, period_unit: "month", currency_code: "usd", status: "active" }],
});

const stripePreview = {
  data: {
    items: [
      { kind: "customer", stripe_id: "cus_1", label: "a@b.com", action: "create" },
      { kind: "plan", stripe_id: "price_1", label: "Pro", action: "create", detail: "49.00 USD" },
    ],
    summary: { "customer.create": 1, "plan.create": 1 },
    warnings: [],
  },
};

const chargebeePreview = {
  data: {
    items: [{ kind: "plan", chargebee_id: "pro", label: "Pro", action: "create", detail: "49.00 USD" }],
    summary: { "plan.create": 1 },
    warnings: [],
  },
};

describe("ImportData wizard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows a source picker first (Stripe, Chargebee, RevenueCat)", () => {
    renderPage();
    expect(screen.getByText(/where are you migrating from/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /stripe/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /chargebee/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /revenuecat/i })).toBeInTheDocument();
  });

  it("routes RevenueCat to the RevenueCat endpoints (not Stripe/Chargebee)", async () => {
    endpoints.revenuecatImportPreview.mockResolvedValue({
      data: { items: [{ kind: "plan", revenuecat_id: "monthly", label: "Pro", action: "create" }], summary: { "plan.create": 1 }, warnings: [] },
    });
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /revenuecat/i }));
    fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: '{"products":[{"id":"monthly"}]}' } });
    fireEvent.click(screen.getByRole("button", { name: /preview import/i }));

    expect(await screen.findByText(/nothing has been imported yet/i)).toBeInTheDocument();
    expect(endpoints.revenuecatImportPreview).toHaveBeenCalledTimes(1);
    expect(endpoints.stripeImportPreview).not.toHaveBeenCalled();
    expect(endpoints.chargebeeImportPreview).not.toHaveBeenCalled();
  });

  it("routes Stripe through preview → commit", async () => {
    endpoints.stripeImportPreview.mockResolvedValue(stripePreview);
    endpoints.stripeImportCommit.mockResolvedValue({ data: { created: { customer: 1, plan: 1 }, failures: [] } });
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: /stripe/i }));
    fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: stripeExport } });
    fireEvent.click(screen.getByRole("button", { name: /preview import/i }));

    expect(await screen.findByText(/nothing has been imported yet/i)).toBeInTheDocument();
    await waitFor(() => expect(endpoints.stripeImportPreview).toHaveBeenCalledTimes(1));
    fireEvent.click(await screen.findByRole("button", { name: /import 2 items/i }));
    expect(await screen.findByText(/import complete/i)).toBeInTheDocument();
    await waitFor(() => expect(endpoints.stripeImportCommit).toHaveBeenCalledTimes(1));
  });

  it("routes Chargebee to the Chargebee endpoints (not Stripe)", async () => {
    endpoints.chargebeeImportPreview.mockResolvedValue(chargebeePreview);
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: /chargebee/i }));
    fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: chargebeeExport } });
    fireEvent.click(screen.getByRole("button", { name: /preview import/i }));

    expect(await screen.findByText(/nothing has been imported yet/i)).toBeInTheDocument();
    expect(endpoints.chargebeeImportPreview).toHaveBeenCalledTimes(1);
    expect(endpoints.stripeImportPreview).not.toHaveBeenCalled();
  });

  it("rejects invalid JSON before calling the API", async () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /stripe/i }));
    fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: "{not json" } });
    fireEvent.click(screen.getByRole("button", { name: /preview import/i }));
    expect(await screen.findByText(/valid JSON/i)).toBeInTheDocument();
    expect(endpoints.stripeImportPreview).not.toHaveBeenCalled();
  });
});

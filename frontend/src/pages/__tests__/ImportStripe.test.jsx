import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, it, expect, vi, beforeEach } from "vitest";
import ImportStripe from "../ImportStripe";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: {
    stripeImportPreview: vi.fn(),
    stripeImportCommit: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const renderPage = () =>
  render(
    <MemoryRouter>
      <ImportStripe />
    </MemoryRouter>
  );

const validExport = JSON.stringify({
  customers: [{ id: "cus_1", email: "a@b.com" }],
  products: [{ id: "prod_1", name: "Pro" }],
  prices: [{ id: "price_1", product: "prod_1", unit_amount: 4900, currency: "usd", recurring: { interval: "month" } }],
});

const previewResponse = {
  data: {
    items: [
      { kind: "customer", stripe_id: "cus_1", label: "a@b.com", action: "create" },
      { kind: "plan", stripe_id: "price_1", label: "Pro", action: "create", detail: "49.00 USD every 1 month" },
    ],
    summary: { "customer.create": 1, "plan.create": 1 },
    warnings: [],
  },
};

function pasteAndPreview() {
  fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: validExport } });
  fireEvent.click(screen.getByRole("button", { name: /preview import/i }));
}

describe("ImportStripe wizard", () => {
  beforeEach(() => vi.clearAllMocks());

  it("rejects invalid JSON before calling the API", async () => {
    renderPage();
    fireEvent.change(screen.getByLabelText(/paste the json/i), { target: { value: "{not json" } });
    fireEvent.click(screen.getByRole("button", { name: /preview import/i }));
    expect(await screen.findByText(/valid JSON/i)).toBeInTheDocument();
    expect(endpoints.stripeImportPreview).not.toHaveBeenCalled();
  });

  it("previews then commits, showing created counts", async () => {
    endpoints.stripeImportPreview.mockResolvedValue(previewResponse);
    endpoints.stripeImportCommit.mockResolvedValue({ data: { created: { customer: 1, plan: 1 }, failures: [] } });
    renderPage();

    pasteAndPreview();

    // Preview step renders the plan table + an import CTA reflecting 2 creates.
    expect(await screen.findByText(/nothing has been imported yet/i)).toBeInTheDocument();
    const importBtn = await screen.findByRole("button", { name: /import 2 items/i });
    fireEvent.click(importBtn);

    expect(await screen.findByText(/import complete/i)).toBeInTheDocument();
    await waitFor(() => expect(endpoints.stripeImportCommit).toHaveBeenCalledTimes(1));
  });

  it("disables the import button when nothing would be created", async () => {
    endpoints.stripeImportPreview.mockResolvedValue({
      data: {
        items: [{ kind: "customer", stripe_id: "cus_1", label: "a@b.com", action: "skip_already_imported" }],
        summary: { "customer.skip_already_imported": 1 },
        warnings: [],
      },
    });
    renderPage();
    pasteAndPreview();

    expect(await screen.findByRole("button", { name: /nothing to import/i })).toBeDisabled();
  });
});

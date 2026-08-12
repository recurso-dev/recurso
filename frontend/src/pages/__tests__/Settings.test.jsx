import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Settings from "../Settings";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getAccount: vi.fn(),
    updateAccount: vi.fn(),
    getEntities: vi.fn(),
    updateEntity: vi.fn(),
  },
}));

vi.mock("../../components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

const primary = (country) => ({
  data: {
    data: [
      { id: "ent-1", is_primary: true, name: "Acme", legal_name: "Acme Inc", invoice_prefix: "INV", country_code: country },
    ],
  },
});

describe("Settings — General section", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getAccount.mockResolvedValue({ data: { data: { name: "Acme", email: "a@b.co" } } });
    endpoints.getEntities.mockResolvedValue(primary("US"));
  });

  it("renders the company identity form with the business-country control", async () => {
    render(<Settings />, { wrapper });
    expect(await screen.findByText("Business country")).toBeInTheDocument();
    expect(screen.getByLabelText("Company name")).toBeInTheDocument();
    expect(screen.getByLabelText("Support email")).toBeInTheDocument();
    // The section-navigation cards moved to the persistent SettingsLayout nav.
    expect(screen.queryByText(/For your region/)).not.toBeInTheDocument();
  });

  it("confirms before switching the business country (tax regime change)", async () => {
    // jsdom lacks these; Radix Select touches them.
    if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
    if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
    endpoints.updateEntity.mockResolvedValue({ data: { data: {} } });
    render(<Settings />, { wrapper });
    const trigger = await screen.findByRole("combobox");
    fireEvent.click(trigger);
    fireEvent.click(await screen.findByRole("option", { name: /india/i }));
    // Picking no longer auto-saves (audit §7) — a confirm dialog intervenes.
    expect(endpoints.updateEntity).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /change country/i }));
    await waitFor(() =>
      expect(endpoints.updateEntity).toHaveBeenCalledWith(
        "ent-1",
        expect.objectContaining({ country_code: "IN" })
      )
    );
  });

  it("hydrates the account fields from the API", async () => {
    render(<Settings />, { wrapper });
    await screen.findByText("Business country");
    expect(screen.getByLabelText("Company name")).toHaveValue("Acme");
    expect(screen.getByLabelText("Support email")).toHaveValue("a@b.co");
  });
});

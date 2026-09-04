import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import EUEInvoiceSettings from "../EUEInvoiceSettings";
import { endpoints } from "../../../lib/api";
import { toast } from "../../../components/ui/sonner";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getEUEInvoiceConfig: vi.fn(),
    updateEUEInvoiceConfig: vi.fn(),
    getEntities: vi.fn(),
  },
}));
vi.mock("../../../components/ui/sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
  >
    {children}
  </QueryClientProvider>
);

const SAVED = {
  enabled: true,
  legal_name: "Acme GmbH",
  vat_number: "DE123456789",
  country_code: "DE",
  street: "Hauptstr. 1",
  city: "Berlin",
  postal_zone: "10115",
};

describe("EUEInvoiceSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEntities.mockResolvedValue({ data: { data: [] } });
    endpoints.updateEUEInvoiceConfig.mockResolvedValue({ data: { data: {} } });
  });

  it("loads the saved seller identity and submits edits (VAT upper-cased)", async () => {
    endpoints.getEUEInvoiceConfig.mockResolvedValue({ data: { data: SAVED } });
    render(<EUEInvoiceSettings />, { wrapper });
    expect(screen.getByText("EU e-invoicing")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText("Legal name")).toHaveValue("Acme GmbH"));
    expect(screen.getByRole("switch", { name: /enable eu e-invoicing/i })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    // Single-entity tenant: no entity scope selector.
    expect(screen.queryByLabelText("Legal entity")).toBeNull();

    fireEvent.change(screen.getByLabelText("VAT number"), { target: { value: "de987654321" } });
    fireEvent.click(screen.getByRole("button", { name: /save settings/i }));
    await waitFor(() =>
      expect(endpoints.updateEUEInvoiceConfig).toHaveBeenCalledWith(
        { ...SAVED, vat_number: "DE987654321" },
        ""
      )
    );
    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("starts disabled and empty when nothing is configured, and surfaces save errors", async () => {
    endpoints.getEUEInvoiceConfig.mockResolvedValue({ data: { data: null } });
    endpoints.updateEUEInvoiceConfig.mockRejectedValue({
      response: { data: { error: { message: "seller identity incomplete" } } },
    });
    render(<EUEInvoiceSettings />, { wrapper });
    const toggle = await screen.findByRole("switch", { name: /enable eu e-invoicing/i });
    expect(toggle).toHaveAttribute("aria-checked", "false");
    expect(screen.getByLabelText("Legal name")).toHaveValue("");

    fireEvent.click(toggle);
    fireEvent.click(screen.getByRole("button", { name: /save settings/i }));
    await waitFor(() =>
      expect(endpoints.updateEUEInvoiceConfig).toHaveBeenCalledWith(
        expect.objectContaining({ enabled: true, legal_name: "" }),
        ""
      )
    );
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("seller identity incomplete"));
  });
});

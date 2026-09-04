import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import InvoiceBranding from "../InvoiceBranding";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: { getInvoiceBranding: vi.fn(), updateInvoiceBranding: vi.fn() },
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

describe("InvoiceBranding", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.updateInvoiceBranding.mockResolvedValue({ data: { data: {} } });
  });

  it("loads saved branding and saves edits", async () => {
    endpoints.getInvoiceBranding.mockResolvedValue({
      data: {
        data: {
          company_name: "Acme Labs",
          logo_data_url: "data:image/png;base64,iVBORw0KGgo=",
          signatory_name: "J. Doe",
          bank_details: "HDFC 000",
          terms: "Net 30",
        },
      },
    });
    render(<InvoiceBranding />, { wrapper });

    await waitFor(() =>
      expect(screen.getByLabelText("Company name")).toHaveValue("Acme Labs")
    );
    // Saved logo shows the replace/remove controls, not the empty-state upload.
    expect(screen.getByRole("button", { name: /replace/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Signatory name"), {
      target: { value: "New Signer" },
    });
    fireEvent.click(screen.getByRole("button", { name: /save branding/i }));

    await waitFor(() => expect(endpoints.updateInvoiceBranding).toHaveBeenCalled());
    expect(endpoints.updateInvoiceBranding.mock.calls[0][0]).toMatchObject({
      company_name: "Acme Labs",
      signatory_name: "New Signer",
      logo_data_url: "data:image/png;base64,iVBORw0KGgo=",
    });
  });

  it("renders empty defaults with upload buttons when nothing is saved", async () => {
    endpoints.getInvoiceBranding.mockResolvedValue({ data: { data: {} } });
    render(<InvoiceBranding />, { wrapper });

    await waitFor(() =>
      expect(screen.getByLabelText("Company name")).toHaveValue("")
    );
    expect(screen.getAllByRole("button", { name: /upload image/i })).toHaveLength(2);
  });
  it("shows a retryable error instead of a blank saveable form when the fetch fails", async () => {
    endpoints.getInvoiceBranding.mockRejectedValueOnce(new Error("boom"));
    render(<InvoiceBranding />, { wrapper });
    await screen.findByText("Couldn't load invoice branding");
    expect(screen.queryByRole("button", { name: /save/i })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(endpoints.getInvoiceBranding).toHaveBeenCalledTimes(2));
  });
});

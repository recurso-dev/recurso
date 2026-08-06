import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import TaxNexusSettings from "../settings/TaxNexusSettings";
import { endpoints } from "@/lib/api";

vi.mock("@/lib/api", () => ({
  endpoints: {
    getTaxNexus: vi.fn(),
    getTaxNexusStatus: vi.fn(),
    getTaxRegistrations: vi.fn(),
    getTaxLiability: vi.fn(),
    setTaxNexus: vi.fn(),
    setTaxRegistrations: vi.fn(),
    getEntities: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("TaxNexusSettings — clearing nexus is guarded", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getTaxNexus.mockResolvedValue({ data: { data: [] } }); // no declared nexus
    endpoints.getTaxNexusStatus.mockResolvedValue({ data: { data: { states: [] } } });
    endpoints.getTaxRegistrations.mockResolvedValue({ data: { data: [] } });
    endpoints.getTaxLiability.mockResolvedValue({ data: { data: null } });
    endpoints.setTaxNexus.mockResolvedValue({ data: {} });
    endpoints.getEntities.mockResolvedValue({ data: { data: [] } });
  });

  it("confirms before saving an empty list (which clears all nexus)", async () => {
    render(<TaxNexusSettings />, { wrapper });
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /save states/i })).toBeInTheDocument()
    );

    fireEvent.click(screen.getByRole("button", { name: /save states/i }));

    // The clear is gated by a confirm — nothing sent yet.
    expect(await screen.findByText(/clear all declared nexus/i)).toBeInTheDocument();
    expect(endpoints.setTaxNexus).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /clear all nexus/i }));
    await waitFor(() => expect(endpoints.setTaxNexus).toHaveBeenCalledTimes(1));
    // Saved an empty list.
    expect(endpoints.setTaxNexus.mock.calls[0][0]).toEqual([]);
  });
});

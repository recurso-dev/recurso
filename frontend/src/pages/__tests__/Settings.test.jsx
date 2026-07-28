import { render, screen, waitFor } from "@testing-library/react";
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

describe("Settings — region-aware tax setup", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getAccount.mockResolvedValue({ data: { data: { name: "Acme", email: "a@b.co" } } });
  });

  it("badges the US sales-tax setup for a US business", async () => {
    endpoints.getEntities.mockResolvedValue(primary("US"));
    render(<Settings />, { wrapper });

    // The business-country control is present.
    expect(await screen.findByText("Business country")).toBeInTheDocument();

    // The US-region setups (sales-tax nexus + W-9 identity) are badged.
    await waitFor(() => expect(screen.getAllByText("For your region")).toHaveLength(2));
    const badgedTitles = screen
      .getAllByText("For your region")
      .map((b) => b.closest("a").textContent);
    expect(badgedTitles.some((t) => t.includes("US sales-tax nexus"))).toBe(true);
    expect(badgedTitles.some((t) => t.includes("US tax identity"))).toBe(true);
    // GST is present but not badged for a US seller.
    expect(screen.getByText("GST configuration").closest("a")).not.toHaveTextContent("For your region");
  });

  it("badges GST + IRP for an India business", async () => {
    endpoints.getEntities.mockResolvedValue(primary("IN"));
    render(<Settings />, { wrapper });

    await waitFor(() => expect(screen.getAllByText("For your region")).toHaveLength(2));
    const badgedTitles = screen
      .getAllByText("For your region")
      .map((b) => b.closest("a").textContent);
    expect(badgedTitles.some((t) => t.includes("GST configuration"))).toBe(true);
    expect(badgedTitles.some((t) => t.includes("E-invoicing (IRP)"))).toBe(true);
  });

  it("badges nothing when the business country is unset", async () => {
    endpoints.getEntities.mockResolvedValue(primary(""));
    render(<Settings />, { wrapper });

    await screen.findByText("Business country");
    expect(screen.queryByText("For your region")).not.toBeInTheDocument();
  });
});

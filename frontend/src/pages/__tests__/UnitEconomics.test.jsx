import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import UnitEconomics from "../UnitEconomics";
import { endpoints } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: { getUnitEconomics: vi.fn() },
}));

const wrapper = ({ children }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}>
    {children}
  </QueryClientProvider>
);

describe("UnitEconomics page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders ARPA / ARPU / LTV from the API", async () => {
    endpoints.getUnitEconomics.mockResolvedValue({
      data: { data: { reporting_currency: "USD", arpa: 5000, arpu: 3000, has_ltv: true, ltv: 60000 } },
    });
    render(<UnitEconomics />, { wrapper });
    await waitFor(() => expect(screen.getByText("ARPA")).toBeInTheDocument());
    expect(screen.getByText("$50.00")).toBeInTheDocument(); // ARPA
    expect(screen.getByText("$30.00")).toBeInTheDocument(); // ARPU
    expect(screen.getByText("$600.00")).toBeInTheDocument(); // LTV
    // Each metric explains how it's computed.
    expect(screen.getByRole("button", { name: /what does arpa mean/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /what does ltv mean/i })).toBeInTheDocument();
  });

  it("shows a dash for LTV when it can't be computed", async () => {
    endpoints.getUnitEconomics.mockResolvedValue({
      data: { data: { reporting_currency: "USD", arpa: 5000, arpu: 3000, has_ltv: false, ltv: 0 } },
    });
    render(<UnitEconomics />, { wrapper });
    await waitFor(() => expect(screen.getByText("LTV")).toBeInTheDocument());
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

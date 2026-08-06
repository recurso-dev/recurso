import { render, screen } from "@testing-library/react";
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

  it("hydrates the account fields from the API", async () => {
    render(<Settings />, { wrapper });
    await screen.findByText("Business country");
    expect(screen.getByLabelText("Company name")).toHaveValue("Acme");
    expect(screen.getByLabelText("Support email")).toHaveValue("a@b.co");
  });
});

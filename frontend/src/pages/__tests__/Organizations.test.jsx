import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Organizations from "../Organizations";
import { endpoints as api } from "../../lib/api";

vi.mock("../../lib/api", () => ({
  endpoints: {
    getOrganizations: vi.fn(),
    getOrgTenants: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getOrgMRR: vi.fn().mockResolvedValue({ data: { data: {} } }),
    createOrganization: vi.fn(),
    addOrgTenant: vi.fn(),
    removeOrgTenant: vi.fn(),
    deleteOrganization: vi.fn(),
  },
}));
vi.mock("@/components/ui/sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }) => (
  <BrowserRouter>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}
    >
      {children}
    </QueryClientProvider>
  </BrowserRouter>
);

describe("Organizations page", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders organizations", async () => {
    api.getOrganizations.mockResolvedValue({ data: { data: [{ id: "org_1", name: "Acme Group" }] } });
    render(<Organizations />, { wrapper });
    await waitFor(() => expect(screen.getByText("Acme Group")).toBeInTheDocument());
  });

  it("shows the empty state with no organizations", async () => {
    api.getOrganizations.mockResolvedValue({ data: { data: [] } });
    render(<Organizations />, { wrapper });
    await waitFor(() => expect(screen.getByText("No organizations yet")).toBeInTheDocument());
  });
});

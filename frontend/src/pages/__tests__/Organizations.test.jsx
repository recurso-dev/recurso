import { render, screen, waitFor, fireEvent } from "@testing-library/react";
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

  it("rejects a malformed tenant ID inline instead of sending it", async () => {
    api.getOrganizations.mockResolvedValue({ data: { data: [{ id: "org_1", name: "Acme Group" }] } });
    render(<Organizations />, { wrapper });
    await waitFor(() => expect(screen.getByText("Acme Group")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Acme Group"));
    const input = await screen.findByLabelText("Tenant ID");
    // No raw "uuid" placeholder; the field explains where the ID comes from.
    expect(screen.queryByPlaceholderText(/tenant uuid/i)).toBeNull();
    expect(screen.getByText(/Account profile page/i)).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "not-a-uuid" } });
    fireEvent.click(screen.getByRole("button", { name: /^add$/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/doesn't look like a tenant ID/i);
    expect(api.addOrgTenant).not.toHaveBeenCalled();
  });

  it("adds a well-formed tenant ID to the organization", async () => {
    api.getOrganizations.mockResolvedValue({ data: { data: [{ id: "org_1", name: "Acme Group" }] } });
    api.addOrgTenant.mockResolvedValue({ data: {} });
    render(<Organizations />, { wrapper });
    await waitFor(() => expect(screen.getByText("Acme Group")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Acme Group"));
    const input = await screen.findByLabelText("Tenant ID");
    const id = "123e4567-e89b-12d3-a456-426614174000";
    fireEvent.change(input, { target: { value: ` ${id} ` } });
    fireEvent.click(screen.getByRole("button", { name: /^add$/i }));
    await waitFor(() => expect(api.addOrgTenant).toHaveBeenCalledWith("org_1", id));
  });

  it("shows the empty state with no organizations", async () => {
    api.getOrganizations.mockResolvedValue({ data: { data: [] } });
    render(<Organizations />, { wrapper });
    await waitFor(() => expect(screen.getByText("No organizations yet")).toBeInTheDocument());
  });
});

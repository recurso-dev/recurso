import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import EntitiesSettings from "../EntitiesSettings";
import { endpoints } from "../../../lib/api";

vi.mock("../../../lib/api", () => ({
  endpoints: {
    getEntities: vi.fn(),
    createEntity: vi.fn(),
    updateEntity: vi.fn(),
    deleteEntity: vi.fn(),
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

const ENTITIES = [
  { id: "e1", name: "Acme US", legal_name: "Acme Inc", invoice_prefix: "ACME", country_code: "US", tb_ledger_id: 1, is_primary: true },
  { id: "e2", name: "Acme India", legal_name: "Acme India Pvt Ltd", invoice_prefix: "ACME-IN", country_code: "IN", tb_ledger_id: 2, is_primary: false },
];

describe("EntitiesSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.getEntities.mockResolvedValue({ data: { data: ENTITIES } });
    endpoints.createEntity.mockResolvedValue({ data: { data: { id: "e3" } } });
    endpoints.deleteEntity.mockResolvedValue({ data: {} });
  });

  it("lists entities and never offers to delete the primary one", async () => {
    render(<EntitiesSettings />, { wrapper });
    expect(screen.getByText("Legal entities")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Acme US")).toBeInTheDocument());
    expect(screen.getByText("Primary")).toBeInTheDocument();
    expect(screen.getByText("ACME-IN")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Edit entity" })).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "Delete entity" })).toHaveLength(1);
  });

  it("creates an entity from the sheet with the country upper-cased", async () => {
    render(<EntitiesSettings />, { wrapper });
    await waitFor(() => expect(screen.getByText("Acme US")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /add entity/i }));
    fireEvent.change(await screen.findByLabelText(/^Name/), { target: { value: "Acme UK" } });
    fireEvent.change(screen.getByLabelText(/Invoice prefix/), { target: { value: "ACME-UK" } });
    fireEvent.change(screen.getByLabelText(/^Country/), { target: { value: "gb" } });
    fireEvent.click(screen.getByRole("button", { name: /^save$/i }));
    await waitFor(() =>
      expect(endpoints.createEntity).toHaveBeenCalledWith({
        name: "Acme UK",
        legal_name: "",
        invoice_prefix: "ACME-UK",
        country_code: "GB",
      })
    );
  });

  it("deletes a non-primary entity only after confirmation", async () => {
    render(<EntitiesSettings />, { wrapper });
    await waitFor(() => expect(screen.getByText("Acme India")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Delete entity" }));
    expect(await screen.findByText("Delete Acme India?")).toBeInTheDocument();
    expect(endpoints.deleteEntity).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /^delete entity$/i }));
    await waitFor(() => expect(endpoints.deleteEntity).toHaveBeenCalledWith("e2"));
  });

  it("shows an empty state with the primary action when there are no entities", async () => {
    endpoints.getEntities.mockResolvedValue({ data: { data: [] } });
    render(<EntitiesSettings />, { wrapper });
    expect(await screen.findByText("No legal entities yet")).toBeInTheDocument();
    // Both the header and the empty state offer the same primary action.
    expect(screen.getAllByRole("button", { name: /add entity/i }).length).toBeGreaterThan(1);
  });

  it("shows an error state with retry when entities fail to load", async () => {
    endpoints.getEntities.mockRejectedValueOnce(new Error("down"));
    render(<EntitiesSettings />, { wrapper });
    expect(await screen.findByText("Couldn't load entities")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    await waitFor(() => expect(screen.getByText("Acme US")).toBeInTheDocument());
  });
});
